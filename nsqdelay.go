package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"sync"
	"time"

	"github.com/bitly/go-nsq"
	"github.com/garyburd/redigo/redis"
)

// Message from nsq delayed queue, in a JSON encoded message body
type Message struct {
	Id		string   `json:"id"`
	Topic	string   `json:"topic"`
	Body	string   `json:"body"`
	SendIn	int64	 `json:"send_in"`
}

var RedisPool *redis.Pool
var redis_address, redis_key string

// pub messages to target nsq topic
var publish chan *Message

func main() {
	// parse command line arguments
	var lookupd, topic, nsqd string

	flag.StringVar(&lookupd, "lookupd_http_address", "http://127.0.0.1:4161", "lookupd HTTP address")
	flag.StringVar(&nsqd, "nsqd_tcp_address", "127.0.0.1:4150", "nsqd TCP address")
	flag.StringVar(&redis_address, "redis_address", "127.0.0.1:6379", "redis address")
	flag.StringVar(&topic, "topic", "delayed", "NSQD topic for delayed messages")
	flag.Parse()

	if lookupd == "" || topic == "" || nsqd == "" || redis_address == "" {
		flag.PrintDefaults()
		log.Fatal("invalid arguments")
	}

	redis_key = "nsqdelay_" + topic

	// initialize a consumer for delayed messages
	c, err := nsq.NewConsumer(topic, "nsqdelay_scheduler", nsq.NewConfig())
	if err != nil {
		log.Fatal(err)
	}

	// consume nsq delayed queue, insert to redis
	c.AddHandler(nsq.HandlerFunc(messageHandler))
	if err := c.ConnectToNSQLookupd(lookupd); err != nil {
		log.Fatal(err)
	}

	// initialize a producer
	p, err := nsq.NewProducer(nsqd, nsq.NewConfig())
	if err != nil {
		log.Fatal(err)
	}

	// messages to target
	publish = make(chan *Message)

	// consume from redis zset, send to publish chan
	go func() {
		for {
			consumeRedis(publish)
			time.Sleep(time.Second)
		}
	}()

	// consume publish chan, send to target nsq topic
	go publishHandler(p, publish)

	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
}

// read redis zset, send to nsq target topic
func consumeRedis(publish chan *Message) {
	conn := newRedisPool().Get()
	defer conn.Close()

	now := time.Now().Unix()
	rows, err := redis.Strings(conn.Do("ZRANGEBYSCORE", redis_key, "-inf", now))
	if err != nil {
		log.Printf(err.Error())
	}

	for _, s := range rows {
		var msg Message

		err := json.Unmarshal([]byte(s), &msg)
		if err != nil {
			log.Printf("message encode error " + s)
			continue
		}

		log.Printf("message from redis " + s)
		publish <- &msg
	}

	conn.Do("ZREMRANGEBYSCORE", redis_key, "-inf", now)
}

// insert into redis zset, fire timestamp as score
func insertToRedis(msg *Message) {
	conn := newRedisPool().Get()
	defer conn.Close()

	now := time.Now().Unix()
	execute_timestamp := now + msg.SendIn

	bytes, _ := json.Marshal(msg)
	_, err := conn.Do("ZADD", redis_key, execute_timestamp, bytes)
	if err != nil {
		log.Printf(err.Error())
	}
	log.Printf("message insert to redis " + string(bytes))
}

// handler nsq incoming messages, insert to redis
func messageHandler(m *nsq.Message) error {
	defer m.Finish()
	var msg Message

	if err := json.Unmarshal(m.Body, &msg); err != nil {
		log.Print(err)
		return err
	}

	// data validation
	if msg.Topic == "" || msg.Body == "" || msg.SendIn == 0 {
		log.Print("invalid delayed message data " + string(m.Body))
		return errors.New("invalid delayed message data")
	}

	msg.Id = string(m.ID[:nsq.MsgIDLength])
	insertToRedis(&msg)
	log.Print("received delayed message from nsq " + string(m.Body))

	return nil
}

// send message to nsq target topic
func publishHandler(p *nsq.Producer, publish chan *Message) {
	for {
		m := <-publish
		if err := p.Publish(m.Topic, []byte(m.Body)); err != nil {
			log.Print(err)

			// retry to send the message in 1 second
			go func(m *Message) {
				time.Sleep(time.Second)
				publish <- m
			}(m)

			continue
		}

		log.Printf("published message '%s' to topic '%s'", m.Body, m.Topic)
	}
}

func newRedisPool() *redis.Pool {
	if RedisPool == nil {
		RedisPool = &redis.Pool{
			MaxIdle:     50,
			MaxActive:   1000,
			IdleTimeout: 3600 * time.Second,
			Dial: func() (c redis.Conn, err error) {
				c, err = redis.Dial("tcp", redis_address)
				if err != nil {
					log.Fatalf(err.Error())
				}
				return c, err
			},
			TestOnBorrow: func(c redis.Conn, t time.Time) error {
				_, err := c.Do("PING")
				return err
			},
		}
	}

	return RedisPool
}
