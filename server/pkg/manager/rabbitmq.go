package manager

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQClient struct {
	amqpURI   string
	queueName string
	logger    *slog.Logger

	connMtx sync.Mutex
	conn    *amqp.Connection
	channel *amqp.Channel

	isClosed    bool
	closeMtx    sync.Mutex
	reconnectCh chan struct{}
}

func CreateRabbitMQClient(amqpURI, queueName string, logger *slog.Logger) (*RabbitMQClient, error) {
	client := &RabbitMQClient{
		amqpURI:     amqpURI,
		queueName:   queueName,
		logger:      logger.With("component", "RabbitMQClient"),
		reconnectCh: make(chan struct{}, 1),
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	go client.handleReconnect()

	return client, nil
}

func (client *RabbitMQClient) connect() error {
	client.connMtx.Lock()
	defer client.connMtx.Unlock()

	var err error
	client.conn, err = amqp.Dial(client.amqpURI)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	client.channel, err = client.conn.Channel()
	if err != nil {
		return fmt.Errorf("failed to open a channel: %w", err)
	}

	if err := client.channel.Confirm(false); err != nil {
		return fmt.Errorf("failed to confirm channel: %w", err)
	}

	_, err = client.channel.QueueDeclare(client.queueName, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("failed to declare a queue: %w", err)
	}

	go func() {
		closeErr := <-client.conn.NotifyClose(make(chan *amqp.Error))
		client.logger.Error("rabbitmq connection closed", "error", closeErr)
		client.triggerReconnect()
	}()

	client.logger.Info("RabbitMQ client connected", "queue", client.queueName)
	return nil
}

func (client *RabbitMQClient) handleReconnect() {
	for range client.reconnectCh {
		client.closeMtx.Lock()
		if client.isClosed {
			client.closeMtx.Unlock()
			return
		}
		client.closeMtx.Unlock()

		client.logger.Info("attempting to reconnect to RabbitMQ")
		backoff := 1 * time.Second
		for {
			err := client.connect()
			if err == nil {
				client.logger.Info("successfully reconnect to RabbitMQ")
				break
			}
			client.logger.Error("failed to reconnect to RabbitMQ", "error", err, "after", backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
		}
	}
}

func (client *RabbitMQClient) triggerReconnect() {
	select {
	case client.reconnectCh <- struct{}{}:
	default:
	}
}

func (client *RabbitMQClient) Publish(ctx context.Context, body string) error {
	client.connMtx.Lock()
	defer client.connMtx.Unlock()

	if client.channel == nil {
		return errors.New("channel is not initialized, possibly disconnected")
	}

	confirms := client.channel.NotifyPublish(make(chan amqp.Confirmation, 1))

	err := client.channel.PublishWithContext(ctx,
		"",
		client.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         []byte(body),
		})

	if err != nil {
		return fmt.Errorf("failed to publish a message: %w", err)
	}

	select {
	case confirm := <-confirms:
		if !confirm.Ack {
			return fmt.Errorf("failed to publish a message: nacked by server")
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		return errors.New("failed to publish a message: timeout")
	}

	client.logger.Debug("successfully published a message")
	return nil
}

func (client *RabbitMQClient) Close() error {
	client.closeMtx.Lock()
	client.isClosed = true
	client.closeMtx.Unlock()

	close(client.reconnectCh)

	client.connMtx.Lock()
	defer client.connMtx.Unlock()

	var finalErr error
	if client.channel != nil {
		if err := client.channel.Close(); err != nil {
			finalErr = fmt.Errorf("channel close error: %w", err)
		}
	}
	if client.conn != nil {
		if err := client.conn.Close(); err != nil {
			if finalErr != nil {
				finalErr = fmt.Errorf("connection close error: %w (previous error: %v)", err, finalErr)
			} else {
				finalErr = fmt.Errorf("connection close error: %w", err)
			}
		}
	}

	if finalErr == nil {
		client.logger.Info("RabbitMQ connection closed gracefully.")
	}
	return finalErr
}
