package queue

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type Consumer struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	tag     string
	done    chan error
	logger  *zap.SugaredLogger
	handler func(deliveries <-chan amqp.Delivery, done chan error)
}

func NewConsumer(amqpURI, exchange, exchangeType, queueName, key, ctag string,
	handler func(deliveries <-chan amqp.Delivery, done chan error),
	logger *zap.SugaredLogger) (*Consumer, error) {
	c := &Consumer{
		conn:    nil,
		channel: nil,
		tag:     ctag,
		done:    make(chan error),
		logger:  logger,
		handler: handler,
	}

	var err error

	logger.Infof("dialing %q", amqpURI)
	c.conn, err = amqp.Dial(amqpURI)
	if err != nil {
		return nil, fmt.Errorf("Dial: %s", err)
	}

	go func() {
		fmt.Printf("closing: %s", <-c.conn.NotifyClose(make(chan *amqp.Error)))
	}()

	logger.Infof("got Connection, getting Channel")
	c.channel, err = c.conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("Channel: %s", err)
	}

	logger.Infof("got Channel, declaring Exchange (%q)", exchange)
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

	logger.Infof("declared Exchange, declaring Queue %q", queueName)
	queue, err := c.channel.QueueDeclare(
		queueName,                        // name of the queue
		true,                             // durable
		false,                            // delete when unused
		false,                            // exclusive
		false,                            // noWait
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("Queue Declare: %s", err)
	}

	logger.Infof("declared Queue (%q %d messages, %d consumers), binding to Exchange (key %q)",
		queue.Name, queue.Messages, queue.Consumers, key)

	if err = c.channel.QueueBind(
		queue.Name, // name of the queue
		key,        // bindingKey
		exchange,   // sourceExchange
		false,      // noWait
		nil,
	); err != nil {
		return nil, fmt.Errorf("Queue Bind: %s", err)
	}

	logger.Infof("Queue bound to Exchange, starting Consume (consumer tag %q)", c.tag)
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

func (c *Consumer) Shutdown() error {
	// will close() the deliveries channel
	if err := c.channel.Cancel(c.tag, true); err != nil {
		return fmt.Errorf("consumer cancel failed: %s", err)
	}

	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("AMQP connection close error: %s", err)
	}

	defer c.logger.Infof("AMQP shutdown OK")

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
