package parachain

import (
	"fmt"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	parachaintypes "github.com/ChainSafe/gossamer/lib/parachain/types"
	"github.com/ChainSafe/gossamer/pkg/scale"
	"github.com/libp2p/go-libp2p/core/peer"
)

type CollationProtocolValues interface {
	CollatorProtocolMessage
}

// CollationProtocol represents all network messages on the collation peer-set.
type CollationProtocol struct {
	inner any
}

func setCollationProtocol[Value CollationProtocolValues](mvdt *CollationProtocol, value Value) {
	mvdt.inner = value
}

func (mvdt *CollationProtocol) SetValue(value any) (err error) {
	switch value := value.(type) {
	case CollatorProtocolMessage:
		setCollationProtocol(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollationProtocol) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case CollatorProtocolMessage:
		return 0, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CollationProtocol) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CollationProtocol) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(CollatorProtocolMessage), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewCollationProtocol returns a new collation protocol varying data type
func NewCollationProtocol() CollationProtocol {
	return CollationProtocol{}
}

type CollatorProtocolMessageValues interface {
	Declare | AdvertiseCollation | CollationSeconded
}

// CollatorProtocolMessage represents Network messages used by the collator protocol subsystem
type CollatorProtocolMessage struct {
	inner any
}

func setCollatorProtocolMessage[Value CollatorProtocolMessageValues](mvdt *CollatorProtocolMessage, value Value) {
	mvdt.inner = value
}

func (mvdt *CollatorProtocolMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case Declare:
		setCollatorProtocolMessage(mvdt, value)
		return

	case AdvertiseCollation:
		setCollatorProtocolMessage(mvdt, value)
		return

	case CollationSeconded:
		setCollatorProtocolMessage(mvdt, value)
		return

	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt CollatorProtocolMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case Declare:
		return 0, mvdt.inner, nil

	case AdvertiseCollation:
		return 1, mvdt.inner, nil

	case CollationSeconded:
		return 4, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt CollatorProtocolMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt CollatorProtocolMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 0:
		return *new(Declare), nil

	case 1:
		return *new(AdvertiseCollation), nil

	case 4:
		return *new(CollationSeconded), nil

	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

// NewCollatorProtocolMessage returns a new collator protocol message varying data type
func NewCollatorProtocolMessage() CollatorProtocolMessage {
	return CollatorProtocolMessage{}
}

// Declare the intent to advertise collations under a collator ID, attaching a
// signature of the `PeerId` of the node using the given collator ID key.
type Declare struct {
	CollatorId        parachaintypes.CollatorID        `scale:"1"`
	ParaID            uint32                           `scale:"2"`
	CollatorSignature parachaintypes.CollatorSignature `scale:"3"`
}

// AdvertiseCollation contains a relay parent hash and is used to advertise a collation to a validator.
// This will only advertise a collation if there exists one for the given relay parent and the given peer is
// set as validator for our para at the given relay parent.
// It can only be sent once the peer has declared that they are a collator with given ID
type AdvertiseCollation common.Hash

// CollationSeconded represents that a collation sent to a validator was seconded.
type CollationSeconded struct {
	Hash                         common.Hash                  `scale:"1"`
	UncheckedSignedFullStatement UncheckedSignedFullStatement `scale:"2"`
}

// Index returns the index of varying data type
func (CollationSeconded) Index() uint {
	return 4
}

const MaxCollationMessageSize uint64 = 100 * 1024

type CollationProtocolV1 struct{}

// Type returns CollationMsgType
func (*CollationProtocolV1) Type() network.MessageType {
	return network.CollationMsgType
}

// Hash returns the hash of the CollationProtocolV1
func (cp *CollationProtocolV1) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := cp.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (cp *CollationProtocolV1) Encode() ([]byte, error) {
	enc, err := scale.Marshal(*cp)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

func decodeCollationMessage(in []byte) (network.NotificationsMessage, error) {
	collationMessage := CollationProtocolV1{}

	err := scale.Unmarshal(in, &collationMessage)
	if err != nil {
		return nil, fmt.Errorf("cannot decode message: %w", err)
	}

	return &collationMessage, nil
}

func handleCollationMessage(_ peer.ID, msg network.NotificationsMessage) (bool, error) {
	// TODO: Add things
	fmt.Println("We got a collation message", msg)
	return false, nil
}

func getCollatorHandshake() (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func decodeCollatorHandshake(_ []byte) (network.Handshake, error) {
	return &collatorHandshake{}, nil
}

func validateCollatorHandshake(_ peer.ID, _ network.Handshake) error {
	return nil
}

type collatorHandshake struct{}

// String formats a collatorHandshake as a string
func (*collatorHandshake) String() string {
	return "collatorHandshake"
}

// Encode encodes a collatorHandshake message using SCALE
func (*collatorHandshake) Encode() ([]byte, error) {
	return []byte{}, nil
}

// Decode the message into a collatorHandshake
func (*collatorHandshake) Decode(_ []byte) error {
	return nil
}

// IsValid returns true
func (*collatorHandshake) IsValid() bool {
	return true
}
