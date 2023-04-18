# Example fast kafka producer in Go

Start the kafka cluster on a local docker

```shell
cd deployment/docker
docker-compose up -d
```

## Producer with a go routine for feedback

Run the test

```shell
go run main.go
```

This will publish 10.000 messages and show the total and average time taken.

By default idempotency is enabled (waiting for ack from all the instances) and linger is set to 50ms instead of 5ms to try to get more messages into a single batch.

## Producer with a custom channel for feedback

This is the slowest of the bunch as it uses a channel to provide feedback but since the messages are produced in a serial way and it waits for feedback, it takes the lingering time (50ms) plus the actual time taken.

Run the test

```shell
go run custom-channel/main.go
```

## Producer with a context in the Opaque field of the message

This is the same as the main test but it passes an int value as the Opaque interface of the Kafka message. That way it can be consumed in the feedback go routine.

Run the test

```shell
go run context/main.go
```

## Producer with a custom http server and correlation to the produced message

This is the actual solution to our problem apparently.
We spawn an http server and listen on GET http://0.0.0.0:3001/message 
Whenever the end-point is invoked, a new message is produced and a context is passed in the message. The https call remains blocked until the feedback comes from the go routine. It is correlated through a Context that contains the channel to use to provide the feedback to the right handler.
This is using a linger value of 0 to force the message to be sent to Kafka as soon as it is produced.

Run the test

```shell
go run http-server/main.go
```

Observed results

```text
ab -n 10000 -c 50 http://localhost:3001/message
This is ApacheBench, Version 2.3 <$Revision: 1901567 $>
Copyright 1996 Adam Twiss, Zeus Technology Ltd, http://www.zeustech.net/
Licensed to The Apache Software Foundation, http://www.apache.org/

Benchmarking localhost (be patient)
Completed 1000 requests
Completed 2000 requests
Completed 3000 requests
Completed 4000 requests
Completed 5000 requests
Completed 6000 requests
Completed 7000 requests
Completed 8000 requests
Completed 9000 requests
Completed 10000 requests
Finished 10000 requests


Server Software:
Server Hostname:        localhost
Server Port:            3001

Document Path:          /message
Document Length:        65 bytes

Concurrency Level:      50
Time taken for tests:   1.563 seconds
Complete requests:      10000
Failed requests:        0
Total transferred:      1820000 bytes
HTML transferred:       650000 bytes
Requests per second:    6397.45 [#/sec] (mean)
Time per request:       7.816 [ms] (mean)
Time per request:       0.156 [ms] (mean, across all concurrent requests)
Transfer rate:          1137.05 [Kbytes/sec] received

Connection Times (ms)
              min  mean[+/-sd] median   max
Connect:        0    1   8.7      1     242
Processing:     2    7  15.2      5     256
Waiting:        2    7  15.2      5     256
Total:          3    8  17.8      6     270

Percentage of the requests served within a certain time (ms)
  50%      6
  66%      6
  75%      7
  80%      7
  90%      9
  95%     11
  98%     25
  99%     39
 100%    270 (longest request)
```