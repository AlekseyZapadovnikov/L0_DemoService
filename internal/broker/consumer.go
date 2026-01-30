package broker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/AlekseyZapadovnikov/L0_DemoService/internal/entity"
	"github.com/segmentio/kafka-go"
)

type OrderSaver interface {
	SaveOrder(ctx context.Context, o entity.Order) error
}

type KafkaConsumer struct {
	reader *kafka.Reader
	saver  OrderSaver
}

func NewKafkaConsumer(brokerAddr string, topic string, groupID string, saver OrderSaver) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddr},
		Topic:    topic,
		GroupID:  groupID,
		MaxBytes: 10e6,
	})
	return &KafkaConsumer{
		reader: reader,
		saver:  saver,
	}
}

func (c *KafkaConsumer) ConsumeAndSave(ctx context.Context) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("failed to read message: %w", err)
		}

		var order entity.Order
		if err := json.Unmarshal(msg.Value, &order); err != nil {
			slog.Error("failed to parse order JSON", "error", err, "message", string(msg.Value))
			continue
		}

		// Валидация данных
		if err := entity.Validate.Struct(order); err != nil {
			slog.Error("failed to validate order data", "error", err, "order_uid", order.OrderUID)
			continue // Пропускаем невалидное сообщение, предварительно логируя его
		}

		slog.Info("Order processed from Kafka", "order_uid", order.OrderUID)

		// Сохранение заказа
		if err := c.saver.SaveOrder(ctx, order); err != nil {
			slog.Error("failed to save order", "order_uid", order.OrderUID, "error", err)
			continue
		}
	}
}

func (c *KafkaConsumer) Close() error {
	return c.reader.Close()
}
