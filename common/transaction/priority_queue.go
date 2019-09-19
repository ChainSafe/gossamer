package transaction

type PriorityQueue struct {
	head *node
}

type node struct {
	data   *ValidTransaction
	parent *node
	child  *node
}

func (q *PriorityQueue) Pop() *ValidTransaction {
	head := q.head
	q.head = head.child
	return head.data
}

func (q *PriorityQueue) Insert(vt *ValidTransaction) {
	curr := q.head
	if curr == nil {
		q.head = &node{data: vt}
		return
	}

	for {
		currPriority := curr.data.validity.priority
		if vt.validity.priority > currPriority {
			newNode := &node{
				data:   vt,
				parent: curr.parent,
				child:  curr,
			}

			if curr.parent == nil {
				q.head = newNode
			} else {
				curr.parent.child = newNode
			}
			curr.parent = newNode

			return
		}
		curr = curr.child
	}
}
