package service

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/entity"
	"github.com/jackc/pgx/v5"
)

type TestData struct {
	Orders []entity.Order `json:"orders"`
}

func loadTestData(t *testing.T) []entity.Order {
	t.Helper()

	file, err := os.Open("tests/testData.json")
	if err != nil {
		t.Fatalf("failed to open test data: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("failed to read test data: %v", err)
	}

	var testData TestData
	if err := json.Unmarshal(data, &testData); err != nil {
		t.Fatalf("failed to unmarshal test data: %v", err)
	}

	return testData.Orders
}

func giveItemsUIDSlice(items []*Item) []string {
	uids := make([]string, 0, len(items))
	for _, item := range items {
		uids = append(uids, item.Value)
	}
	return uids
}

func TestPriorityQueue(t *testing.T) {
	orders := loadTestData(t)

	// Подготовка эталонных данных
	items := make([]*Item, 0, len(orders))
	for _, ord := range orders {
		items = append(items, makeItem(ord.OrderUID))
	}

	// Копия для проверки обратного порядка (LIFO/Priority Logic)
	revItems := make([]*Item, len(items))
	copy(revItems, items)
	slices.Reverse(revItems)

	revItemsUID := giveItemsUIDSlice(revItems)
	itemsUID := giveItemsUIDSlice(items)

	// Базовая проверка длины
	t.Run("Check Length", func(t *testing.T) {
		prq := NewSafePriorityQueue(10)
		for _, item := range items {
			prq.Push(item)
		}
		if got := prq.Len(); got != len(orders) {
			t.Errorf("expected length %d, got %d", len(orders), got)
		}
	})

	testCases := []struct {
		name             string
		action           func(prq *SafePriorityQueue, items []*Item)
		expectedPopOrder []string
	}{
		{
			name:             "Normal push order",
			action:           func(prq *SafePriorityQueue, items []*Item) {},
			expectedPopOrder: revItemsUID,
		},
		{
			name: "Push with priority update",
			action: func(prq *SafePriorityQueue, items []*Item) {
				// Обновляем приоритет у 4-го элемента (индекс 3)
				prq.Update(items[3], time.Now())
			},
			// Ожидаем, что первым выйдет items[3], так как у него самое свежее время
			expectedPopOrder: []string{itemsUID[3]},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prq := NewSafePriorityQueue(10)

			// Пересоздаем items для каждого теста, чтобы не было сайд-эффектов от pointer'ов
			currentItems := make([]*Item, 0, len(orders))
			for _, ord := range orders {
				item := makeItem(ord.OrderUID)
				currentItems = append(currentItems, item)
				prq.Push(item)
			}

			// Выполняем действие теста (например, Update)
			tc.action(prq, currentItems)

			var popped []string
			for prq.Len() > 0 {
				item := prq.Pop()
				if item != nil {
					popped = append(popped, item.Value)
				}
			}

			// Проверяем
			if len(tc.expectedPopOrder) == 0 {
				return
			}

			// Сравниваем только начало списка, если expectedPopOrder короче полного списка
			limit := len(tc.expectedPopOrder)
			if len(popped) < limit {
				t.Fatalf("popped items count %d less than expected to check %d", len(popped), limit)
			}

			if !slices.Equal(popped[:limit], tc.expectedPopOrder) {
				t.Errorf("pop order mismatch.\nWant: %v\nGot:  %v", tc.expectedPopOrder, popped[:limit])
			}
		})
	}
}

// Mocks

type mockStorage struct {
	mockDB map[string]entity.Order
}

func (m *mockStorage) GetOrderByUID(ctx context.Context, uid string) (entity.Order, error) {
	if order, ok := m.mockDB[uid]; ok {
		return order, nil
	}
	return entity.Order{}, pgx.ErrNoRows
}

func (m *mockStorage) GetLastNOrders(ctx context.Context, n int) ([]entity.Order, error) {
	return []entity.Order{}, nil
}

func (m *mockStorage) SaveOrder(ctx context.Context, o entity.Order) error {
	return nil
}

func TestCache(t *testing.T) {
	mockOrders := map[string]entity.Order{
		"order-1": {OrderUID: "order-1", TrackNumber: "TRACK_A"},
		"order-2": {OrderUID: "order-2", TrackNumber: "TRACK_B"},
		"order-3": {OrderUID: "order-3", TrackNumber: "TRACK_C"},
		"order-4": {OrderUID: "order-4", TrackNumber: "TRACK_D"},
	}
	storage := &mockStorage{mockDB: mockOrders}

	t.Run("Get from empty cache (miss and fill)", func(t *testing.T) {
		cache := NewCache(storage, 3)

		order, err := cache.GiveOrderByUID("order-1")
		if err != nil {
			t.Fatalf("expected no error, but got: %v", err)
		}
		if order.OrderUID != "order-1" {
			t.Errorf("expected to get order-1, but got: %s", order.OrderUID)
		}

		if _, exists := cache.OrderMap["order-1"]; !exists {
			t.Error("order-1 was not added to the cache after a miss")
		}
		if len(cache.OrderMap) != 1 {
			t.Errorf("expected cache size to be 1, but got: %d", len(cache.OrderMap))
		}
	})

	t.Run("Eviction of least recently used item", func(t *testing.T) {
		cache := NewCache(storage, 2)

		mustGet := func(uid string) {
			t.Helper()
			if _, err := cache.GiveOrderByUID(uid); err != nil {
				t.Fatalf("failed to prepare cache with %s: %v", uid, err)
			}
		}

		mustGet("order-1")
		time.Sleep(10 * time.Millisecond)
		mustGet("order-2")
		time.Sleep(10 * time.Millisecond)

		if len(cache.OrderMap) != 2 {
			t.Fatalf("expected cache size to be 2, got: %d", len(cache.OrderMap))
		}

		mustGet("order-3")

		if len(cache.OrderMap) != 2 {
			t.Errorf("expected cache size to be 2 after eviction, got: %d", len(cache.OrderMap))
		}
		if _, exists := cache.OrderMap["order-3"]; !exists {
			t.Error("new item order-3 was not added to cache")
		}
		if _, exists := cache.OrderMap["order-2"]; !exists {
			t.Error("item order-2 should not have been evicted")
		}
		if _, exists := cache.OrderMap["order-1"]; exists {
			t.Error("least recently used item order-1 was not evicted")
		}
	})

	t.Run("Accessing an item updates its priority", func(t *testing.T) {
		cache := NewCache(storage, 2)

		mustGet := func(uid string) {
			t.Helper()
			if _, err := cache.GiveOrderByUID(uid); err != nil {
				t.Fatalf("failed to prepare cache with %s: %v", uid, err)
			}
		}

		mustGet("order-1")
		time.Sleep(10 * time.Millisecond)
		mustGet("order-2")
		time.Sleep(10 * time.Millisecond)

		mustGet("order-1")
		time.Sleep(10 * time.Millisecond)

		mustGet("order-3")

		if len(cache.OrderMap) != 2 {
			t.Errorf("expected cache size to be 2, but got: %d", len(cache.OrderMap))
		}
		if _, exists := cache.OrderMap["order-1"]; !exists {
			t.Error("order-1 should have been kept in cache (recently accessed)")
		}
		if _, exists := cache.OrderMap["order-2"]; exists {
			t.Error("order-2 should have been evicted")
		}
	})

	t.Run("Getting a non-existent item returns an error", func(t *testing.T) {
		cache := NewCache(storage, 3)
		_, err := cache.GiveOrderByUID("non-existent-order")
		if err == nil {
			t.Fatal("expected an error for a non-existent item, but got nil")
		}
	})
}
