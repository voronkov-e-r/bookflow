package main

import (
	"context"
	"encoding/json"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rabbitmq/amqp091-go"
)

type OrderCreatedEvent struct {
	Event     string `json:"event"`
	OrderId   string `json:"order_id"`
	UserId    string `json:"user_id"`
	BookId    string `json:"book_id"`
	BookTitle string `json:"book_title"`
	CreatedAt int64  `json:"created_at"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		rabbitURL = "amqp://admin:secret@localhost:5672/"
	}

	conn, err := amqp091.Dial(rabbitURL)
	if err != nil {
		log.Fatalf("RabbitMQ connection error: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Channel error: %v", err)
	}
	defer ch.Close()

	if err := ch.ExchangeDeclare("orders", "fanout", true, false, false, false, nil); err != nil {
		log.Fatalf("Exchange declare error: %v", err)
	}

	queue, err := ch.QueueDeclare("notification", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Queue declare error: %v", err)
	}

	queueAnalitycs, err := ch.QueueDeclare("analityc", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Queue declare error: %v", err)
	}

	err = ch.QueueBind(queueAnalitycs.Name, "", "orders", false, nil)
	if err != nil {
		log.Fatalf("Queue bind error: %v", err)
	}

	if err := ch.QueueBind(queue.Name, "", "orders", false, nil); err != nil {
		slog.Error("bind", "err", err)
		os.Exit(1)
	}

	if err := ch.Qos(10, 0, false); err != nil {
		log.Fatalf("Qos error: %v", err)
	}

	msgs, err := ch.Consume(queue.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Consumer error: %v", err)
	}

	analitycMsgs, err := ch.Consume(queueAnalitycs.Name, "", false, false, false, false, nil)
	if err != nil {
		log.Fatalf("Consumer error: %v", err)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		slog.Info("shutting down...")
		ch.Cancel("", false)
		cancel()
	}()

	slog.Info("notification worker started", "queue", queue.Name)

	for {
		select {
		case <-ctx.Done():
			slog.Info("worker stopped")
			return

		case msg, ok := <-msgs:
			if !ok {
				return
			}
			handle(msg)

		case msg, ok := <-analitycMsgs:
			if !ok {
				return
			}
			handleAnalitycs(msg)
		}
	}
}

func handle(msg amqp091.Delivery) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in handler", "panic", r)
			msg.Nack(false, false)
		}
	}()

	var ev OrderCreatedEvent
	err := json.Unmarshal(msg.Body, &ev)
	if err != nil {
		slog.Error("Unmarshal error", "error", err)
		return
	}

	if ev.Event != "order.created" {
		msg.Ack(false)
		return
	}

	slog.Info("sending notification",
		"event", ev.Event,
		"order_id", ev.OrderId,
		"user_id", ev.UserId,
		"book_title", ev.BookTitle,
	)

	time.Sleep(50 * time.Millisecond)

	msg.Ack(false)
	log.Println("notification sent")
}

func handleAnalitycs(msg amqp091.Delivery) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic in handler", "panic", r)
			msg.Nack(false, false)
		}
	}()

	msg.Ack(false)
	slog.Info("saved to analitycs")
}
