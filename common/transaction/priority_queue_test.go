package transaction

import (
	"reflect"
	"testing"
)

func TestPriorityQueue(t *testing.T) {
	a := &ValidTransaction{
		validity: Validity{priority: 1},
	}
	b := &ValidTransaction{
		validity: Validity{priority: 3},
	}
	c := &ValidTransaction{
		validity: Validity{priority: 2},
	}
	d := &ValidTransaction{
		validity: Validity{priority: 17},
	}
	e := &ValidTransaction{
		validity: Validity{priority: 2},
	}

	pq := new(PriorityQueue)
	pq.Insert(a)
	pq.Insert(b)
	pq.Insert(c)
	pq.Insert(d)
	pq.Insert(e)

	n := pq.Pop()
	if !reflect.DeepEqual(n, d) {
		t.Fatalf("Fail: got %v expected %v", n, d)
	}

	n = pq.Pop()
	if !reflect.DeepEqual(n, b) {
		t.Fatalf("Fail: got %v expected %v", n, b)
	}

	n = pq.Pop()
	if !reflect.DeepEqual(n, c) {
		t.Fatalf("Fail: got %v expected %v", n, c)
	}

	n = pq.Pop()
	if !reflect.DeepEqual(n, e) {
		t.Fatalf("Fail: got %v expected %v", n, e)
	}

	n = pq.Pop()
	if !reflect.DeepEqual(n, a) {
		t.Fatalf("Fail: got %v expected %v", n, a)
	}
}
