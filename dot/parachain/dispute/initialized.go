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
	"github.com/gammazero/deque"
	"time"
)

const ChainImportMaxBatchSize = 6

type Initialized struct {
	// TODO: keystore
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
		defaultConfirm := func() error { return nil }
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
			} else if message.Subsystem != nil {
				switch {
				case message.Signal.Concluded:
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
					i.Scraper.ProcessFinalisedBlock(message.Signal.BlockFinalised.BlockNumber)

				// TODO: case: FromOrchestra::Communication
				default:
					logger.Errorf("OverseerSignal::Unknown")
				}
			}
		}

		if !overlayDB.IsEmpty() {
			if err := overlayDB.WriteToDB(); err != nil {
				return fmt.Errorf("write overlay backend to db: %w", err)
			}
		}

		err := defaultConfirm()
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

			if _, err := types.NewCheckedSignedDisputeStatement(disputeStatement, candidateHash, votes.Session, validatorPublic, parachain.ValidatorSignature(validatorSignature)); err != nil {
				logger.Errorf("scraped backing votes had invalid signature. Candidate: %v, session: %v, validatorPublic: %v, validatorIndex: %v",
					candidateHash,
					votes.Session,
					validatorPublic,
					backers.ValidatorIndex,
				)
				return fmt.Errorf("new checked signed dispute statement: %w", err)
			}

			signedDisputeStatement := types.NewSignedDisputeStatement(disputeStatement, candidateHash, votes.Session, validatorPublic, parachain.ValidatorSignature(validatorSignature))
			statements = append(statements, types.Statement{
				SignedDisputeStatement: signedDisputeStatement,
				ValidatorIndex:         backers.ValidatorIndex,
			})

			// Importantly, handling import statements for backing votes also
			// clears spam slots for any newly backed candidates
			if err := i.HandleImportStatements(context,
				backend,
				backingValidators.CandidateReceipt,
				votes.Session,
				statements,
				now,
			); err != nil {
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

			sessionInfo, err := i.runtime.ParachainHostSessionInfo(
				blockHash,
				parachainTypes.SessionIndex(dispute.Session),
			)
			if err != nil {
				logger.Warnf("could not retrieve session info for recently concluded dispute. "+
					"session: %v, candidateHash: %v, error: %v",
					dispute.Session,
					dispute.CandidateHash,
					err,
				)
				continue
			}

			var filteredStatements []types.Statement
			for _, statement := range statements {
				if int(statement.ValidatorIndex) < len(sessionInfo.Validators) {
					logger.Errorf("missing validator public key that participated in concluded dispute. "+
						"session: %v, validatorIndex: %v",
						statement.SignedDisputeStatement.SessionIndex,
						statement.ValidatorIndex,
					)
					continue
				}

				validatorPublic := sessionInfo.Validators[statement.ValidatorIndex]
				disputeStatement := types.NewSignedDisputeStatement(statement.SignedDisputeStatement.DisputeStatement,
					statement.SignedDisputeStatement.CandidateHash,
					statement.SignedDisputeStatement.SessionIndex,
					validatorPublic,
					statement.SignedDisputeStatement.ValidatorSignature,
				)
				filteredStatements = append(filteredStatements, types.Statement{
					SignedDisputeStatement: disputeStatement,
					ValidatorIndex:         statement.ValidatorIndex,
				})
			}

			if len(filteredStatements) == 0 {
				logger.Errorf("skipping empty from chain dispute import. session: %v, candidateHash: %v",
					votes.Session,
					candidateHash,
				)
				continue
			}

			if err := i.HandleImportStatements(context,
				backend,
				backingValidators.CandidateReceipt,
				votes.Session,
				filteredStatements,
				now,
			); err != nil {
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
) error {
	switch {
	case message.ImportStatements != nil:
		logger.Tracef("in HandleIncoming::ImportStatements")
	case message.RecentDisputes != nil:
	case message.ActiveDisputes != nil:
	case message.QueryCandidateVotes != nil:
	case message.IssueLocalStatement != nil:
	case message.DetermineUndisputedChain != nil:
	default:
		return fmt.Errorf("unknown dispute coordinator message")
	}

	return nil
}

func (i *Initialized) HandleImportStatements(
	context overseer.Context,
	backend OverlayBackend,
	candidateReceipt parachainTypes.CandidateReceipt,
	session parachainTypes.SessionIndex,
	statements []types.Statement,
	now uint64,
) error {
	logger.Tracef("in HandleImportStatements")

	if i.sessionIsAncient(session) {

	}
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
	//TODO: implement
	panic("Initialized.IssueLocalStatement not implemented")
}

func (i *Initialized) sessionIsAncient(session parachainTypes.SessionIndex) bool {
	diff := session - (DisputeWindow - 1)
	return session < diff || session < i.HighestSessionSeen
}

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

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
