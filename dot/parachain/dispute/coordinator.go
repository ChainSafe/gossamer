package dispute

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/overseer"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/scraping"
	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachain "github.com/ChainSafe/gossamer/dot/parachain/runtime"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/internal/log"
	"github.com/ChainSafe/gossamer/lib/keystore"
	"github.com/dgraph-io/badger/v4"
	"github.com/emirpasic/gods/sets/treeset"
	"time"
)

const Window = 6

var logger = log.NewFromGlobal(log.AddContext("parachain", "disputes"))

// Coordinator implements the CoordinatorSubsystem interface.
type Coordinator struct {
	keystore     keystore.Keystore
	store        *overlayBackend
	runtime      parachain.RuntimeInstance
	maxSpamVotes uint32

	overseer chan<- any
	receiver chan any
}

type startupResult struct {
	participation    []ParticipationData
	votes            []parachainTypes.ScrapedOnChainVotes
	spamSlots        SpamSlots
	orderingProvider *scraping.ChainScraper
	highestSession   parachainTypes.SessionIndex
	gapsInCache      bool
}

type initializeResult struct {
	participation []ParticipationData
	votes         []parachainTypes.ScrapedOnChainVotes
	activatedLeaf *overseer.ActivatedLeaf
	initialized   *Initialized
}

func (d *Coordinator) sendDisputeMessages(
	env types.CandidateEnvironment,
	voteState types.CandidateVoteState,
) {
	ownVotes, err := voteState.Own.Votes()
	if err != nil {
		logger.Errorf("get own votes: %s", err)
		return
	}

	for _, vote := range ownVotes {
		keypair, err := types.GetValidatorKeyPair(d.keystore, env.Session.Validators, vote.ValidatorIndex)
		if err != nil {
			logger.Errorf("get validator key pair: %s", err)
			continue
		}

		candidateHash, err := voteState.Votes.CandidateReceipt.Hash()
		if err != nil {
			logger.Errorf("get candidate hash: %s", err)
			continue
		}

		isValid, err := vote.DisputeStatement.IsValid()
		if err != nil {
			logger.Errorf("check if dispute statement is valid: %s", err)
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

		if err := sendMessage(d.overseer, disputeMessage); err != nil {
			logger.Errorf("send dispute message: %s", err)
		}
	}
}

func (d *Coordinator) waitForFirstLeaf() (*overseer.ActivatedLeaf, error) {
	for {
		select {
		case overseerMessage := <-d.receiver:
			switch message := overseerMessage.(type) {
			case overseer.Signal[overseer.ActiveLeavesUpdate]:
				return message.Data.Activated, nil
			default:
				logger.Warnf("Received message before first active leaves update. "+
					"This is not expected - message will be dropped. %T", message)
			}
		}
	}
}

func (d *Coordinator) initialize() (
	*initializeResult,
	error,
) {
	firstLeaf, err := d.waitForFirstLeaf()
	if err != nil {
		return nil, fmt.Errorf("wait for first leaf: %w", err)
	}

	startupData, err := d.handleStartup(firstLeaf)
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
		initialized: NewInitializedState(d.overseer,
			d.receiver,
			d.runtime,
			startupData.spamSlots,
			startupData.orderingProvider,
			startupData.highestSession,
			startupData.gapsInCache,
			d.keystore,
		),
	}, nil
}

func (d *Coordinator) handleStartup(initialHead *overseer.ActivatedLeaf) (
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

	gapsInCache := false
	for idx := saturatingSub(uint32(highestSession), Window-1); idx <= uint32(highestSession); idx++ {
		sessionInfo, err := d.runtime.ParachainHostSessionInfo(initialHead.Hash, parachainTypes.SessionIndex(idx))
		if err != nil || sessionInfo == nil {
			logger.Debugf("no session info for session %d", idx)
			gapsInCache = true
			continue
		}
	}

	// prune obsolete disputes
	earliestSession := parachainTypes.SessionIndex(saturatingSub(uint32(highestSession), Window-1))
	if err := d.store.NoteEarliestSession(earliestSession); err != nil {
		return nil, fmt.Errorf("note earliest session: %w", err)
	}

	// for every dispute in activeDisputes
	// get candidate votes
	// check if it is a potential spam
	// participate if needed, if not distribute the own vote
	var participationRequests []ParticipationData
	spamDisputes := make(map[unconfirmedKey]*treeset.Set)
	leafHash := initialHead.Hash
	scraper, scrapedVotes, err := scraping.NewChainScraper(d.overseer, d.runtime, initialHead)
	if err != nil {
		return nil, fmt.Errorf("new chain scraper: %w", err)
	}

	activeDisputes.Descend(nil, func(i interface{}) bool {
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

		voteState, err := types.NewCandidateVoteState(*votes, env, uint64(now))
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
		} else if voteState.Own.VoteMissing() {
			logger.Tracef("found valid dispute, with no vote from us on startup - participating. %s")
			priority := ParticipationPriorityHigh
			if !isIncluded {
				priority = ParticipationPriorityBestEffort
			}

			participationRequests = append(participationRequests, ParticipationData{
				request: ParticipationRequest{
					candidateHash:    dispute.Comparator.CandidateHash,
					candidateReceipt: voteState.Votes.CandidateReceipt,
					session:          dispute.Comparator.SessionIndex,
				},
				priority: priority,
			})
		} else {
			logger.Tracef("found valid dispute, with vote from us on startup - distributing. %s")
			d.sendDisputeMessages(*env, voteState)
		}

		return true
	})

	return &startupResult{
		participation:    participationRequests,
		votes:            scrapedVotes.OnChainVotes,
		spamSlots:        NewSpamSlotsFromState(spamDisputes, d.maxSpamVotes),
		orderingProvider: scraper,
		highestSession:   0,
		gapsInCache:      gapsInCache,
	}, nil
}

func (d *Coordinator) Run() error {
	initResult, err := d.initialize()
	if err != nil {
		return fmt.Errorf("initialize dispute coordinator: %w", err)
	}

	initData := InitialData{
		Participation: initResult.participation,
		Votes:         initResult.votes,
		Leaf:          initResult.activatedLeaf,
	}
	initResult.initialized.Run(d.overseer, d.store.inner, &initData)
	return nil
}

func NewDisputesCoordinator(db *badger.DB, overseer chan<- any, receiver chan any) (*Coordinator, error) {
	dbBackend := NewDBBackend(db)
	backend := newOverlayBackend(dbBackend)

	return &Coordinator{
		store:        backend,
		overseer:     overseer,
		receiver:     receiver,
		maxSpamVotes: MaxSpamVotes,
	}, nil
}
