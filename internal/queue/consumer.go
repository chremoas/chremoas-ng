package queue

import (
	"context"
	"fmt"

	sl "github.com/bhechinger/spiffylogger"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
	handler func(deliveries <-chan amqp.Delivery, done chan error)
}

func NewConsumer(ctx context.Context, amqpURI, exchange, exchangeType, queueName, key, ctag string,
	handler func(deliveries <-chan amqp.Delivery, done chan error)) (*Consumer, error) {

	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
		handler: handler,
	}

	var err error

	sp.Info("dialing queue", zap.String("queue URI", sanitizeURI(amqpURI)))
	c.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	go func() {
		fmt.Printf("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	sp.Info("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	sp.Info("got Channel, declaring Exchange", zap.String("exchange", exchange))
	if err = c.channel.ExchangeDeclare(
		exchange,     // name of the exchange
		exchangeType, // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return nil, fmt.Errorf("Exchange Declare: %s", err)
	}

	sp.Info("declared Exchange, declaring Queue", zap.String("queue", queueName))
	queue, err := c.channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Declare: %s", err)
	}

	sp.Info("declared Queue, binding to Exchange",
		zap.String("name", queue.Name),
		zap.Int("messages", queue.Messages),
		zap.Int("consumers", queue.Consumers),
		zap.String("exchange key", key),
	)

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		exchange,   // sourceExchange
		false,      // noWait
		nil,
	); err != nil {
		return nil, fmt.Errorf("Queue Bind: %s", err)
	}

	sp.Info("Queue bound to Exchange, starting Consume", zap.String("consumer tag", c.tag))
	deliveries, err := c.channel.Consume(
		queue.Name, // name
		c.tag,      // consumerTag,
		false,      // noAck
		false,      // exclusive
		false,      // noLocal
		false,      // noWait
		nil,        // arguments
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Consume: %s", err)
	}

	go c.handler(deliveries, c.done)

	return c, nil
}

func (c *Consumer) Shutdown(ctx context.Context) error {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		return fmt.Errorf("consumer cancel failed: %s", err)
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("AMQP connection close error: %s", err)
	}

	defer sp.Info("AMQP shutdown OK")

	// wait for handle() to exit
	return <-c.done
}

// func handle(deliveries <-chan amqp.Delivery, done chan error, logger *zap.SugaredLogger) {
// 	for d := range deliveries {
// 		logger.Infof(
// 			"got %dB delivery: [%v] %q",
// 			len(d.Body),
// 			d.DeliveryTag,
// 			d.Body,
// 		)
// 		d.Ack(false)
// 	}
// 	logger.Infof("handle: deliveries channel closed")
// 	done <- nil
// }
