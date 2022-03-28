package queue

import (
	"context"
	"fmt"

	sl "github.com/bhechinger/spiffylogger"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Producer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	exchange   string
	routingKey string
}

func NewPublisher(ctx context.Context, amqpURI, exchange, exchangeType, routingKey string) (*Producer, error) {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var err error

	p := &Producer{
		conn:       nil,
		channel:    nil,
		exchange:   exchange,
		routingKey: routingKey,
	}

	sp.Info("dialing queue", zap.String("queue URI", sanitizeURI(amqpURI)))
	p.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	sp.Info("got Connection, getting Channel")
	p.channel, err = p.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	sp.Info("got Channel, declaring Exchange",
		zap.String("exchange type", exchangeType), zap.String("exchange", exchange))
	if err := p.channel.ExchangeDeclare(
		p.exchange,   // name
		exchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return nil, fmt.Errorf("Exchange Declare: %s", err)
	}

	sp.Info("declared Exchange")

	return p, nil
}

func (p Producer) Publish(ctx context.Context, body []byte) error {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	var err error

	if err = p.channel.Publish(
		p.exchange,   // publish to an exchange
		p.routingKey, // routing to 0 or more queues
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "text/plain",
			ContentEncoding: "",
			Body:            body,
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
			// a bunch of application/implementation-specific fields
		},
	); err != nil {
		sp.Error("exchange publish", zap.Error(err))
		return fmt.Errorf("exchange publish: %w", err)
	}

	return nil
}

func (p Producer) Shutdown(ctx context.Context) {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	err := p.conn.Close()
	if err != nil {
		sp.Error("Error closing connection", zap.Error(err))
	}
}
