package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/entity"
	"github.com/brianvoe/gofakeit/v6"
	"github.com/segmentio/kafka-go"
)

// generateValidOrder генерирует полностью валидный объект entity.Order.
func generateValidOrder() entity.Order {
	var order entity.Order
	if err := gofakeit.Struct(&order); err != nil {
		log.Fatalf("failed to fake struct: %v", err)
	}

	// Основные поля Order
	order.OrderUID = gofakeit.UUID()
	order.TrackNumber = gofakeit.Regex(`^[A-Z0-9]{10,20}$`)
	order.Entry = gofakeit.Word()
	order.Locale = gofakeit.RandomString([]string{"ru", "en", "de", "fr", "es", "it", "ja", "zh"})
	order.CustomerID = gofakeit.UUID()
	order.DeliveryService = gofakeit.Company()
	order.DateCreated = time.Now()

	// Ограничиваем SmID диапазоном int4, чтобы избежать переполнения в БД.
	order.SmID = gofakeit.Number(0, 2147483647)

	// Delivery
	order.Delivery.Name = gofakeit.Name()
	order.Delivery.Phone = fmt.Sprintf("+%d%d", gofakeit.Number(1, 999), gofakeit.Number(1000000000, 9999999999))
	order.Delivery.Zip = gofakeit.Zip()
	order.Delivery.City = gofakeit.City()
	order.Delivery.Address = gofakeit.Street()
	order.Delivery.Region = gofakeit.State()
	order.Delivery.Email = gofakeit.Email()

	// Payment
	order.Payment.OrderUID = order.OrderUID
	order.Payment.Currency = gofakeit.CurrencyShort()
	order.Payment.Provider = gofakeit.Company()
	order.Payment.Bank = gofakeit.Company()
	order.Payment.DeliveryCost = gofakeit.Number(100, 2000)
	order.Payment.GoodsTotal = gofakeit.Number(500, 50000)
	order.Payment.CustomFee = gofakeit.Number(0, 5000)
	order.Payment.Amount = order.Payment.GoodsTotal + order.Payment.DeliveryCost - order.Payment.CustomFee
	order.Payment.PaymentDt = gofakeit.DateRange(order.DateCreated, time.Now())

	// Items
	numItems := rand.Intn(5) + 1
	order.Items = make([]entity.Item, numItems)
	for i := 0; i < numItems; i++ {
		var item entity.Item
		if err := gofakeit.Struct(&item); err != nil {
			log.Fatalf("failed to fake item struct: %v", err)
		}

		item.OrderUID = order.OrderUID
		item.TrackNumber = order.TrackNumber
		item.ChrtID = gofakeit.Number(100000, 999999)
		item.Price = gofakeit.Number(100, 10000)
		item.Sale = gofakeit.Number(10, 50)
		item.TotalPrice = item.Price - (item.Price * item.Sale / 100)
		item.NmID = gofakeit.Number(1000000, 9999999)
		item.Status = gofakeit.Number(200, 299)
		item.Rid = gofakeit.UUID()
		item.Name = gofakeit.ProductName()
		item.Brand = gofakeit.Company()

		order.Items[i] = item
	}

	return order
}

func main() {
	writer := &kafka.Writer{
		Addr:     kafka.TCP("localhost:9092"),
		Topic:    "orders",
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	for i := 0; i < 1000; i++ {
		go func(i int) {
			order := generateValidOrder()

			if err := entity.Validate.Struct(order); err != nil {
				log.Printf("Generated invalid order: %v", err)
			}

			jsonData, err := json.Marshal(&order)
			if err != nil {
				log.Printf("Marshal error: %v", err)
			}

			err = writer.WriteMessages(context.Background(), kafka.Message{Value: jsonData})
			if err != nil {
				log.Printf("Kafka write error: %v", err)
			}

			fmt.Printf("Message %d sent\n", i+1)
		}(i)
	}
	time.Sleep(5 * time.Second)
}
