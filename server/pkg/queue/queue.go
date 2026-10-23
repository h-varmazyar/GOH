package queue

import (
	"errors"
	"sync"
)

// Queue represents a queue that holds any type.
type Queue[T any] struct {
	mu       sync.Mutex
	elements []T
}

// Enqueue adds an element to the back of the queue.
func (q *Queue[T]) Enqueue(element T) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.elements = append(q.elements, element)
}

// Dequeue removes and returns the element at the front of the queue.
func (q *Queue[T]) Dequeue() (T, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	if q.IsEmpty() {
		var zero T // Returns the zero value of type T
		return zero, errors.New("queue is empty")
	}

	element := q.elements[0]
	q.elements = q.elements[1:] // Reslice to remove the first element
	return element, nil
}

// Front returns the element at the front without removing it.
func (q *Queue[T]) Front() (T, error) {
	if q.IsEmpty() {
		var zero T
		return zero, errors.New("queue is empty")
	}
	return q.elements[0], nil
}

// IsEmpty returns true if the queue is empty.
func (q *Queue[T]) IsEmpty() bool {
	return len(q.elements) == 0
}

// Size returns the number of elements in the queue.
func (q *Queue[T]) Size() int {
	return len(q.elements)
}
