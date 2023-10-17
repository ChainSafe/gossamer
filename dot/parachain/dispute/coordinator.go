package dispute

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/scraping"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/crypto/sr25519"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/dgraph-io/badger/v4"
	"github.com/emirpasic/gods/sets/treeset"
	"time"
)

const DisputeWindow = 6

var logger = log.NewFromGlobal(log.AddContext("parachain", "disputes"))

// CoordinatorSubsystem is the dispute coordinator subsystem interface.
type CoordinatorSubsystem interface {
	// Run runs the dispute coordinator subsystem.
	Run(context overseer.Context) error
}

// disputeCoordinator implements the CoordinatorSubsystem interface.
type disputeCoordinator struct {
	keystore keystore.Keystore
	store    *overlayBackend
	runtime  parachain.RuntimeInstance
}

func (d *disputeCoordinator) Run(context overseer.Context) error {
	initResult, err := d.initialize(context)
	if err != nil {
		return fmt.Errorf("initialize dispute coordinator: %w", err)
	}

	//TODO: run the subsystem
	initData := InitialData{
		Participation: initResult.participation,
		Votes:         initResult.votes,
		Leaf:          initResult.activatedLeaf,
	}

	if err := initResult.initialized.Run(context, d.store.inner, &initData); err != nil {
		return fmt.Errorf("run initialized state: %w", err)
	}

	return nil
}

type startupResult struct {
	participation    []ParticipationRequestWithPriority
	votes            []parachainTypes.ScrapedOnChainVotes
	spamSlots        SpamSlots
	orderingProvider scraping.ChainScraper
	highestSession   parachainTypes.SessionIndex
	gapsInCache      bool
}

type initializeResult struct {
	participation []ParticipationRequestWithPriority
	votes         []parachainTypes.ScrapedOnChainVotes
	activatedLeaf *overseer.ActivatedLeaf
	initialized   *Initialized
}

func (d *disputeCoordinator) waitForFirstLeaf(context overseer.Context) (*overseer.ActivatedLeaf, error) {
	// TODO: implement
	for {
		select {
		case overseerSignal := <-context.Receiver:
			if overseerSignal == nil {
				return nil, fmt.Errorf("received nil signal from overseer")
			}
		}
	}
}

func (d *disputeCoordinator) initialize(context overseer.Context) (
	*initializeResult,
	error,
) {
	firstLeaf, err := d.waitForFirstLeaf(context)
	if err != nil {
		return nil, fmt.Errorf("wait for first leaf: %w", err)
	}

	startupData, err := d.handleStartup(context, firstLeaf)
	if err != nil {
		return nil, fmt.Errorf("handle startup: %w", err)
	}

	if !d.store.IsEmpty() {
		if err := d.store.WriteToDB(); err != nil {
			return nil, fmt.Errorf("write to db: %w", err)
		}
	}

	return &initializeResult{
		participation: startupData.participation,
		votes:         startupData.votes,
		activatedLeaf: firstLeaf,
		initialized: NewInitializedState(context.Sender,
			d.runtime,
			startupData.spamSlots,
			&startupData.orderingProvider,
			startupData.highestSession,
			startupData.gapsInCache,
		),
	}, nil
}

func (d *disputeCoordinator) handleStartup(context overseer.Context, initialHead *overseer.ActivatedLeaf) (
	*startupResult,
	error,
) {
	var now = time.Now().Unix()
	activeDisputes, err := d.store.GetActiveDisputes(uint64(now))
	if err != nil {
		return nil, fmt.Errorf("get active disputes: %w", err)
	}

	// get the highest session
	highestSession, err := d.runtime.ParachainHostSessionIndexForChild(initialHead.Hash)
	if err != nil {
		return nil, fmt.Errorf("getting highest session: %w", err)
	}

	// TODO: we need to cache the sessionInfo. Polkadot has a module for it in subsystems
	// https://github.com/paritytech/polkadot/blob/master/node/subsystem-util/src/runtime/mod.rs
	gapsInCache := false
	for idx := highestSession - (DisputeWindow - 1); idx <= highestSession; idx++ {
		_, err = d.runtime.ParachainHostSessionInfo(initialHead.Hash, idx)
		if err != nil {
			logger.Debugf("no session info for session %d", idx)
			gapsInCache = true
			continue
		}
	}

	// prune obsolete disputes
	if err := d.store.NoteEarliestSession(highestSession); err != nil {
		return nil, fmt.Errorf("note earliest session: %w", err)
	}

	// for every dispute in activeDisputes
	// get candidate votes
	// check if it is a potential spam
	// participate if needed, if not distribute the own vote
	var participationRequests []ParticipationRequestWithPriority
	spamDisputes := make(map[unconfirmedKey]*treeset.Set)
	leafHash := initialHead.Hash
	scraper, scrapedVotes, err := scraping.NewChainScraper(context.Sender, d.runtime, initialHead)
	if err != nil {
		return nil, fmt.Errorf("new chain scraper: %w", err)
	}

	activeDisputes.Value.Descend(nil, func(i interface{}) bool {
		dispute, ok := i.(*types.Dispute)
		if !ok {
			logger.Error("active dispute is not a dispute")
			return true
		}

		env, err := types.NewCandidateEnvironment(d.keystore, d.runtime, highestSession, leafHash)
		if err != nil {
			logger.Errorf("we are lacking a `SessionInfo` for handling db votes on startup.: %s", err)
			return true
		}

		votes, err := d.store.GetCandidateVotes(highestSession, dispute.Comparator.CandidateHash)
		if err != nil {
			logger.Errorf("failed to get initial candidate votes for dispute %s: %s",
				dispute.Comparator.CandidateHash, err)
			return false
		}

		voteState, err := types.NewCandidateVoteState(*votes, uint64(now))
		if err != nil {
			logger.Errorf("failed to create candidate vote state for dispute %s: %s",
				dispute.Comparator.CandidateHash, err)
			return false
		}

		potentialSpam, err := scraper.IsPotentialSpam(voteState, dispute.Comparator.CandidateHash)
		if err != nil {
			logger.Errorf("failed to check if dispute %s is potential spam: %s",
				dispute.Comparator.CandidateHash, err)
			return false
		}
		isIncluded := scraper.IsCandidateIncluded(dispute.Comparator.CandidateHash)

		if potentialSpam {
			logger.Tracef("found potential spam dispute on startup %s", dispute.Comparator.CandidateHash)

			disputeKey := unconfirmedKey{
				session:   dispute.Comparator.SessionIndex,
				candidate: dispute.Comparator.CandidateHash,
			}
			if _, ok := spamDisputes[disputeKey]; !ok {
				spamDisputes[disputeKey] = treeset.NewWithIntComparator()
			}
			spamDisputes[disputeKey].Add(voteState.Votes.VotedIndices())
		} else {
			if voteState.Own.VoteMissing() {
				logger.Tracef("found valid dispute, with no vote from us on startup - participating. %s")
				priority := ParticipationPriorityHigh
				if !isIncluded {
					priority = ParticipationPriorityBestEffort
				}

				participationRequests = append(participationRequests, ParticipationRequestWithPriority{
					request: ParticipationRequest{
						candidateHash:    dispute.Comparator.CandidateHash,
						candidateReceipt: voteState.Votes.CandidateReceipt,
						session:          dispute.Comparator.SessionIndex,
					},
					priority: priority,
				})
			} else {
				logger.Tracef("found valid dispute, with vote from us on startup - distributing. %s")
				d.sendDisputeMessages(context, *env, voteState)
			}
		}

		return true
	})

	return &startupResult{
		participation:    participationRequests,
		votes:            scrapedVotes.OnChainVotes,
		spamSlots:        NewSpamSlotsFromState(spamDisputes, MaxSpamVotes),
		orderingProvider: scraping.ChainScraper{},
		highestSession:   0,
		gapsInCache:      gapsInCache,
	}, nil
}

func (d *disputeCoordinator) sendDisputeMessages(
	context overseer.Context,
	env types.CandidateEnvironment,
	voteState types.CandidateVoteState,
) {
	ownVotes, err := voteState.Own.Votes()
	if err != nil {
		logger.Errorf("failed to get own votes: %s", err)
		return
	}

	var publicKey parachainTypes.ValidatorID
	for _, vote := range ownVotes {
		if int(vote.ValidatorIndex) < len(env.Session.Validators) {
			publicKey = env.Session.Validators[vote.ValidatorIndex]
		} else {
			logger.Errorf("failed to get validator public key for index %d", vote.ValidatorIndex)
			continue
		}
		pubKey, err := sr25519.NewPublicKey(publicKey[:])
		if err != nil {
			logger.Errorf("failed to create public key: %s", err)
			continue
		}
		keypair := d.keystore.GetKeypairFromAddress(pubKey.Address())

		candidateHash, err := voteState.Votes.CandidateReceipt.Hash()
		if err != nil {
			logger.Errorf("failed to get candidate hash: %s", err)
			continue
		}

		isValid, err := vote.DisputeStatement.IsValid()
		if err != nil {
			logger.Errorf("failed to check if dispute statement is valid: %s", err)
			continue
		}

		signedDisputeStatement, err := types.NewSignedDisputeStatement(keypair, isValid, candidateHash, env.SessionIndex)
		if err != nil {
			logger.Errorf("create signed dispute statement: %s", err)
			continue
		}

		disputeMessage, err := types.NewDisputeMessage(keypair, voteState.Votes, &signedDisputeStatement, vote.ValidatorIndex, env.Session)
		if err != nil {
			logger.Errorf("create dispute message: %s", err)
			continue
		}

		if err := context.Sender.SendMessage(disputeMessage); err != nil {
			logger.Errorf("send dispute message: %s", err)
		}
	}
}

func NewDisputeCoordinator(path string) (CoordinatorSubsystem, error) {
	db, err := badger.Open(badger.DefaultOptions(path))
	if err != nil {
		return nil, fmt.Errorf("open badger db: %w", err)
	}

	dbBackend := NewDBBackend(db)
	backend := newOverlayBackend(dbBackend)

	return &disputeCoordinator{
		store: backend,
	}, nil
}
