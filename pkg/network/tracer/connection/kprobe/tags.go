//+build linux_bpf

package kprobe

import (
	"bytes"
	"unsafe"

	"github.com/DataDog/datadog-agent/pkg/network"
	netebpf "github.com/DataDog/datadog-agent/pkg/network/ebpf"
	"github.com/DataDog/ebpf"
)

type tagsSet struct {
	m            *ebpf.Map
	set          map[string]uint32
	nextTagValue uint32
}

func newTagsSet(m *ebpf.Map) *tagsSet {
	return &tagsSet{
		m:            m,
		set:          make(map[string]uint32),
		nextTagValue: uint32(len(netebpf.StaticTagsStrings)),
	}
}

func (ts *tagsSet) getTagsStrings() (strs network.Tags) {
	max := uint32(len(netebpf.StaticTagsStrings) - 1)
	rset := make(map[uint32]string)
	for k, v := range ts.set {
		if v > max {
			max = v
		}
		rset[v] = k
	}
	for i, t := range netebpf.StaticTagsStrings {
		rset[uint32(i)] = t
	}

	for i := uint32(0); i <= max; i++ {
		v, found := rset[i]
		if !found {
			/* we should never hit this */
			continue
		}
		strs = append(strs, v)
	}
	return strs
}

// getIndexes lookup on ConnTuple, update the internal map and return the tags indexes
func (ts *tagsSet) getIndexes(tuple *netebpf.ConnTuple) (uintTags []uint32) {
	tags := new(netebpf.Tags)
	err := ts.m.Lookup(unsafe.Pointer(tuple), unsafe.Pointer(tags))
	if err == nil {
		return
	}

	tagsValues := []byte{}
	for _, v := range tags.Value {
		tagsValues = append(tagsValues, byte(v))
	}
	// split the multiple tags, thanks to the tailing \0
	for _, t := range bytes.Split(tagsValues, []byte{0}) {
		if len(t) == 0 {
			continue
		}
		tag := string(t)
		if v, found := ts.set[tag]; found {
			uintTags = append(uintTags, v)
			continue
		}
		ts.set[tag] = ts.nextTagValue
		uintTags = append(uintTags, ts.nextTagValue)
		ts.nextTagValue++
	}
	return uintTags
}

func getStaticTagsFromConnStats(s *netebpf.ConnStats) (r []uint32) {
	var v uint64 = s.Tags
	for i := 0; i < 8; /*sizeof(uint64)/sizeof(uint8)*/ i++ {
		o := i * 8
		tag := uint32((v >> o) & 0xff)
		if tag > uint32(0) {
			r = append(r, tag)
		}
	}
	return r
}
