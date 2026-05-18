package main

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/sting8k/jump/services/jumpd/internal/hostactions"
)

type hostActionService interface {
	DisplaySleepStatus(context.Context) hostactions.DisplaySleepCapability
	SleepDisplay(context.Context) (hostactions.DisplaySleepCapability, error)
}

type defaultHostActionService struct{}

func (defaultHostActionService) DisplaySleepStatus(ctx context.Context) hostactions.DisplaySleepCapability {
	return hostactions.GetDisplaySleepStatus(ctx)
}

func (defaultHostActionService) SleepDisplay(ctx context.Context) (hostactions.DisplaySleepCapability, error) {
	return hostactions.SleepDisplay(ctx)
}

func registerHostActionRoutes(mux *http.ServeMux, service hostActionService) {
	mux.HandleFunc("GET /v1/host-actions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"ok": true,
			"data": map[string]any{
				"display_sleep": service.DisplaySleepStatus(r.Context()),
			},
		})
	})

	mux.HandleFunc("POST /v1/host-actions/display-sleep", func(w http.ResponseWriter, r *http.Request) {
		status, err := service.SleepDisplay(r.Context())
		if err != nil {
			if errors.Is(err, hostactions.ErrUnavailable) {
				writeError(w, http.StatusNotImplemented, "display_sleep_unavailable", status.Reason)
				return
			}
			log.Printf("hostactions: display sleep failed: %v", err)
			writeError(w, http.StatusInternalServerError, "display_sleep_failed", "failed to sleep display")
			return
		}
		writeJSON(w, map[string]any{"ok": true, "data": map[string]any{"display_sleep": status}})
	})
}
