package web

import (
	"fmt"
	"github.com/djcrock/prospect/internal/game"
	"github.com/djcrock/prospect/internal/util"
	"github.com/djcrock/prospect/internal/web/static"
	"github.com/djcrock/prospect/internal/web/templates"
	"log"
	"math/rand/v2"
	"net/http"
	"sync"
)

const randomIdRetries = 10
const gameIdLength = 12
const playerIdLength = 12

type server struct {
	mu    sync.RWMutex
	games map[string]*game.Game

	logger *log.Logger
}

func NewApp(
	logger *log.Logger,
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
	mux.HandleFunc("POST /game/{id}/leave", s.handlePostGameLeave)
	mux.HandleFunc("POST /game/{id}/start", s.handlePostGameStart)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("%s %s", r.Method, r.URL.Path)
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
			g := &game.Game{
				Id:      gameId,
				Players: []game.Player{{Id: util.RandomString(playerIdLength), Name: userName}},
				Rand:    rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
			}
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

func (s *server) removeGame(gameId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.games, gameId)
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

func redirectToGame(w http.ResponseWriter, r *http.Request, gameId string) {
	http.Redirect(w, r, "/game/"+gameId, http.StatusSeeOther)
}

func (s *server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	err := templates.Index.ExecuteFull(w, nil)
	if err != nil {
		s.logger.Printf("Failed to execute template: %v", err)
	}
}

func (s *server) handlePostGame(w http.ResponseWriter, r *http.Request) {
	g := s.createGame(r.FormValue("name"))
	if g == nil {
		http.Error(w, "Failed to create game. Please try again later.", http.StatusInternalServerError)
		return
	}

	setPlayerIdCookie(w, g.Id, g.Players[0].Id)
	redirectToGame(w, r, g.Id)
}

func (s *server) handlePostGamePlayers(w http.ResponseWriter, r *http.Request) {
	g := s.retrieveGame(r.PathValue("id"))
	if g == nil {
		w.WriteHeader(http.StatusNotFound)
		err := templates.NotFound.ExecuteFull(w, nil)
		if err != nil {
			s.logger.Printf("Failed to execute template: %v", err)
		}
		return
	}

	g.Mu.Lock()
	defer g.Mu.Unlock()
	playerIdCookie, err := r.Cookie("playerId")
	if err == nil {
		p := g.GetPlayerById(playerIdCookie.Value)
		if p != nil {
			redirectToGame(w, r, g.Id)
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
			redirectToGame(w, r, g.Id)
			return
		}
	}

	http.Error(w, "Failed to join game. Please try again later.", http.StatusInternalServerError)
}

func (s *server) handlePostGameLeave(w http.ResponseWriter, r *http.Request) {
	g := s.retrieveGame(r.PathValue("id"))
	if g == nil {
		w.WriteHeader(http.StatusNotFound)
		err := templates.NotFound.ExecuteFull(w, nil)
		if err != nil {
			s.logger.Printf("Failed to execute template: %v", err)
		}
		return
	}

	g.Mu.Lock()
	defer g.Mu.Unlock()
	playerIdCookie, err := r.Cookie("playerId")
	if err == nil {
		g.RemovePlayer(playerIdCookie.Value)
		if g.IsEmpty() {
			s.removeGame(g.Id)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
	}

	redirectToGame(w, r, g.Id)
}

func (s *server) handlePostGameStart(w http.ResponseWriter, r *http.Request) {
	g := s.retrieveGame(r.PathValue("id"))
	if g == nil {
		w.WriteHeader(http.StatusNotFound)
		err := templates.NotFound.ExecuteFull(w, nil)
		if err != nil {
			s.logger.Printf("Failed to execute template: %v", err)
		}
		return
	}

	g.Mu.Lock()
	defer g.Mu.Unlock()
	playerIdCookie, err := r.Cookie("playerId")
	if err != nil {
		redirectToGame(w, r, g.Id)
		return
	}
	p := g.GetPlayerById(playerIdCookie.Value)
	if p == nil {
		redirectToGame(w, r, g.Id)
		return
	}
	err = g.Start()
	if err != nil {
		s.logger.Printf("failed to start game: %v", err)
	}

	redirectToGame(w, r, g.Id)
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
			s.logger.Printf("Failed to execute template: %v", err)
		}
		return
	}

	g.Mu.RLock()
	defer g.Mu.RUnlock()

	data := &gameData{Base: baseData{Title: "Game"}, Game: g}

	playerIdCookie, err := r.Cookie("playerId")
	if err == nil {
		data.Player = g.GetPlayerById(playerIdCookie.Value)
	}

	err = templates.Game.ExecuteFull(w, data)
	if err != nil {
		s.logger.Printf("Failed to execute template: %v", err)
	}
}
