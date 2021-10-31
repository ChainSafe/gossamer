package offchain

import (
	"net/http"
	"sync"
)

var (
	once    sync.Once
	HTTPSet *Set

	_ = OnceHTTPSet()
)

type Set struct {
	mtx  *sync.Mutex
	reqs []*http.Request
}

// OnceHTTPSet
func OnceHTTPSet() *Set {
	once.Do(func() {
		HTTPSet = &Set{
			mtx:  new(sync.Mutex),
			reqs: make([]*http.Request, 0),
		}
	})

	return HTTPSet
}

// StartRequest create a new request using the method and the uri, adds the request into the list
// and then return the position of the request inside the list
func (p *Set) StartRequest(method, uri string) (int16, error) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	req, err := http.NewRequest(method, uri, nil)
	if err != nil {
		return 0, err
	}

	p.reqs = append(p.reqs, req)
	return int16(len(p.reqs) - 1), nil
}

// Remove just remove a expecific request from reqs
func (p *Set) Remove(id int) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.reqs = append(p.reqs[:id], p.reqs[id+1:]...)
}

func (p *Set) Get(id int) *http.Request {
	if len(p.reqs) <= id {
		return nil
	}

	return p.reqs[id]
}
