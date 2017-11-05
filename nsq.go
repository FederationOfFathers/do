package main

import (
	"encoding/json"
	"os"

	nsq "github.com/nsqio/go-nsq"
	"go.uber.org/zap"
)

var nsqTopic = "fof-work"
var nsqChannel = "do"
var nsqAddress = "127.0.0.1:4150"
var nsqC *nsq.Consumer
var nsqP *nsq.Producer
var handlers = map[int]map[string]func(json.RawMessage) error{
	1: map[string]func(json.RawMessage) error{},
}

func init() {
	if t := os.Getenv("NSQ_TOPIC"); t != "" {
		nsqTopic = t
	}

	if c := os.Getenv("NSQ_CHANNEL"); c != "" {
		nsqChannel = c
	}

}

func initConsumer() {
	consumer, err := nsq.NewConsumer(nsqTopic, nsqChannel, nsq.NewConfig())
	if err != nil {
		logger.Fatal("Error creating NSQ consumer", zap.Error(err))
	}
	nsqC = consumer
	nsqC.AddHandler(nsq.HandlerFunc(handle))
	if err := nsqC.ConnectToNSQD(nsqAddress); err != nil {
		logger.Fatal("Error connecting to NSQ", zap.Error(err))
	}
}

func initProducer() {
	producer, err := nsq.NewProducer(nsqAddress, nsq.NewConfig())
	if err != nil {
		logger.Fatal("Error creating NSQ Producer", zap.Error(err))
	}
	if err := producer.Ping(); err != nil {
		logger.Fatal("Error connecting to NSQ Producer", zap.Error(err))
	}
	nsqP = producer
}

func handle(m *nsq.Message) error {
	logger.Info("Got NSQ Message", zap.Int64("key", m.Timestamp))
	return doJob(m.Body)
}
