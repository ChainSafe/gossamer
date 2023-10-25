package dispute

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/scraping"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainRuntime "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/babe/inherents"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
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
	ParticipationReceiver chan MuxedMessage
	ChainImportBacklog    *deque.Deque[parachainTypes.ScrapedOnChainVotes]
	// TODO: metrics
}

type InitialData struct {
	Participation []ParticipationRequestWithPriority
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

func (i *Initialized) Run(context overseer.Context, backend DBBackend, initialData *InitialData) error {
	for {
		if err := i.runUntilError(context, backend, initialData); err == nil {
			logger.Info("received `Conclude` signal, exiting")
			return nil
		} else {
			logger.Errorf("error in dispute coordinator, restarting", "error", err)
		}
	}
}

func (i *Initialized) runUntilError(context overseer.Context, backend DBBackend, initialData *InitialData) error {
	if initialData != nil {
		for _, p := range initialData.Participation {
			if err := i.Participation.Queue(context, p.request, p.priority); err != nil {
				return fmt.Errorf("queue participation request: %w", err)
			}
		}

		overlayDB := newOverlayBackend(backend)
		if err := i.ProcessChainImportBacklog(context,
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

	for {
		overlayDB := newOverlayBackend(backend)
		confirmWrite := func() error { return nil }
		select {
		case message := <-i.ParticipationReceiver:
			if message.Participation != nil {
				if err := i.Participation.Clear(message.Participation.CandidateHash); err != nil {
					return fmt.Errorf("clear participation: %w", err)
				}

				valid, err := message.Participation.Outcome.Validity()
				if err != nil {
					logger.Warnf("dispute participation failed. Session: %v, "+
						"candidateHash: %v, Error: %v",
						message.Participation.Session,
						message.Participation.CandidateHash,
						err)
				} else {
					logger.Tracef(
						"issuing local statement based on participation outcome. Session: %v, "+
							"candidateHash: %v, Valid: %v",
						message.Participation.Session,
						message.Participation.CandidateHash,
						valid,
					)
					if err := i.IssueLocalStatement(context,
						overlayDB,
						message.Participation.CandidateHash,
						message.Participation.CandidateReceipt,
						message.Participation.Session,
						valid,
						uint64(time.Now().Unix())); err != nil {
						return fmt.Errorf("issue local statement: %w", err)
					}
				}
			} else if message.Signal != nil {
				switch {
				case message.Signal.Conclude == true:
					return nil
				case message.Signal.ActiveLeaves != nil:
					logger.Tracef("OverseerSignal::ActiveLeavesUpdate")
					if err := i.ProcessActiveLeavesUpdate(context,
						overlayDB,
						*message.Signal.ActiveLeaves,
						uint64(time.Now().Unix())); err != nil {
						return fmt.Errorf("process active leaves update: %w", err)
					}
				case message.Signal.BlockFinalised != nil:
					logger.Tracef("OverseerSignal::BlockFinalised")
					i.Scraper.ProcessFinalisedBlock(message.Signal.BlockFinalised.Number)

				default:
					logger.Errorf("OverseerSignal::Unknown")
				}
			} else if message.Communication != nil {
				var err error
				confirmWrite, err = i.HandleIncoming(context,
					overlayDB,
					*message.Communication,
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

		err := confirmWrite()
		if err != nil {
			return fmt.Errorf("default confirm: %w", err)
		}
	}
}

func (i *Initialized) ProcessActiveLeavesUpdate(
	context overseer.Context,
	backend OverlayBackend,
	update overseer.ActiveLeavesUpdate,
	now uint64,
) error {
	logger.Tracef("Processing ActiveLeavesUpdate")
	scrappedUpdates, err := i.Scraper.ProcessActiveLeavesUpdate(context.Sender, update)
	if err != nil {
		return fmt.Errorf("scraper: process active leaves update: %w", err)
	}

	i.Participation.BumpPriority(context, scrappedUpdates.IncludedReceipts)
	i.Participation.ProcessActiveLeavesUpdate(update)

	if update.Activated != nil {
		sessionIDx, err := i.runtime.ParachainHostSessionIndexForChild(update.Activated.Hash)
		if err != nil {
			logger.Debugf("failed to update session cache for disputes - can't fetch session index: %v", err)
		} else {
			// If error has occurred during last session caching - fetch the whole window
			// Otherwise - cache only the new sessions
			var lowerBound parachainTypes.SessionIndex
			if i.GapsInCache {
				lowerBound = sessionIDx - (DisputeWindow - 1)
				if sessionIDx < lowerBound {
					lowerBound = sessionIDx
				}
			} else {
				lowerBound = i.HighestSessionSeen + 1
			}

			// There is a new session. Perform a dummy fetch to cache it.
			for session := lowerBound; session <= sessionIDx; session++ {
				if _, err := i.runtime.ParachainHostSessionInfo(update.Activated.Hash, session); err != nil {
					logger.Debugf("error caching SessionInfo on ActiveLeaves update. "+
						"Session: %v, Hash: %v, Error: %v",
						session,
						update.Activated.Hash,
						err)
					i.GapsInCache = true
				}
			}

			i.HighestSessionSeen = sessionIDx

			earliestSession := saturatingSub(uint32(sessionIDx), DisputeWindow-1)
			if err := backend.NoteEarliestSession(parachainTypes.SessionIndex(earliestSession)); err != nil {
				logger.Tracef("error noting earliest session: %w", err)
			}

			i.SpamSlots.PruneOld(parachainTypes.SessionIndex(earliestSession))
		}

		logger.Tracef("will process %v onchain votes", len(scrappedUpdates.OnChainVotes))

		if err := i.ProcessChainImportBacklog(context,
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
	context overseer.Context,
	backend OverlayBackend,
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
		if err := i.ProcessOnChainVotes(context, backend, votes, now, blockHash); err != nil {
			logger.Errorf("skipping scraping block due to error: %w", err)
		}
	}

	i.ChainImportBacklog = chainImportBacklog
	return nil
}

func (i *Initialized) ProcessOnChainVotes(
	context overseer.Context,
	backend OverlayBackend,
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
			}

			disputeStatement := inherents.NewDisputeStatement()
			if err := disputeStatement.Set(inherents.ValidDisputeStatementKind(validStatementKind)); err != nil {
				logger.Errorf("set dispute statement: %s", err)
				continue
			}

			signedDisputeStatement, err := types.NewCheckedSignedDisputeStatement(disputeStatement,
				candidateHash,
				votes.Session,
				validatorPublic,
				parachain.ValidatorSignature(validatorSignature),
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
			if outcome, err := i.HandleImportStatements(context,
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
			if outcome, err := i.HandleImportStatements(context,
				backend,
				candidateReceipt,
				votes.Session,
				statements,
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
	context overseer.Context,
	backend OverlayBackend,
	message types.DisputeCoordinatorMessage,
	now uint64,
) (func() error, error) {
	switch {
	case message.ImportStatements != nil:
		logger.Tracef("HandleIncoming::ImportStatements")
		candidateReceipt := MaybeCandidateReceipt{
			CandidateReceipt: &message.ImportStatements.CandidateReceipt,
		}
		outcome, err := i.HandleImportStatements(context,
			backend,
			candidateReceipt,
			message.ImportStatements.Session,
			message.ImportStatements.Statements,
			now,
		)
		if err != nil {
			return nil, fmt.Errorf("handle import statements: %w", err)
		}

		report := func() error {
			if message.ImportStatements.PendingConfirmation != nil {
				if err := message.ImportStatements.PendingConfirmation.SendMessage(outcome); err != nil {
					return fmt.Errorf("confirm import statements: %w", err)
				}
			}

			return nil
		}

		if outcome == InvalidImport {
			return nil, report()
		}

		return report, nil
	case message.RecentDisputes != nil:
		logger.Tracef("HandleIncoming::RecentDisputes")
		recentDisputes, err := backend.GetRecentDisputes()
		if err != nil {
			return nil, fmt.Errorf("get recent disputes: %w", err)
		}

		if err := message.RecentDisputes.Sender.SendMessage(recentDisputes); err != nil {
			return nil, fmt.Errorf("send recent disputes: %w", err)
		}
	case message.ActiveDisputes != nil:
		logger.Tracef("HandleIncoming::ActiveDisputes")
		activeDisputes, err := backend.GetActiveDisputes(now)
		if err != nil {
			return nil, fmt.Errorf("get active disputes: %w", err)
		}

		if err := message.ActiveDisputes.Sender.SendMessage(activeDisputes); err != nil {
			return nil, fmt.Errorf("send active disputes: %w", err)
		}
	case message.QueryCandidateVotes != nil:
		logger.Tracef("HandleIncoming::QueryCandidateVotes")

		var queryOutput []types.QueryCandidateVotesResponse
		for _, query := range message.QueryCandidateVotes.Queries {
			candidateVotes, err := backend.GetCandidateVotes(query.Session, query.CandidateHash)
			if err != nil {
				logger.Debugf("no candidate votes found for query. session: %v, candidateHash: %v",
					query.Session,
					query.CandidateHash,
				)
				return nil, fmt.Errorf("get candidate votes: %w", err)
			}

			queryOutput = append(queryOutput, types.QueryCandidateVotesResponse{
				Session:       query.Session,
				CandidateHash: query.CandidateHash,
				Votes:         *candidateVotes,
			})
		}

		if err := message.QueryCandidateVotes.Sender.SendMessage(queryOutput); err != nil {
			return nil, fmt.Errorf("send candidate votes: %w", err)
		}
	case message.IssueLocalStatement != nil:
		logger.Tracef("HandleIncoming::IssueLocalStatement")
		if err := i.IssueLocalStatement(context,
			backend,
			message.IssueLocalStatement.CandidateHash,
			message.IssueLocalStatement.CandidateReceipt,
			message.IssueLocalStatement.Session,
			message.IssueLocalStatement.Valid,
			now,
		); err != nil {
			return nil, fmt.Errorf("issue local statement: %w", err)
		}
	case message.DetermineUndisputedChain != nil:
		logger.Tracef("HandleIncoming::DetermineUndisputedChain")
		undisputedChain, err := i.determineUndisputedChain(backend,
			message.DetermineUndisputedChain.Base,
			message.DetermineUndisputedChain.BlockDescriptions,
		)
		if err != nil {
			return nil, fmt.Errorf("determine undisputed chain: %w", err)
		}

		if err := message.DetermineUndisputedChain.Tx.SendMessage(undisputedChain); err != nil {
			return nil, fmt.Errorf("send undisputed chain: %w", err)
		}
	default:
		return nil, fmt.Errorf("unknown dispute coordinator message")
	}

	return nil, nil
}

func (i *Initialized) HandleImportStatements(
	context overseer.Context,
	backend OverlayBackend,
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
		oldState, err = types.NewCandidateVoteState(*votesInDB, now)
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

		// Use of unbounded channels justified because:
		// 1. Only triggered twice per dispute.
		// 2. Raising a dispute is costly (requires validation + recovery) by honest nodes,
		// dishonest nodes are limited by spam slots.
		// 3. Concluding a dispute is even more costly.
		// Therefore, it is reasonable to expect a simple vote request to succeed way faster
		// than disputes are raised.
		// 4. We are waiting (and blocking the whole subsystem) on a response right after -
		// therefore even with all else failing we will never have more than
		// one message in flight at any given time.
		responseChan := make(chan *overseer.ApprovalSignatureResponse, 1)
		message := overseer.ApprovalVotingMessage{
			GetApprovalSignature: &overseer.GetApprovalSignatureForCandidate{
				CandidateHash: candidateHash,
				ResponseChan:  responseChan,
			},
		}
		if err := context.Sender.SendUnboundedMessage(message); err != nil {
			logger.Warnf("failed to fetch approval signatures for candidate %s: %s",
				candidateHash,
				err,
			)
			importResult = intermediateResult
		} else {
			response := <-responseChan
			if response.Error != nil {
				return InvalidImport, fmt.Errorf("approval signature response: %w", response.Error)
			}

			result, err := intermediateResult.ImportApprovalVotes(response.Signature, now)
			if err != nil {
				return InvalidImport, fmt.Errorf("import approval votes: %w", err)
			}

			var ok bool
			importResult, ok = result.(*ImportResultHandler)
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
		participationRequest := ParticipationRequest{
			candidateHash:    candidateHash,
			candidateReceipt: newState.Votes.CandidateReceipt,
			session:          session,
		}
		if err := i.Participation.Queue(context, participationRequest, priority); err != nil {
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

			validatorPublic := env.Session.Validators[vote.ValidatorIndex]
			keypair, err := getValidatorKeyPair(validatorPublic, i.keystore)
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

			if err := context.Sender.SendMessage(disputeMessage); err != nil {
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

			dispute, err := types.NewDispute()
			if err != nil {
				return InvalidImport, fmt.Errorf("new dispute: %w", err)
			}

			dispute.Comparator.CandidateHash = candidateHash
			dispute.Comparator.SessionIndex = session
			dispute.DisputeStatus = *newState.DisputeStatus
			if existing := recentDisputes.Value.Get(dispute); existing == nil {
				activeStatus, err := types.NewDisputeStatusVDT()
				if err != nil {
					return InvalidImport, fmt.Errorf("new dispute status: %w", err)
				}
				if err := activeStatus.Set(types.ActiveStatus{}); err != nil {
					return InvalidImport, fmt.Errorf("set active status: %w", err)
				}
				dispute.DisputeStatus = activeStatus
				recentDisputes.Value.Set(dispute)
				logger.Infof("new dispute initiated for candidate %s, session %d",
					candidateHash,
					session,
				)
			}

			logger.Tracef("writing recent disputes with updates for candidate. "+
				"candidateHash: %v, session: %v, status:%v",
				candidateHash,
				session,
				dispute.DisputeStatus,
			)

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
			message := overseer.ChainSelectionMessage{
				RevertBlocks: &overseer.RevertBlocksRequest{Blocks: blocks},
			}
			if err := context.Sender.SendMessage(message); err != nil {
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
	if importResult.VotesChanged() {
		if err := backend.SetCandidateVotes(session, candidateHash, &newState.Votes); err != nil {
			return InvalidImport, fmt.Errorf("set candidate votes: %w", err)
		}
	}

	return ValidImport, nil
}

func (i *Initialized) IssueLocalStatement(
	context overseer.Context,
	backend OverlayBackend,
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

		if int(index) > len(env.Session.Validators) {
			return fmt.Errorf("missing validator public key. session: %v, validatorIndex: %v",
				session,
				index,
			)
		}

		validatorPublic := env.Session.Validators[index]
		keypair, err := getValidatorKeyPair(validatorPublic, i.keystore)
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
		keypair, err := getValidatorKeyPair(env.Session.Validators[statement.ValidatorIndex], i.keystore)
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

		if err := context.Sender.SendMessage(disputeMessage); err != nil {
			logger.Warnf("failed to send dispute message for validator index %d: %s",
				statement.ValidatorIndex,
				err,
			)
			continue
		}
	}

	// Do import
	if len(statements) > 0 {
		if outcome, err := i.HandleImportStatements(context,
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
	diff := session - (DisputeWindow - 1)
	return session < diff || session < i.HighestSessionSeen
}

func (i *Initialized) determineUndisputedChain(backend OverlayBackend,
	baseBlock overseer.Block,
	blockDescriptions []types.BlockDescription,
) (overseer.Block, error) {
	last := overseer.NewBlock(baseBlock.Number+uint32(len(blockDescriptions)),
		blockDescriptions[len(blockDescriptions)-1].BlockHash,
	)

	recentDisputes, err := backend.GetRecentDisputes()
	if err != nil {
		return overseer.Block{}, fmt.Errorf("get recent disputes: %w", err)
	}

	if recentDisputes.Value.Len() == 0 {
		return last, nil
	}

	isPossiblyInvalid := func(session parachainTypes.SessionIndex, candidateHash common.Hash) bool {
		disputeStatus := recentDisputes.Value.Get(types.NewDisputeComparator(session, candidateHash))
		status, ok := disputeStatus.(types.DisputeStatusVDT)
		if !ok {
			logger.Errorf("cast to dispute status. Expected types.DisputeStatusVDT, got %T", disputeStatus)
			return false
		}

		isPossiblyInvalid, err := status.IsPossiblyInvalid()
		if err != nil {
			logger.Errorf("is possibly invalid: %s", err)
			return false
		}

		return isPossiblyInvalid
	}

	for i, blockDescription := range blockDescriptions {
		for _, candidate := range blockDescription.Candidates {
			if isPossiblyInvalid(blockDescription.Session, candidate.Value) {
				if i == 0 {
					return baseBlock, nil
				} else {
					return overseer.NewBlock(baseBlock.Number+uint32(i-1),
						blockDescriptions[i-1].BlockHash,
					), nil
				}
			}
		}
	}

	return last, nil
}

// getValidatorKeyPair returns the keypair for the given validator public key.
func getValidatorKeyPair(validatorPublic parachainTypes.ValidatorID,
	keystore keystore.Keystore,
) (keystore.KeyPair, error) {
	pubKey, err := sr25519.NewPublicKey(validatorPublic[:])
	if err != nil {
		return nil, fmt.Errorf("new public key: %w", err)
	}
	return keystore.GetKeypairFromAddress(pubKey.Address()), nil
}

// NewInitializedState creates a new initialized state.
func NewInitializedState(sender overseer.Sender,
	runtime parachainRuntime.RuntimeInstance,
	spamSlots SpamSlots,
	scraper *scraping.ChainScraper,
	highestSessionSeen parachainTypes.SessionIndex,
	gapsInCache bool,
) *Initialized {
	return &Initialized{
		runtime:               runtime,
		SpamSlots:             spamSlots,
		Scraper:               scraper,
		HighestSessionSeen:    highestSessionSeen,
		GapsInCache:           gapsInCache,
		ParticipationReceiver: make(chan MuxedMessage),
		ChainImportBacklog:    deque.New[parachainTypes.ScrapedOnChainVotes](),
		Participation:         NewParticipation(sender, runtime),
	}
}

// saturatingSub returns the result of a - b, saturating at 0.
func saturatingSub(a, b uint32) uint32 {
	result := int(a) - int(b)
	if result < 0 {
		return 0
	}
	return uint32(result)
}

// minInt returns the smallest of a or b.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
