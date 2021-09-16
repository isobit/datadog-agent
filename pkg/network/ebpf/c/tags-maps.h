#ifndef __TAGS_MAPS_H
#define __TAGS_MAPS_H

#include "tags-types.h"

/* This is a key/value store with the keys being a conn_tuple_t, the values being tags_t.
 */
struct bpf_map_def SEC("maps/conn_tags") conn_tags = {
    .type = BPF_MAP_TYPE_PERF_EVENT_ARRAY,
    .key_size = sizeof(__u32),
    .value_size = sizeof(__u32),
    .max_entries = 0, // This will get overridden at runtime using max_tracked_connections
    .pinning = 0,
    .namespace = "",
};

#endif
