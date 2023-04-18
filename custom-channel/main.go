package main

import (
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"os"
	"time"
)

func main() {
	bootstrapServers := "localhost:9092"
	topic := "test-topic"
	totalMsgCount := 50

	p, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": bootstrapServers,
		"linger.ms":         50,
		//"request.required.acks": 0,
		"enable.idempotence": true,
		//"debug":     "msg",
	})

	if err != nil {
		fmt.Printf("Failed to create producer: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("Created Producer %v\n", p)

	c := make(chan kafka.Event, 100)

	start := time.Now()
	msgcnt := 0
	for msgcnt < totalMsgCount {
		value := fmt.Sprintf("Producer example, message #%d", msgcnt)

		err = p.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          []byte(value),
			Headers:        []kafka.Header{{Key: "myTestHeader", Value: []byte("header values are binary")}},
		}, c)

		if err != nil {
			if err.(kafka.Error).Code() == kafka.ErrQueueFull {
				// Producer queue is full, wait 100ms for messages
				// to be delivered then try again.
				time.Sleep(100 * time.Millisecond)
				continue
			}
			fmt.Printf("Failed to produce message: %v\n", err)
		}

		e := <-c
		switch ev := e.(type) {
		case *kafka.Message:
			// The message delivery report, indicating success or
			// permanent failure after retries have been exhausted.
			// Application level retries won't help since the client
			// is already configured to do that.
			m := ev
			if m.TopicPartition.Error != nil {
				fmt.Printf("Delivery failed: %v\n", m.TopicPartition.Error)
			} else {
				//fmt.Printf("Delivered message to topic %s [%d] at offset %v\n",
				//	*m.TopicPartition.Topic, m.TopicPartition.Partition, m.TopicPartition.Offset)
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
		msgcnt++
	}

	// Flush and close the producer and the events channel
	for p.Flush(1000) > 0 {
		fmt.Print("Still waiting to flush outstanding messages\n")
	}
	elapsed := time.Since(start)
	mean := fmt.Sprintf("%fs", elapsed.Seconds()/float64(totalMsgCount))
	average, err := time.ParseDuration(mean)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Elapsed time for %d messages: %s. Average: %s\n", totalMsgCount, elapsed, average)
	p.Close()
}
