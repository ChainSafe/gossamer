package dispute

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/parachain/dispute/types"
	parachainTypes "github.com/ChainSafe/gossamer/dot/parachain/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/dgraph-io/badger/v4"
)

const (
	earliestSessionKey   = "earliestSession"
	recentDisputesPrefix = "recentDisputes_"
	candidateVotesPrefix = "candidateVotes_"
	watermarkKey         = "watermark"
)

// Question: do we wanna scale encode the key as well?
func newEarliestSessionKey() []byte {
	return []byte(earliestSessionKey)
}

func newRecentDisputesKey(session parachainTypes.SessionIndex, candidateHash common.Hash) []byte {
	key := append([]byte(recentDisputesPrefix), session.Bytes()...)
	key = append(key, candidateHash[:]...)
	return key
}

func newCandidateVotesKey(session parachainTypes.SessionIndex, candidateHash common.Hash) []byte {
	key := append([]byte(candidateVotesPrefix), session.Bytes()...)
	key = append(key, candidateHash[:]...)
	return key
}

func newCandidateVotesSessionPrefix(session parachainTypes.SessionIndex) []byte {
	if session == 0 {
		return append([]byte(candidateVotesPrefix), 0, 0)
	}

	return append([]byte(candidateVotesPrefix), session.Bytes()...)
}

func newWatermarkKey() []byte {
	return []byte(watermarkKey)
}

const MaxCleanBatchSize = 300

type BadgerBackend struct {
	db *badger.DB
}

func (b *BadgerBackend) GetEarliestSession() (*parachainTypes.SessionIndex, error) {
	var earliestSession *parachainTypes.SessionIndex
	if err := b.db.View(func(txn *badger.Txn) error {
		key := newEarliestSessionKey()
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return scale.Unmarshal(val, &earliestSession)
		})
	}); err != nil {
		return nil, fmt.Errorf("get earliest session from db: %w", err)
	}

	return earliestSession, nil
}

func (b *BadgerBackend) GetRecentDisputes() (scale.BTree, error) {
	recentDisputes := scale.NewBTree[types.Dispute](types.CompareDisputes)

	if err := b.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := []byte(recentDisputesPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			dispute, err := types.NewDispute()
			if err != nil {
				return err
			}

			if err := item.Value(func(val []byte) error {
				return scale.Unmarshal(val, &dispute)
			}); err != nil {
				return err
			}
			recentDisputes.Set(dispute)
		}

		return nil
	}); err != nil {
		return recentDisputes, fmt.Errorf("get recent disputes from db: %w", err)
	}

	return recentDisputes, nil
}

func (b *BadgerBackend) GetCandidateVotes(session parachainTypes.SessionIndex,
	candidateHash common.Hash,
) (*types.CandidateVotes, error) {
	candidateVotes := types.NewCandidateVotes()
	if err := b.db.View(func(txn *badger.Txn) error {
		key := newCandidateVotesKey(session, candidateHash)
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		return item.Value(func(val []byte) error {
			return scale.Unmarshal(val, &candidateVotes)
		})
	}); err != nil {
		return nil, fmt.Errorf("get candidate votes from db: %w", err)
	}

	return candidateVotes, nil
}

// setEarliestSessionTxn sets the badger txn to store the earliest session.
func (b *BadgerBackend) setEarliestSessionTxn(txn *badger.Txn, session *parachainTypes.SessionIndex) error {
	key := newEarliestSessionKey()
	val, err := scale.Marshal(session)
	if err != nil {
		return err
	}

	return txn.Set(key, val)
}

// setRecentDisputesTxn sets the badger txn to store the recent disputes.
func (b *BadgerBackend) setRecentDisputesTxn(txn *badger.Txn, recentDisputes scale.BTree) error {
	var (
		val []byte
		err error
	)
	recentDisputes.Descend(nil, func(item interface{}) bool {
		dispute := item.(*types.Dispute)
		key := newRecentDisputesKey(dispute.Comparator.SessionIndex, dispute.Comparator.CandidateHash)
		val, err = scale.Marshal(dispute)
		if err != nil {
			return false
		}

		if err := txn.Set(key, val); err != nil {
			return false
		}

		return true
	})

	return err
}

// setCandidateVotesTxn sets the badger txn to store the candidate votes.
func (b *BadgerBackend) setCandidateVotesTxn(txn *badger.Txn,
	session parachainTypes.SessionIndex,
	candidateHash common.Hash,
	votes *types.CandidateVotes,
) error {
	key := newCandidateVotesKey(session, candidateHash)
	val, err := scale.Marshal(votes)
	if err != nil {
		return fmt.Errorf("marshal candidate votes: %w", err)
	}

	return txn.Set(key, val)
}

func (b *BadgerBackend) SetEarliestSession(session *parachainTypes.SessionIndex) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return b.setEarliestSessionTxn(txn, session)
	})
}

func (b *BadgerBackend) SetRecentDisputes(recentDisputes scale.BTree) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return b.setRecentDisputesTxn(txn, recentDisputes)
	})
}

func (b *BadgerBackend) SetCandidateVotes(session parachainTypes.SessionIndex,
	candidateHash common.Hash,
	votes *types.CandidateVotes,
) error {
	return b.db.Update(func(txn *badger.Txn) error {
		return b.setCandidateVotesTxn(txn, session, candidateHash, votes)
	})
}

// setWatermarkTxn sets the badger txn to store the session watermark.
func (b *BadgerBackend) setWatermarkTxn(txn *badger.Txn, session parachainTypes.SessionIndex) error {
	key := newWatermarkKey()
	val, err := scale.Marshal(session)
	if err != nil {
		return err
	}

	return txn.Set(key, val)
}

// getWatermark gets the session watermark.
// session watermark is used to cleanup old candidate votes.
func (b *BadgerBackend) getWatermark() (parachainTypes.SessionIndex, error) {
	var watermark parachainTypes.SessionIndex
	if err := b.db.View(func(txn *badger.Txn) error {
		key := newWatermarkKey()
		item, err := txn.Get(key)
		if err != nil {
			if err.Error() != badger.ErrKeyNotFound.Error() {
				return err
			}

			watermark = 0
			return nil
		}

		return item.Value(func(val []byte) error {
			return scale.Unmarshal(val, &watermark)
		})
	}); err != nil {
		return 0, fmt.Errorf("get watermark from db: %w", err)
	}

	return watermark, nil
}

// setVotesCleanupTxn sets the badger txn to cleanup old candidate votes.
func (b *BadgerBackend) setVotesCleanupTxn(txn *badger.Txn, earliestSession parachainTypes.SessionIndex) error {
	// Get watermark
	watermark, err := b.getWatermark()
	if err != nil {
		return fmt.Errorf("get watermark: %w", err)
	}

	cleanUntil := earliestSession - watermark
	if cleanUntil > MaxCleanBatchSize {
		cleanUntil = MaxCleanBatchSize
	}

	for i := watermark; i < cleanUntil; i++ {
		prefix := newCandidateVotesSessionPrefix(i)
		it := txn.NewIterator(badger.DefaultIteratorOptions)

		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			item := it.Item()
			err := txn.Delete(item.Key())
			if err != nil {
				it.Close()
				return fmt.Errorf("delete candidate votes: %w", err)
			}
		}

		it.Close()
	}

	// new watermark
	if err := b.setWatermarkTxn(txn, cleanUntil); err != nil {
		return fmt.Errorf("set watermark: %w", err)
	}

	return nil
}

func (b *BadgerBackend) Write(earliestSession *parachainTypes.SessionIndex,
	recentDisputes scale.BTree,
	candidateVotes map[types.Comparator]*types.CandidateVotes,
) error {
	return b.db.Update(func(txn *badger.Txn) error {
		if err := b.setEarliestSessionTxn(txn, earliestSession); err != nil {
			return fmt.Errorf("set earliest session in db: %w", err)
		}

		// cleanup old candidate votes
		if err := b.setVotesCleanupTxn(txn, *earliestSession); err != nil {
			return fmt.Errorf("cleanup votes: %w", err)
		}

		if err := b.setRecentDisputesTxn(txn, recentDisputes); err != nil {
			return fmt.Errorf("set recent disputes in db: %w", err)
		}

		for comparator, votes := range candidateVotes {
			if votes == nil {
				if err := b.setVotesCleanupTxn(txn, comparator.SessionIndex); err != nil {
					return fmt.Errorf("delete candidate votes: %w", err)
				}
			} else {
				if err := b.setCandidateVotesTxn(txn,
					comparator.SessionIndex,
					comparator.CandidateHash,
					votes); err != nil {
					return fmt.Errorf("set candidate votes in db: %w", err)
				}
			}

		}

		return nil
	})
}

var _ DBBackend = (*BadgerBackend)(nil)

// NewDBBackend creates a new DBBackend backed by a badger db.
func NewDBBackend(db *badger.DB) *BadgerBackend {
	return &BadgerBackend{
		db: db,
	}
}
