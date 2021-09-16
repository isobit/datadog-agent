#ifndef __TAGS_TYPES_H
#define __TAGS_TYPES_H

#include "tracer.h"

// dynamic tags
#define TAGS_MAX_LENGTH 16

typedef struct {
    conn_tuple_t tup;
    __u8 value[TAGS_MAX_LENGTH];
} tags_t;

// static tags limited to 64 tags per unique connection
enum static_tags {
    HTTP = (1<<0),
    LIBSSL = (1<<1),
};

#endif
