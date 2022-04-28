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

	sp.With(
		zap.String("ampq_uri", sanitizeURI(amqpURI)),
		zap.String("exchange", exchange),
		zap.String("exchange_type", exchangeType),
		zap.String("queue_name", queueName),
		zap.String("routing_key", key),
		zap.String("ctag", ctag),
	)

	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
		handler: handler,
	}

	var err error

	sp.Info("dialing queue")
	c.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		sp.Error("error dialing queue", zap.Error(err))
		return nil, err
	}

	go func() {
		fmt.Printf("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	sp.Info("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		sp.Error("error getting channel", zap.Error(err))
		return nil, err
	}

	sp.Info("got Channel, declaring Exchange")
	if err = c.channel.ExchangeDeclare(
		exchange,     // name of the exchange
		exchangeType, // type
		true,         // durable
		false,        // delete when complete
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		sp.Error("error declaring exchange", zap.Error(err))
		return nil, err
	}

	sp.Info("declared Exchange, declaring Queue")
	queue, err := c.channel.QueueDeclare(
		queueName, // name of the queue
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // noWait
		nil,
	)
	if err != nil {
		sp.Error("error declaring queue", zap.Error(err))
		return nil, err
	}

	sp.Info("declared Queue, binding to Exchange",
		zap.String("queue.name", queue.Name),
		zap.Int("queue.messages", queue.Messages),
		zap.Int("queue.consumers", queue.Consumers),
	)

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		exchange,   // sourceExchange
		false,      // noWait
		nil,
	); err != nil {
		sp.Error("error binding queue", zap.Error(err))
		return nil, err
	}

	sp.Info("Queue bound to Exchange, starting Consume")
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
		sp.Error("error starting consumer", zap.Error(err))
		return nil, err
	}

	go c.handler(deliveries, c.done)

	return c, nil
}

func (c *Consumer) Shutdown(ctx context.Context) error {
	_, sp := sl.OpenSpan(ctx)
	defer sp.Close()

	sp.With(
		zap.String("ctag", c.tag),
	)

	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		sp.Error("error canceling consumer", zap.Error(err))
		return err
	}

	if err := c.conn.Close(); err != nil {
		sp.Error("error closing connection", zap.Error(err))
		return err
	}

	defer sp.Info("AMQP shutdown OK")

	// wait for handle() to exit
	return <-c.done
}
