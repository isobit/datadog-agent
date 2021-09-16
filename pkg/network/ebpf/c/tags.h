#ifndef __TAGS_H
#define __TAGS_H

#include "tracer-stats.h"
#include "tags-types.h"
#include "tags-maps.h"

// Dynamic perf map tags
static __always_inline int write_map_tags(struct pt_regs* ctx, conn_tuple_t *t, __u8 *value, const size_t len) {
    tags_t tag;
    __builtin_memcpy(&tag.tup, t, sizeof(tag.tup));
    int i;
#pragma unroll
    for (i = 0; i < TAGS_MAX_LENGTH && i < len; i++) {
        tag->value[i] = value[i];
    }

    u32 cpu = bpf_get_smp_processor_id();
    bpf_perf_event_output(ctx, &conn_tags, cpu, &tag, sizeof(tag));
    return i;
}

// Static tags
static __always_inline void add_tags_stats(conn_stats_ts_t *stats, __u64 tags) {
    stats->tags |= tags;
}

static __always_inline void add_tags_tuple(conn_tuple_t *t, __u64 tags) {
    conn_stats_ts_t *stats = get_conn_stats(t);
    if (!stats) {
        return;
    }
    add_tags_stats(stats, tags);
}

#endif
