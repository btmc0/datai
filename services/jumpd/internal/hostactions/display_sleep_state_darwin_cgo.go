//go:build darwin && cgo

package hostactions

/*
#cgo LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>

static int jump_display_sleep_state(void) {
	CGDirectDisplayID displays[16];
	uint32_t count = 0;
	CGError err = CGGetOnlineDisplayList(16, displays, &count);
	if (err != kCGErrorSuccess || count == 0) {
		return 0;
	}

	for (uint32_t i = 0; i < count; i++) {
		if (!CGDisplayIsAsleep(displays[i])) {
			return 1;
		}
	}
	return 2;
}
*/
import "C"

import "context"

func currentDisplaySleepState(context.Context) string {
	switch C.jump_display_sleep_state() {
	case 1:
		return "awake"
	case 2:
		return "asleep"
	default:
		return "unknown"
	}
}
