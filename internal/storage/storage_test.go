package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/entity"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
)

// cols определяет точный порядок и имена колонок, которые мы ожидаем от SQL-запроса.
// Это центральное место для синхронизации тестов с реальным запросом.
var cols = []string{
	"order_uid", "track_number", "entry", "locale", "internal_signature", "customer_id",
	"delivery_service", "shardkey", "sm_id", "date_created", "oof_shard",
	"name", "phone", "zip", "city", "address", "region", "email",
	"payment_uid", "request_id", "currency", "provider", "amount", "payment_dt", "bank",
	"delivery_cost", "goods_total", "custom_fee",
	"rid", "chrt_id", "item_track_number", "price", "item_name",
	"sale", "size", "total_price", "nm_id", "brand", "status",
}

// loadTemplateOrder загружает базовый заказ-шаблон из JSON файла.
func loadTemplateOrder() (entity.Order, error) {
	jsonData, err := os.ReadFile("../../cmd/helpCMD/model.json")
	if err != nil {
		return entity.Order{}, fmt.Errorf("failed to read model.json: %w", err)
	}

	var testOrder entity.Order
	if err := json.Unmarshal(jsonData, &testOrder); err != nil {
		return entity.Order{}, fmt.Errorf("failed to unmarshal model.json: %w", err)
	}
	return testOrder, nil
}

// generateTestOrder создаёт уникальный заказ на основе шаблона и индекса.
func generateTestOrder(templOrder entity.Order, i int) entity.Order {
	order := templOrder
	order.Items = make([]entity.Item, len(templOrder.Items))
	copy(order.Items, templOrder.Items)

	order.OrderUID = fmt.Sprintf("test-uid-%d", i)
	order.TrackNumber = fmt.Sprintf("TRACK-%d", i)
	if len(order.Items) > 0 {
		order.Items[0].Rid = fmt.Sprintf("rid-%d", i)
	}
	return order
}

func orderToRow(order entity.Order, itemIndex int) []any {
	item := entity.Item{}
	if len(order.Items) > itemIndex {
		item = order.Items[itemIndex]
	}
	return []any{
		order.OrderUID, order.TrackNumber, order.Entry, order.Locale, order.InternalSignature, order.CustomerID,
		order.DeliveryService, order.ShardKey, order.SmID, order.DateCreated, order.OofShard,
		order.Delivery.Name, order.Delivery.Phone, order.Delivery.Zip, order.Delivery.City, order.Delivery.Address, order.Delivery.Region, order.Delivery.Email,

		order.Payment.OrderUID,

		order.Payment.RequestID, order.Payment.Currency, order.Payment.Provider, order.Payment.Amount, order.Payment.PaymentDt, order.Payment.Bank,
		order.Payment.DeliveryCost, order.Payment.GoodsTotal, order.Payment.CustomFee,
		item.Rid, item.ChrtID, item.TrackNumber, item.Price, item.Name,
		item.Sale, item.Size, item.TotalPrice, item.NmID, item.Brand, item.Status,
	}
}

func TestGetOrderByUID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать мок-пул: %v", err)
	}
	defer mock.Close()

	s := Storage{pool: mock}

	testOrder, err := loadTemplateOrder()
	if err != nil {
		t.Fatalf("не удалось загрузить шаблон заказа: %v", err)
	}
	testOrderUID := testOrder.OrderUID

	testCases := []struct {
		name          string
		orderUID      string
		mockSetup     func()
		expectedOrder entity.Order
		expectedErr   error
	}{
		{
			name:     "Успех: Заказ с одним товаром найден",
			orderUID: testOrderUID,
			mockSetup: func() {
				rows := pgxmock.NewRows(cols).AddRow(orderToRow(testOrder, 0)...)
				mock.ExpectQuery(`SELECT .* WHERE o.order_uid = \$1`).
					WithArgs(testOrderUID).
					WillReturnRows(rows)
			},
			expectedOrder: testOrder,
			expectedErr:   nil,
		},
		{
			name:     "Ошибка: Заказ не найден",
			orderUID: "nonexistent-uid",
			mockSetup: func() {
				rows := pgxmock.NewRows(cols)
				mock.ExpectQuery(`SELECT .* WHERE o.order_uid = \$1`).
					WithArgs("nonexistent-uid").
					WillReturnRows(rows)
			},
			expectedOrder: entity.Order{},
			expectedErr:   pgx.ErrNoRows,
		},
		{
			name:     "Ошибка: Ошибка базы данных",
			orderUID: testOrderUID,
			mockSetup: func() {
				mock.ExpectQuery(`SELECT .* WHERE o.order_uid = \$1`).
					WithArgs(testOrderUID).
					WillReturnError(fmt.Errorf("something went wrong"))
			},
			expectedOrder: entity.Order{},
			expectedErr:   fmt.Errorf("failed to query order: something went wrong"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			order, err := s.GetOrderByUID(context.Background(), tc.orderUID)

			assertError(t, err, tc.expectedErr)

			if tc.expectedErr == nil {
				if !reflect.DeepEqual(order, tc.expectedOrder) {
					assertJSONEqual(t, order, tc.expectedOrder)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("были невыполненные ожидания мока: %s", err)
			}
		})
	}
}

func TestGetLastNOrders(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("не удалось создать мок-пул: %v", err)
	}
	defer mock.Close()

	s := Storage{pool: mock}

	baseOrder, err := loadTemplateOrder()
	if err != nil {
		t.Fatalf("не удалось загрузить шаблон заказа: %v", err)
	}

	order1 := generateTestOrder(baseOrder, 1)
	order2 := generateTestOrder(baseOrder, 2)

	orderWithTwoItems := generateTestOrder(baseOrder, 3)
	item2 := orderWithTwoItems.Items[0]
	item2.Rid = "rid-3-item-2"
	item2.Name = "Второй товар"
	orderWithTwoItems.Items = append(orderWithTwoItems.Items, item2)

	testCases := []struct {
		name           string
		n              int
		mockSetup      func()
		expectedOrders []entity.Order
		expectedErr    error
	}{
		{
			name: "Успех: Получение 2 заказов",
			n:    2,
			mockSetup: func() {
				rows := pgxmock.NewRows(cols).
					AddRow(orderToRow(order1, 0)...).
					AddRow(orderToRow(order2, 0)...)
				mock.ExpectQuery(`SELECT .* LIMIT \$1`).
					WithArgs(2).
					WillReturnRows(rows)
			},
			expectedOrders: []entity.Order{order1, order2},
			expectedErr:    nil,
		},
		{
			name: "Успех: Получение 1 заказа с 2 товарами",
			n:    1,
			mockSetup: func() {
				rows := pgxmock.NewRows(cols).
					AddRow(orderToRow(orderWithTwoItems, 0)...).
					AddRow(orderToRow(orderWithTwoItems, 1)...)
				mock.ExpectQuery(`SELECT .* LIMIT \$1`).
					WithArgs(1).
					WillReturnRows(rows)
			},
			expectedOrders: []entity.Order{orderWithTwoItems},
			expectedErr:    nil,
		},
		{
			name: "Успех: В БД меньше заказов чем запрошено",
			n:    5,
			mockSetup: func() {
				rows := pgxmock.NewRows(cols).AddRow(orderToRow(order1, 0)...)
				mock.ExpectQuery(`SELECT .* LIMIT \$1`).
					WithArgs(5).
					WillReturnRows(rows)
			},
			expectedOrders: []entity.Order{order1},
			expectedErr:    nil,
		},
		{
			name: "Ошибка: Ошибка базы данных",
			n:    3,
			mockSetup: func() {
				mock.ExpectQuery(`SELECT .* LIMIT \$1`).
					WithArgs(3).
					WillReturnError(fmt.Errorf("db connection failed"))
			},
			expectedOrders: nil,
			expectedErr:    fmt.Errorf("failed to query all orders: db connection failed"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.mockSetup()
			orders, err := s.GetLastNOrders(context.Background(), tc.n)

			assertError(t, err, tc.expectedErr)

			if tc.expectedErr == nil {
				if len(orders) == 0 {
					orders = nil
				}
				if len(tc.expectedOrders) == 0 {
					tc.expectedOrders = nil
				}

				if !reflect.DeepEqual(orders, tc.expectedOrders) {
					assertJSONEqual(t, orders, tc.expectedOrders)
				}
			}

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("были невыполненные ожидания мока: %s", err)
			}
		})
	}
}

func assertError(t *testing.T, got, want error) {
	t.Helper()
	if want == nil {
		if got != nil {
			t.Errorf("неожиданная ошибка: %v", got)
		}
		return
	}
	if got == nil || got.Error() != want.Error() {
		t.Errorf("ожидалась ошибка '%v', а получили '%v'", want, got)
	}
}

func assertJSONEqual(t *testing.T, got, want any) {
	t.Helper()
	gotJSON, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("не удалось сериализовать полученный результат в JSON: %v", err)
	}
	wantJSON, err := json.MarshalIndent(want, "", "  ")
	if err != nil {
		t.Fatalf("не удалось сериализовать ожидаемый результат в JSON: %v", err)
	}

	if string(gotJSON) != string(wantJSON) {
		t.Errorf("результат не совпадает:\n\nожидали:\n%s\n\nполучили:\n%s\n", string(wantJSON), string(gotJSON))
	}
}
