package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ThreeHats/foundryvtt-rest-api-relay/go-relay/internal/ws"
	"github.com/go-chi/chi/v5"
)

func TestAdminInteractiveSessionsList(t *testing.T) {
	manager := ws.NewInteractiveSessionManager(3)

	// Test with no sessions
	r := chi.NewRouter()
	r.Mount("/", AdminInteractiveSessionsRouter(nil, manager))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"] != float64(0) {
		t.Errorf("expected total=0, got %v", resp["total"])
	}

	if sessions, ok := resp["sessions"].([]interface{}); !ok || len(sessions) != 0 {
		t.Errorf("expected empty sessions array, got %v", resp["sessions"])
	}
}

func TestAdminInteractiveSessionsWithSessions(t *testing.T) {
	manager := ws.NewInteractiveSessionManager(3)

	// Create a session manually (simulating a connection)
	sessionID, _ := manager.CreateSession("client1", "key1", "consumer1", nil, ws.InteractiveSessionMetadata{
		Quality: 0.8,
		Scale:   2.0,
	})

	// Verify we can retrieve it
	if session := manager.GetSession(sessionID); session == nil {
		t.Fatalf("expected session to exist")
	}

	// Activate it
	manager.ActivateSession(sessionID)

	// Now test the endpoint
	r := chi.NewRouter()
	r.Mount("/", AdminInteractiveSessionsRouter(nil, manager))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"] != float64(1) {
		t.Errorf("expected total=1, got %v", resp["total"])
	}

	sessions, ok := resp["sessions"].([]interface{})
	if !ok || len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %v", resp["sessions"])
	}

	session := sessions[0].(map[string]interface{})
	if session["sessionId"] != sessionID {
		t.Errorf("expected sessionId=%s, got %s", sessionID, session["sessionId"])
	}

	if session["clientId"] != "client1" {
		t.Errorf("expected clientId=client1, got %s", session["clientId"])
	}

	if session["state"] != "active" {
		t.Errorf("expected state=active, got %s", session["state"])
	}

	if session["quality"] != 0.8 {
		t.Errorf("expected quality=0.8, got %v", session["quality"])
	}
}

func TestAdminInteractiveSessionsEndSession(t *testing.T) {
	manager := ws.NewInteractiveSessionManager(3)

	sessionID, _ := manager.CreateSession("client1", "key1", "consumer1", nil, ws.InteractiveSessionMetadata{})
	manager.ActivateSession(sessionID)

	r := chi.NewRouter()
	r.Mount("/", AdminInteractiveSessionsRouter(nil, manager))

	// Delete the session
	w := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/"+sessionID, nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify it's gone
	if manager.GetSession(sessionID) != nil {
		t.Errorf("expected session to be deleted")
	}
}

func TestAdminInteractiveSessionsNilManager(t *testing.T) {
	r := chi.NewRouter()
	r.Mount("/", AdminInteractiveSessionsRouter(nil, nil))

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("GET", "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)

	if resp["total"] != float64(0) {
		t.Errorf("expected total=0 when manager is nil, got %v", resp["total"])
	}
}
