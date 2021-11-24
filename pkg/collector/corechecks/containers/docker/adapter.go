// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package docker

import "github.com/DataDog/datadog-agent/pkg/collector/corechecks/containers/generic"

var metricsNameMapping map[string]string = map[string]string{
	"container.uptime":                "docker.uptime",
	"container.cpu.usage":             "docker.cpu.usage",
	"container.cpu.user":              "docker.cpu.user",
	"container.cpu.system":            "docker.cpu.system",
	"container.cpu.throttled":         "docker.cpu.throttled.time",
	"container.cpu.throttled.periods": "docker.cpu.throttled",
	"container.cpu.limit":             "docker.cpu.limit",
	"container.memory.usage":          "", // Not present in legacy Docker check
	"container.memory.kernel":         "docker.kmem.usage",
	"container.memory.limit":          "docker.mem.limit",
	"container.memory.soft_limit":     "docker.mem.soft_limit",
	"container.memory.rss":            "docker.mem.rss",
	"container.memory.cache":          "docker.mem.cache",
	"container.memory.swap":           "docker.mem.swap",
	"container.memory.oom_events":     "docker.mem.failed_count",
	"container.memory.working_set":    "docker.mem.private_working_set",
	"container.memory.commit":         "docker.mem.commit_bytes",
	"container.memory.commit.peak":    "docker.mem.commit_peak_bytes",
}

var metricsValuesConverter map[string]func(float64) float64 = map[string]func(float64) float64{
	"container.cpu.usage":     generic.ConvertNanosecondsToHz,
	"container.cpu.user":      generic.ConvertNanosecondsToHz,
	"container.cpu.system":    generic.ConvertNanosecondsToHz,
	"container.cpu.throttled": generic.ConvertNanosecondsToHz,
	"container.cpu.limit":     generic.ConvertNanosecondsToHz,
}
