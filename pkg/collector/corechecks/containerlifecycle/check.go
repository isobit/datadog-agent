// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-present Datadog, Inc.

package containerlifecycle

import (
	"time"

	yaml "gopkg.in/yaml.v2"

	"github.com/DataDog/datadog-agent/pkg/aggregator"
	"github.com/DataDog/datadog-agent/pkg/autodiscovery/integration"
	"github.com/DataDog/datadog-agent/pkg/collector/check"
	core "github.com/DataDog/datadog-agent/pkg/collector/corechecks"
	"github.com/DataDog/datadog-agent/pkg/util/log"
	"github.com/DataDog/datadog-agent/pkg/workloadmeta"
)

const checkName = "container_lifecycle"

// Config holds the container_lifecycle check configuration
type Config struct{}

// Parse parses the container_lifecycle check config and set default values
func (c *Config) Parse(data []byte) error {
	return yaml.Unmarshal(data, c)
}

// Check reports container lifecycle events
type Check struct {
	core.CheckBase
	workloadmetaStore workloadmeta.Store
	instance          *Config
	processor         *processor
	stopCh            chan struct{}
}

// Configure parses the check configuration and initializes the container_lifecycle check
func (c *Check) Configure(config, initConfig integration.Data, source string) error {
	var err error

	err = c.CommonConfigure(config, source)
	if err != nil {
		return err
	}

	err = c.instance.Parse(config)
	if err != nil {
		return err
	}

	sender, err := aggregator.GetSender(c.ID())
	if err != nil {
		return err
	}

	c.processor, err = newProcessor(sender)

	return err
}

// Run starts the container_lifecycle check
func (c *Check) Run() error {
	log.Infof("Starting long-running check %q", c.ID())
	defer log.Infof("Shutting down long-running check %q", c.ID())

	contEventsCh := c.workloadmetaStore.Subscribe(
		checkName+"-cont",
		workloadmeta.NewFilter(
			[]workloadmeta.Kind{workloadmeta.KindContainer},
			[]workloadmeta.Source{workloadmeta.SourceDocker, workloadmeta.SourceContainerd},
		),
	)

	podEventsCh := c.workloadmetaStore.Subscribe(
		checkName+"-pod",
		workloadmeta.NewFilter(
			[]workloadmeta.Kind{workloadmeta.KindKubernetesPod},
			[]workloadmeta.Source{workloadmeta.SourceKubelet},
		),
	)

	for {
		select {
		case eventBundle := <-contEventsCh:
			c.processor.processEvents(eventBundle)
		case eventBundle := <-podEventsCh:
			c.processor.processEvents(eventBundle)
		case <-c.stopCh:
			return nil
		}
	}
}

// Stop stops the container_lifecycle check
func (c *Check) Stop() { close(c.stopCh) }

// Interval returns 0, it makes container_lifecycle a long-running check
func (c *Check) Interval() time.Duration { return 0 }

// CheckFactory registers the container_lifecycle check
func CheckFactory() check.Check {
	return &Check{
		CheckBase:         core.NewCheckBase(checkName),
		workloadmetaStore: workloadmeta.GetGlobalStore(),
		instance:          &Config{},
		stopCh:            make(chan struct{}),
	}
}

func init() {
	core.RegisterCheck(checkName, CheckFactory)
}
