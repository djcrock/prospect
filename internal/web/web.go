package web

import (
	"fmt"
	"github.com/djcrock/prospect/internal/game"
	"github.com/djcrock/prospect/internal/util"
	"github.com/djcrock/prospect/internal/web/static"
	"github.com/djcrock/prospect/internal/web/templates"
	"log/slog"
	"net/http"
	"sync"
)

const randomIdRetries = 10
const gameIdLength = 12
const playerIdLength = 12

type server struct {
	mu    sync.RWMutex
	games map[string]*game.Game

	logger *slog.Logger
}

func NewApp(
	logger *slog.Logger,
) http.Handler {
	s := &server{
		games:  make(map[string]*game.Game),
		logger: logger,
	}
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", static.FileServer))
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("POST /game", s.handlePostGame)
	mux.HandleFunc("GET /game/{id}", s.handleGetGame)
	mux.HandleFunc("POST /game/{id}/players", s.handlePostGamePlayers)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.InfoContext(r.Context(), fmt.Sprintf("%s %s", r.Method, r.URL.Path))
		mux.ServeHTTP(w, r)
	})
}

type baseData struct {
	Title string
}

func (s *server) createGame(userName string) *game.Game {
	s.mu.Lock()
	defer s.mu.Unlock()
	for range randomIdRetries {
		gameId := util.RandomString(gameIdLength)
		_, ok := s.games[gameId]
		if !ok {
			g := &game.Game{Id: gameId, Players: []game.Player{{Id: util.RandomString(playerIdLength), Name: userName}}}
			s.games[gameId] = g
			return g
		}
	}

	return nil
}

func (s *server) retrieveGame(gameId string) *game.Game {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.games[gameId]
}

func setPlayerIdCookie(w http.ResponseWriter, gameId, playerId string) {
	gamePath := "/game/" + gameId
	http.SetCookie(w, &http.Cookie{
		Name:     "playerId",
		Value:    playerId,
		Path:     gamePath,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	err := templates.Index.ExecuteFull(w, nil)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "Failed to execute template: %v", err)
	}
}

func (s *server) handlePostGame(w http.ResponseWriter, r *http.Request) {
	g := s.createGame(r.FormValue("name"))
	if g == nil {
		http.Error(w, "Failed to create game. Please try again later.", http.StatusInternalServerError)
		return
	}

	gamePath := "/game/" + g.Id
	setPlayerIdCookie(w, g.Id, g.Players[0].Id)
	http.Redirect(w, r, gamePath, http.StatusSeeOther)

}

func (s *server) handlePostGamePlayers(w http.ResponseWriter, r *http.Request) {
	g := s.retrieveGame(r.PathValue("id"))
	if g == nil {
		w.WriteHeader(http.StatusNotFound)
		err := templates.NotFound.ExecuteFull(w, nil)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "Failed to execute template: %v", err)
		}
		return
	}

	gamePath := "/game/" + g.Id

	playerIdCookie, err := r.Cookie("playerId")
	if err == nil {
		p := g.GetPlayerById(playerIdCookie.Value)
		if p != nil {
			http.Redirect(w, r, gamePath, http.StatusSeeOther)
			return
		}
	}

	playerName := r.FormValue("name")

	for range randomIdRetries {
		playerId := util.RandomString(playerIdLength)
		existingPlayer := g.GetPlayerById(playerId)
		if existingPlayer == nil {
			err = g.AddPlayer(playerId, playerName)
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to add player: %v", err), http.StatusBadRequest)
				return
			}
			setPlayerIdCookie(w, g.Id, playerId)
			http.Redirect(w, r, gamePath, http.StatusSeeOther)
			return
		}
	}

	http.Error(w, "Failed to join game. Please try again later.", http.StatusInternalServerError)
}

type gameData struct {
	Base   baseData
	Game   *game.Game
	Player *game.Player
}

func (s *server) handleGetGame(w http.ResponseWriter, r *http.Request) {
	g := s.retrieveGame(r.PathValue("id"))
	if g == nil {
		w.WriteHeader(http.StatusNotFound)
		err := templates.NotFound.ExecuteFull(w, nil)
		if err != nil {
			s.logger.ErrorContext(r.Context(), "Failed to execute template: %v", err)
		}
		return
	}

	data := &gameData{Base: baseData{Title: "Game"}, Game: g}

	playerIdCookie, err := r.Cookie("playerId")
	if err == nil {
		data.Player = g.GetPlayerById(playerIdCookie.Value)
	}

	s.logger.InfoContext(r.Context(), fmt.Sprintf("%+v, %+v", data.Game, data.Player))
	err = templates.Game.ExecuteFull(w, data)
	if err != nil {
		s.logger.ErrorContext(r.Context(), "Failed to execute template: %v", err)
	}
}
