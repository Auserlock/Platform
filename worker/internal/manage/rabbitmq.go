package queue

import (
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	log "github.com/sirupsen/logrus"
)

type Delivery struct {
	Body        []byte
	deliveryTag uint64
	channel     *amqp.Channel
}

func (d *Delivery) Ack() error {
	if d.channel == nil {
		return errors.New("channel is nil, cannot ack")
	}
	return d.channel.Ack(d.deliveryTag, false)
}

func (d *Delivery) Nack(requeue bool) error {
	if d.channel == nil {
		return errors.New("channel is nil, cannot nack")
	}
	return d.channel.Nack(d.deliveryTag, false, requeue)
}

type RabbitMQClient struct {
	amqpURI   string
	queueName string

	connMtx sync.Mutex
	conn    *amqp.Connection
	channel *amqp.Channel

	isClosed    bool
	closeMtx    sync.Mutex
	reconnectCh chan struct{}

	deliveryCh chan Delivery
}

func NewClient(amqpURI, queueName string) (*RabbitMQClient, error) {
	client := &RabbitMQClient{
		amqpURI:     amqpURI,
		queueName:   queueName,
		reconnectCh: make(chan struct{}, 1),

		deliveryCh: make(chan Delivery),
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	go client.handleReconnect()

	return client, nil
}

func (c *RabbitMQClient) connect() error {
	c.connMtx.Lock()
	defer c.connMtx.Unlock()

	var err error
	c.conn, err = amqp.Dial(c.amqpURI)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	c.channel, err = c.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	if err := c.channel.Confirm(false); err != nil {
		return fmt.Errorf("failed to set confirm mode: %w", err)
	}

	_, err = c.channel.QueueDeclare(
		c.queueName,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	go func() {
		closeErr := <-c.conn.NotifyClose(make(chan *amqp.Error))
		c.closeMtx.Lock()
		isClosedByApp := c.isClosed
		c.closeMtx.Unlock()
		if !isClosedByApp {
			log.Errorf("RabbitMQ connection closed unexpectedly. error: %v", closeErr)
			c.triggerReconnect()
		} else {
			log.Infoln("RabbitMQ connection closed gracefully by application.")
		}
	}()

	go func() {
		if c.channel != nil {
			channelCloseErr := <-c.channel.NotifyClose(make(chan *amqp.Error))
			c.closeMtx.Lock()
			isClosedByApp := c.isClosed
			c.closeMtx.Unlock()
			if !isClosedByApp {
				log.Errorf("RabbitMQ channel closed unexpectedly, error: %v", channelCloseErr)
				c.triggerReconnect()
			}
		}
	}()

	log.Infof("Successfully connected to RabbitMQ, queue: %s", c.queueName)
	return nil
}

func (c *RabbitMQClient) handleReconnect() {
	for range c.reconnectCh {
		c.closeMtx.Lock()
		if c.isClosed {
			c.closeMtx.Unlock()
			log.Infoln("Client is closed, stopping reconnect handler.")
			return
		}
		c.closeMtx.Unlock()

		log.Infoln("Attempting to reconnect to RabbitMQ...")
		backoff := 1 * time.Second
		for {
			err := c.connect()
			if err == nil {
				log.Infoln("Successfully reconnected to RabbitMQ")
				break
			}
			log.Errorf("Failed to reconnect, will retry..., error: %v, after: %v", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func (c *RabbitMQClient) triggerReconnect() {
	select {
	case c.reconnectCh <- struct{}{}:
	default:
	}
}

func (c *RabbitMQClient) Consume() (<-chan Delivery, error) {
	log.Infoln("Starting persistent consumer...")

	go func() {
		for {
			c.closeMtx.Lock()
			if c.isClosed {
				c.closeMtx.Unlock()
				log.Infoln("Consumer stopping because client is closed.")
				close(c.deliveryCh)
				return
			}
			c.closeMtx.Unlock()

			c.connMtx.Lock()
			channel := c.channel
			c.connMtx.Unlock()

			if channel == nil || channel.IsClosed() {
				log.Warningln("Consumer waiting for channel to be ready...")
				time.Sleep(2 * time.Second)
				continue
			}

			err := channel.Qos(1, 0, false)
			if err != nil {
				log.Errorf("Failed to set QOS, will retry..., error: %v", err)
				time.Sleep(2 * time.Second)
				continue
			}

			msgs, err := channel.Consume(
				c.queueName,
				"",
				false,
				false,
				false,
				false,
				nil,
			)
			if err != nil {
				log.Error("Failed to register consumer, will retry...", "error", err)
				time.Sleep(2 * time.Second)
				continue
			}

			log.Infoln("Consumer registered and waiting for messages...")

			for msg := range msgs {
				delivery := Delivery{
					Body:        msg.Body,
					deliveryTag: msg.DeliveryTag,
					channel:     channel,
				}

				c.deliveryCh <- delivery
			}

			log.Warnln("RabbitMQ message channel closed. Attempting to re-establish consumption...")
		}
	}()

	return c.deliveryCh, nil
}

func (c *RabbitMQClient) Close() error {
	c.closeMtx.Lock()
	if c.isClosed {
		c.closeMtx.Unlock()
		return errors.New("client is already closed")
	}
	c.isClosed = true
	c.closeMtx.Unlock()

	close(c.reconnectCh)

	c.connMtx.Lock()
	defer c.connMtx.Unlock()

	var finalErr error
	if c.channel != nil && !c.channel.IsClosed() {
		if err := c.channel.Close(); err != nil {
			finalErr = fmt.Errorf("channel close error: %w", err)
		}
	}
	if c.conn != nil && !c.conn.IsClosed() {
		if err := c.conn.Close(); err != nil {
			finalErr = fmt.Errorf("connection close error: %w (previous error: %v)", err, finalErr)
		}
	}

	if finalErr == nil {
		log.Infoln("RabbitMQ connection closed gracefully.")
	}
	return finalErr
}
