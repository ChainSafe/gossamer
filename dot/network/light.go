package network

import (
	"fmt"
	"io"

	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/lib/common/optional"
	"github.com/ChainSafe/gossamer/lib/scale"
	"github.com/libp2p/go-libp2p-core/peer"
)

// Pair is a pair of arbitrary bytes.
type Pair struct {
	first  []byte
	second []byte
}

// LightRequest is all possible light client related requests.
type LightRequest struct {
	RmtCallRequest      RemoteCallRequest
	RmtReadRequest      RemoteReadRequest
	RmtHeaderRequest    RemoteHeaderRequest
	RmtReadChildRequest RemoteReadChildRequest
	RmtChangesRequest   RemoteChangesRequest
}

// LightResponse is all possible light client response messages.
type LightResponse struct {
	RmtCallResponse   RemoteCallResponse
	RmtReadResponse   RemoteReadResponse
	RmtHeaderResponse RemoteHeaderResponse
	RmtChangeResponse RemoteChangesResponse
}

// RemoteCallRequest ...
type RemoteCallRequest struct {
	Block  []byte
	Method string
	Data   []byte
}

// RemoteReadRequest ...
type RemoteReadRequest struct {
	Block []byte
	Keys  [][]byte
}

// RemoteReadChildRequest ...
type RemoteReadChildRequest struct {
	Block      []byte
	StorageKey []byte
	Keys       [][]byte
}

// RemoteHeaderRequest ...
type RemoteHeaderRequest struct {
	Block []byte
}

// RemoteChangesRequest ...
type RemoteChangesRequest struct {
	FirstBlock *optional.Hash
	LastBlock  *optional.Hash
	Min        []byte
	Max        []byte
	StorageKey *optional.Bytes
	key        []byte
}

// RemoteCallResponse ...
type RemoteCallResponse struct {
	Proof []byte
}

// RemoteReadResponse ...
type RemoteReadResponse struct {
	Proof []byte
}

// RemoteHeaderResponse ...
type RemoteHeaderResponse struct {
	Header []*optional.Header
	proof  []byte
}

// RemoteChangesResponse ...
type RemoteChangesResponse struct {
	Max        []byte
	Proof      [][]byte
	Roots      [][]Pair
	RootsProof []byte
}

// String formats a RemoteCallRequest as a string
func (rc *RemoteCallRequest) String() string {
	return fmt.Sprintf("Block =%s method=%s Data=%s",
		string(rc.Block), rc.Method, string(rc.Data))
}

// Encode encodes a RemoteCallRequest message using SCALE and appends the type byte to the start
func (rc *RemoteCallRequest) Encode() ([]byte, error) {
	enc, err := scale.Encode(rc)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteCallRequestType}, enc...), nil
}

// Decode the message into a RemoteCallRequest, it assumes the type byte has been removed
func (rc *RemoteCallRequest) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rc)
	return err
}

// Type returns RemoteCallRequestType
func (rc *RemoteCallRequest) Type() int {
	return RemoteCallRequestType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rc *RemoteCallRequest) IDString() string {
	return ""
}

// String formats a RemoteChangesRequest as a string
func (rc *RemoteChangesRequest) String() string {
	return fmt.Sprintf("FirstBlock =%s LastBlock=%s Min=%s Max=%s Storagekey=%s key=%s",
		rc.FirstBlock.String(),
		rc.LastBlock.String(),
		string(rc.Min),
		string(rc.Max),
		rc.StorageKey.String(),
		string(rc.key),
	)
}

// Encode encodes a RemoteChangesRequest message using SCALE and appends the type byte to the start
func (rc *RemoteChangesRequest) Encode() ([]byte, error) {
	enc, err := scale.Encode(rc)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteChangesRequestType}, enc...), nil
}

// Decode the message into a RemoteChangesRequest, it assumes the type byte has been removed
func (rc *RemoteChangesRequest) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rc)
	return err
}

// Type returns RemoteChangesRequestType
func (rc *RemoteChangesRequest) Type() int {
	return RemoteChangesRequestType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rc *RemoteChangesRequest) IDString() string {
	return ""
}

// String formats a RemoteHeaderRequest as a string
func (rh *RemoteHeaderRequest) String() string {
	return fmt.Sprintf("Block =%s", string(rh.Block))
}

// Encode encodes a RemoteHeaderRequest message using SCALE and appends the type byte to the start
func (rh *RemoteHeaderRequest) Encode() ([]byte, error) {
	enc, err := scale.Encode(rh)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteHeaderRequestType}, enc...), nil
}

// Decode the message into a RemoteHeaderRequest, it assumes the type byte has been removed
func (rh *RemoteHeaderRequest) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rh)
	return err
}

// Type returns RemoteHeaderRequestType
func (rh *RemoteHeaderRequest) Type() int {
	return RemoteHeaderRequestType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rh *RemoteHeaderRequest) IDString() string {
	return ""
}

// String formats a RemoteReadRequest as a string
func (rr *RemoteReadRequest) String() string {
	return fmt.Sprintf("Block =%s", string(rr.Block))
}

// Encode encodes a RemoteReadRequest message using SCALE and appends the type byte to the start
func (rr *RemoteReadRequest) Encode() ([]byte, error) {
	enc, err := scale.Encode(rr)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteReadRequestType}, enc...), nil
}

// Decode the message into a RemoteReadRequest, it assumes the type byte has been removed
func (rr *RemoteReadRequest) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rr)
	return err
}

// Type returns RemoteReadRequestType
func (rr *RemoteReadRequest) Type() int {
	return RemoteReadRequestType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rr *RemoteReadRequest) IDString() string {
	return ""
}

// String formats a RemoteReadChildRequest as a string
func (rr *RemoteReadChildRequest) String() string {
	var strKeys []string
	for _, v := range rr.Keys {
		strKeys = append(strKeys, string(v))
	}
	return fmt.Sprintf("Block =%s StorageKey=%s Keys=%v",
		string(rr.Block),
		string(rr.StorageKey),
		strKeys,
	)
}

// Encode encodes a RemoteReadChildRequest message using SCALE and appends the type byte to the start
func (rr *RemoteReadChildRequest) Encode() ([]byte, error) {
	enc, err := scale.Encode(rr)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteReadChildRequestType}, enc...), nil
}

// Decode the message into a RemoteReadChildRequest, it assumes the type byte has been removed
func (rr *RemoteReadChildRequest) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rr)
	return err
}

// Type returns RemoteReadChildRequestType
func (rr *RemoteReadChildRequest) Type() int {
	return RemoteReadChildRequestType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rr *RemoteReadChildRequest) IDString() string {
	return ""
}

// String formats a RemoteCallResponse as a string
func (rc *RemoteCallResponse) String() string {
	return fmt.Sprintf("Proof =%s", string(rc.Proof))
}

// Encode encodes a RemoteCallResponse message using SCALE and appends the type byte to the start
func (rc *RemoteCallResponse) Encode() ([]byte, error) {
	enc, err := scale.Encode(rc)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteCallResponseType}, enc...), nil
}

// Decode the message into a RemoteCallResponse, it assumes the type byte has been removed
func (rc *RemoteCallResponse) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rc)
	return err
}

// Type returns RemoteCallResponseType
func (rc *RemoteCallResponse) Type() int {
	return RemoteCallResponseType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rc *RemoteCallResponse) IDString() string {
	return ""
}

// String formats a RemoteChangesResponse as a string
func (rc *RemoteChangesResponse) String() string {
	var strRoots []string
	var strProof []string
	for _, v := range rc.Proof {
		strProof = append(strProof, string(v))
	}
	for _, v := range rc.Roots {
		for _, p := range v {
			strRoots = append(strRoots, string(p.first), string(p.second))
		}
	}
	return fmt.Sprintf("Max =%s Proof =%s Roots=%v RootsProof=%s",
		string(rc.Max),
		strProof,
		strRoots,
		string(rc.RootsProof),
	)
}

// Encode encodes a RemoteChangesResponse message using SCALE and appends the type byte to the start
func (rc *RemoteChangesResponse) Encode() ([]byte, error) {
	enc, err := scale.Encode(rc)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteChangesResponseType}, enc...), nil
}

// Decode the message into a RemoteChangesResponse, it assumes the type byte has been removed
func (rc *RemoteChangesResponse) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rc)
	return err
}

// Type returns RemoteChangesResponseType
func (rc *RemoteChangesResponse) Type() int {
	return RemoteChangesResponseType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rc *RemoteChangesResponse) IDString() string {
	return ""
}

// String formats a RemoteReadResponse as a string
func (rr *RemoteReadResponse) String() string {
	return fmt.Sprintf("Proof =%s", string(rr.Proof))
}

// Encode encodes a RemoteReadResponse message using SCALE and appends the type byte to the start
func (rr *RemoteReadResponse) Encode() ([]byte, error) {
	enc, err := scale.Encode(rr)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteReadResponseType}, enc...), nil
}

// Decode the message into a RemoteReadRequest, it assumes the type byte has been removed
func (rr *RemoteReadResponse) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rr)
	return err
}

// Type returns RemoteReadResponseType
func (rr *RemoteReadResponse) Type() int {
	return RemoteReadResponseType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rr *RemoteReadResponse) IDString() string {
	return ""
}

// String formats a RemoteHeaderResponse as a string
func (rh *RemoteHeaderResponse) String() string {
	return fmt.Sprintf("Header =%s Proof =%s", rh.Header, string(rh.proof))
}

// Encode encodes a RemoteHeaderResponse message using SCALE and appends the type byte to the start
func (rh *RemoteHeaderResponse) Encode() ([]byte, error) {
	enc, err := scale.Encode(rh)
	if err != nil {
		return enc, err
	}
	return append([]byte{RemoteReadResponseType}, enc...), nil
}

// Decode the message into a RemoteHeaderResponse, it assumes the type byte has been removed
func (rh *RemoteHeaderResponse) Decode(r io.Reader) error {
	sd := scale.Decoder{Reader: r}
	_, err := sd.Decode(rh)
	return err
}

// Type returns RemoteReadResponseType
func (rh *RemoteHeaderResponse) Type() int {
	return RemoteReadResponseType
}

// IDString Returns an empty string to ensure we don't rebroadcast it
func (rh *RemoteHeaderResponse) IDString() string {
	return ""
}

func remoteCallResp(peer peer.ID, req *RemoteCallRequest) (*RemoteCallResponse, error) {
	return &RemoteCallResponse{}, nil
}
func remoteChangeResp(peer peer.ID, req *RemoteChangesRequest) (*RemoteChangesResponse, error) {
	return &RemoteChangesResponse{}, nil
}
func remoteHeaderResp(peer peer.ID, req *RemoteHeaderRequest) (*RemoteCallResponse, error) {
	return &RemoteCallResponse{}, nil
}
func remoteReadChildResp(peer peer.ID, req *RemoteReadChildRequest) (*RemoteReadResponse, error) {
	return &RemoteReadResponse{}, nil
}
func remoteReadResp(peer peer.ID, req *RemoteReadRequest) (*RemoteReadResponse, error) {
	return &RemoteReadResponse{}, nil
}
