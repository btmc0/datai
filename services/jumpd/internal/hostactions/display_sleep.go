package hostactions

import (
	"context"
	"errors"
)

var ErrUnavailable = errors.New("display sleep unavailable")

type DisplaySleepCapability struct {
	Available bool   `json:"available"`
	Status    string `json:"status"`
	Platform  string `json:"platform"`
	State     string `json:"state"`
	Reason    string `json:"reason,omitempty"`
}

func GetDisplaySleepStatus(ctx context.Context) DisplaySleepCapability {
	return getDisplaySleepStatus(ctx)
}

func SleepDisplay(ctx context.Context) (DisplaySleepCapability, error) {
	return sleepDisplay(ctx)
}
