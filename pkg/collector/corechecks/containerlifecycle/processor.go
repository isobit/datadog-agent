// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package containerlifecycle

import (
	"context"
	"encoding/json"

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/collector/corechecks/containerlifecycle/model"
	"github.com/DataDog/datadog-agent/pkg/epforwarder"
	"github.com/DataDog/datadog-agent/pkg/util"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/workloadmeta"
)

const (
	payloadV1           = "v1"
	eventNameDelete     = "del"
	objectKindContainer = "cont"
	objectKindPod       = "pod"
)

type processor struct {
	hostname string
	sender   aggregator.Sender
}

func newProcessor(sender aggregator.Sender) (*processor, error) {
	hostname, err := util.GetHostname(context.TODO())
	if err != nil {
		return nil, err
	}

	return &processor{
		hostname: hostname,
		sender:   sender,
	}, nil
}

// processEvents handles workloadmeta events, supports pods and container unset events.
func (p *processor) processEvents(evBundle workloadmeta.EventBundle) {
	close(evBundle.Ch)

	log.Tracef("Processing %d events", len(evBundle.Events))

	for _, event := range evBundle.Events {
		entityID := event.Entity.GetID()

		switch event.Type {
		case workloadmeta.EventTypeUnset:
			switch entityID.Kind {
			case workloadmeta.KindContainer:
				container, ok := event.Entity.(*workloadmeta.Container)
				if !ok {
					log.Debugf("Expected workloadmeta.Container got %T, skipping", event.Entity)
					continue
				}

				payload, err := p.transformContainer(container, event.Sources)
				if err != nil {
					log.Debugf("Couldn't generate container lifecycle payload: %w - Entity dump: %+v", err, container)
					continue
				}

				log.Tracef("Sending container lifecycle payload %q", payload)
				p.sender.EventPlatformEvent(payload, epforwarder.EventTypeContainerLifecycle)

			case workloadmeta.KindKubernetesPod:
				payload, err := p.transformPod(event.Entity)
				if err != nil {
					log.Debugf("Couldn't generate container lifecycle payload: %w - Entity dump: %+v", err, event.Entity)
					continue
				}

				log.Tracef("Sending container lifecycle payload %q", payload)
				p.sender.EventPlatformEvent(payload, epforwarder.EventTypeContainerLifecycle)

			case workloadmeta.KindECSTask: // not supported
			default:
				log.Debugf("Cannot handle event for entity %q with kind %q", entityID.ID, entityID.Kind)
			}

		case workloadmeta.EventTypeSet: // not supported
		default:
			log.Debugf("Cannot handle event of type %d", event.Type)
		}
	}

	p.sender.Commit()
}

// transformContainer builds a payload json string from a workloadmeta.Container object
func (p *processor) transformContainer(container *workloadmeta.Container, sources []workloadmeta.Source) (string, error) {
	event := model.Event{}
	if len(sources) > 0 {
		event.Source = string(sources[0])
	}

	if !container.State.FinishedAt.IsZero() {
		event.ExitTimestamp = container.State.FinishedAt.Unix()
	}

	if container.State.ExitCode != nil {
		exitCode := *container.State.ExitCode
		event.ExitCode = &exitCode
	}

	payload := &model.Payload{
		Version:    payloadV1,
		EventType:  eventNameDelete,
		ObjectKind: objectKindContainer,
		ObjectID:   container.ID,
		Host:       p.hostname,
		Event:      event,
	}

	return toJSONString(payload)
}

// transformPod builds a payload json string from a workloadmeta.Entity representing a pod object
func (p *processor) transformPod(pod workloadmeta.Entity) (string, error) {
	payload := &model.Payload{
		Version:    payloadV1,
		EventType:  eventNameDelete,
		ObjectKind: objectKindPod,
		ObjectID:   pod.GetID().ID,
		Host:       p.hostname,
		Event: model.Event{
			Source: string(workloadmeta.SourceKubelet),
		},
	}

	return toJSONString(payload)
}

func toJSONString(payload *model.Payload) (string, error) {
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return string(jsonPayload), nil
}
