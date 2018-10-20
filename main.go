// Copyright (c) OpenFaaS Project 2018. All rights reserved.
// Copyright (c) Keiran Smith 2018. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/affix/sidekiq-connector/types"
	"github.com/jrallison/go-workers"
)

type connectorConfig struct {
	gatewayURL      string
	upstreamTimeout time.Duration
	queues          []string
	printResponse   bool
	rebuildInterval time.Duration
	redis_host      string
}

func main() {
	config := buildConnectorConfig()

	topicMap := types.NewTopicMap()

	lookupBuilder := types.FunctionLookupBuilder{
		GatewayURL: config.gatewayURL,
		Client:     types.MakeClient(config.upstreamTimeout),
	}

	ticker := time.NewTicker(config.rebuildInterval)
	go synchronizeLookups(ticker, &lookupBuilder, &topicMap)

	workers.Configure(map[string]string{
		"server":   config.redis_host,
		"database": "0",
		"pool":     "30",
		"process":  "1",
	})

	for _, queue := range config.queues {
		handler := makeMessageHandler(&topicMap, config, queue)
		workers.Process("myqueue", handler, 10)
	}

	workers.Run()
}

func synchronizeLookups(ticker *time.Ticker,
	lookupBuilder *types.FunctionLookupBuilder,
	topicMap *types.TopicMap) {

	for {
		<-ticker.C
		lookups, err := lookupBuilder.Build()
		if err != nil {
			log.Fatalln(err)
		}

		log.Println("Syncing topic map")
		topicMap.Sync(&lookups)
	}
}

func makeMessageHandler(topicMap *types.TopicMap, config connectorConfig, queue string) func(msg *workers.Msg) {

	invoker := types.Invoker{
		PrintResponse: config.printResponse,
		Client:        types.MakeClient(config.upstreamTimeout),
		GatewayURL:    config.gatewayURL,
	}

	mcb := func(msg *workers.Msg) {
		msgJson, err := json.Marshal(msg.Args)

		if err != nil {
			log.Fatal(err.Error())
		}
		invoker.Invoke(topicMap, queue, &msgJson)
	}
	return mcb
}

func buildConnectorConfig() connectorConfig {

	redis := "redis_host"
	if val, exists := os.LookupEnv("redis_host"); exists {
		redis = val
	}

	queues := []string{}
	if val, exists := os.LookupEnv("queues"); exists {
		for _, topic := range strings.Split(val, ",") {
			if len(topic) > 0 {
				queues = append(queues, topic)
			}
		}
	}
	if len(queues) == 0 {
		log.Fatal(`Provide a list of queues i.e. queues="payment_published,slack_joined"`)
	}

	gatewayURL := "http://gateway:8080"
	if val, exists := os.LookupEnv("gateway_url"); exists {
		gatewayURL = val
	}

	upstreamTimeout := time.Second * 30
	rebuildInterval := time.Second * 3

	if val, exists := os.LookupEnv("upstream_timeout"); exists {
		parsedVal, err := time.ParseDuration(val)
		if err == nil {
			upstreamTimeout = parsedVal
		}
	}

	if val, exists := os.LookupEnv("rebuild_interval"); exists {
		parsedVal, err := time.ParseDuration(val)
		if err == nil {
			rebuildInterval = parsedVal
		}
	}

	printResponse := false
	if val, exists := os.LookupEnv("print_response"); exists {
		printResponse = (val == "1" || val == "true")
	}

	return connectorConfig{
		gatewayURL:      gatewayURL,
		upstreamTimeout: upstreamTimeout,
		queues:          queues,
		rebuildInterval: rebuildInterval,
		redis_host:      redis,
		printResponse:   printResponse,
	}
}
