// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package containerlifecycle

import (
	"testing"
	"time"

	"github.com/DataDog/datadog-agent/pkg/aggregator/mocksender"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	"github.com/DataDog/datadog-agent/pkg/workloadmeta"

	"github.com/stretchr/testify/mock"
)

func TestProcessEvents(t *testing.T) {
	var mockTime = func(timeStr string) time.Time {
		t, _ := time.Parse("2006-01-02 15:04:05", timeStr)
		return t
	}

	commonTS := mockTime("2022-01-01 00:00:00")
	commonExitCode := uint32(0)

	anotherCommonTS := mockTime("2022-01-01 01:02:03")
	anotherCommonExitCode := uint32(1)

	p := &processor{hostname: "host"}
	tests := []struct {
		name     string
		evBundle workloadmeta.EventBundle
		wantFunc func(t *testing.T, s *mocksender.MockSender)
	}{
		{
			name: "docker",
			evBundle: workloadmeta.EventBundle{
				Events: []workloadmeta.Event{
					{
						Type:    workloadmeta.EventTypeUnset,
						Sources: []workloadmeta.Source{workloadmeta.SourceDocker},
						Entity:  container("cont-id", &commonExitCode, commonTS),
					},
				},
				Ch: make(chan struct{}),
			},
			wantFunc: func(t *testing.T, s *mocksender.MockSender) {
				s.AssertNumberOfCalls(t, "EventPlatformEvent", 1)

				payload := `{"version":"v1","obj_kind":"cont","obj_id":"cont-id","ev_type":"del","host":"host","ev":{"src":"docker","exit_ts":1640995200,"exit_code":0}}`
				s.AssertEventPlatformEvent(t, payload, "container-lifecycle")

				s.AssertCalled(t, "Commit")
			},
		},
		{
			name: "containerd",
			evBundle: workloadmeta.EventBundle{
				Events: []workloadmeta.Event{
					{
						Type:    workloadmeta.EventTypeUnset,
						Sources: []workloadmeta.Source{workloadmeta.SourceContainerd},
						Entity:  container("cont-id", &commonExitCode, commonTS),
					},
				},
				Ch: make(chan struct{}),
			},
			wantFunc: func(t *testing.T, s *mocksender.MockSender) {
				s.AssertNumberOfCalls(t, "EventPlatformEvent", 1)

				payload := `{"version":"v1","obj_kind":"cont","obj_id":"cont-id","ev_type":"del","host":"host","ev":{"src":"containerd","exit_ts":1640995200,"exit_code":0}}`
				s.AssertEventPlatformEvent(t, payload, "container-lifecycle")

				s.AssertCalled(t, "Commit")
			},
		},
		{
			name: "ignore set event",
			evBundle: workloadmeta.EventBundle{
				Events: []workloadmeta.Event{
					{
						Type:    workloadmeta.EventTypeSet,
						Sources: []workloadmeta.Source{workloadmeta.SourceDocker},
						Entity:  container("cont-id", &commonExitCode, commonTS),
					},
				},
				Ch: make(chan struct{}),
			},
			wantFunc: func(t *testing.T, s *mocksender.MockSender) {
				s.AssertNotCalled(t, "EventPlatformEvent")
			},
		},
		{
			name: "multiple set and unset events",
			evBundle: workloadmeta.EventBundle{
				Events: []workloadmeta.Event{
					{
						Type:    workloadmeta.EventTypeUnset,
						Sources: []workloadmeta.Source{workloadmeta.SourceContainerd},
						Entity:  container("cont-id-1", &commonExitCode, commonTS),
					},
					{
						Type:    workloadmeta.EventTypeSet,
						Sources: []workloadmeta.Source{workloadmeta.SourceContainerd},
						Entity:  container("imposter", &commonExitCode, commonTS),
					},
					{
						Type:    workloadmeta.EventTypeUnset,
						Sources: []workloadmeta.Source{workloadmeta.SourceContainerd},
						Entity:  container("cont-id-2", &anotherCommonExitCode, anotherCommonTS),
					},
				},
				Ch: make(chan struct{}),
			},
			wantFunc: func(t *testing.T, s *mocksender.MockSender) {
				s.AssertNumberOfCalls(t, "EventPlatformEvent", 2)

				payload1 := `{"version":"v1","obj_kind":"cont","obj_id":"cont-id-1","ev_type":"del","host":"host","ev":{"src":"containerd","exit_ts":1640995200,"exit_code":0}}`
				s.AssertEventPlatformEvent(t, payload1, "container-lifecycle")

				payload2 := `{"version":"v1","obj_kind":"cont","obj_id":"cont-id-2","ev_type":"del","host":"host","ev":{"src":"containerd","exit_ts":1640998923,"exit_code":1}}`
				s.AssertEventPlatformEvent(t, payload2, "container-lifecycle")

				s.AssertCalled(t, "Commit")
			},
		},
		{
			name: "pod",
			evBundle: workloadmeta.EventBundle{
				Events: []workloadmeta.Event{
					{
						Type:    workloadmeta.EventTypeUnset,
						Sources: []workloadmeta.Source{workloadmeta.SourceKubelet},
						Entity:  pod("pod-id"),
					},
				},
				Ch: make(chan struct{}),
			},
			wantFunc: func(t *testing.T, s *mocksender.MockSender) {
				s.AssertNumberOfCalls(t, "EventPlatformEvent", 1)

				payload := `{"version":"v1","obj_kind":"pod","obj_id":"pod-id","ev_type":"del","host":"host","ev":{"src":"kubelet"}}`
				s.AssertEventPlatformEvent(t, payload, "container-lifecycle")

				s.AssertCalled(t, "Commit")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := mocksender.NewMockSender(check.ID(tt.name))
			sender.On("EventPlatformEvent", mock.Anything, mock.Anything).Return()
			sender.On("Commit").Return()
			p.sender = sender

			p.processEvents(tt.evBundle)
			tt.wantFunc(t, sender)
		})
	}
}

func container(id string, exitCode *uint32, exitTS time.Time) *workloadmeta.Container {
	return &workloadmeta.Container{
		EntityID: workloadmeta.EntityID{
			Kind: workloadmeta.KindContainer,
			ID:   id,
		},
		State: workloadmeta.ContainerState{
			ExitCode:   exitCode,
			FinishedAt: exitTS,
		},
	}
}

func pod(id string) *workloadmeta.EntityID {
	return &workloadmeta.EntityID{
		Kind: workloadmeta.KindKubernetesPod,
		ID:   id,
	}
}
