package web

import (
	"net/http"
)

const cookieNamePlayerId = "playerId"

func setPlayerIdCookie(w http.ResponseWriter, _ *http.Request, gameId, playerId string) {
	gamePath := "/game/" + gameId
	cookie := &http.Cookie{
		Name:     cookieNamePlayerId,
		Value:    playerId,
		Path:     gamePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

func getPlayerIdCookie(r *http.Request) string {
	playerIdCookie, err := r.Cookie(cookieNamePlayerId)
	if err == nil {
		return playerIdCookie.Value
	}
	return ""
}
