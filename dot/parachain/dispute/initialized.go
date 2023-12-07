package dispute

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/scraping"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainRuntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/gammazero/deque"
	"time"
)

type ImportStatementResult uint

const (
	InvalidImport ImportStatementResult = iota
	ValidImport
)

const ChainImportMaxBatchSize = 6

type Initialized struct {
	keystore              keystore.Keystore
	runtime               parachainRuntime.RuntimeInstance
	HighestSessionSeen    parachainTypes.SessionIndex
	GapsInCache           bool
	SpamSlots             SpamSlots
	Participation         Participation
	Scraper               *scraping.ChainScraper
	ParticipationReceiver <-chan any
	ChainImportBacklog    *deque.Deque[parachainTypes.ScrapedOnChainVotes]
	// TODO: metrics
}

type InitialData struct {
	Participation []ParticipationData
	Votes         []parachainTypes.ScrapedOnChainVotes
	Leaf          *overseer.ActivatedLeaf
}

type MaybeCandidateReceipt struct {
	CandidateReceipt *parachainTypes.CandidateReceipt
	CandidateHash    common.Hash
}

func (m MaybeCandidateReceipt) Hash() (common.Hash, error) {
	if m.CandidateReceipt != nil {
		return m.CandidateReceipt.Hash()
	}

	return m.CandidateHash, nil
}

func (i *Initialized) Run(overseerChannel chan<- any, backend DBBackend, initialData *InitialData) {
	go func() {
		for {
			if err := i.runUntilError(overseerChannel, backend, initialData); err == nil {
				logger.Info("received `Conclude` signal, exiting")
				break
			} else {
				logger.Errorf("error in dispute coordinator, restarting: %v", err)
			}
		}
	}()
}

func (i *Initialized) runUntilError(overseerChannel chan<- any, backend DBBackend, initialData *InitialData) error {
	if initialData != nil {
		for _, p := range initialData.Participation {
			if err := i.Participation.Queue(overseerChannel, p); err != nil {
				return fmt.Errorf("queue participation request: %w", err)
			}
		}

		overlayDB := newOverlayBackend(backend)
		if err := i.ProcessChainImportBacklog(overseerChannel,
			overlayDB,
			initialData.Votes,
			uint64(time.Now().Unix()),
			initialData.Leaf.Hash); err != nil {
			return fmt.Errorf("process chain import backlog: %w", err)
		}

		if !overlayDB.IsEmpty() {
			if err := overlayDB.WriteToDB(); err != nil {
				return fmt.Errorf("write overlay backend to db: %w", err)
			}
		}

		update := overseer.ActiveLeavesUpdate{Activated: initialData.Leaf}
		i.Participation.ProcessActiveLeavesUpdate(update)
	}

	overlayDB := newOverlayBackend(backend)
	for {
		var confirmWrite func() error
		select {
		case msg := <-i.ParticipationReceiver:
			switch message := msg.(type) {
			case overseer.Signal[overseer.Conclude]:
				logger.Tracef("OverseerSignal::Conclude")
				return nil
			case overseer.Signal[overseer.ActiveLeavesUpdate]:
				logger.Tracef("OverseerSignal::ActiveLeavesUpdate")
				if err := i.ProcessActiveLeavesUpdate(overseerChannel,
					overlayDB,
					message.Data,
					uint64(time.Now().Unix())); err != nil {
					return fmt.Errorf("process active leaves update: %w", err)
				}

			case overseer.Signal[overseer.BlockFinalized]:
				logger.Tracef("OverseerSignal::BlockFinalised")
				i.Scraper.ProcessFinalisedBlock(message.Data.Number)

			default:
				var err error
				confirmWrite, err = i.HandleIncoming(overseerChannel,
					overlayDB,
					message,
					uint64(time.Now().Unix()),
				)
				if err != nil {
					return fmt.Errorf("handle incoming: %w", err)
				}
			}
		}

		if !overlayDB.IsEmpty() {
			if err := overlayDB.WriteToDB(); err != nil {
				return fmt.Errorf("write overlay backend to db: %w", err)
			}
		}

		if confirmWrite != nil {
			err := confirmWrite()
			if err != nil {
				return fmt.Errorf("default confirm: %w", err)
			}
		}
	}
}

func (i *Initialized) ProcessActiveLeavesUpdate(
	overseerChannel chan<- any,
	backend *overlayBackend,
	update overseer.ActiveLeavesUpdate,
	now uint64,
) error {
	logger.Tracef("Processing ActiveLeavesUpdate")
	scrappedUpdates, err := i.Scraper.ProcessActiveLeavesUpdate(overseerChannel, update)
	if err != nil {
		return fmt.Errorf("scraper: process active leaves update: %w", err)
	}

	i.Participation.BumpPriority(overseerChannel, scrappedUpdates.IncludedReceipts)
	i.Participation.ProcessActiveLeavesUpdate(update)

	if update.Activated != nil {
		sessionIDx, err := i.runtime.ParachainHostSessionIndexForChild(update.Activated.Hash)
		if err != nil {
			logger.Debugf("failed to update session cache for disputes - can't fetch session index: %v", err)
		} else {
			// If error has occurred during last session caching - fetch the whole window
			// Otherwise - cache only the new sessions
			lowerBound := saturatingSub(uint32(sessionIDx), Window-1)
			if !i.GapsInCache && uint32(i.HighestSessionSeen) > lowerBound {
				lowerBound = uint32(i.HighestSessionSeen + 1)
			}

			// There is a new session. Perform a dummy fetch to cache it.
			for session := lowerBound; session <= uint32(sessionIDx); session++ {
				if _, err := i.runtime.ParachainHostSessionInfo(update.Activated.Hash, parachainTypes.SessionIndex(session)); err != nil {
					logger.Debugf("error caching SessionInfo on ActiveLeaves update. "+
						"Session: %v, Hash: %v, Error: %v",
						session,
						update.Activated.Hash,
						err,
					)
					i.GapsInCache = true
				}
			}

			i.HighestSessionSeen = sessionIDx
			earliestSession := saturatingSub(uint32(sessionIDx), Window-1)
			if err := backend.NoteEarliestSession(parachainTypes.SessionIndex(earliestSession)); err != nil {
				logger.Tracef("error noting earliest session: %w", err)
			}

			i.SpamSlots.PruneOld(parachainTypes.SessionIndex(earliestSession))
		}

		logger.Tracef("will process %v onchain votes", len(scrappedUpdates.OnChainVotes))

		if err := i.ProcessChainImportBacklog(overseerChannel,
			backend,
			scrappedUpdates.OnChainVotes,
			now,
			update.Activated.Hash,
		); err != nil {
			return fmt.Errorf("process chain import backlog: %w", err)
		}
	}

	logger.Tracef("finished processing ActiveLeavesUpdate")
	return nil
}

func (i *Initialized) ProcessChainImportBacklog(
	overseerChannel chan<- any,
	backend *overlayBackend,
	newVotes []parachainTypes.ScrapedOnChainVotes,
	now uint64,
	blockHash common.Hash,
) error {
	chainImportBacklog := deque.New[parachainTypes.ScrapedOnChainVotes]()
	for k := 0; k < i.ChainImportBacklog.Len(); k++ {
		chainImportBacklog.PushBack(i.ChainImportBacklog.At(k))
	}
	for _, newVote := range newVotes {
		chainImportBacklog.PushBack(newVote)
	}

	importRange := minInt(ChainImportMaxBatchSize, chainImportBacklog.Len())
	for k := 0; k < importRange; k++ {
		votes := chainImportBacklog.PopFront()
		if err := i.ProcessOnChainVotes(overseerChannel, backend, votes, now, blockHash); err != nil {
			logger.Errorf("skipping scraping block due to error: %w", err)
		}
	}

	i.ChainImportBacklog = chainImportBacklog
	return nil
}

func (i *Initialized) ProcessOnChainVotes(
	overseerChannel chan<- any,
	backend *overlayBackend,
	votes parachainTypes.ScrapedOnChainVotes,
	now uint64,
	blockHash common.Hash,
) error {
	if len(votes.BackingValidators) == 0 && len(votes.Disputes) == 0 {
		return nil
	}

	// Scraped on-chain backing votes for the candidates with
	// the new active leaf as if we received them via gossip.
	for _, backingValidators := range votes.BackingValidators {
		sessionInfo, err := i.runtime.ParachainHostSessionInfo(
			backingValidators.CandidateReceipt.Descriptor.RelayParent,
			votes.Session,
		)
		if err != nil {
			logger.Warnf("failed to get session info for candidate %s, session %d: %s",
				backingValidators.CandidateReceipt,
				votes.Session,
				err)
			return nil
		}

		candidateHash, err := backingValidators.CandidateReceipt.Hash()
		if err != nil {
			logger.Warnf("hash candidate receipt: %s", err)
			return nil
		}

		logger.Infof("importing backing votes from chain for candidate %s and relay parent %s",
			candidateHash,
			backingValidators.CandidateReceipt.Descriptor.RelayParent)

		var statements []types.Statement
		for _, backers := range backingValidators.BackingValidators {
			if len(sessionInfo.Validators) < int(backers.ValidatorIndex) {
				logger.Errorf("missing validator public key. session: %v, validatorIndex: %v",
					votes.Session,
					backers.ValidatorIndex,
				)
				continue
			}

			validatorPublic := sessionInfo.Validators[backers.ValidatorIndex]
			validatorSignature, err := backers.ValidityAttestation.Signature()
			if err != nil {
				logger.Errorf("get signature: %s", err)
				continue
			}

			compactStatement, err := types.NewCompactStatementFromAttestation(backers.ValidityAttestation,
				candidateHash)
			if err != nil {
				logger.Errorf("get compact statement: %s", err)
				continue
			}
			compactStatementValue, err := compactStatement.Value()
			if err != nil {
				logger.Errorf("get compact statement value: %s", err)
				continue
			}

			validStatementKind := inherents.NewValidDisputeStatementKind()
			switch compactStatementValue.(type) {
			case types.SecondedCompactStatement:
				if err := validStatementKind.Set(
					inherents.BackingSeconded(backingValidators.CandidateReceipt.Descriptor.RelayParent),
				); err != nil {
					logger.Errorf("set valid statement kind: %s", err)
					continue
				}
			case types.ValidCompactStatement:
				if err := validStatementKind.Set(
					inherents.BackingValid(backingValidators.CandidateReceipt.Descriptor.RelayParent),
				); err != nil {
					logger.Errorf("set valid statement kind: %s", err)
					continue
				}
			default:
				logger.Errorf("invalid compact statement value type: %T", compactStatementValue)
				continue
			}

			disputeStatement := inherents.NewDisputeStatement()
			if err := disputeStatement.Set(validStatementKind); err != nil {
				logger.Errorf("set dispute statement: %s", err)
				continue
			}

			validatorID, err := types.GetValidatorID(sessionInfo.Validators, backers.ValidatorIndex)
			if err != nil {
				logger.Errorf("get validator id: %s", err)
				continue
			}

			signedDisputeStatement, err := types.NewCheckedSignedDisputeStatement(disputeStatement,
				candidateHash,
				votes.Session,
				parachainTypes.ValidatorSignature(validatorSignature),
				validatorID,
			)
			if err != nil {
				logger.Errorf("scraped backing votes had invalid signature. "+
					"Candidate: %v, session: %v, validatorPublic: %v, validatorIndex: %v",
					candidateHash,
					votes.Session,
					validatorPublic,
					backers.ValidatorIndex,
				)
				return fmt.Errorf("new checked signed dispute statement: %w", err)
			}

			statements = append(statements, types.Statement{
				SignedDisputeStatement: *signedDisputeStatement,
				ValidatorIndex:         backers.ValidatorIndex,
			})

			// Importantly, handling import statements for backing votes also
			// clears spam slots for any newly backed candidates
			candidateReceipt := MaybeCandidateReceipt{
				CandidateReceipt: &backingValidators.CandidateReceipt,
			}
			if outcome, err := i.HandleImportStatements(overseerChannel,
				backend,
				candidateReceipt,
				votes.Session,
				statements,
				now,
			); err != nil || outcome == InvalidImport {
				logger.Errorf("attempted import of on-chain backing votes failed. session: %v, relayParent: %v",
					votes.Session,
					backingValidators.CandidateReceipt.Descriptor.RelayParent,
				)
			}
		}

		// Import disputes from on-chain, this already went through a vote, so it's assumed
		// as verified. This will only be stored, gossiping it is not necessary.
		for _, dispute := range votes.Disputes {
			logger.Tracef("importing dispute votes from chain for candidate. candidateHash: %v, session: %v",
				dispute.CandidateHash,
				dispute.Session,
			)

			disputeSessionInfo, err := i.runtime.ParachainHostSessionInfo(blockHash, parachainTypes.SessionIndex(dispute.Session))
			if err != nil || disputeSessionInfo == nil {
				logger.Warnf("no session info for disputeSession %d", dispute.Session)
				continue
			}

			var filteredStatements []types.Statement
			for _, statement := range statements {
				if int(statement.ValidatorIndex) >= len(disputeSessionInfo.Validators) {
					logger.Warnf("invalid validator index %d for dispute session %d",
						statement.ValidatorIndex,
						dispute.Session)
					continue
				}

				statement.SignedDisputeStatement.ValidatorPublic = disputeSessionInfo.Validators[statement.ValidatorIndex]
				filteredStatements = append(filteredStatements, statement)
			}

			if len(statements) == 0 {
				logger.Errorf("skipping empty from chain dispute import. session: %v, candidateHash: %v",
					votes.Session,
					candidateHash,
				)
				continue
			}

			candidateReceipt := MaybeCandidateReceipt{
				CandidateHash: candidateHash,
			}
			if outcome, err := i.HandleImportStatements(overseerChannel,
				backend,
				candidateReceipt,
				votes.Session,
				filteredStatements,
				now,
			); err != nil || outcome == InvalidImport {
				logger.Errorf("attempted import of on-chain dispute votes failed. "+
					"session: %v, candidateHash: %v",
					votes.Session,
					candidateHash,
				)
				continue
			}

			logger.Tracef("imported dispute votes from chain for candidate. candidateHash: %v, session: %v",
				candidateHash,
				votes.Session,
			)
		}
	}

	return nil
}

func (i *Initialized) HandleIncoming(
	overseerChannel chan<- any,
	backend *overlayBackend,
	msg any,
	now uint64,
) (func() error, error) {
	switch message := msg.(type) {
	case types.Message[ParticipationStatement]:
		if err := i.Participation.Clear(message.Data.CandidateHash); err != nil {
			return nil, fmt.Errorf("clear participation: %w", err)
		}

		valid, err := message.Data.Outcome.Validity()
		if err != nil {
			logger.Warnf("dispute participation failed. Session: %v, "+
				"candidateHash: %v, Error: %v",
				message.Data.Session,
				message.Data.CandidateHash,
				err,
			)
		} else {
			logger.Tracef(
				"issuing local statement based on participation outcome. Session: %v, "+
					"candidateHash: %v, Valid: %v",
				message.Data.Session,
				message.Data.CandidateHash,
				valid,
			)
			if err := i.IssueLocalStatement(overseerChannel,
				backend,
				message.Data.CandidateHash,
				message.Data.CandidateReceipt,
				message.Data.Session,
				valid,
				uint64(time.Now().Unix())); err != nil {
				return nil, fmt.Errorf("issue local statement: %w", err)
			}
		}
	case types.Message[types.ImportStatements]:
		logger.Tracef("HandleIncoming::ImportStatements")
		candidateReceipt := MaybeCandidateReceipt{
			CandidateReceipt: &message.Data.CandidateReceipt,
		}
		outcome, err := i.HandleImportStatements(overseerChannel,
			backend,
			candidateReceipt,
			message.Data.Session,
			message.Data.Statements,
			now,
		)
		if err != nil {
			return nil, fmt.Errorf("handle import statements: %w", err)
		}

		report := func() error {
			if message.ResponseChannel != nil {
				if err := sendMessage(message.ResponseChannel, outcome); err != nil {
					return fmt.Errorf("confirm import statements: %w", err)
				}
			}

			return nil
		}

		if outcome == InvalidImport {
			return nil, report()
		}

		return report, nil
	case types.Message[types.RecentDisputesMessage]:
		logger.Tracef("HandleIncoming::RecentDisputes")
		recentDisputes, err := backend.GetRecentDisputes()
		if err != nil {
			return nil, fmt.Errorf("get recent disputes: %w", err)
		}

		if err := sendMessage(message.ResponseChannel, recentDisputes); err != nil {
			return nil, fmt.Errorf("send recent disputes: %w", err)
		}
	case types.Message[types.ActiveDisputes]:
		logger.Tracef("HandleIncoming::ActiveDisputes")
		activeDisputes, err := backend.GetActiveDisputes(now)
		if err != nil {
			return nil, fmt.Errorf("get active disputes: %w", err)
		}

		if err := sendMessage(message.ResponseChannel, activeDisputes); err != nil {
			return nil, fmt.Errorf("send active disputes: %w", err)
		}
	case types.Message[types.QueryCandidateVotes]:
		logger.Tracef("HandleIncoming::QueryCandidateVotes")
		var queryOutput []types.QueryCandidateVotesResponse
		for _, query := range message.Data.Queries {
			candidateVotes, err := backend.GetCandidateVotes(query.Session, query.CandidateHash)
			if err != nil {
				logger.Debugf("no candidate votes found for query. session: %v, candidateHash: %v",
					query.Session,
					query.CandidateHash,
				)
				return nil, fmt.Errorf("get candidate votes: %w", err)
			}

			if candidateVotes != nil {
				queryOutput = append(queryOutput, types.QueryCandidateVotesResponse{
					Session:       query.Session,
					CandidateHash: query.CandidateHash,
					Votes:         candidateVotes,
				})
			}
		}

		if err := sendMessage(message.ResponseChannel, queryOutput); err != nil {
			return nil, fmt.Errorf("send candidate votes: %w", err)
		}
	case types.Message[types.IssueLocalStatementMessage]:
		logger.Tracef("HandleIncoming::IssueLocalStatement")
		if err := i.IssueLocalStatement(overseerChannel,
			backend,
			message.Data.CandidateHash,
			message.Data.CandidateReceipt,
			message.Data.Session,
			message.Data.Valid,
			now,
		); err != nil {
			return nil, fmt.Errorf("issue local statement: %w", err)
		}
	case types.Message[types.DetermineUndisputedChainMessage]:
		logger.Tracef("HandleIncoming::DetermineUndisputedChain")
		undisputedChain, err := i.determineUndisputedChain(backend,
			message.Data.Base,
			message.Data.BlockDescriptions,
		)
		resp := types.DetermineUndisputedChainResponse{
			Block: undisputedChain,
			Err:   err,
		}
		if err := sendMessage(message.ResponseChannel, resp); err != nil {
			return nil, fmt.Errorf("send undisputed chain: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown dispute coordinator message type %T", msg)
	}

	return nil, nil
}

func (i *Initialized) HandleImportStatements(
	overseerChannel chan<- any,
	backend *overlayBackend,
	maybeCandidateReceipt MaybeCandidateReceipt,
	session parachainTypes.SessionIndex,
	statements []types.Statement,
	now uint64,
) (ImportStatementResult, error) {
	logger.Tracef("in HandleImportStatements")
	if i.sessionIsAncient(session) {
		return InvalidImport, nil
	}

	candidateHash, err := maybeCandidateReceipt.Hash()
	if err != nil {
		return InvalidImport, fmt.Errorf("hash candidate receipt: %w", err)
	}

	votesInDB, err := backend.GetCandidateVotes(session, candidateHash)
	if err != nil {
		return InvalidImport, fmt.Errorf("get candidate votes: %w", err)
	}

	var relayParent common.Hash
	if maybeCandidateReceipt.CandidateReceipt != nil {
		relayParent = maybeCandidateReceipt.CandidateReceipt.Descriptor.RelayParent
	} else {
		if votesInDB == nil {
			return InvalidImport, fmt.Errorf("cannot obtain relay parent without `CandidateReceipt` available")
		}

		relayParent = votesInDB.CandidateReceipt.Descriptor.RelayParent
	}

	env, err := types.NewCandidateEnvironment(i.keystore, i.runtime, session, relayParent)
	if err != nil {
		return InvalidImport, fmt.Errorf("new candidate environment: %w", err)
	}

	logger.Tracef("number of validators: %v, candidateHash: %v, session: %v",
		len(env.Session.Validators),
		candidateHash,
		session,
	)

	// In case we are not provided with a candidate receipt
	// we operate under the assumption, that a previous vote
	// which included a `CandidateReceipt` was seen.
	// This holds since every block is preceded by the `Backing`-phase.
	//
	// There is one exception: A sufficiently sophisticated attacker could prevent
	// us from seeing the backing votes by withholding arbitrary blocks, and hence we do
	// not have a `CandidateReceipt` available.
	var oldState types.CandidateVoteState
	if votesInDB != nil {
		oldState, err = types.NewCandidateVoteState(*votesInDB, env, now)
		if err != nil {
			return InvalidImport, fmt.Errorf("new candidate vote state: %w", err)
		}
	} else {
		if maybeCandidateReceipt.CandidateReceipt != nil {
			oldState, err = types.NewCandidateVoteStateFromReceipt(*maybeCandidateReceipt.CandidateReceipt)
			if err != nil {
				return InvalidImport, fmt.Errorf("new candidate vote state from receipt: %w", err)
			}
		} else {
			return InvalidImport, fmt.Errorf("cannot import votes without `CandidateReceipt` available")
		}
	}

	logger.Tracef("votes loaded. candidateHash: %v, session: %v",
		candidateHash,
		session,
	)

	var importResult *ImportResultHandler
	intermediateResult, err := NewImportResultFromStatements(env, statements, oldState, now)
	if err != nil {
		return InvalidImport, fmt.Errorf("new import result from statements: %w", err)
	}

	isFreshlyConcluded, err := intermediateResult.IsFreshlyConcluded()
	if err != nil {
		return InvalidImport, fmt.Errorf("is freshly concluded: %w", err)
	}

	if intermediateResult.IsFreshlyDisputed() || isFreshlyConcluded {
		logger.Tracef("requesting approval signatures. candidateHash: %v, session: %v",
			candidateHash,
			session,
		)

		message := overseer.ApprovalVotingMessage[overseer.ApprovalSignatureForCandidate]{
			Message: overseer.ApprovalSignatureForCandidate{
				CandidateHash: candidateHash,
			},
			ResponseChan: make(chan any),
		}

		// TODO: we need to send this to a prioritised channel
		res, err := call(overseerChannel, message, message.ResponseChan)
		if err != nil {
			logger.Warnf("failed to fetch approval signatures for candidate %s: %s",
				candidateHash,
				err,
			)
		} else {
			response, ok := res.(overseer.ApprovalSignatureResponse)
			if !ok {
				return InvalidImport, fmt.Errorf("invalid approval signature response")
			}

			if response.Error != nil {
				return InvalidImport, fmt.Errorf("approval signature response: %w", response.Error)
			}

			result, err := intermediateResult.ImportApprovalVotes(i.keystore, response.Signature, env, now)
			if err != nil {
				return InvalidImport, fmt.Errorf("import approval votes: %w", err)
			}

			intermediateResult, ok = result.(*ImportResultHandler)
			if !ok {
				return InvalidImport, fmt.Errorf("invalid import result")
			}
		}
	} else {
		logger.Tracef("not requested approval signatures. candidateHash: %v, session: %v",
			candidateHash,
			session,
		)
	}
	importResult = intermediateResult

	logger.Tracef("import result ready. candidateHash: %v, session: %v",
		candidateHash,
		session,
	)

	newState := importResult.newState
	isIncluded := i.Scraper.IsCandidateIncluded(candidateHash)
	ownVoteMissing := newState.Own.VoteMissing()
	isDisputed := newState.IsDisputed()
	isConfirmed, err := newState.IsConfirmed()
	if err != nil {
		return InvalidImport, fmt.Errorf("is confirmed: %w", err)
	}
	potentialSpam, err := i.Scraper.IsPotentialSpam(newState, candidateHash)
	if err != nil {
		return InvalidImport, fmt.Errorf("is potential spam: %w", err)
	}
	allowParticipation := !potentialSpam
	logger.Tracef("ownVoteMissing: %v, potentialSpam: %v, isIncluded: %v, "+
		"candidateHash: %v, confirmed: %v, hasInvalidVoters: %v",
		ownVoteMissing,
		potentialSpam,
		isIncluded,
		candidateHash,
		isConfirmed,
		len(importResult.newInvalidVoters) == 0,
	)

	// This check is responsible for all clearing of spam slots. It runs
	// whenever a vote is imported from on or off chain, and decrements
	// slots whenever a candidate is newly backed, confirmed, or has our
	// own vote.
	if !potentialSpam {
		i.SpamSlots.Clear(session, candidateHash)
	} else if len(importResult.newInvalidVoters) > 0 {
		freeSpamSlotsAvailable := false
		// Only allow import if at least one validator voting invalid, has not exceeded
		// its spam slots:
		for _, index := range importResult.newInvalidVoters {
			// Disputes can only be triggered via an invalidity stating vote, thus we only
			// need to increase spam slots on invalid votes. (If we did not, we would also
			// increase spam slots for backing validators for example - as validators have to
			// provide some opposing vote for dispute-distribution).
			freeSpamSlotsAvailable = freeSpamSlotsAvailable ||
				i.SpamSlots.AddUnconfirmed(session, candidateHash, index)
		}

		if !freeSpamSlotsAvailable {
			logger.Debugf("rejecting import because of full spam slots. "+
				"candidateHash: %v, session: %v, invalidVoters: %v",
				candidateHash,
				session,
				importResult.newInvalidVoters,
			)
			return InvalidImport, nil
		}
	}

	// Participate in dispute if we did not cast a vote before and actually have keys to cast a
	// local vote. Disputes should fall in one of the categories below, otherwise we will refrain
	// from participation:
	// - `isIncluded` lands in prioritised queue
	// - `isConfirmed` | `isBacked` lands in the best effort queue
	// We don't participate in disputes on finalized candidates.
	if ownVoteMissing && isDisputed && allowParticipation {
		priority := ParticipationPriorityBestEffort
		if isIncluded {
			priority = ParticipationPriorityHigh
		}
		logger.Tracef("queuing participation for candidate: %v, session: %v, priority: %v",
			candidateHash,
			session,
			priority,
		)
		// TODO: metrics
		participationData := ParticipationData{
			ParticipationRequest{
				candidateHash:    candidateHash,
				candidateReceipt: newState.Votes.CandidateReceipt,
				session:          session,
			},
			priority,
		}
		if err := i.Participation.Queue(overseerChannel, participationData); err != nil {
			logger.Errorf("failed to queue participation request: %s", err)
		}
	} else {
		logger.Tracef("will not queue participation for candidate: %v, "+
			"session: %v, ownVoteMissing: %v, isDisputed: %v, allowParticipation: %v",
			candidateHash,
			session,
			ownVoteMissing,
			isDisputed,
			allowParticipation,
		)
		// TODO: metrics
	}

	// Also send any already existing approval vote on new disputes
	if importResult.IsFreshlyDisputed() {
		ourApprovalVotes, err := newState.Own.ApprovalVotes()
		if err != nil {
			return InvalidImport, fmt.Errorf("own approval votes: %w", err)
		}

		for _, vote := range ourApprovalVotes {
			if int(vote.ValidatorIndex) >= len(env.Session.Validators) {
				logger.Errorf("missing validator public key. session: %v, validatorIndex: %v",
					session,
					vote.ValidatorIndex,
				)
				continue
			}

			keypair, err := types.GetValidatorKeyPair(i.keystore, env.Session.Validators, vote.ValidatorIndex)
			if err != nil {
				logger.Warnf("missing validator keypair for validator index %d: %s",
					vote.ValidatorIndex,
					err,
				)
				continue
			}

			statement, err := types.NewSignedDisputeStatement(keypair, true, candidateHash, session)
			if err != nil {
				logger.Warnf("failed to construct dispute statement for validator index %d: %s",
					vote.ValidatorIndex,
					err,
				)
				continue
			}

			logger.Tracef("sending out own approval vote. candidateHash: %v, session: %v, validatorIndex: %v",
				candidateHash,
				session,
				vote.ValidatorIndex,
			)

			disputeMessage, err := types.NewDisputeMessage(keypair,
				newState.Votes,
				&statement,
				vote.ValidatorIndex,
				env.Session,
			)
			if err != nil {
				return InvalidImport, fmt.Errorf("new dispute message: %w", err)
			}

			if err := sendMessage(overseerChannel, disputeMessage); err != nil {
				return InvalidImport, fmt.Errorf("send dispute message: %w", err)
			}
		}
	}

	// All good, update recent disputes if state has changed
	if newState.DisputeStatus != nil {
		stateChanged, err := importResult.DisputeStateChanged()
		if err != nil {
			return InvalidImport, fmt.Errorf("dispute state changed: %w", err)
		}

		if stateChanged {
			recentDisputes, err := backend.GetRecentDisputes()
			if err != nil {
				return InvalidImport, fmt.Errorf("get recent disputes: %w", err)
			}

			existingDispute := false
			recentDisputes.Ascend(nil, func(i interface{}) bool {
				dispute, ok := i.(*types.Dispute)
				if !ok {
					return true
				}

				if dispute.Comparator.CandidateHash == candidateHash && dispute.Comparator.SessionIndex == session {
					existingDispute = true
					dispute.DisputeStatus = *newState.DisputeStatus
					return false
				}

				return true
			})

			if !existingDispute {
				dispute, err := types.NewDispute()
				if err != nil {
					return InvalidImport, fmt.Errorf("new dispute: %w", err)
				}

				dispute.Comparator.CandidateHash = candidateHash
				dispute.Comparator.SessionIndex = session
				dispute.DisputeStatus = *newState.DisputeStatus
				recentDisputes.Set(dispute)
			}

			if err := backend.SetRecentDisputes(recentDisputes); err != nil {
				return InvalidImport, fmt.Errorf("set recent disputes: %w", err)
			}
		}
	}

	// Notify ChainSelection if a dispute has concluded against a candidate. ChainSelection
	// will need to mark the candidate's relay parent as reverted.
	isFreshlyConcludedAgainst, err := importResult.IsFreshlyConcludedAgainst()
	if err != nil {
		return InvalidImport, fmt.Errorf("is freshly concluded against: %w", err)
	}
	if isFreshlyConcludedAgainst {
		inclusions := i.Scraper.GetBlocksIncludingCandidate(candidateHash)
		blocks := make([]overseer.Block, len(inclusions))
		for _, inclusion := range inclusions {
			logger.Tracef("dispute has just concluded against the candidate hash noted."+
				"Its parent will be marked as reverted. candidateHash: %v, parentBlockNumber: %v, parentBlockHash: %v",
				candidateHash,
				inclusion.BlockNumber,
				inclusion.BlockHash,
			)
			blocks = append(blocks, overseer.Block{
				Number: inclusion.BlockNumber,
				Hash:   inclusion.BlockHash,
			})
		}

		if len(blocks) > 0 {
			message := overseer.ChainSelectionMessage[overseer.RevertBlocks]{
				Message: overseer.RevertBlocks{Blocks: blocks},
			}
			if err := sendMessage(overseerChannel, message); err != nil {
				return InvalidImport, fmt.Errorf("send revert blocks request: %w", err)
			}
		} else {
			logger.Debugf("could not find an including block for candidate against which"+
				"a dispute has concluded candidateHash: %v, session: %v",
				candidateHash,
				session,
			)
		}
	}

	// TODO: update metrics

	// Only write when votes have changed.
	votes := importResult.IntoUpdatedVotes()
	if votes != nil {
		if err := backend.SetCandidateVotes(session, candidateHash, &newState.Votes); err != nil {
			return InvalidImport, fmt.Errorf("set candidate votes: %w", err)
		}
	}

	return ValidImport, nil
}

func (i *Initialized) IssueLocalStatement(
	overseerChannel chan<- any,
	backend *overlayBackend,
	candidateHash common.Hash,
	candidateReceipt parachainTypes.CandidateReceipt,
	session parachainTypes.SessionIndex,
	valid bool,
	now uint64,
) error {
	logger.Tracef("issuing local statement for candidate %s", candidateHash)

	env, err := types.NewCandidateEnvironment(i.keystore, i.runtime, session, candidateReceipt.Descriptor.RelayParent)
	if err != nil {
		logger.Warnf("missing session info for candidate %s, session %d: %s",
			candidateHash,
			session,
		)
		return fmt.Errorf("new candidate environment: %w", err)
	}

	votes, err := backend.GetCandidateVotes(session, candidateHash)
	if err != nil {
		return fmt.Errorf("get candidate votes: %w", err)
	}
	if votes == nil {
		candidateVotes := types.NewCandidateVotesFromReceipt(candidateReceipt)
		votes = &candidateVotes
	}

	// Sign a statement for each validator index we control which has
	// not already voted. This should generally be maximum 1 statement.
	votedIndices := votes.VotedIndices()
	var statements []types.Statement

	for index := range env.ControlledIndices {
		if votedIndices.Contains(index) {
			continue
		}

		keypair, err := types.GetValidatorKeyPair(i.keystore, env.Session.Validators, index)
		if err != nil {
			return fmt.Errorf("get validator key pair: %w", err)
		}

		signedDisputeStatement, err := types.NewSignedDisputeStatement(keypair, valid, candidateHash, session)
		if err != nil {
			return fmt.Errorf("new signed dispute statement: %w", err)
		}

		statements = append(statements, types.Statement{
			SignedDisputeStatement: signedDisputeStatement,
			ValidatorIndex:         index,
		})
	}

	// Get the message out
	for _, statement := range statements {
		keypair, err := types.GetValidatorKeyPair(i.keystore, env.Session.Validators, statement.ValidatorIndex)
		if err != nil {
			logger.Warnf("missing validator keypair for validator index %d: %s",
				statement.ValidatorIndex,
				err,
			)
			continue
		}

		disputeMessage, err := types.NewDisputeMessage(keypair,
			*votes,
			&statement.SignedDisputeStatement,
			statement.ValidatorIndex,
			env.Session,
		)
		if err != nil {
			logger.Warnf("failed to construct dispute message for validator index %d: %s",
				statement.ValidatorIndex,
				err,
			)
			continue
		}

		if err := sendMessage(overseerChannel, disputeMessage); err != nil {
			logger.Warnf("failed to send dispute message for validator index %d: %s",
				statement.ValidatorIndex,
				err,
			)
			continue
		}
	}

	// Do import
	if len(statements) > 0 {
		if outcome, err := i.HandleImportStatements(overseerChannel,
			backend,
			MaybeCandidateReceipt{
				CandidateReceipt: &candidateReceipt,
			},
			session,
			statements,
			now,
		); err != nil || outcome == InvalidImport {
			logger.Errorf("attempted import of our own votes failed. session: %v, candidateHash: %v",
				session,
				candidateHash,
			)
		} else {
			logger.Tracef("successfully imported our own votes. session: %v, candidateHash: %v",
				session,
				candidateHash,
			)
		}
	}

	return nil
}

func (i *Initialized) sessionIsAncient(session parachainTypes.SessionIndex) bool {
	return uint32(session) < saturatingSub(uint32(i.HighestSessionSeen), Window-1)
}

func (i *Initialized) determineUndisputedChain(backend OverlayBackend,
	baseBlock overseer.Block,
	blockDescriptions []types.BlockDescription,
) (overseer.Block, error) {
	if len(blockDescriptions) == 0 {
		return baseBlock, nil
	}

	e := blockDescriptions[len(blockDescriptions)-1]
	last := overseer.NewBlock(baseBlock.Number+uint32(len(blockDescriptions)),
		e.BlockHash,
	)

	recentDisputes, err := backend.GetRecentDisputes()
	if err != nil {
		return overseer.Block{}, fmt.Errorf("get recent disputes: %w", err)
	}

	if recentDisputes.Len() == 0 {
		return last, nil
	}

	isPossiblyInvalid := func(session parachainTypes.SessionIndex, candidateHash common.Hash) bool {
		isPossiblyInvalid := false
		// TODO: replace btree with btreemap
		recentDisputes.Ascend(nil, func(i interface{}) bool {
			dispute, ok := i.(*types.Dispute)
			if !ok {
				return true
			}

			if dispute.Comparator.SessionIndex != session && dispute.Comparator.CandidateHash != candidateHash {
				return true
			}

			isPossiblyInvalid, err = dispute.DisputeStatus.IsPossiblyInvalid()
			if err != nil {
				logger.Errorf("is possibly invalid: %s", err)
				return true
			}

			return false
		})

		return isPossiblyInvalid
	}

	for i, blockDescription := range blockDescriptions {
		for _, candidate := range blockDescription.Candidates {
			if isPossiblyInvalid(blockDescription.Session, candidate.Value) {
				if i == 0 {
					return baseBlock, nil
				} else {
					return overseer.NewBlock(baseBlock.Number+uint32(i),
						blockDescriptions[i-1].BlockHash,
					), nil
				}
			}
		}
	}

	return last, nil
}

// NewInitializedState creates a new initialized state.
func NewInitializedState(overseerChannel chan<- any,
	receiver chan any,
	runtime parachainRuntime.RuntimeInstance,
	spamSlots SpamSlots,
	scraper *scraping.ChainScraper,
	highestSessionSeen parachainTypes.SessionIndex,
	gapsInCache bool,
	keystore keystore.Keystore,
) *Initialized {
	return &Initialized{
		runtime:               runtime,
		SpamSlots:             spamSlots,
		Scraper:               scraper,
		HighestSessionSeen:    highestSessionSeen,
		GapsInCache:           gapsInCache,
		ParticipationReceiver: receiver,
		ChainImportBacklog:    deque.New[parachainTypes.ScrapedOnChainVotes](),
		Participation:         NewParticipation(overseerChannel, receiver, runtime),
		keystore:              keystore,
	}
}

// saturatingSub returns the result of a - b, saturating at 0.
func saturatingSub(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return 0
}

// minInt returns the smallest of a or b.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
