// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"net/url"
	"strings"

	kafka "github.com/Shopify/sarama"
	"go.uber.org/zap"
)

func init() {
	err := zap.RegisterSink("kafka", kafkaSinkFactory)
	if err != nil {
		panic(err)
	}
}

// kafkaSink is a Kafka sink
type kafkaSink struct {
	producer kafka.SyncProducer
	topic    string
	key      string
}

// kafkaSinkFactory is a factory for the Kafka sink
func kafkaSinkFactory(u *url.URL) (zap.Sink, error) {
	topic := "kafka_default_topic"
	key := "kafka_default_key"
	m, _ := url.ParseQuery(u.RawQuery)
	if len(m["topic"]) != 0 {
		topic = m["topic"][0]
	}

	if len(m["key"]) != 0 {
		key = m["key"][0]
	}

	brokers := strings.Split(u.Host, ",")
	config := kafka.NewConfig()
	config.Producer.Return.Successes = true

	producer, err := kafka.NewSyncProducer(brokers, config)
	if err != nil {
		return kafkaSink{}, err
	}

	return kafkaSink{
		producer: producer,
		topic:    topic,
		key:      key,
	}, nil
}

// Write implements zap.Sink Write function
func (s kafkaSink) Write(b []byte) (int, error) {
	var returnErr error
	for _, topic := range strings.Split(s.topic, ",") {
		if s.key != "" {
			_, _, err := s.producer.SendMessage(&kafka.ProducerMessage{
				Topic: topic,
				Key:   kafka.StringEncoder(s.key),
				Value: kafka.ByteEncoder(b),
			})
			if err != nil {
				returnErr = err
			}
		} else {
			_, _, err := s.producer.SendMessage(&kafka.ProducerMessage{
				Topic: topic,
				Value: kafka.ByteEncoder(b),
			})
			if err != nil {
				returnErr = err
			}
		}

	}
	return len(b), returnErr
}

// Sync implement zap.Sink func Sync
func (s kafkaSink) Sync() error {
	return nil
}

// Close implements zap.Sink Close function
func (s kafkaSink) Close() error {
	return nil
}
