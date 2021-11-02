// +build linux,!linux_bpf

package constantfetch

import (
	"github.com/DataDog/datadog-agent/pkg/security/config"
	"github.com/DataDog/datadog-agent/pkg/security/ebpf/kernel"
)

func GetAvailableConstantFetchers(config *config.Config, kv *kernel.Version) []ConstantFetcher {
	fallbackConstantFetcher := NewFallbackConstantFetcher(kv)
	return []ConstantFetcher{
		fallbackConstantFetcher,
	}
}
