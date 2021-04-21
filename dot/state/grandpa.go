package state

import (
	"encoding/binary"
	"errors"

	"github.com/ChainSafe/chaindb"
	"github.com/ChainSafe/gossamer/lib/grandpa"
	"github.com/ChainSafe/gossamer/lib/scale"
)

var (
	grandpaPrefix     = "grandpa"
	authoritiesPrefix = []byte("auth")
	currentSetIDKey   = []byte("setID")
)

// GrandpaState tracks information related to grandpa
type GrandpaState struct {
	baseDB chaindb.Database
	db     chaindb.Database
}

// NewGrandpaStateFromGenesis returns a new GrandpaState given the grandpa genesis authorities
func NewGrandpaStateFromGenesis(db chaindb.Database, genesisAuthorities []*grandpa.Voter) (*GrandpaState, error) {
	grandpaDB := chaindb.NewTable(db, grandpaPrefix)
	s := &GrandpaState{
		baseDB: db,
		db:     grandpaDB,
	}

	err := s.SetAuthorities(1, genesisAuthorities)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// NewGrandpaState returns a new GrandpaState
func NewGrandpaState(db chaindb.Database) (*GrandpaState, error) {
	return &GrandpaState{
		baseDB: db,
		db:     chaindb.NewTable(db, grandpaPrefix),
	}, nil
}

func authoritiesKey(setID uint64) []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, setID)
	return append(authoritiesPrefix, buf...)
}

// SetAuthorities sets the authorities for a given setID
func (s *GrandpaState) SetAuthorities(setID uint64, authorities []*grandpa.Voter) error {
	enc, err := scale.Encode(authorities)
	if err != nil {
		return err
	}

	return s.db.Put(authoritiesKey(setID), enc)
}

// GetAuthorities returns the authorities for the given setID
func (s *GrandpaState) GetAuthorities(setID uint64) ([]*grandpa.Voter, error) {
	enc, err := s.db.Get(authoritiesKey(setID))
	if err != nil {
		return nil, err
	}

	v, err := scale.Decode(enc, []*grandpa.Voter{})
	if err != nil {
		return nil, err
	}

	return v.([]*grandpa.Voter), nil
}

func (s *GrandpaState) SetCurrentSetID(setID uint64) error {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, setID)
	return s.db.Put(currentSetIDKey, buf[:])
}

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

// IncrementAndSetAuthorities increments the current set ID and sets the authorities for the new current set ID
func (s *GrandpaState) IncrementAndSetAuthorities(authorities []*grandpa.Voter) error {
	currSetID, err := s.GetCurrentSetID()
	if err != nil {
		return err
	}

	newSetID := currSetID + 1
	err = s.SetAuthorities(newSetID, authorities)
	if err != nil {
		return err
	}

	return s.SetCurrentSetID(newSetID)
}
