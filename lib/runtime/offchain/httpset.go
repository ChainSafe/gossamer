// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package offchain

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

const maxConcurrentRequests = 1000

var (
	errIntBufferEmpty        = errors.New("int buffer exhausted")
	errIntBufferFull         = errors.New("int buffer is full")
	errRequestIDNotAvailable = errors.New("request id not available")
	errRequestInvalid        = errors.New("request is invalid")
	errRequestAlreadyStarted = errors.New("request has already started")
	errInvalidHeaderKey      = errors.New("invalid header key")
)

// requestIDBuffer created to control the amount of available non-duplicated ids
type requestIDBuffer chan int16

// newIntBuffer creates the request id buffer starting from 1 till @buffSize (by default @buffSize is 1000)
func newIntBuffer(buffSize int16) requestIDBuffer {
	b := make(chan int16, buffSize)
	for i := int16(1); i <= buffSize; i++ {
		b <- i
	}

	return b
}

func (b requestIDBuffer) get() (int16, error) {
	select {
	case v := <-b:
		return v, nil
	default:
		return 0, errIntBufferEmpty
	}
}

func (b requestIDBuffer) put(i int16) error {
	select {
	case b <- i:
		return nil
	default:
		return errIntBufferFull
	}
}

// Request holds the request object and update the invalid and waiting status whenever
// the request starts or is waiting to be read
type Request struct {
	Request          *http.Request
	invalid, waiting bool
}

// AddHeader adds a new HTTP header into request property, only if request is valid and has not started yet
func (r *Request) AddHeader(name, value string) error {
	if r.invalid {
		return errRequestInvalid
	}

	if r.waiting {
		return errRequestAlreadyStarted
	}

	name = strings.TrimSpace(name)
	if len(name) == 0 {
		return fmt.Errorf("%w: empty header key", errInvalidHeaderKey)
	}

	r.Request.Header.Add(name, value)
	return nil
}

// HTTPSet holds a pool of concurrent http request calls
type HTTPSet struct {
	*sync.Mutex
	reqs   map[int16]*Request
	idBuff requestIDBuffer
}

// NewHTTPSet creates a offchain http set that can be used
// by runtime as HTTP clients, the max concurrent requests is 1000
func NewHTTPSet() *HTTPSet {
	return &HTTPSet{
		new(sync.Mutex),
		make(map[int16]*Request),
		newIntBuffer(maxConcurrentRequests),
	}
}

// StartRequest create a new request using the method and the uri, adds the request into the list
// and then return the position of the request inside the list
func (p *HTTPSet) StartRequest(method, uri string) (int16, error) {
	p.Lock()
	defer p.Unlock()

	id, err := p.idBuff.get()
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

	p.reqs[id] = &Request{
		Request: req,
	}

	return id, nil
}

// Remove just remove a expecific request from reqs
func (p *HTTPSet) Remove(id int16) error {
	p.Lock()
	defer p.Unlock()

	delete(p.reqs, id)

	return p.idBuff.put(id)
}

// Get returns a request or nil if request not found
func (p *HTTPSet) Get(id int16) *Request {
	p.Lock()
	defer p.Unlock()

	return p.reqs[id]
}
