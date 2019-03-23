package inet

import (
	"bytes"
	"testing"
)

func TestQueue(t *testing.T) {
	q := Queue{}
	if exp, got := q.length, 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got := q.front; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if got := q.back; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
}

func TestQueue_Queuing(t *testing.T) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}

	q.Enqueue(test1)
	q.Enqueue(test2)

	dq1 := q.Dequeue()
	if exp, got := bytes.Compare(test1, dq1), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	dq2 := q.Dequeue()
	if exp, got := bytes.Compare(test2, dq2), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestQueue_queue(t *testing.T) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}
	q.Enqueue(nil) // Should be consequenceless test cov
	q.Enqueue(test1)
	if exp, got := q.length, 1; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := q.front, q.back; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	q.Enqueue(test2)
	if exp, got := q.length, 2; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := q.front, q.back; exp == got {
		t.Errorf("Did not want: %v, got: %v", exp, got)
	}

	if exp, got := bytes.Compare(*q.front.data, test1), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := bytes.Compare(*q.front.next.data, test2), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
}

func TestQueue_dequeue(t *testing.T) {
	test1 := []byte{1, 2, 3}
	test2 := []byte{4, 5, 6}

	q := Queue{}
	if got := q.Dequeue(); got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}

	q.Enqueue(test1)
	q.Enqueue(test2)

	if exp, got := q.front, q.back; exp == got {
		t.Errorf("Did not want: %v, got: %v", exp, got)
	}
	dq1 := q.Dequeue()
	if exp, got := bytes.Compare(test1, dq1), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if exp, got := q.front, q.back; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	dq2 := q.Dequeue()
	if exp, got := bytes.Compare(test2, dq2), 0; exp != got {
		t.Errorf("Expected: %v, got: %v", exp, got)
	}
	if got := q.front; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
	if got := q.back; got != nil {
		t.Errorf("Expected: %v to be nil.", got)
	}
}
