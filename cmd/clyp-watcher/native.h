#ifndef CLYP_WATCHER_NATIVE_H
#define CLYP_WATCHER_NATIVE_H

#include <stddef.h>

int clyp_x11_run(char *error, size_t error_size);
int clyp_wayland_run(char *error, size_t error_size);

#endif
