// Copyright 2021 ChainSafe Systems (ON)
// SPDX-License-Identifier: LGPL-3.0-only

package transaction

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPriorityQueue(t *testing.T) {
	tests := []*ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &Validity{Priority: 1},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &Validity{Priority: 4},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &Validity{Priority: 17},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &Validity{Priority: 2},
		},
	}

	pq := NewPriorityQueue()
	expected := []int{3, 1, 2, 4, 0}

	for _, node := range tests {
		pq.Push(node)
	}

	for i, exp := range expected {
		n := pq.Pop()
		if !reflect.DeepEqual(n, tests[exp]) {
			t.Log(n.Validity)
			t.Log(tests[exp].Validity)
			t.Fatalf("Fail: iteration %d got %v expected %v", i, n, tests[exp])
		}
	}
}

func TestPriorityQueueAgain(t *testing.T) {
	tests := []*ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &Validity{Priority: 3},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &Validity{Priority: 3},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &Validity{Priority: 1},
		},
	}

	pq := NewPriorityQueue()
	expected := []int{1, 3, 0, 2, 4}

	for _, node := range tests {
		pq.Push(node)
	}

	for i, exp := range expected {
		n := pq.Pop()
		if !reflect.DeepEqual(n, tests[exp]) {
			t.Fatalf("Fail: iteration %d got %v expected %v", i, n, tests[exp])
		}
	}
}

func TestPeek_Empty(t *testing.T) {
	pq := NewPriorityQueue()
	vt := pq.Peek()
	if vt != nil {
		t.Fatalf("Fail: expected nil for empty queue")
	}
}

func TestPriorityQueue_Pop(t *testing.T) {
	pq := NewPriorityQueue()

	val := pq.Pop()

	if val != nil {
		t.Errorf("pop on empty list should return nil")
	}
	val = pq.Peek()
	if val != nil {
		t.Errorf("pop on empty list should return nil")
	}

	pq.Push(&ValidTransaction{
		Extrinsic: []byte{},
		Validity:  new(Validity),
	})

	peek := pq.Peek()
	if peek == nil {
		t.Errorf("expected item, got nil Peek()")
	}

	pop := pq.Pop()
	if pop == nil {
		t.Errorf("expected item, got nil for Pop()")
	}

	if !reflect.DeepEqual(peek, pop) {
		t.Error("Peek() did not return the same value as Pop()")
	}
}

func TestPeek(t *testing.T) {
	tests := []*ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &Validity{Priority: 3},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &Validity{Priority: 3},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &Validity{Priority: 1},
		},
	}

	pq := NewPriorityQueue()
	expected := []int{1, 3, 0, 2, 4}

	for _, node := range tests {
		pq.Push(node)
	}

	for _, exp := range expected {
		n := pq.Peek()
		if !reflect.DeepEqual(n, tests[exp]) {
			t.Fatalf("Fail: got %v expected %v", n, tests[exp])
		}
		pq.Pop()
	}
}

func TestPriorityQueueConcurrentCalls(_ *testing.T) {
	pq := NewPriorityQueue()

	const parallelism = 2

	var startWg, endWg sync.WaitGroup
	startWg.Add(parallelism)
	endWg.Add(parallelism)

	timedOut := make(chan struct{})
	go func() {
		startWg.Wait()
		const duration = 100 * time.Millisecond
		timer := time.NewTimer(duration)
		<-timer.C
		close(timedOut)
	}()

	for i := 0; i < parallelism; i++ {
		go func() {
			defer endWg.Done()
			startWg.Done()
			for {
				select {
				case <-timedOut:
					return
				default:
				}
				_, _ = pq.Push(&ValidTransaction{Validity: &Validity{Priority: 1}})
				_ = pq.Peek()
				_ = pq.Pop()
			}
		}()
	}

	endWg.Wait()
}

func TestPending(t *testing.T) {
	tests := []*ValidTransaction{
		{
			Extrinsic: []byte("a"),
			Validity:  &Validity{Priority: 5},
		},
		{
			Extrinsic: []byte("b"),
			Validity:  &Validity{Priority: 4},
		},
		{
			Extrinsic: []byte("c"),
			Validity:  &Validity{Priority: 3},
		},
		{
			Extrinsic: []byte("d"),
			Validity:  &Validity{Priority: 2},
		},
		{
			Extrinsic: []byte("e"),
			Validity:  &Validity{Priority: 1},
		},
	}

	pq := NewPriorityQueue()

	for _, node := range tests {
		pq.Push(node)
	}

	pending := pq.Pending()
	if !reflect.DeepEqual(pending, tests) {
		t.Fatalf("Fail: got %v expected %v", pending, tests)
	}
}

func TestRemoveExtrinsic(t *testing.T) {
	tests := []*ValidTransaction{
		{
			Extrinsic: []byte("rats"),
			Validity:  &Validity{Priority: 5},
		},
		{
			Extrinsic: []byte("arecool"),
			Validity:  &Validity{Priority: 4},
		},
	}

	pq := NewPriorityQueue()

	for _, node := range tests {
		pq.Push(node)
	}

	pq.RemoveExtrinsic(tests[0].Extrinsic)

	res := pq.Pop()
	if !reflect.DeepEqual(res, tests[1]) {
		t.Fatalf("Fail: got %v expected %v", res, tests[1])
	}
}

func TestPopChannel(t *testing.T) {
	pq := NewPriorityQueue()
	slotTimer := time.NewTimer(time.Second)

	popChan := pq.PopChannel(slotTimer)

	go func() {
		counter := 0
		for {
			time.Sleep(time.Millisecond * 100)
			pq.Push(&ValidTransaction{Validity: &Validity{Priority: uint64(counter)}})
			counter++
		}
	}()

	counter := 0
	for txn := range popChan {
		assert.Equal(t, uint64(counter), txn.Validity.Priority)
		counter++
	}
	assert.Equal(t, 9, counter)
}
