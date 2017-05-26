nsqdelay - a delayed message queue for NSQ
==========================================

[![Build Status](https://travis-ci.org/henter/nsqdelay.svg?branch=master)](https://travis-ci.org/henter/nsqdelay)

__nsqdelay__ can be used for sending delayed messages on top of NSQ,
it listens on the __delayed__ topic by default (configurable) and receives JSON encoded messages with the following structure:

```
{
  "topic": "my_topic",
  "body": "message_body",
  "send_in": 10
}
```
send "message_body" to nsq topic "my_topic" after 10 seconds


It persists the messages to redis and publish them when the time comes.

Usage
-----
For all command line arguments use `docker run --rm henter/nsqdelay -h`

```
  -lookupd_http_address string
    	lookupd HTTP address (default "http://127.0.0.1:4161")
  -nsqd_tcp_address string
    	nsqd TCP address (default "127.0.0.1:4150")
  -redis_address string
    	redis address (default "127.0.0.1:6379")
  -topic string
    	NSQD topic for delayed messages (default "delayed")
```

Otherwise run it with `docker-compose up` (build container, links to nsq and redis)
