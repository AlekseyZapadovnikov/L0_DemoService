// пакет сервис реализует слой бизнес логики

package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/entity"
	"github.com/jackc/pgx/v5"
)

var (
	ErrOrderNotFound = errors.New("order not found")
)

type getOrder interface {
	GetOrderByUID(ctx context.Context, in string) (entity.Order, error)
	GetLastNOrders(ctx context.Context, numberOfgetOrders int) ([]entity.Order, error)
	saver
}

type saver interface {
	SaveOrder(ctx context.Context, o entity.Order) error
}

type Cache struct {
	OrderMap   map[string]entity.Order // Хранилище данных
	orderItems map[string]*Item        // Быстрый доступ к элементам в очереди по UID
	OrderTaker getOrder                // Интерфейс для получения заказов из хранилища
	prQ        *SafePriorityQueue      // Указатель, чтобы избежать копирования
	cacheCap   int
	mu         sync.RWMutex
}

func NewCache(storage getOrder, cacheCap int) *Cache {
	return &Cache{
		OrderMap:   make(map[string]entity.Order, cacheCap),
		orderItems: make(map[string]*Item, cacheCap),
		OrderTaker: storage,
		prQ:        NewSafePriorityQueue(cacheCap),
		cacheCap:   cacheCap,
		mu:         sync.RWMutex{},
	}
}

// загружаем cacheCap элементов в наш Cache при запусте сервиса
func (s *Cache) LoadCache(ctx context.Context) error {
	orders, err := s.OrderTaker.GetLastNOrders(ctx, s.cacheCap) // достаём из хранилища N заказов
	if err != nil {
		slog.Info("s.OrderTaker.GetLastNOrders(ctx, s.cacheCap)", "error", err)
		return fmt.Errorf("error occured while tryed load cache in service.LoadCache() %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, ord := range orders {
		s.OrderMap[ord.OrderUID] = ord
		item := makeItem(ord.OrderUID)
		s.prQ.Push(item)
		s.orderItems[ord.OrderUID] = item
	}
	return nil
}

// возвращает Order по UID
func (s *Cache) GiveOrderByUID(UID string) (entity.Order, error) {
	s.mu.RLock()

	ord, isIn := s.OrderMap[UID]
	s.mu.RUnlock()

	if isIn {
		s.updateOrderPriority(UID)
		return ord, nil
	}

	ord, err := s.OrderTaker.GetOrderByUID(context.Background(), UID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return entity.Order{}, fmt.Errorf("%w: order with UID %s not found", ErrOrderNotFound, UID)
		}
		return entity.Order{}, fmt.Errorf("error occurred while trying to get order with UID %s: %w", UID, err)
	}

	s.addToCache(ord)
	return ord, nil
}

// сохраняет Order в БД и в Cache
func (s *Cache) SaveOrder(ctx context.Context, o entity.Order) error {
	if err := s.OrderTaker.SaveOrder(ctx, o); err != nil {
		slog.Error("Failed to save order to database", "order_uid", o.OrderUID, "error", err)
		return fmt.Errorf("error occurred while trying to save order: %w", err)
	}
	s.addToCache(o)
	return nil
}

// добавляет Order в cache
func (s *Cache) addToCache(ord entity.Order) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Если кэш заполнен, вытесняем самый старый элемент.
	if len(s.OrderMap) >= s.cacheCap {
		item := s.prQ.Pop()
		if item != nil {
			slog.Info("Evicting order from cache", "order_uid", item.Value)
			delete(s.OrderMap, item.Value)
			delete(s.orderItems, item.Value)
		}
	}

	// Добавляем новый элемент.
	itm := makeItem(ord.OrderUID)
	s.prQ.Push(itm)
	s.OrderMap[ord.OrderUID] = ord
	s.orderItems[ord.OrderUID] = itm
	slog.Info("Order added to cache", "order_uid", ord.OrderUID)
}

func (s *Cache) updateOrderPriority(UID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.orderItems[UID]
	if !exists {
		return
	}

	s.prQ.Update(item, time.Now())
}

func (s *Cache) GetPriorityQueue() *SafePriorityQueue {
	return s.prQ
}

func (s *Cache) PrinPriorityQueue() string {
	return s.prQ.String()
}
