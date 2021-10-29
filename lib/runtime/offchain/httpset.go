package offchain

import (
	"errors"
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
	list []*http.Request
}

// OnceHTTPSet
func OnceHTTPSet() *Set {
	once.Do(func() {
		HTTPSet = &Set{
			mtx:  new(sync.Mutex),
			list: make([]*http.Request, 0),
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

	p.list = append(p.list, req)
	return int16(len(p.list) - 1), nil
}

func (p *Set) ExecRequest(id int) error {
	if len(p.list) <= id {
		return errors.New("http list does not contains id %v")
	}

	req := p.list[id]

	client := new(http.Client)
	_, err := client.Do(req)
	return err
}

// Remove just remove a expecific request from list
func (p *Set) Remove(id int) {
	p.mtx.Lock()
	defer p.mtx.Unlock()

	p.list = append(p.list[:id], p.list[id+1:]...)
}

func (p *Set) Get(id int) *http.Request {
	if len(p.list) <= id {
		return nil
	}

	return p.list[id]
}
