package networkbridge

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/ChainSafe/gossamer/dot/network"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

type WireMessage struct {
	inner any
}

type WireMessageValues interface {
	ViewUpdate | ProtocolMessage
}

func setMyVaryingDataType[Value WireMessageValues](mvdt *WireMessage, value Value) {
	mvdt.inner = value
}

func (mvdt *WireMessage) SetValue(value any) (err error) {
	switch value := value.(type) {
	case ViewUpdate:
		setMyVaryingDataType(mvdt, value)
		return
	case ProtocolMessage:
		setMyVaryingDataType(mvdt, value)
		return
	default:
		return fmt.Errorf("unsupported type")
	}
}

func (mvdt WireMessage) IndexValue() (index uint, value any, err error) {
	switch mvdt.inner.(type) {
	case ProtocolMessage:
		return 1, mvdt.inner, nil
	case ViewUpdate:
		return 2, mvdt.inner, nil

	}
	return 0, nil, scale.ErrUnsupportedVaryingDataTypeValue
}

func (mvdt WireMessage) Value() (value any, err error) {
	_, value, err = mvdt.IndexValue()
	return
}

func (mvdt WireMessage) ValueAt(index uint) (value any, err error) {
	switch index {
	case 1:
		return ProtocolMessage{}, nil
	case 2:
		return ViewUpdate{}, nil
	}
	return nil, scale.ErrUnknownVaryingDataTypeValue
}

func (w WireMessage) Type() network.MessageType {
	// TODO: create a wire message type and return that
	return network.CollationMsgType
}

func (w WireMessage) Hash() (common.Hash, error) {
	// scale encode each extrinsic
	encMsg, err := w.Encode()
	if err != nil {
		return common.Hash{}, fmt.Errorf("cannot encode message: %w", err)
	}

	return common.Blake2bHash(encMsg)
}

// Encode a collator protocol message using scale encode
func (w WireMessage) Encode() ([]byte, error) {
	enc, err := scale.Marshal(w)
	if err != nil {
		return nil, err
	}
	return enc, nil
}

type ViewUpdate View

// View is a succinct representation of a peer's view. This consists of a bounded amount of chain heads
// and the highest known finalized block number.
//
// Up to `N` (5?) chain heads.
type View struct {
	// a bounded amount of chain heads
	heads []common.Hash
	// the highest known finalized number
	finalizedNumber uint32
}

type SortableHeads []common.Hash

func (s SortableHeads) Len() int {
	return len(s)
}

func (s SortableHeads) Less(i, j int) bool {
	return s[i].String() > s[j].String()
}

func (s SortableHeads) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// checkHeadsEqual checks if the heads of the view are equal to the heads of the other view.
func (v View) checkHeadsEqual(other View) bool {
	if len(v.heads) != len(other.heads) {
		return false
	}

	localHeads := v.heads
	sort.Sort(SortableHeads(localHeads))
	otherHeads := other.heads
	sort.Sort(SortableHeads(otherHeads))

	return reflect.DeepEqual(localHeads, otherHeads)
}

type ProtocolMessage struct{}
