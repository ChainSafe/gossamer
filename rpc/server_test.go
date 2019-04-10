package rpc

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"testing"
)


// ------------- Example Service -----------------------

type ServiceRequest struct {
	N int
}

type ServiceResponse struct {
	Result int
}

type Service struct {}

var ErrResponse = errors.New("error response")

func (s *Service) Echo(r *http.Request, req *ServiceRequest, res *ServiceResponse) error {
	log.Printf("ECHO -- Got N: %d", req.N)
	res.Result = req.N
	return nil
}

func (s *Service) Fail(r *http.Request, req *ServiceRequest, res *ServiceResponse) error {
	return ErrResponse
}

// -------------------------------------------------------

// --------------- Mock Codec ------------------------------

type MockCodec struct {
	N int
}

func (c MockCodec) NewRequest(r *http.Request) CodecRequest {
	return MockCodecRequest{c.N}
}

type MockCodecRequest struct {
	N int
}

func (r MockCodecRequest) Method() (string, error) {
	return "Service.Echo", nil
}

func (r MockCodecRequest) ReadRequest(args interface{}) error {
	req := args.(*ServiceRequest)
	req.N = r.N
	return nil
}

func (r MockCodecRequest) WriteResponse(w http.ResponseWriter, reply interface{}) {
	res := reply.(*ServiceResponse)
	w.Write([]byte(strconv.Itoa(res.Result)))
}

func (r MockCodecRequest) WriteError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	w.Write([]byte(err.Error()))
}

type MockResponseWriter struct {
	header http.Header
	Status int
	Body string
}

func NewMockResponseWriter() *MockResponseWriter {
	header := make(http.Header)
	return &MockResponseWriter{header:header}
}

func (w *MockResponseWriter) Header() http.Header {
	return w.header
}

func (w *MockResponseWriter) Write(p []byte) (int, error) {
	w.Body = string(p)
	if w.Status == 0 {
		w.Status = 200
	}
	return len(p), nil
}

func (w *MockResponseWriter) WriteHeader (status int) {
	w.Status = status
}

func TestServeHTTP(t *testing.T) {
	s := NewServer()
	s.RegisterService(new(Service), "")
	s.RegisterCodec(MockCodec{10})
	r, err := http.NewRequest("POST", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/json")
	w := NewMockResponseWriter()
	s.ServeHTTP(w, r)
	if w.Status != 200 {
		t.Errorf("unexpected status. got: %d expected: %d", w.Status, 200)
	}
	if w.Body != strconv.Itoa(10) {
		t.Errorf("unexpected body content. got: %s expected %s", w.Body, strconv.Itoa(10))
	}
}




