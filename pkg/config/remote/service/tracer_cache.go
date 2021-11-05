// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// TracerCache is a thread-safe, auto-cleaning cache for tracer infos.
type TracerCache struct {
	tracerInfos map[string]*entry
	maxEntries  int
	ttl         time.Duration
	mutex       sync.Mutex
	ticker      *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
}

type entry struct {
	tracerInfo *pbgo.TracerInfo
	lastSeen   time.Time
}

// NewTracerCache returns a new TracerCache and starts periodic cleanup.
func NewTracerCache(maxEntries int, ttl time.Duration, cleanupInterval time.Duration) *TracerCache {
	ctx, cancelFunc := context.WithCancel(context.Background())
	tc := &TracerCache{
		tracerInfos: make(map[string]*entry),
		maxEntries:  maxEntries,
		ttl:         ttl,
		ticker:      time.NewTicker(cleanupInterval),
		ctx:         ctx,
		cancel:      cancelFunc,
	}

	go tc.startLoop()
	log.Infof("Started tracer cache with maxEntries = %d and ttl = %s", maxEntries, ttl)

	return tc
}

// Stop stops the automatic cleanup.
func (tc *TracerCache) Stop() {
	tc.cancel()
}

func (tc *TracerCache) startLoop() {
	for {
		select {
		case <-tc.ticker.C:
			tc.cleanup()
		case <-tc.ctx.Done():
			return
		}
	}
}

func (tc *TracerCache) cleanup() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	log.Debugf("Running tracer cleanup, currently %d entries", len(tc.tracerInfos))
	for runtimeID, tracer := range tc.tracerInfos {
		if time.Now().Sub(tracer.lastSeen) >= tc.ttl {
			log.Debugf("Expiring tracer %s, last seen: %s", tracer.tracerInfo.RuntimeId, tracer.lastSeen.UTC())
			delete(tc.tracerInfos, runtimeID)
		}
	}
}

// AddTracer add a tracer to the cache.
func (tc *TracerCache) AddTracer(tracer *pbgo.TracerInfo) error {
	log.Debugf("Adding tracer %s", tracer)
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	if len(tc.tracerInfos) >= tc.maxEntries {
		return fmt.Errorf("TracerCache maxCapacity reached. Refusing to add tracer %s", tracer)
	}
	tc.tracerInfos[tracer.RuntimeId] = &entry{tracer, time.Now()}
	return nil
}

// Tracers get all tracers currently considered alive in the cache.
func (tc *TracerCache) Tracers() []*pbgo.TracerInfo {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tracers := make([]*pbgo.TracerInfo, len(tc.tracerInfos))
	for _, e := range tc.tracerInfos {
		tracers = append(tracers, e.tracerInfo)
	}
	return tracers
}
