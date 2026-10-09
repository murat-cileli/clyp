#include "native.h"
#include "_cgo_export.h"

#include <X11/Xatom.h>
#include <X11/Xlib.h>
#include <X11/extensions/Xfixes.h>
#include <errno.h>
#include <poll.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>

#define CLYP_MAX_CLIPBOARD_BYTES (256U * 1024U * 1024U)

struct x11_context {
  Display *display;
  Window window;
  Atom clipboard;
  Atom property;
  Atom targets;
  Atom incr;
  int xfixes_event_base;
  int pending_change;
  Time last_selection_timestamp;
};

static void set_error(char *error, size_t error_size, const char *message) {
  if (error != NULL && error_size > 0) {
    snprintf(error, error_size, "%s", message);
  }
}
static int64_t monotonic_milliseconds(void) {
  struct timespec now;
  clock_gettime(CLOCK_MONOTONIC, &now);
  return (int64_t)now.tv_sec * 1000 + now.tv_nsec / 1000000;
}

static int next_event_until(struct x11_context *ctx, XEvent *event,
                            int64_t deadline) {
  while (1) {
    if (XPending(ctx->display) > 0) {
      XNextEvent(ctx->display, event);
      if (event->type == ctx->xfixes_event_base + XFixesSelectionNotify) {
        XFixesSelectionNotifyEvent *selection =
            (XFixesSelectionNotifyEvent *)event;
        if (selection->selection == ctx->clipboard &&
            selection->subtype == XFixesSetSelectionOwnerNotify &&
            selection->owner != None &&
            selection->selection_timestamp != ctx->last_selection_timestamp) {
          ctx->last_selection_timestamp = selection->selection_timestamp;
          ctx->pending_change = 1;
        }
      }
      return 1;
    }

    int64_t remaining = deadline - monotonic_milliseconds();
    if (remaining <= 0) {
      return 0;
    }
    struct pollfd descriptor = {
        .fd = ConnectionNumber(ctx->display), .events = POLLIN, .revents = 0};
    int result = poll(&descriptor, 1, (int)remaining);
    if (result < 0 && errno == EINTR) {
      continue;
    }
    if (result <= 0) {
      return 0;
    }
  }
}

static int append_bytes(unsigned char **buffer, size_t *length,
                        const unsigned char *chunk, size_t chunk_length) {
  if (chunk_length == 0) {
    return 1;
  }
  if (*length > CLYP_MAX_CLIPBOARD_BYTES - chunk_length) {
    return 0;
  }
  unsigned char *resized = realloc(*buffer, *length + chunk_length);
  if (resized == NULL) {
    return 0;
  }
  memcpy(resized + *length, chunk, chunk_length);
  *buffer = resized;
  *length += chunk_length;
  return 1;
}

static int read_property(struct x11_context *ctx, Atom *actual_type,
                         int *actual_format, unsigned char **output,
                         size_t *output_length) {
  unsigned long item_count = 0;
  unsigned long bytes_after = 0;
  unsigned char *property_data = NULL;
  int result = XGetWindowProperty(
      ctx->display, ctx->window, ctx->property, 0, 0x1fffffff, False,
      AnyPropertyType, actual_type, actual_format, &item_count, &bytes_after,
      &property_data);
  if (result != Success) {
    fprintf(stderr, "clyp-watcher: XGetWindowProperty failed (%d)\n", result);
    return 0;
  }

  if (*actual_type != ctx->incr) {
    size_t item_size = *actual_format == 32 ? sizeof(long)
                       : *actual_format == 16 ? 2
                                              : 1;
    size_t byte_count = item_count * item_size;
    int ok = append_bytes(output, output_length, property_data, byte_count);
    if (property_data != NULL) {
      XFree(property_data);
    }
    XDeleteProperty(ctx->display, ctx->window, ctx->property);
    return ok;
  }

  if (property_data != NULL) {
    XFree(property_data);
  }
  XDeleteProperty(ctx->display, ctx->window, ctx->property);
  XFlush(ctx->display);

  int64_t deadline = monotonic_milliseconds() + 30000;
  while (1) {
    XEvent event;
    if (!next_event_until(ctx, &event, deadline)) {
      fprintf(stderr,
              "clyp-watcher: incremental clipboard transfer timed out\n");
      return 0;
    }
    if (event.type != PropertyNotify || event.xproperty.window != ctx->window ||
        event.xproperty.atom != ctx->property ||
        event.xproperty.state != PropertyNewValue) {
      continue;
    }

    Atom chunk_type = None;
    int chunk_format = 0;
    item_count = 0;
    bytes_after = 0;
    property_data = NULL;
    result = XGetWindowProperty(
        ctx->display, ctx->window, ctx->property, 0, 0x1fffffff, True,
        AnyPropertyType, &chunk_type, &chunk_format, &item_count, &bytes_after,
        &property_data);
    if (result != Success) {
      fprintf(stderr,
              "clyp-watcher: incremental XGetWindowProperty failed (%d)\n",
              result);
      return 0;
    }
    size_t item_size = chunk_format == 32 ? sizeof(long)
                       : chunk_format == 16 ? 2
                                            : 1;
    size_t byte_count = item_count * item_size;
    if (byte_count == 0) {
      if (property_data != NULL) {
        XFree(property_data);
      }
      return 1;
    }
    *actual_type = chunk_type;
    *actual_format = chunk_format;
    int ok = append_bytes(output, output_length, property_data, byte_count);
    XFree(property_data);
    if (!ok) {
      return 0;
    }
    deadline = monotonic_milliseconds() + 5000;
  }
}

static int request_selection(struct x11_context *ctx, Atom target,
                             Atom *actual_type, int *actual_format,
                             unsigned char **output, size_t *output_length) {
  *output = NULL;
  *output_length = 0;
  XDeleteProperty(ctx->display, ctx->window, ctx->property);
  XConvertSelection(ctx->display, ctx->clipboard, target, ctx->property,
                    ctx->window, CurrentTime);
  XFlush(ctx->display);

  int64_t deadline = monotonic_milliseconds() + 3000;
  while (1) {
    XEvent event;
    if (!next_event_until(ctx, &event, deadline)) {
      return 0;
    }
    if (event.type != SelectionNotify ||
        event.xselection.requestor != ctx->window ||
        event.xselection.selection != ctx->clipboard ||
        event.xselection.target != target) {
      continue;
    }
    if (event.xselection.property == None) {
      return 0;
    }
    return read_property(ctx, actual_type, actual_format, output, output_length);
  }
}

static int atom_is_offered(const Atom *atoms, size_t count, Atom candidate) {
  for (size_t index = 0; index < count; index++) {
    if (atoms[index] == candidate) {
      return 1;
    }
  }
  return 0;
}

static void capture_selection(struct x11_context *ctx) {
  Window selection_owner = XGetSelectionOwner(ctx->display, ctx->clipboard);
  if (selection_owner == None) {
    return;
  }

  Atom actual_type = None;
  int actual_format = 0;
  unsigned char *target_data = NULL;
  size_t target_length = 0;
  Atom selected_target = None;
  const char *selected_mime = NULL;

  Atom utf8 = XInternAtom(ctx->display, "UTF8_STRING", False);
  Atom text_utf8 =
      XInternAtom(ctx->display, "text/plain;charset=utf-8", False);
  Atom text_plain = XInternAtom(ctx->display, "text/plain", False);
  Atom image_png = XInternAtom(ctx->display, "image/png", False);
  Atom image_jpeg = XInternAtom(ctx->display, "image/jpeg", False);
  Atom image_webp = XInternAtom(ctx->display, "image/webp", False);
  Atom image_bmp = XInternAtom(ctx->display, "image/bmp", False);
  Atom image_tiff = XInternAtom(ctx->display, "image/tiff", False);

  if (request_selection(ctx, ctx->targets, &actual_type, &actual_format,
                        &target_data, &target_length) &&
      actual_format == 32) {
    Atom *offered = (Atom *)target_data;
    size_t offered_count = target_length / sizeof(Atom);
    if (atom_is_offered(offered, offered_count, text_utf8)) {
      selected_target = text_utf8;
      selected_mime = "text/plain;charset=utf-8";
    } else if (atom_is_offered(offered, offered_count, utf8)) {
      selected_target = utf8;
      selected_mime = "text/plain;charset=utf-8";
    } else if (atom_is_offered(offered, offered_count, text_plain)) {
      selected_target = text_plain;
      selected_mime = "text/plain";
    } else if (atom_is_offered(offered, offered_count, XA_STRING)) {
      selected_target = XA_STRING;
      selected_mime = "text/plain";
    } else if (atom_is_offered(offered, offered_count, image_png)) {
      selected_target = image_png;
      selected_mime = "image/png";
    } else if (atom_is_offered(offered, offered_count, image_jpeg)) {
      selected_target = image_jpeg;
      selected_mime = "image/jpeg";
    } else if (atom_is_offered(offered, offered_count, image_webp)) {
      selected_target = image_webp;
      selected_mime = "image/webp";
    } else if (atom_is_offered(offered, offered_count, image_bmp)) {
      selected_target = image_bmp;
      selected_mime = "image/bmp";
    } else if (atom_is_offered(offered, offered_count, image_tiff)) {
      selected_target = image_tiff;
      selected_mime = "image/tiff";
    }
  }
  free(target_data);

  if (selected_target == None) {
    selected_target = utf8;
    selected_mime = "text/plain;charset=utf-8";
  }

  unsigned char *content = NULL;
  size_t content_length = 0;
  if (!request_selection(ctx, selected_target, &actual_type, &actual_format,
                         &content, &content_length) ||
      actual_format != 8 || content_length == 0) {
    if (XGetSelectionOwner(ctx->display, ctx->clipboard) != selection_owner) {
      free(content);
      return;
    }
    char *target_name = XGetAtomName(ctx->display, selected_target);
    fprintf(stderr,
            "clyp-watcher: failed to read X11 clipboard target %s "
            "(format=%d, bytes=%zu)\n",
            target_name == NULL ? "unknown" : target_name, actual_format,
            content_length);
    if (target_name != NULL) {
      XFree(target_name);
    }
    free(content);
    return;
  }

  goX11ClipboardData((char *)selected_mime, content, content_length);
  free(content);
}

int clyp_x11_run(char *error, size_t error_size) {
  XInitThreads();
  struct x11_context ctx = {0};
  ctx.display = XOpenDisplay(NULL);
  if (ctx.display == NULL) {
    set_error(error, error_size, "failed to open X11 display");
    return 1;
  }

  int xfixes_error_base = 0;
  if (!XFixesQueryExtension(ctx.display, &ctx.xfixes_event_base,
                            &xfixes_error_base)) {
    set_error(error, error_size, "XFixes extension is unavailable");
    XCloseDisplay(ctx.display);
    return 1;
  }

  ctx.window = XCreateSimpleWindow(ctx.display, DefaultRootWindow(ctx.display),
                                   0, 0, 1, 1, 0, 0, 0);
  XSelectInput(ctx.display, ctx.window, PropertyChangeMask);
  ctx.clipboard = XInternAtom(ctx.display, "CLIPBOARD", False);
  ctx.property = XInternAtom(ctx.display, "CLYP_SELECTION", False);
  ctx.targets = XInternAtom(ctx.display, "TARGETS", False);
  ctx.incr = XInternAtom(ctx.display, "INCR", False);

  XFixesSelectSelectionInput(ctx.display, ctx.window, ctx.clipboard,
                             XFixesSetSelectionOwnerNotifyMask);
  XFlush(ctx.display);

  if (XGetSelectionOwner(ctx.display, ctx.clipboard) != None) {
    capture_selection(&ctx);
  }

  while (1) {
    XEvent event;
    XNextEvent(ctx.display, &event);
    if (event.type != ctx.xfixes_event_base + XFixesSelectionNotify) {
      continue;
    }
    XFixesSelectionNotifyEvent *selection =
        (XFixesSelectionNotifyEvent *)&event;
    if (selection->selection != ctx.clipboard ||
        selection->subtype != XFixesSetSelectionOwnerNotify ||
        selection->owner == None ||
        selection->selection_timestamp == ctx.last_selection_timestamp) {
      continue;
    }
    ctx.last_selection_timestamp = selection->selection_timestamp;
    do {
      ctx.pending_change = 0;
      capture_selection(&ctx);
    } while (ctx.pending_change);
  }

  return 0;
}
