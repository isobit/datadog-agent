#ifndef __TAGS_TYPES_H
#define __TAGS_TYPES_H

#include "tracer.h"

// dynamic tags
#define TAGS_MAX_LENGTH 16

// value of the tags map
typedef struct {
    __u8 value[TAGS_MAX_LENGTH];
} tags_t;

// static tags are limited to 255
enum static_tags {
    NOTAGS = 0,
    HTTP = 1,
    LIBSSL = 2,
};

#endif
