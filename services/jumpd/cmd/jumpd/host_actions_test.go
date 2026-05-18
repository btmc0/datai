package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sting8k/jump/services/jumpd/internal/hostactions"
)

type fakeHostActionService struct {
	status hostactions.DisplaySleepCapability
	err    error
	calls  int
}

func (f *fakeHostActionService) DisplaySleepStatus(context.Context) hostactions.DisplaySleepCapability {
	return f.status
}

func (f *fakeHostActionService) SleepDisplay(context.Context) (hostactions.DisplaySleepCapability, error) {
	f.calls++
	return f.status, f.err
}

func TestHostActionRoutesExposeDisplaySleepStatus(t *testing.T) {
	mux := http.NewServeMux()
	registerHostActionRoutes(mux, &fakeHostActionService{status: hostactions.DisplaySleepCapability{
		Available: true,
		Status:    "available",
		Platform:  "darwin",
		State:     "awake",
	}})

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/v1/host-actions", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"\"display_sleep\"", "\"available\":true", "\"platform\":\"darwin\"", "\"state\":\"awake\""} {
		if !strings.Contains(body, want) {
			t.Fatalf("response %q missing %s", body, want)
		}
	}
}

func TestHostActionRoutesRejectUnavailableDisplaySleep(t *testing.T) {
	service := &fakeHostActionService{status: hostactions.DisplaySleepCapability{
		Available: false,
		Status:    "unsupported",
		Platform:  "linux",
		State:     "unknown",
		Reason:    "display sleep is only available on macOS",
	}, err: hostactions.ErrUnavailable}
	mux := http.NewServeMux()
	registerHostActionRoutes(mux, service)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/host-actions/display-sleep", nil))
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}
	if service.calls != 1 {
		t.Fatalf("SleepDisplay calls = %d, want 1", service.calls)
	}
	if !strings.Contains(rec.Body.String(), "display_sleep_unavailable") {
		t.Fatalf("response = %q, want unavailable code", rec.Body.String())
	}
}

func TestHostActionRoutesHideExecutionErrors(t *testing.T) {
	service := &fakeHostActionService{status: hostactions.DisplaySleepCapability{
		Available: true,
		Status:    "available",
		Platform:  "darwin",
		State:     "awake",
	}, err: errors.New("raw pmset failure")}
	mux := http.NewServeMux()
	registerHostActionRoutes(mux, service)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/v1/host-actions/display-sleep", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "display_sleep_failed") || strings.Contains(body, "raw pmset failure") {
		t.Fatalf("response = %q, want sanitized failure", body)
	}
}
