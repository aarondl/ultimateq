package inet

// queueNode is the node structure underneath the Queue type.
type queueNode struct {
	next *queueNode
	data *[]byte
}

// Queue implements a singly-linked queue data structure for byte slices.
type Queue struct {
	front  *queueNode
	back   *queueNode
	length int
}

// Enqueue adds the byte slice to the queue
func (q *Queue) Enqueue(bytes []byte) {
	if len(bytes) == 0 {
		return
	}

	node := &queueNode{data: &bytes}

	if q.length == 0 {
		q.front = node
		q.back = q.front
	} else {
		q.back.next = node
		q.back = node
	}

	q.length++
}

// Dequeue dequeues from the front of the queue.
func (q *Queue) Dequeue() []byte {
	if q.length == 0 {
		return nil
	}
	data := q.front.data
	q.front = q.front.next
	if q.length == 1 {
		q.back = nil
	}
	q.length--

	return *data
}
