package service

import (
	"container/heap"
	"sync"
	"time"
)

// SafePriorityQueue — это потокобезопасная обертка для PriorityQueue.
type SafePriorityQueue struct {
	pq PriorityQueue
	mu sync.Mutex
}

// NewSafePriorityQueue создает новую безопасную приоритетную очередь.
func NewSafePriorityQueue(cacheCap int) *SafePriorityQueue {
	spq := &SafePriorityQueue{
		pq: make(PriorityQueue, 0, cacheCap),
	}
	// Инициализируем кучу
	heap.Init(&spq.pq)
	return spq
}

// Push добавляет элемент в очередь безопасно.
func (spq *SafePriorityQueue) Push(item *Item) {
	spq.mu.Lock()
	defer spq.mu.Unlock()

	heap.Push(&spq.pq, item)
}

// Pop извлекает и возвращает элемент с наивысшим приоритетом безопасно.
func (spq *SafePriorityQueue) Pop() *Item {
	spq.mu.Lock()
	defer spq.mu.Unlock()

	if spq.pq.Len() == 0 {
		return nil // Или вернуть ошибку, в зависимости от вашей логики
	}

	return heap.Pop(&spq.pq).(*Item)
}

// Update безопасно изменяет приоритет элемента в очереди.
// Эта функция принимает указатель на элемент, который нужно обновить.
func (spq *SafePriorityQueue) Update(item *Item, newPriority time.Time) {
	spq.mu.Lock()
	defer spq.mu.Unlock()

	item.Priority = newPriority
	heap.Fix(&spq.pq, item.Index)
}

// Len возвращает количество элементов в очереди безопасно.
func (spq *SafePriorityQueue) Len() int {
	spq.mu.Lock()
	defer spq.mu.Unlock()

	return spq.pq.Len()
}

func makeItem(UID string) *Item {
	return &Item{
		Value:    UID,
		Priority: time.Now(),
	}
}

func (sprq *SafePriorityQueue) GetPrq() PriorityQueue {
	return sprq.pq
}

func (sprq *SafePriorityQueue) LenPrq() int {
	return sprq.pq.Len()
}

func (sprq *SafePriorityQueue) String() string {
	sprq.mu.Lock()
	defer sprq.mu.Unlock()

	s := "PriorityQueue: ["
	for i, item := range sprq.pq {
		if i > 0 {
			s += ", "
		}
		s += item.Value + "(" + item.Priority.String() + ")"
	}
	return s + "]"
}
