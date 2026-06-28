package handler

import (
	"net/http"

	"github.com/ThreeHats/foundryvtt-rest-api-relay/go-relay/internal/database"
	"github.com/ThreeHats/foundryvtt-rest-api-relay/go-relay/internal/handler/helpers"
	"github.com/ThreeHats/foundryvtt-rest-api-relay/go-relay/internal/ws"
	"github.com/go-chi/chi/v5"
)

// AdminInteractiveSessionsRouter exposes admin endpoints for interactive (screenshot/view) sessions.
func AdminInteractiveSessionsRouter(db *database.DB, manager *ws.InteractiveSessionManager) chi.Router {
	r := chi.NewRouter()

	// GET /admin/api/interactive-sessions
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		if manager == nil {
			helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
				"total":    0,
				"sessions": []interface{}{},
			})
			return
		}

		// Return all interactive sessions in a safe format (no credentials)
		sessions := manager.ListSessions()
		safeSessions := make([]map[string]interface{}, 0, len(sessions))
		for _, s := range sessions {
			safeSessions = append(safeSessions, map[string]interface{}{
				"sessionId":    s.SessionID,
				"clientId":     s.ClientID,
				"consumerId":   s.ConsumerID,
				"state":        s.State,
				"createdAt":    s.CreatedAt,
				"lastActivity": s.LastActivity,
				"quality":      s.Metadata.Quality,
				"scale":        s.Metadata.Scale,
			})
		}

		helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"total":    len(safeSessions),
			"sessions": safeSessions,
		})
	})

	// DELETE /admin/api/interactive-sessions/{id}
	r.Delete("/{id}", func(w http.ResponseWriter, req *http.Request) {
		if manager == nil {
			helpers.WriteError(w, http.StatusServiceUnavailable, "Interactive sessions disabled")
			return
		}
		id := chi.URLParam(req, "id")
		if !manager.EndSession(id) {
			helpers.WriteError(w, http.StatusNotFound, "Session not found")
			return
		}
		auditAdmin(req, db, "session.kill", "interactive_session", id, "")
		helpers.WriteJSON(w, http.StatusOK, map[string]string{"message": "Session ended"})
	})

	return r
}
