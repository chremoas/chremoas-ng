package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Producer struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	logger     *zap.SugaredLogger
	exchange   string
	routingKey string
}

func NewPublisher(amqpURI, exchange, exchangeType, routingKey string, logger *zap.SugaredLogger) (*Producer, error) {
	var err error

	p := &Producer{
		conn:       nil,
		channel:    nil,
		logger:     logger,
		exchange:   exchange,
		routingKey: routingKey,
	}

	logger.Infof("dialing %q", sanitizeURI(amqpURI))
	p.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	logger.Info("got Connection, getting Channel")
	p.channel, err = p.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	logger.Infof("got Channel, declaring %q Exchange (%q)", exchangeType, exchange)
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

	logger.Info("declared Exchange")

	return p, nil
}

func (p Producer) Publish(body []byte) error {
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
		return fmt.Errorf("Exchange Publish: %s", err)
	}

	return nil
}

func (p Producer) Shutdown() {
	err := p.conn.Close()
	if err != nil {
		p.logger.Errorf("Error closing connection: %s", err, err)
	}
}
