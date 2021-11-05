// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package service

import (
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/DataDog/datadog-agent/pkg/proto/pbgo"
)

func TestExpiration(t *testing.T) {
	tc := NewTracerCache(1000, time.Nanosecond, 100*time.Nanosecond) // 10 cleanups per millisecond
	defer tc.Stop()
	tracer := &pbgo.TracerInfo{RuntimeId: "foo", ServiceId: "service", Env: "env", Version: "version"}
	if err := tc.AddTracer(tracer); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond) // allow for some cleanups
	tc.cleanup()
	assert.Assert(t, len(tc.Tracers()) == 0, "Tracers should be empty eventually.")
}

func TestMaxTracers(t *testing.T) {
	tc := NewTracerCache(0, time.Hour, time.Hour)
	defer tc.Stop()
	tracer := &pbgo.TracerInfo{RuntimeId: "", ServiceId: "", Env: "", Version: ""}
	err := tc.AddTracer(tracer)
	assert.Error(t, err, "TracerCache maxCapacity reached. Refusing to add tracer ")
}
