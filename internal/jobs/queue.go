package jobs

import (
	"errors"
)

var ErrQueueFull = errors.New("job queue is full")

type Queue struct {
	ch chan string
}

func NewQueue(size int) *Queue {
	if size <= 0 {
		size = 256
	}
	return &Queue{ch: make(chan string, size)}
}

func (q *Queue) Enqueue(path string) error {
	select {
	case q.ch <- path:
		return nil
	default:
		return ErrQueueFull
	}
}

func (q *Queue) Dequeue() <-chan string { return q.ch }
func (q *Queue) Len() int               { return len(q.ch) }
func (q *Queue) Close()                 { close(q.ch) }
