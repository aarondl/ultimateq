package inet

import (
	"bytes"
	. "launchpad.net/gocheck"
)

func (s *s) TestQueue(c *C) {
	q := Queue{}
	c.Assert(q.length, Equals, 0)
	c.Assert(q.front, IsNil)
	c.Assert(q.back, IsNil)
	c.Assert(q.mutex, NotNil)
}

func (s *s) TestQueue_Queuing(c *C) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}

	q.Enqueue()
	c.Assert(q.length, Equals, 0)
	dq := q.Dequeue(1)
	c.Assert(dq, IsNil)

	q.Enqueue(test1)
	q.Enqueue(test2, test1)

	dq1 := q.Dequeue(1)
	c.Assert(bytes.Compare(test1, dq1[0]), Equals, 0)
	dq2 := q.Dequeue(20)
	c.Assert(bytes.Compare(test2, dq2[0]), Equals, 0)
	c.Assert(bytes.Compare(test1, dq2[1]), Equals, 0)
}

func (s *s) TestQueue_queue(c *C) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}
	q.enqueue(&test1)
	c.Assert(q.length, Equals, 1)
	c.Assert(q.front, Equals, q.back)
	q.enqueue(&test2)
	c.Assert(q.length, Equals, 2)
	c.Assert(q.front, Not(Equals), q.back)

	c.Assert(bytes.Compare(*q.front.data, test1), Equals, 0)
	c.Assert(bytes.Compare(*q.front.next.data, test2), Equals, 0)
}

func (s *s) TestQueue_dequeue(c *C) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}
	c.Assert(q.dequeue(), IsNil)

	q.enqueue(&test1)
	q.enqueue(&test2)

	c.Assert(q.front, Not(Equals), q.back)
	dq1 := *q.dequeue()
	c.Assert(bytes.Compare(test1, dq1), Equals, 0)
	c.Assert(q.front, Equals, q.back)
	dq2 := *q.dequeue()
	c.Assert(bytes.Compare(test2, dq2), Equals, 0)
	c.Assert(q.front, IsNil)
	c.Assert(q.back, IsNil)
}
