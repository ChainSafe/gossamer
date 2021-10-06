package network

import (
	"fmt"
	"github.com/ChainSafe/gossamer/dot/types"
	"github.com/ChainSafe/gossamer/lib/common"
	"github.com/ChainSafe/gossamer/pkg/scale"
)

// Pair is a pair of arbitrary bytes.
type Pair struct {
	First  []byte
	Second []byte
}

// LightRequest is all possible light client related requests.
type LightRequest struct {
	RmtCallRequest      *RemoteCallRequest
	RmtReadRequest      *RemoteReadRequest
	RmtHeaderRequest    *RemoteHeaderRequest
	RmtReadChildRequest *RemoteReadChildRequest
	RmtChangesRequest   *RemoteChangesRequest
}

// Request is a scale compatible struct containing all possible light client related requests.
type Request struct {
	RmtCallRequest      RemoteCallRequest
	RmtReadRequest      RemoteReadRequest
	RmtHeaderRequest    RemoteHeaderRequest
	RmtReadChildRequest RemoteReadChildRequest
	RmtChangesRequest   RemoteChangesRequest
}

// NewLightRequest returns a new LightRequest
func NewLightRequest() *LightRequest {
	rcr := newRemoteChangesRequest()
	return &LightRequest{
		RmtCallRequest:      newRemoteCallRequest(),
		RmtReadRequest:      newRemoteReadRequest(),
		RmtHeaderRequest:    newRemoteHeaderRequest(),
		RmtReadChildRequest: newRemoteReadChildRequest(),
		RmtChangesRequest:   &rcr,
	}
}

func newRequest() *Request {
	return &Request{
		RmtCallRequest:      *newRemoteCallRequest(),
		RmtReadRequest:      *newRemoteReadRequest(),
		RmtHeaderRequest:    *newRemoteHeaderRequest(),
		RmtReadChildRequest: *newRemoteReadChildRequest(),
		RmtChangesRequest:   newRemoteChangesRequest(),
	}
}

// SubProtocol returns the light sub-protocol
func (l *LightRequest) SubProtocol() string {
	return lightID
}

// Encode encodes a LightRequest message using SCALE and appends the type byte to the start
func (l *LightRequest) Encode() ([]byte, error) {
	req := Request{
		RmtCallRequest:      *l.RmtCallRequest,
		RmtReadRequest:      *l.RmtReadRequest,
		RmtHeaderRequest:    *l.RmtHeaderRequest,
		RmtReadChildRequest: *l.RmtReadChildRequest,
		RmtChangesRequest:   *l.RmtChangesRequest,
	}
	return scale.Marshal(req)
}

// Decode the message into a LightRequest, it assumes the type byte has been removed
func (l *LightRequest) Decode(in []byte) error {
	msg := newRequest()
	err := scale.Unmarshal(in, msg)
	if err != nil {
		return err
	}

	l.RmtCallRequest = &msg.RmtCallRequest
	l.RmtReadRequest = &msg.RmtReadRequest
	l.RmtHeaderRequest = &msg.RmtHeaderRequest
	l.RmtReadChildRequest = &msg.RmtReadChildRequest
	l.RmtChangesRequest = &msg.RmtChangesRequest
	return nil
}

// String formats a LightRequest as a string
func (l LightRequest) String() string {
	return fmt.Sprintf(
		"RemoteCallRequest=%s RemoteReadRequest=%s RemoteHeaderRequest=%s "+
			"RemoteReadChildRequest=%s RemoteChangesRequest=%s",
		l.RmtCallRequest, l.RmtReadRequest, l.RmtHeaderRequest, l.RmtReadChildRequest, l.RmtChangesRequest)
}

// LightResponse is all possible light client response messages.
// TODO Utilize custom VDT types in the header or this won't decode correctly
type LightResponseNew struct {
	RmtCallResponse   *RemoteCallResponse
	RmtReadResponse   *RemoteReadResponse
	RmtHeaderResponse *RemoteHeaderResponse
	RmtChangeResponse *RemoteChangesResponse
}

// LightResponse is all possible light client response messages.
type Response struct {
	RmtCallResponse   RemoteCallResponse
	RmtReadResponse   RemoteReadResponse
	RmtHeaderResponse RemoteHeaderResponse
	RmtChangeResponse RemoteChangesResponse
}

// NewLightResponse returns a new LightResponse
func NewLightResponse() *LightResponseNew {
	return &LightResponseNew{
		RmtCallResponse:   newRemoteCallResponse(),
		RmtReadResponse:   newRemoteReadResponse(),
		RmtHeaderResponse: newRemoteHeaderResponse(),
		RmtChangeResponse: newRemoteChangesResponse(),
	}
}

func newResponse() *Response {
	return &Response{
		RmtCallResponse:   *newRemoteCallResponse(),
		RmtReadResponse:   *newRemoteReadResponse(),
		RmtHeaderResponse: *newRemoteHeaderResponse(),
		RmtChangeResponse: *newRemoteChangesResponse(),
	}
}

// SubProtocol returns the light sub-protocol
func (l *LightResponseNew) SubProtocol() string {
	return lightID
}

// Encode encodes a LightResponse message using SCALE and appends the type byte to the start
func (l *LightResponseNew) Encode() ([]byte, error) {
	response := Response{
		RmtCallResponse:   *l.RmtCallResponse,
		RmtReadResponse:   *l.RmtReadResponse,
		RmtHeaderResponse: *l.RmtHeaderResponse,
		RmtChangeResponse: *l.RmtChangeResponse,
	}
	return scale.Marshal(response)
}

// Decode the message into a LightResponse, it assumes the type byte has been removed
func (l *LightResponseNew) Decode(in []byte) error {
	msg := newResponse()
	err := scale.Unmarshal(in, msg)
	if err != nil {
		return err
	}

	l.RmtCallResponse = &msg.RmtCallResponse
	l.RmtReadResponse = &msg.RmtReadResponse
	l.RmtHeaderResponse = &msg.RmtHeaderResponse
	l.RmtChangeResponse = &msg.RmtChangeResponse
	return nil
}

// String formats a RemoteReadRequest as a string
func (l LightResponseNew) String() string {
	return fmt.Sprintf(
		"RemoteCallResponse=%s RemoteReadResponse=%s RemoteHeaderResponse=%s RemoteChangesResponse=%s",
		l.RmtCallResponse, l.RmtReadResponse, l.RmtHeaderResponse, l.RmtChangeResponse)
}

// RemoteCallRequest ...
type RemoteCallRequest struct {
	Block  []byte
	Method string
	Data   []byte
}

func newRemoteCallRequest() *RemoteCallRequest{
	return &RemoteCallRequest{
		Block: []byte{},
		Method: "",
		Data: []byte{},
	}
}

// RemoteReadRequest ...
type RemoteReadRequest struct {
	Block []byte
	Keys  [][]byte
}

func newRemoteReadRequest() *RemoteReadRequest{
	return &RemoteReadRequest{
		Block: []byte{},
	}
}

// RemoteReadChildRequest ...
type RemoteReadChildRequest struct {
	Block      []byte
	StorageKey []byte
	Keys       [][]byte
}

func newRemoteReadChildRequest() *RemoteReadChildRequest{
	return &RemoteReadChildRequest{
		Block: []byte{},
		StorageKey: []byte{},
	}
}

// RemoteHeaderRequest ...
type RemoteHeaderRequest struct {
	Block []byte
}

func newRemoteHeaderRequest() *RemoteHeaderRequest{
	return &RemoteHeaderRequest{
		Block: []byte{},
	}
}

// RemoteChangesRequest ...
type RemoteChangesRequest struct {
	FirstBlock *common.Hash
	LastBlock  *common.Hash
	Min        []byte
	Max        []byte
	StorageKey *[]byte
	key        []byte
}

func newRemoteChangesRequest() RemoteChangesRequest {
	return RemoteChangesRequest{
		FirstBlock: nil,
		LastBlock:  nil,
		Min:        []byte{},
		Max:        []byte{},
		StorageKey: nil,
	}
}

// RemoteCallResponse ...
type RemoteCallResponse struct {
	Proof []byte
}

func newRemoteCallResponse() *RemoteCallResponse{
	return &RemoteCallResponse{
		Proof: []byte{},
	}
}

// RemoteReadResponse ...
type RemoteReadResponse struct {
	Proof []byte
}

func newRemoteReadResponse() *RemoteReadResponse{
	return &RemoteReadResponse{
		Proof: []byte{},
	}
}

// RemoteHeaderResponse ...
type RemoteHeaderResponse struct {
	Header []*types.Header
	proof  []byte
}

func newRemoteHeaderResponse() *RemoteHeaderResponse{
	return &RemoteHeaderResponse{
		Header: nil,
	}
}

// RemoteChangesResponse ...
type RemoteChangesResponse struct {
	Max        []byte
	Proof      [][]byte
	Roots      [][]Pair
	RootsProof []byte
}

func newRemoteChangesResponse() *RemoteChangesResponse{
	return &RemoteChangesResponse{
		Max: []byte{},
		RootsProof: []byte{},
	}
}

// String formats a RemoteCallRequest as a string
func (rc *RemoteCallRequest) String() string {
	return fmt.Sprintf("Block =%s method=%s Data=%s",
		string(rc.Block), rc.Method, string(rc.Data))
}

// String formats a RemoteChangesRequest as a string
func (rc *RemoteChangesRequest) String() string {
	first := common.Hash{}
	last := common.Hash{}
	storageKey := []byte{0}
	if rc.FirstBlock != nil {
		first = *rc.FirstBlock
	}
	if rc.LastBlock != nil {
		last = *rc.LastBlock
	}
	if rc.StorageKey != nil {
		storageKey = *rc.StorageKey
	}
	return fmt.Sprintf("FirstBlock =%s LastBlock=%s Min=%s Max=%s Storagekey=%s key=%s",
		first,
		last,
		string(rc.Min),
		string(rc.Max),
		storageKey,
		string(rc.key),
	)
}

// String formats a RemoteHeaderRequest as a string
func (rh *RemoteHeaderRequest) String() string {
	return fmt.Sprintf("Block =%s", string(rh.Block))
}

// String formats a RemoteReadRequest as a string
func (rr *RemoteReadRequest) String() string {
	return fmt.Sprintf("Block =%s", string(rr.Block))
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

// String formats a RemoteCallResponse as a string
func (rc *RemoteCallResponse) String() string {
	return fmt.Sprintf("Proof =%s", string(rc.Proof))
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
			strRoots = append(strRoots, string(p.First), string(p.Second))
		}
	}
	return fmt.Sprintf("Max =%s Proof =%s Roots=%v RootsProof=%s",
		string(rc.Max),
		strProof,
		strRoots,
		string(rc.RootsProof),
	)
}

// String formats a RemoteReadResponse as a string
func (rr *RemoteReadResponse) String() string {
	return fmt.Sprintf("Proof =%s", string(rr.Proof))
}

// String formats a RemoteHeaderResponse as a string
func (rh *RemoteHeaderResponse) String() string {
	return fmt.Sprintf("Header =%+v Proof =%s", rh.Header, string(rh.proof))
}

func remoteCallResp(_ *RemoteCallRequest) (*RemoteCallResponse, error) {
	return &RemoteCallResponse{}, nil
}
func remoteChangeResp(_ *RemoteChangesRequest) (*RemoteChangesResponse, error) {
	return &RemoteChangesResponse{}, nil
}
func remoteHeaderResp(_ *RemoteHeaderRequest) (*RemoteHeaderResponse, error) {
	return &RemoteHeaderResponse{}, nil
}
func remoteReadChildResp(_ *RemoteReadChildRequest) (*RemoteReadResponse, error) {
	return &RemoteReadResponse{}, nil
}
func remoteReadResp(_ *RemoteReadRequest) (*RemoteReadResponse, error) {
	return &RemoteReadResponse{}, nil
}
