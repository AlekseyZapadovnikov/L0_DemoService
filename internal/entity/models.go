// крутое архитектурное решение - сделать сущности глобальными,
// не нужны никакие дополнительные сервисы для передачи данных между слоями

package entity

import (
	"time"

	"github.com/go-playground/validator/v10"
)

var Validate *validator.Validate

func init() {
	Validate = validator.New()
}

type Order struct {
	OrderUID          string    `json:"order_uid" db:"order_uid" validate:"required"`
	TrackNumber       string    `json:"track_number" db:"track_number" validate:"required"`
	Entry             string    `json:"entry" db:"entry" validate:"required"`
	Locale            string    `json:"locale" db:"locale" validate:"required,len=2"`
	InternalSignature string    `json:"internal_signature" db:"internal_signature"`
	CustomerID        string    `json:"customer_id" db:"customer_id" validate:"required"`
	DeliveryService   string    `json:"delivery_service" db:"delivery_service" validate:"required"`
	ShardKey          string    `json:"shardkey" db:"shardkey"`
	SmID              int       `json:"sm_id" db:"sm_id"`
	DateCreated       time.Time `json:"date_created" db:"date_created" validate:"required"`
	OofShard          string    `json:"oof_shard" db:"oof_shard"`

	// Вложенные объекты (хранятся в отдельных таблицах payment и delivery)
	Delivery Delivery `json:"delivery" db:"-" validate:"required"`
	Payment  Payment  `json:"payment" db:"-" validate:"required"`
	Items    []Item   `json:"items" db:"-" validate:"required,min=1,dive"`
}

type Delivery struct {
	// В SQL delivery.order_uid — первичный ключ, ссылается на orders(order_uid)
	OrderUID string `json:"order_uid,omitempty" db:"order_uid"`

	Name    string `json:"name" db:"name" validate:"required"`
	Phone   string `json:"phone" db:"phone" validate:"required,e164"` // Валидация номера телефона в формате E.164
	Zip     string `json:"zip" db:"zip"`
	City    string `json:"city" db:"city" validate:"required"`
	Address string `json:"address" db:"address" validate:"required"`
	Region  string `json:"region" db:"region"`
	Email   string `json:"email" db:"email" validate:"email"`
}

type Payment struct {
	// В JSON поле "transaction" соответствует payment.order_uid в SQL (см. комментарий в скрипте)
	OrderUID     string    `json:"transaction" db:"order_uid"`
	RequestID    string    `json:"request_id" db:"request_id"`
	Currency     string    `json:"currency" db:"currency"`
	Provider     string    `json:"provider" db:"provider"`
	Amount       int       `json:"amount" db:"amount"`
	PaymentDt    time.Time `json:"payment_dt" db:"payment_dt" validate:"required"`
	Bank         string    `json:"bank" db:"bank"`
	DeliveryCost int       `json:"delivery_cost" db:"delivery_cost" validate:"gte=0"`
	GoodsTotal   int       `json:"goods_total" db:"goods_total" validate:"gte=0"`
	CustomFee    int       `json:"custom_fee" db:"custom_fee"`
}

type Item struct {
	// rid — первичный ключ строки заказа
	Rid string `json:"rid" db:"rid" validate:"required"`
	// order_uid — внешний ключ на orders(order_uid)
	OrderUID string `json:"order_uid,omitempty" db:"order_uid"`

	ChrtID      int    `json:"chrt_id" db:"chrt_id"`
	TrackNumber string `json:"track_number" db:"track_number"`
	Price       int    `json:"price" db:"price" validate:"gte=0"`
	Name        string `json:"name" db:"name" validate:"required"`
	Sale        int    `json:"sale" db:"sale" validate:"gte=0"`
	Size        string `json:"size" db:"size"`
	TotalPrice  int    `json:"total_price" db:"total_price"`
	NmID        int    `json:"nm_id" db:"nm_id"`
	Brand       string `json:"brand" db:"brand"`
	Status      int    `json:"status" db:"status"`
}
