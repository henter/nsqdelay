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

Otherwise run it with `docker-compose up`
