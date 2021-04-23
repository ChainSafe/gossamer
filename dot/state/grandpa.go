package state

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ChainSafe/chaindb"
	//"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/scale"
)

var (
	grandpaPrefix     = "grandpa"
	authoritiesPrefix = []byte("auth")
	setIDChangePrefix = []byte("change")
	currentSetIDKey   = []byte("setID")
)

// GrandpaState tracks information related to grandpa
type GrandpaState struct {
	baseDB     chaindb.Database
	db         chaindb.Database
	blockState *BlockState
}

// NewGrandpaStateFromGenesis returns a new GrandpaState given the grandpa genesis authorities
func NewGrandpaStateFromGenesis(db chaindb.Database, genesisAuthorities []*types.GrandpaVoter) (*GrandpaState, error) {
	grandpaDB := chaindb.NewTable(db, grandpaPrefix)
	s := &GrandpaState{
		baseDB: db,
		db:     grandpaDB,
	}

	err := s.SetCurrentSetID(1)
	if err != nil {
		return nil, err
	}

	err = s.SetAuthorities(1, genesisAuthorities)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// NewGrandpaState returns a new GrandpaState
func NewGrandpaState(db chaindb.Database, blockState *BlockState) (*GrandpaState, error) {
	return &GrandpaState{
		baseDB:     db,
		db:         chaindb.NewTable(db, grandpaPrefix),
		blockState: blockState,
	}, nil
}

func authoritiesKey(setID uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, setID)
	return append(authoritiesPrefix, buf...)
}

func setIDChangeKey(setID uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, setID)
	return append(setIDChangePrefix, buf...)
}

// SetAuthorities sets the authorities for a given setID
func (s *GrandpaState) SetAuthorities(setID uint64, authorities []*types.GrandpaVoter) error {
	enc, err := scale.Encode(authorities)
	if err != nil {
		return err
	}

	return s.db.Put(authoritiesKey(setID), enc)
}

// GetAuthorities returns the authorities for the given setID
func (s *GrandpaState) GetAuthorities(setID uint64) ([]*types.GrandpaVoter, error) {
	enc, err := s.db.Get(authoritiesKey(setID))
	if err != nil {
		return nil, err
	}

	v, err := scale.Decode(enc, []*types.GrandpaVoter{})
	if err != nil {
		return nil, err
	}

	return v.([]*types.GrandpaVoter), nil
}

// SetCurrentSetID sets the current set ID
func (s *GrandpaState) SetCurrentSetID(setID uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, setID)
	return s.db.Put(currentSetIDKey, buf[:])
}

// GetCurrentSetID retrieves the current set ID
func (s *GrandpaState) GetCurrentSetID() (uint64, error) {
	id, err := s.db.Get(currentSetIDKey)
	if err != nil {
		return 0, err
	}

	if len(id) < 8 {
		return 0, errors.New("invalid setID")
	}

	return binary.LittleEndian.Uint64(id), nil
}

// SetNextChange sets the next authority change
func (s *GrandpaState) SetNextChange(authorities []*types.GrandpaVoter, number *big.Int) error {
	currSetID, err := s.GetCurrentSetID()
	if err != nil {
		return err
	}

	nextSetID := currSetID + 1
	err = s.SetAuthorities(nextSetID, authorities)
	if err != nil {
		return err
	}

	err = s.SetSetIDChangeAtBlock(nextSetID, number)
	if err != nil {
		return err
	}

	return nil
}

// IncrementSetID increments the set ID
func (s *GrandpaState) IncrementSetID() error {
	currSetID, err := s.GetCurrentSetID()
	if err != nil {
		return err
	}

	nextSetID := currSetID + 1
	return s.SetCurrentSetID(nextSetID)
}

// SetSetIDChangeAtBlock sets a set ID change at a certain block
func (s *GrandpaState) SetSetIDChangeAtBlock(setID uint64, number *big.Int) error {
	return s.db.Put(setIDChangeKey(setID), number.Bytes())
}

// GetSetIDChange returs the block number where the set ID was updated
func (s *GrandpaState) GetSetIDChange(setID uint64) (*big.Int, error) {
	num, err := s.db.Get(setIDChangeKey(setID))
	if err != nil {
		return nil, err
	}

	return big.NewInt(0).SetBytes(num), nil
}
