package jobs

import "testing"

func TestQueueRejectsWhenFull(t *testing.T) {
	q := NewQueue(1)
	if err := q.Enqueue("a"); err != nil {
		t.Fatalf("enqueue first: %v", err)
	}
	if err := q.Enqueue("b"); err != ErrQueueFull {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
}
