package service

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeHats/foundryvtt-rest-api-relay/go-relay/internal/model"
)

type fakeNotificationSettingsLookup struct {
	settings *model.NotificationSettings
}

func (f *fakeNotificationSettingsLookup) FindByUser(ctx context.Context, userID int64) (*model.NotificationSettings, error) {
	return f.settings, nil
}

type fakeApiKeyNotificationSettingsLookup struct {
	settings *model.ApiKeyNotificationSettings
}

func (f *fakeApiKeyNotificationSettingsLookup) FindByApiKey(ctx context.Context, apiKeyID int64) (*model.ApiKeyNotificationSettings, error) {
	return f.settings, nil
}

func newTestDispatcher(acct *model.NotificationSettings) *Dispatcher {
	return NewDispatcher(NotificationStores{
		NotificationSettings: &fakeNotificationSettingsLookup{settings: acct},
	}, nil, "https://example.test")
}

func TestDispatchNoUserIDIsNoOp(t *testing.T) {
	d := newTestDispatcher(&model.NotificationSettings{NotifyOnConnect: true})
	// Should not panic and should simply return without sending anything.
	d.Dispatch(NotificationContext{Event: EventConnect})
}

func TestDispatchSkipsWhenEventDisabled(t *testing.T) {
	acct := &model.NotificationSettings{NotifyOnConnect: false}
	d := newTestDispatcher(acct)
	// No webhook/email configured, so sendToDestination would be a no-op anyway,
	// but this exercises the accountEventEnabled gate directly.
	if d.accountEventEnabled(acct, EventConnect) {
		t.Error("expected EventConnect to be disabled")
	}
}

func TestDispatchDebounceSuppressesRepeatedEvents(t *testing.T) {
	acct := &model.NotificationSettings{
		NotifyOnConnect:                true,
		NotificationDebounceWindowSecs: 60,
	}
	d := newTestDispatcher(acct)

	nc := NotificationContext{Event: EventConnect, UserID: 1, ClientID: "client-1"}
	d.Dispatch(nc) // first call records the debounce timestamp
	d.Dispatch(nc) // second call within the window should be suppressed

	key := "1:connect:client-1"
	val, ok := d.debounce.Load(key)
	if !ok {
		t.Fatal("expected debounce entry to be recorded")
	}
	if time.Since(val.(time.Time)) > time.Minute {
		t.Error("debounce timestamp unexpectedly stale")
	}
}

func TestDuplicateConnectionRejectedAlwaysEnabled(t *testing.T) {
	acct := &model.NotificationSettings{} // all toggles false
	d := newTestDispatcher(acct)
	if !d.accountEventEnabled(acct, EventDuplicateConnectionRejected) {
		t.Error("expected EventDuplicateConnectionRejected to always be enabled")
	}
}

func TestKeyEventEnabledOnlyCoversKeyScopedEvents(t *testing.T) {
	d := &Dispatcher{}
	settings := &model.ApiKeyNotificationSettings{
		NotifyOnExecuteJs: true,
		NotifyOnRateLimit: true,
	}
	if !d.keyEventEnabled(settings, EventExecuteJs) {
		t.Error("expected EventExecuteJs to be enabled")
	}
	if !d.keyEventEnabled(settings, EventRateLimit) {
		t.Error("expected EventRateLimit to be enabled")
	}
	if d.keyEventEnabled(settings, EventConnect) {
		t.Error("expected EventConnect to be unsupported at key scope")
	}
}

func TestBuildDescriptionIncludesFields(t *testing.T) {
	nc := NotificationContext{
		WorldTitle: "My World",
		ClientID:   "abc123",
		Reason:     "test reason",
	}
	desc := buildDescription(nc)
	if desc == "(no additional details)" {
		t.Error("expected description to include provided fields")
	}
}

func TestBuildDescriptionEmptyFallback(t *testing.T) {
	if got := buildDescription(NotificationContext{}); got != "(no additional details)" {
		t.Errorf("expected fallback description, got %q", got)
	}
}
