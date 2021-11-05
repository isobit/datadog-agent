// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/DataDog/datadog-agent/pkg/config/remote/service/tuf"
	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

const errorRetryInterval = 3 * time.Second

// SubscriberCallback defines the function called when a new configuration was fetched
type SubscriberCallback func(config *pbgo.ConfigResponse) error

// Subscriber describes a product's configuration subscriber
type Subscriber struct {
	product     pbgo.Product
	refreshRate time.Duration
	lastUpdate  time.Time
	lastVersion uint64
	callback    SubscriberCallback
}

// NewSubscriber returns a new subscriber with the specified refresh rate and a callback
func NewSubscriber(product pbgo.Product, refreshRate time.Duration, callback SubscriberCallback) *Subscriber {
	return &Subscriber{
		product:     product,
		refreshRate: refreshRate,
		callback:    callback,
	}
}

// NewChanSubscriber returns a new subscriber that will put the ConfigResponse objects onto a provided channel.
func NewChanSubscriber(product pbgo.Product, refreshRate time.Duration, channel chan *pbgo.ConfigResponse) *Subscriber {
	return NewSubscriber(product, refreshRate, func(config *pbgo.ConfigResponse) error {
		select {
		case channel <- config:
			return nil
		default:
			return errors.New("failed to put config onto channel")
		}
	})
}

// NewGRPCSubscriber returns a new gRPC stream based subscriber.
func NewGRPCSubscriber(product pbgo.Product, callback SubscriberCallback) (context.CancelFunc, error) {
	currentConfigSnapshotVersion := uint64(0)
	client := tuf.NewDirectorPartialClient()

	cas, err := newCoreAgentStream()
	if err != nil {
		cas.cancel()
		return nil, fmt.Errorf("failed to create core agent stream: %w", err)
	}
	defer cas.streamCancel()

	go func() {
		log.Debug("Waiting for configuration from remote config management")

		for {
			request := pbgo.SubscribeConfigRequest{
				CurrentConfigSnapshotVersion: currentConfigSnapshotVersion,
				Product:                      product,
			}
			stream := cas.getStream()
			err := stream.Send(&request)
			if err != nil {
				log.Errorf("Error sending message to core agent: %s", err)
				continue
			}

			for {
				// Get new event from stream
				configResponse, err := stream.Recv()
				if err == io.EOF {
					continue
				} else if err != nil {
					log.Warnf("Stopped listening for configuration from remote config management: %s", err)
					time.Sleep(errorRetryInterval)
					break
				}

				if err := client.Verify(configResponse); err != nil {
					log.Errorf("Partial verify failed: %s", err)
					continue
				}

				log.Infof("Got config for product %s", product)
				if err := callback(configResponse); err == nil {
					currentConfigSnapshotVersion = configResponse.ConfigSnapshotVersion
				}
			}
		}
	}()

	return cas.cancel, nil
}

// NewTracerGRPCSubscriber returns a new gRPC stream based subscriber. The subscriber sends tracer infos to core agent
// and listens for configuration updates from core agent asynchronously.
func NewTracerGRPCSubscriber(product pbgo.Product, callback SubscriberCallback, tracerInfos chan *pbgo.TracerInfo) (context.CancelFunc, error) {
	cas, err := newCoreAgentStream()
	if err != nil {
		return nil, fmt.Errorf("error creating core agent stream: %s", err)
	}

	go cas.sendTracerInfos(tracerInfos, product)
	go cas.readConfigs(product, callback)

	return cas.cancel, nil
}
