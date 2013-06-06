package inet

import (
	"bytes"
	. "launchpad.net/gocheck"
)

func (s *s) TestQueue(c *C) {
	q := Queue{}
	c.Check(q.length, Equals, 0)
	c.Check(q.front, IsNil)
	c.Check(q.back, IsNil)
}

func (s *s) TestQueue_Queuing(c *C) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}

	q.Enqueue(test1)
	q.Enqueue(test2)

	dq1 := q.Dequeue()
	c.Check(bytes.Compare(test1, dq1), Equals, 0)
	dq2 := q.Dequeue()
	c.Check(bytes.Compare(test2, dq2), Equals, 0)
}

func (s *s) TestQueue_queue(c *C) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}
	q.Enqueue(nil) // Should be consequenceless test cov
	q.Enqueue(test1)
	c.Check(q.length, Equals, 1)
	c.Check(q.front, Equals, q.back)
	q.Enqueue(test2)
	c.Check(q.length, Equals, 2)
	c.Check(q.front, Not(Equals), q.back)

	c.Check(bytes.Compare(*q.front.data, test1), Equals, 0)
	c.Check(bytes.Compare(*q.front.next.data, test2), Equals, 0)
}

func (s *s) TestQueue_dequeue(c *C) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}
	c.Check(q.Dequeue(), IsNil)

	q.Enqueue(test1)
	q.Enqueue(test2)

	c.Check(q.front, Not(Equals), q.back)
	dq1 := q.Dequeue()
	c.Check(bytes.Compare(test1, dq1), Equals, 0)
	c.Check(q.front, Equals, q.back)
	dq2 := q.Dequeue()
	c.Check(bytes.Compare(test2, dq2), Equals, 0)
	c.Check(q.front, IsNil)
	c.Check(q.back, IsNil)
}
