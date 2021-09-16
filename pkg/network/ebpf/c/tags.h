#ifndef __TAGS_H
#define __TAGS_H

#include "tracer-stats.h"
#include "tags-types.h"
#include "tags-maps.h"

// map tags
static __always_inline tags_t * conn_map_tags(conn_tuple_t *t) {
    // initialize-if-no-exist the connection tags, and load it
    tags_t empty = { 0 };
    if (bpf_map_update_elem(&conn_tags, t, &empty, BPF_NOEXIST) == -E2BIG) {
        increment_telemetry_count(conn_tags_max_entries_hit);
    }
    return bpf_map_lookup_elem(&conn_tags, t);
}

static __always_inline int write_map_tags(conn_tuple_t *t, const int once, const size_t offset, __u8 *value, const size_t len) {
    tags_t *tags = conn_map_tags(t);
    if ((tags == NULL) || (once && tags->value[offset] != 0)) {
        return offset;
    }
    if (offset >= TAGS_MAX_LENGTH) {
        return 0;
    }
    int i;
#pragma unroll
    for (i = offset; i < TAGS_MAX_LENGTH && (i-offset) < len; i++) {
        tags->value[i] = value[(i-offset)];
    }
    return i;
}

// Static tags
static __always_inline void add_tags(conn_stats_ts_t *stats, const int once, enum static_tags tags) {
    if (once && stats->tags == NOTAGS) {
        stats->tags = tags;
    } else {
        stats->tags = (stats->tags << 8) | tags;
    }
}

static __always_inline void add_tags_tuple(conn_tuple_t *t, const int once, enum static_tags tags) {
    conn_stats_ts_t *stats = get_conn_stats(t);
    if (!stats) {
        return;
    }
    add_tags(stats, once, tags);
}

#endif
