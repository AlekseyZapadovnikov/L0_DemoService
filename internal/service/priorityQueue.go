// C:/Users/Asus/Desktop/go_p/WB/L0_DemoServise/internal/service/priorityQueue.go

package service

import (
	"time"
)

// Item — это элемент в нашей приоритетной очереди
type Item struct {
	Value    string    // UID заказа (с большой буквы)
	Priority time.Time // Приоритет (с большой буквы)
	Index    int       // Индекс элемента в куче (с большой буквы)
}

// PriorityQueue реализует heap.Interface
type PriorityQueue []*Item

// Len использует pointer receiver
func (pq *PriorityQueue) Len() int { return len(*pq) }

// Less использует pointer receiver
func (pq *PriorityQueue) Less(i, j int) bool {
	// Мы хотим, чтобы старые элементы имели более высокий приоритет (были "меньше"),
	// чтобы они всплывали наверх и удалялись первыми.
	return (*pq)[i].Priority.Before((*pq)[j].Priority)
}

// Swap использует pointer receiver
func (pq *PriorityQueue) Swap(i, j int) {
	(*pq)[i], (*pq)[j] = (*pq)[j], (*pq)[i]
	(*pq)[i].Index = i
	(*pq)[j].Index = j
}

// Push использует pointer receiver
func (pq *PriorityQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Item)
	item.Index = n
	*pq = append(*pq, item)
}

// Pop использует pointer receiver
func (pq *PriorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // избегаем утечки памяти
	item.Index = -1 // для безопасности
	*pq = old[0 : n-1]
	return item
}

// update (не используется, но для полноты картины реализован)
// func (pq *PriorityQueue) update(item *Item, Value string, Priority time.Time) {
// 	item.Value = Value
// 	item.Priority = Priority
// 	heap.Fix(pq, item.Index)
// }
