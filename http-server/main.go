package main

import (
	"context"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"io"
	"net/http"
	"os"
	"time"
)

func main() {
	bootstrapServers := "localhost:9092"
	topic := "test-topic"

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"linger.ms":         0,
		//"request.required.acks": 0,
		"enable.idempotence": true,
		//"debug":     "msg",
	})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created Producer %v\n", p)

	// Listen to all the events on the default events channel
	go func() {
		for e := range p.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				// The message delivery report, indicating success or
				// permanent failure after retries have been exhausted.
				// Application level retries won't help since the client
				// is already configured to do that.
				m := ev
				ctx := m.Opaque.(context.Context)
				c := ctx.Value("channel").(chan kafka.Event)
				c <- e
				if m.TopicPartition.Error != nil {
					fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
				} else {
					fmt.Printf("Delivered message with opaque %d to topic %s [%d] at offset %v\n",
						m.Opaque, *m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
				}
			case kafka.Error:
				// Generic client instance-level errors, such as
				// broker connection failures, authentication issues, etc.
				//
				// These errors should generally be considered informational
				// as the underlying client will automatically try to
				// recover from any errors encountered, the application
				// does not need to take action on them.
				fmt.Printf("Error: %v\n", ev)
			default:
				fmt.Printf("Ignored event: %s\n", ev)
			}
		}
	}()

	msgcnt := 0

	handler := func(w http.ResponseWriter, req *http.Request) {
		c := make(chan kafka.Event)
		ctx := context.WithValue(context.TODO(), "msgcnt", msgcnt)
		ctx = context.WithValue(ctx, "channel", c)
		start := time.Now()
		value := fmt.Sprintf("Producer example, message #%d", msgcnt)

		err = p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(value),
			Headers:        []kafka.Header{{Key: "myTestHeader", Value: []byte("header values are binary")}},
			Opaque:         ctx,
		}, nil)

		if err != nil {
			if err.(kafka.Error).Code() == kafka.ErrQueueFull {
				// Producer queue is full, wait 100ms for messages
				// to be delivered then try again.
				time.Sleep(100 * time.Millisecond)
				return
			}
			fmt.Printf("Failed to produce message: %v\n", err)
		}
		msgcnt++
		io.WriteString(w, "Waiting for the message to be delivered!\n")
		e := <-c
		elapsed := time.Since(start)
		fmt.Printf("Elapsed time for event %v: %s\n", e, elapsed)
		io.WriteString(w, fmt.Sprintf("Got %v\n", e))
	}

	http.HandleFunc("/message", handler)
	err = http.ListenAndServe("0.0.0.0:3001", nil)
	if err != nil {
		fmt.Printf("Failed to create http server: #{err}\n")
		os.Exit(1)
	}

	// Flush and close the producer and the events channel
	for p.Flush(1000) > 0 {
		fmt.Print("Still waiting to flush outstanding messages\n")
	}
	p.Close()
}
