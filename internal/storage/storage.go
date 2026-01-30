package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/AlekseyZapadovnikov/L0_DemoService/config"
	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/entity"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	orderQuery = `
		SELECT 
			o.order_uid, o.track_number, o.entry, o.locale, o.internal_signature, o.customer_id, 
			o.delivery_service, o.shardkey, o.sm_id, o.date_created, o.oof_shard,
			d.name, d.phone, d.zip, d.city, d.address, d.region, d.email,
			p.order_uid AS payment_uid,
			p.request_id, p.currency, p.provider, p.amount, p.payment_dt, p.bank, 

			p.delivery_cost, p.goods_total, p.custom_fee,
			i.rid, i.chrt_id, i.track_number AS item_track_number, i.price, i.name AS item_name, 
			i.sale, i.size, i.total_price, i.nm_id, i.brand, i.status
		FROM orders o
		LEFT JOIN delivery d ON o.order_uid = d.order_uid
		LEFT JOIN payment p ON o.order_uid = p.order_uid
		LEFT JOIN items i ON o.order_uid = i.order_uid
		`
)

func scanDataFromRows(rows pgx.Rows, order *entity.Order, item *entity.Item) error {
	return rows.Scan(
		&order.OrderUID, &order.TrackNumber, &order.Entry, &order.Locale, &order.InternalSignature, &order.CustomerID,
		&order.DeliveryService, &order.ShardKey, &order.SmID, &order.DateCreated, &order.OofShard,
		&order.Delivery.Name, &order.Delivery.Phone, &order.Delivery.Zip, &order.Delivery.City, &order.Delivery.Address, &order.Delivery.Region, &order.Delivery.Email,

		&order.Payment.OrderUID,
		&order.Payment.RequestID, &order.Payment.Currency, &order.Payment.Provider, &order.Payment.Amount, &order.Payment.PaymentDt, &order.Payment.Bank,

		&order.Payment.DeliveryCost, &order.Payment.GoodsTotal, &order.Payment.CustomFee,
		&item.Rid, &item.ChrtID, &item.TrackNumber, &item.Price, &item.Name,
		&item.Sale, &item.Size, &item.TotalPrice, &item.NmID, &item.Brand, &item.Status,
	)
}

// интерфейс, для того чтобы можно было запускать тесты
type DBPool interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Begin(ctx context.Context) (pgx.Tx, error)
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
	Close()
}

type Storage struct {
	pool DBPool
}

func NewStorage(cfg *config.Storage) (*Storage, error) {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.DBUser,
		cfg.DBPassword,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
	)

	pool, err := pgxpool.New(context.Background(), connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	return &Storage{pool: pool}, nil
}

func (s *Storage) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

// SaveOrder сохраняет заказ в БД в рамках одной транзакции
func (s *Storage) SaveOrder(ctx context.Context, o entity.Order) (err error) {
	tx, err := s.pool.Begin(ctx)

	if err != nil {
		return fmt.Errorf("error while starting transaction %w", err)
	}

	// 2. Defer с обработкой Rollback
	defer func() {
		if err != nil {
			// Если была ошибка в логике, пытаемся откатить
			rbErr := tx.Rollback(ctx)

			// pgx.ErrTxClosed возникает, когда мы пытаемся откатить закомиченную транзакцию
			// то есть в строчке ниже условие такое - если транзакция не была закомичена и не получилось ролбэкнуть то
			if rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
				err = errors.Join(err, fmt.Errorf("rollback failed: %w", rbErr))
			}
		}
	}()

	_, err = tx.Exec(ctx,
		`INSERT INTO orders
		(order_uid, track_number, entry, locale, internal_signature, customer_id,
		 delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature, o.CustomerID, o.DeliveryService, o.ShardKey, o.SmID, o.DateCreated, o.OofShard,
	)
	if err != nil {
		return fmt.Errorf("failed to insert into orders: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO delivery
		(order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City, o.Delivery.Address, o.Delivery.Region, o.Delivery.Email,
	)
	if err != nil {
		return fmt.Errorf("failed to insert into delivery: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO payment (order_uid, request_id, currency, provider, amount,
		 payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		o.OrderUID, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider, o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal, o.Payment.CustomFee,
	)
	if err != nil {
		return fmt.Errorf("failed to insert into payment: %w", err)
	}

	// Вставка в items
	if len(o.Items) > 0 { // если у заказа нет товаров — пропускаем этот блок;
		cols := []string{
			"rid", "order_uid", "chrt_id", "track_number", "price", "name", "sale",
			"size", "total_price", "nm_id", "brand", "status",
		}
		rows := make([][]interface{}, 0, len(o.Items)) // это все items
		for _, it := range o.Items {
			rows = append(rows, []interface{}{
				it.Rid, o.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.Name,
				it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status,
			})
		}

		// CopyFrom: Копируем данные в таблицу
		_, err = tx.CopyFrom(
			ctx,
			pgx.Identifier{"items"}, // Имя таблицы, в которую грузим
			cols,                    // список колонок
			pgx.CopyFromRows(rows),  // Данные
		)
		if err != nil {
			return fmt.Errorf("failed to copy into items: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

/* Items — это список. Если заказов много или товаров в заказе >10-20,
обычные вставки (Exec в цикле) будут медленными, потому что каждый Exec — отдельный запрос к серверу БД =>
много сетевых вызовов -> это дорого, поэтому, я думаю, что тут лучше использовать CopyForm или хотя бы Batch */

func (s *Storage) GetLastNOrders(ctx context.Context, n int) ([]entity.Order, error) {
	secondPart := "\nLIMIT $1"
	query := orderQuery + secondPart

	rows, err := s.pool.Query(ctx, query, n)
	if err != nil {
		return nil, fmt.Errorf("failed to query all orders: %w", err)
	}
	defer rows.Close()

	// Map для сборки заказов: ключ - order_uid, значение - указатель на Order
	ordersMap := make(map[string]*entity.Order)

	for rows.Next() {
		var order entity.Order
		var item entity.Item

		err = scanDataFromRows(rows, &order, &item)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Проверяем, существует ли заказ в map
		existingOrder, exists := ordersMap[order.OrderUID]
		if !exists {
			//(используем копию, чтобы избежать перезаписи)
			newOrder := order // Копируем структуру
			ordersMap[order.OrderUID] = &newOrder
			existingOrder = &newOrder
		}

		// Добавляем item, если он есть (rid != "")
		if item.Rid != "" {
			existingOrder.Items = append(existingOrder.Items, item)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	orders := make([]entity.Order, 0, len(ordersMap))
	for _, ord := range ordersMap {
		orders = append(orders, *ord)
	}

	return orders, nil
}

// GetOrderByUID находит один заказ по его ID
func (s *Storage) GetOrderByUID(ctx context.Context, orderUID string) (entity.Order, error) {
	query := orderQuery + "\nWHERE o.order_uid = $1" // выбираем все заказы с данным UID

	rows, err := s.pool.Query(ctx, query, orderUID)
	if err != nil {
		return entity.Order{}, fmt.Errorf("failed to query order: %w", err)
	}
	defer rows.Close()

	var order entity.Order
	var items []entity.Item
	var firstRow bool = true // это флаг для проверки была ли найдена хоть одна строка

	for rows.Next() {
		var item entity.Item
		err = scanDataFromRows(rows, &order, &item)
		if err != nil {
			return entity.Order{}, fmt.Errorf("failed to scan row: %w", err)
		}

		// Заполняем основные поля заказа только из первой строки (чтобы избежать дубликатов)
		if firstRow {
			firstRow = false
		}

		// Добавляем item, если rid не NULL (те, если есть item)
		if item.Rid != "" {
			items = append(items, item)
		}
	}
	// были ли ошибки во время итерации по строкам
	if err = rows.Err(); err != nil {
		return entity.Order{}, fmt.Errorf("error during rows iteration: %w", err)
	}

	if firstRow {
		// Если не было ни одной строки, заказ не найден
		return entity.Order{}, pgx.ErrNoRows
	}

	order.Items = items

	// Заполняем поля OrderUID в связанных структурах, почему-то неработало до этого
	order.Payment.OrderUID = order.OrderUID

	return order, nil
}

// GetAllOrders загружает все заказы из БД для восстановления кэша
func (s *Storage) GetAllOrders(ctx context.Context) ([]entity.Order, error) {
	query := orderQuery + "\nORDER BY o.date_created DESC"

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query all orders: %w", err)
	}
	defer rows.Close()

	// Map для сборки заказов: ключ - order_uid, значение - указатель на Order
	ordersMap := make(map[string]*entity.Order)

	for rows.Next() {
		var order entity.Order
		var item entity.Item

		err = scanDataFromRows(rows, &order, &item)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Проверяем, существует ли заказ в map
		existingOrder, exists := ordersMap[order.OrderUID]
		if !exists {
			//(используем копию, чтобы избежать перезаписи)
			newOrder := order // Копируем структуру
			ordersMap[order.OrderUID] = &newOrder
			existingOrder = &newOrder
		}

		// Добавляем item, если он есть (rid != "")
		if item.Rid != "" {
			existingOrder.Items = append(existingOrder.Items, item)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error during rows iteration: %w", err)
	}

	orders := make([]entity.Order, 0, len(ordersMap))
	for _, ord := range ordersMap {
		orders = append(orders, *ord)
	}

	return orders, nil
}
