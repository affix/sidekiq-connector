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

	"github.com/jrallison/go-workers"
	"github.com/openfaas-incubator/connector-sdk/types"
)

type connectorConfig struct {
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
		Client: types.MakeClient(config.upstreamTimeout),
	}

	creds := types.GetCredentials()
	controllerconfig := &types.ControllerConfig{
		RebuildInterval:   time.Millisecond * 1000,
		GatewayURL:        "http://127.0.0.1:8080",
		PrintResponse:     true,
		PrintResponseBody: true,
	}

	controller := types.NewController(creds, controllerconfig)

	receiver := ResponseReceiver{}
	controller.Subscribe(&receiver)

	controller.BeginMapBuilder()

	ticker := time.NewTicker(config.rebuildInterval)
	go synchronizeLookups(ticker, &lookupBuilder, &topicMap)

	workers.Configure(map[string]string{
		"server":   config.redis_host,
		"database": "0",
		"pool":     "30",
		"process":  "1",
	})

	for _, queue := range config.queues {
		handler := makeMessageHandler(controller, queue)
		workers.Process(queue, handler, 10)
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

func makeMessageHandler(controller *types.Controller, queue string) func(msg *workers.Msg) {

	mcb := func(msg *workers.Msg) {
		msgJson, err := json.Marshal(msg.Args)

		if err != nil {
			log.Fatal(err.Error())
		}
		controller.Invoke(queue, &msgJson)
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
		upstreamTimeout: upstreamTimeout,
		queues:          queues,
		rebuildInterval: rebuildInterval,
		redis_host:      redis,
		printResponse:   printResponse,
	}
}

type ResponseReceiver struct {
}

func (ResponseReceiver) Response(res types.InvokerResponse) {
	if res.Error != nil {
		log.Printf("tester got error: %s", res.Error.Error())
	} else {
		log.Printf("tester got result: [%d] %s => %s (%d) bytes", res.Status, res.Topic, res.Function, len(*res.Body))
	}
}
