package offchain

import (
	"errors"
	"net/http"
	"sync"
)

const maxConcurrentRequests = 1000

var (
	errIntBufferEmpty        = errors.New("int buffer exhausted")
	errIntBufferFull         = errors.New("int buffer is full")
	errRequestIDNotAvailable = errors.New("request id not available")
)

type requestIDBuffer chan int16

func newIntBuffer(buffSize int16) *requestIDBuffer {
	b := make(chan int16, buffSize)
	for i := int16(0); i < buffSize; i++ {
		b <- i
	}

	intb := requestIDBuffer(b)
	return &intb
}

func (b *requestIDBuffer) Get() (int16, error) {
	select {
	case v := <-*b:
		return v, nil
	default:
		return 0, errIntBufferEmpty
	}
}

func (b *requestIDBuffer) Put(i int16) error {
	select {
	case *b <- i:
		return nil
	default:
		return errIntBufferFull
	}
}

type Set struct {
	mtx    *sync.Mutex
	reqs   map[int16]*http.Request
	idBuff *requestIDBuffer
}

func NewSet() *Set {
	return &Set{
		mtx:    new(sync.Mutex),
		reqs:   make(map[int16]*http.Request),
		idBuff: newIntBuffer(maxConcurrentRequests),
	}
}

// StartRequest create a new request using the method and the uri, adds the request into the list
// and then return the position of the request inside the list
func (p *Set) StartRequest(method, uri string) (int16, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	id, err := p.idBuff.Get()
	if err != nil {
		return 0, err
	}

	if _, ok := p.reqs[id]; ok {
		return 0, errRequestIDNotAvailable
	}

	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return 0, err
	}

	p.reqs[id] = req
	return id, nil
}

// Remove just remove a expecific request from reqs
func (p *Set) Remove(id int16) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	delete(p.reqs, id)
}

// Get returns a request or nil if request not found
func (p *Set) Get(id int16) *http.Request {
	return p.reqs[id]
}
