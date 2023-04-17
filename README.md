# Example fast kafka producer in Go

Start the kafka cluster on a local docker

```shell
cd deployment/docker
docker-compose up -d
```

Run the test

```shell
go run main.go
```

This will publish 10.000 messages and show the total and average time taken.

By default idempotency is enabled (waiting for ack from all the instances) and linger is set to 50ms instead of 5ms to try to get more messages into a single batch.
