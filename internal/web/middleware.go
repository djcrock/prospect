package web

import (
	"context"
	"github.com/djcrock/prospect/internal/web/room"
	"github.com/djcrock/prospect/internal/web/templates"
	"net/http"
)

type contextKey int

const (
	contextKeyPlayerId contextKey = iota
	contextKeyGameRoom
)

func withPlayerIdContext(r *http.Request, playerId string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKeyPlayerId, playerId))
}

func withGameRoomContext(r *http.Request, gameRoom *room.Room) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKeyGameRoom, gameRoom))
}

func getPlayerId(r *http.Request) string {
	val := r.Context().Value(contextKeyPlayerId)
	if val == nil {
		return ""
	}
	return val.(string)
}

func getGameRoom(r *http.Request) *room.Room {
	return r.Context().Value(contextKeyGameRoom).(*room.Room)
}

func (s *server) withGameRoom(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gr := s.retrieveGameRoom(r.PathValue("id"))
		if gr == nil {
			w.WriteHeader(http.StatusNotFound)
			err := templates.NotFound.ExecuteFull(w, nil)
			if err != nil {
				s.logger.Printf("Failed to execute template: %v", err)
			}
			return
		}
		r = withGameRoomContext(r, gr)

		playerId := gr.EnsurePlayer(getPlayerIdCookie(r))
		setPlayerIdCookie(w, r, gr.Game.Id, playerId)
		r = withPlayerIdContext(r, playerId)

		next.ServeHTTP(w, r)
	})
}
