// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package model

// Payload holds the container lifecycle payload content
type Payload struct {
	Version    string `json:"version"`
	ObjectKind string `json:"obj_kind"`
	ObjectID   string `json:"obj_id"`
	EventType  string `json:"ev_type"`
	Host       string `json:"host"`
	Event      Event  `json:"ev"`
}

// Event holds the container lifecycle event details
type Event struct {
	Source        string  `json:"src"`
	ExitTimestamp int64   `json:"exit_ts,omitempty"`
	ExitCode      *uint32 `json:"exit_code,omitempty"`
}
