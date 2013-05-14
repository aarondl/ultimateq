package inet

import (
	"sync"
)

// queueNode is the node structure underneath the Queue type.
type queueNode struct {
	next *queueNode
	data *[]byte
}

// Queue implements a singly-linked queue data structure for byte slices.
// Due to the intention of having goroutines access it, it's sync-locked.
// It is not meant to be used as a generic re-usable container.
type Queue struct {
	front  *queueNode
	back   *queueNode
	length int
	mutex  sync.Mutex
}

// Enqueue copies the byte slices (because of goroutines a copy must be made)
// lockes the mutex, and adds the newly allocated copies. To avoid lock thrash
// when a large queue is made it copies all byte slices outside of the mutex
// lock. This requires an extra allocation.
func (q *Queue) Enqueue(slices ...[]byte) {
	if len(slices) == 0 {
		return
	}

	if len(slices) == 1 {
		q.mutex.Lock()
		q.enqueue(&slices[0])
		q.mutex.Unlock()
		return
	}

	copies := make([]*[]byte, len(slices))
	for i, v := range slices {
		cpy := make([]byte, len(v))
		copy(cpy, v)
		copies[i] = &cpy
	}

	q.mutex.Lock()
	for i := 0; i < len(copies); i++ {
		q.enqueue(copies[i])
	}
	q.mutex.Unlock()
}

// queue is the function that actually updates the internal structure of the
// queue to reflect the enqueue; adjusts length, front/back ptrs etc.
func (q *Queue) enqueue(slice *[]byte) {
	node := &queueNode{data: slice}

	if q.length == 0 {
		q.front = node
		q.back = q.front
	} else {
		q.back.next = node
		q.back = node
	}

	q.length++
}

// Dequeue takes a number of elements to dequeue and returns them in a slice of
// byte slices.
func (q *Queue) Dequeue(n int) [][]byte {
	q.mutex.Lock()
	if n > q.length {
		n = q.length
	}

	if 0 == n {
		q.mutex.Unlock()
		return nil
	}

	data := make([][]byte, n)

	for i := 0; i < n; i++ {
		data[i] = *q.dequeue()
	}
	q.mutex.Unlock()

	return data
}

// dequeue is the function that updates the internal structure of the queue
// reflect the dequeue; adjusts length, front/back ptrs etc.
func (q *Queue) dequeue() *[]byte {
	if q.length == 0 {
		return nil
	}

	data := q.front.data
	q.front = q.front.next
	if q.length == 1 {
		q.back = nil
	}
	q.length--

	return data
}
