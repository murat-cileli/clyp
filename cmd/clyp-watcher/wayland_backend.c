#define _GNU_SOURCE

#include "native.h"
#include "_cgo_export.h"
#include "protocol/ext-data-control-v1-client-protocol.h"

#include <errno.h>
#include <fcntl.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <wayland-client.h>

#include "protocol/ext-data-control-v1-protocol.c"

#define CLYP_MAX_CLIPBOARD_BYTES (256U * 1024U * 1024U)

struct wayland_offer {
  struct ext_data_control_offer_v1 *proxy;
  char *mime_type;
  int mime_rank;
};

struct wayland_context {
  struct wl_display *display;
  struct wl_registry *registry;
  struct wl_seat *seat;
  struct ext_data_control_manager_v1 *manager;
  struct ext_data_control_device_v1 *device;
  struct wayland_offer *selection;
  int running;
};

static void set_error(char *error, size_t error_size, const char *message) {
  if (error != NULL && error_size > 0) {
    snprintf(error, error_size, "%s", message);
  }
}
static int mime_rank(const char *mime_type) {
  if (strcmp(mime_type, "text/plain;charset=utf-8") == 0) {
    return 100;
  }
  if (strcmp(mime_type, "UTF8_STRING") == 0) {
    return 95;
  }
  if (strcmp(mime_type, "text/plain") == 0 ||
      strcmp(mime_type, "STRING") == 0) {
    return 90;
  }
  if (strcmp(mime_type, "image/png") == 0) {
    return 80;
  }
  if (strcmp(mime_type, "image/jpeg") == 0) {
    return 75;
  }
  if (strcmp(mime_type, "image/webp") == 0 ||
      strcmp(mime_type, "image/bmp") == 0 ||
      strcmp(mime_type, "image/tiff") == 0) {
    return 70;
  }
  return 0;
}

static void destroy_offer(struct wayland_offer *offer) {
  if (offer == NULL) {
    return;
  }
  if (offer->proxy != NULL) {
    ext_data_control_offer_v1_destroy(offer->proxy);
  }
  free(offer->mime_type);
  free(offer);
}

static int read_offer(struct wayland_context *context,
                      struct wayland_offer *offer) {
  if (offer == NULL || offer->mime_type == NULL || offer->mime_rank == 0) {
    return 1;
  }

  int descriptors[2];
  if (pipe2(descriptors, O_CLOEXEC) != 0) {
    return 0;
  }
  ext_data_control_offer_v1_receive(offer->proxy, offer->mime_type,
                                    descriptors[1]);
  close(descriptors[1]);
  if (wl_display_flush(context->display) < 0 && errno != EAGAIN) {
    close(descriptors[0]);
    return 0;
  }

  unsigned char *content = NULL;
  size_t content_length = 0;
  unsigned char chunk[65536];
  while (1) {
    ssize_t count = read(descriptors[0], chunk, sizeof(chunk));
    if (count < 0 && errno == EINTR) {
      continue;
    }
    if (count <= 0) {
      break;
    }
    if (content_length > CLYP_MAX_CLIPBOARD_BYTES - (size_t)count) {
      free(content);
      close(descriptors[0]);
      return 0;
    }
    unsigned char *resized = realloc(content, content_length + (size_t)count);
    if (resized == NULL) {
      free(content);
      close(descriptors[0]);
      return 0;
    }
    content = resized;
    memcpy(content + content_length, chunk, (size_t)count);
    content_length += (size_t)count;
  }
  close(descriptors[0]);

  if (content_length > 0) {
    const char *reported_mime = offer->mime_type;
    if (strcmp(reported_mime, "UTF8_STRING") == 0 ||
        strcmp(reported_mime, "STRING") == 0) {
      reported_mime = "text/plain;charset=utf-8";
    }
    goWaylandClipboardData((char *)reported_mime, content, content_length);
  }
  free(content);
  return 1;
}

static void offer_mime(void *data, struct ext_data_control_offer_v1 *proxy,
                       const char *mime_type) {
  (void)proxy;
  struct wayland_offer *offer = data;
  int rank = mime_rank(mime_type);
  if (rank <= offer->mime_rank) {
    return;
  }
  char *copy = strdup(mime_type);
  if (copy == NULL) {
    return;
  }
  free(offer->mime_type);
  offer->mime_type = copy;
  offer->mime_rank = rank;
}

static const struct ext_data_control_offer_v1_listener offer_listener = {
    .offer = offer_mime,
};

static void device_data_offer(void *data,
                              struct ext_data_control_device_v1 *device,
                              struct ext_data_control_offer_v1 *proxy) {
  (void)data;
  (void)device;
  struct wayland_offer *offer = calloc(1, sizeof(*offer));
  if (offer == NULL) {
    ext_data_control_offer_v1_destroy(proxy);
    return;
  }
  offer->proxy = proxy;
  ext_data_control_offer_v1_add_listener(proxy, &offer_listener, offer);
}

static void device_selection(void *data,
                             struct ext_data_control_device_v1 *device,
                             struct ext_data_control_offer_v1 *proxy) {
  (void)device;
  struct wayland_context *context = data;
  struct wayland_offer *next =
      proxy == NULL ? NULL : ext_data_control_offer_v1_get_user_data(proxy);
  if (context->selection != NULL && context->selection != next) {
    destroy_offer(context->selection);
  }
  context->selection = next;
  if (next != NULL) {
    read_offer(context, next);
  }
}

static void device_finished(void *data,
                            struct ext_data_control_device_v1 *device) {
  (void)device;
  struct wayland_context *context = data;
  context->running = 0;
}

static void device_primary_selection(
    void *data, struct ext_data_control_device_v1 *device,
    struct ext_data_control_offer_v1 *proxy) {
  (void)data;
  (void)device;
  if (proxy != NULL) {
    struct wayland_offer *offer = ext_data_control_offer_v1_get_user_data(proxy);
    destroy_offer(offer);
  }
}

static const struct ext_data_control_device_v1_listener device_listener = {
    .data_offer = device_data_offer,
    .selection = device_selection,
    .finished = device_finished,
    .primary_selection = device_primary_selection,
};

static void registry_global(void *data, struct wl_registry *registry,
                            uint32_t name, const char *interface,
                            uint32_t version) {
  struct wayland_context *context = data;
  if (strcmp(interface, ext_data_control_manager_v1_interface.name) == 0) {
    context->manager = wl_registry_bind(
        registry, name, &ext_data_control_manager_v1_interface,
        version < 1 ? version : 1);
  } else if (strcmp(interface, wl_seat_interface.name) == 0 &&
             context->seat == NULL) {
    context->seat = wl_registry_bind(registry, name, &wl_seat_interface,
                                     version < 1 ? version : 1);
  }
}

static void registry_global_remove(void *data, struct wl_registry *registry,
                                   uint32_t name) {
  (void)data;
  (void)registry;
  (void)name;
}

static const struct wl_registry_listener registry_listener = {
    .global = registry_global,
    .global_remove = registry_global_remove,
};

int clyp_wayland_run(char *error, size_t error_size) {
  struct wayland_context context = {.running = 1};
  context.display = wl_display_connect(NULL);
  if (context.display == NULL) {
    set_error(error, error_size, "failed to connect to Wayland display");
    return 1;
  }

  context.registry = wl_display_get_registry(context.display);
  wl_registry_add_listener(context.registry, &registry_listener, &context);
  if (wl_display_roundtrip(context.display) < 0) {
    set_error(error, error_size, "failed to read Wayland registry");
    wl_registry_destroy(context.registry);
    wl_display_disconnect(context.display);
    return 1;
  }

  if (context.manager == NULL || context.seat == NULL) {
    if (context.manager != NULL) {
      ext_data_control_manager_v1_destroy(context.manager);
    }
    if (context.seat != NULL) {
      wl_seat_destroy(context.seat);
    }
    wl_registry_destroy(context.registry);
    wl_display_disconnect(context.display);
    return 2;
  }

  context.device =
      ext_data_control_manager_v1_get_data_device(context.manager, context.seat);
  ext_data_control_device_v1_add_listener(context.device, &device_listener,
                                          &context);

  while (context.running && wl_display_dispatch(context.display) >= 0) {
  }

  destroy_offer(context.selection);
  ext_data_control_device_v1_destroy(context.device);
  ext_data_control_manager_v1_destroy(context.manager);
  wl_seat_destroy(context.seat);
  wl_registry_destroy(context.registry);
  int protocol_error = wl_display_get_error(context.display);
  wl_display_disconnect(context.display);
  if (protocol_error != 0) {
    set_error(error, error_size, "Wayland clipboard connection failed");
    return 1;
  }
  return 0;
}
