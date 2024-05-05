package web

import (
	"bytes"
	"fmt"
	"github.com/djcrock/prospect/internal/game"
	"github.com/djcrock/prospect/internal/web/room"
	"github.com/djcrock/prospect/internal/web/static"
	"github.com/djcrock/prospect/internal/web/templates"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type server struct {
	mu    sync.RWMutex
	rooms *room.Collection

	logger *log.Logger
}

func NewApp(
	logger *log.Logger,
) http.Handler {
	s := &server{
		rooms:  room.NewCollection(),
		logger: logger,
	}
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.StripPrefix("/static/", static.FileServer))
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("POST /game", s.handlePostGame)
	mux.Handle("GET /game/{id}", s.withGameRoom(http.HandlerFunc(s.handleGetGame)))
	mux.Handle("GET /game/{id}/sse", s.withGameRoom(http.HandlerFunc(s.handleGetGameSse)))
	mux.Handle("POST /game/{id}/players", s.withGameRoom(http.HandlerFunc(s.handlePostGamePlayers)))
	mux.Handle("POST /game/{id}/leave", s.withGameRoom(http.HandlerFunc(s.handlePostGameLeave)))
	mux.Handle("POST /game/{id}/start", s.withGameRoom(http.HandlerFunc(s.handlePostGameStart)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Vary", "HX-Request")
		logger.Printf("%s %s", r.Method, r.URL.Path)
		mux.ServeHTTP(w, r)
	})
}

type baseData struct {
	Title string
}

func (s *server) retrieveGameRoom(gameId string) *room.Room {
	return s.rooms.GetRoom(gameId)
}

func (s *server) removeGame(gameId string) {
	s.rooms.RemoveRoom(gameId)
}

func (s *server) redirectToGame(w http.ResponseWriter, r *http.Request, g *game.Game) {
	gameUrl := "/game/" + g.Id
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Push-Url", gameUrl)
		data := &gameData{
			Base:   baseData{Title: "Game"},
			Game:   g,
			Player: g.GetPlayerById(getPlayerId(r)),
		}

		err := templates.Game.ExecutePartial(w, data)
		if err != nil {
			s.logger.Printf("Failed to execute template: %v", err)
		}
		return
	}
	http.Redirect(w, r, gameUrl, http.StatusSeeOther)
}

func (s *server) renderGame(w io.Writer, r *http.Request, g *game.Game) {
	data := &gameData{
		Base:   baseData{Title: "Game"},
		Game:   g,
		Player: g.GetPlayerById(getPlayerId(r)),
	}
	var err error
	if r.Header.Get("HX-Request") == "true" {
		err = templates.Game.ExecutePartial(w, data)
	} else {
		err = templates.Game.ExecuteFull(w, data)
	}
	if err != nil {
		s.logger.Printf("failed to execute template: %v", err)
	}
}

func (s *server) renderGameSse(w io.Writer, r *http.Request, g *game.Game) {
	data := &gameData{
		Base:   baseData{Title: "Game"},
		IsSse:  true,
		Game:   g,
		Player: g.GetPlayerById(getPlayerId(r)),
	}
	err := templates.Game.ExecutePartial(w, data)
	if err != nil {
		s.logger.Printf("failed to execute template: %v", err)
	}
}

func (s *server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	err := templates.Index.ExecuteFull(w, nil)
	if err != nil {
		s.logger.Printf("Failed to execute template: %v", err)
	}
}

func (s *server) handlePostGame(w http.ResponseWriter, r *http.Request) {
	gameRoom := s.rooms.NewRoom()
	playerId := gameRoom.EnsurePlayer("")
	err := gameRoom.Game.AddPlayer(playerId, r.FormValue("name"))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create game: %v", err), http.StatusBadRequest)
		return
	}

	r = withPlayerIdContext(r, gameRoom.Game.Players[0].Id)
	setPlayerIdCookie(w, r, gameRoom.Game.Id, gameRoom.Game.Players[0].Id)
	s.redirectToGame(w, r, gameRoom.Game)
}

func (s *server) handlePostGamePlayers(w http.ResponseWriter, r *http.Request) {
	gr := getGameRoom(r)
	playerId := getPlayerId(r)

	gr.Mu.Lock()
	defer gr.Mu.Unlock()
	p := gr.Game.GetPlayerById(playerId)
	if p != nil {
		s.renderGame(w, r, gr.Game)
		return
	}

	playerName := r.FormValue("name")

	err := gr.Game.AddPlayer(playerId, playerName)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to add player: %v", err), http.StatusBadRequest)
		return
	}
	gr.Notify()
	s.renderGame(w, r, gr.Game)
	//s.redirectToGame(w, r, gr.Game)
	return
}

func (s *server) handlePostGameLeave(w http.ResponseWriter, r *http.Request) {
	gr := getGameRoom(r)

	gr.Mu.Lock()
	defer gr.Mu.Unlock()
	gr.Game.RemovePlayer(getPlayerId(r))
	gr.Notify()

	// TODO: is this necessary?
	if gr.Game.IsEmpty() {
		s.removeGame(gr.Game.Id)
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	s.renderGame(w, r, gr.Game)
}

func (s *server) handlePostGameStart(w http.ResponseWriter, r *http.Request) {
	gr := getGameRoom(r)

	gr.Mu.Lock()
	defer gr.Mu.Unlock()
	p := gr.Game.GetPlayerById(getPlayerId(r))
	if p == nil {
		s.renderGame(w, r, gr.Game)
		return
	}
	err := gr.Game.Start()
	if err != nil {
		s.logger.Printf("failed to start game: %v", err)
	}

	s.renderGame(w, r, gr.Game)
}

type gameData struct {
	Base   baseData
	IsSse  bool
	Game   *game.Game
	Player *game.Player
}

func (s *server) handleGetGame(w http.ResponseWriter, r *http.Request) {
	gr := getGameRoom(r)

	gr.Mu.RLock()
	defer gr.Mu.RUnlock()
	s.renderGame(w, r, gr.Game)
}

const sseKeepAliveInterval = time.Second * 10

func (s *server) handleGetGameSse(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	gr := getGameRoom(r)

	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Content-Type", "text/event-stream")
	flusher.Flush()

	keepAliveTicker := time.NewTicker(sseKeepAliveInterval)
	defer func() {
		keepAliveTicker.Stop()
	}()

	notify := gr.Listen(r.Context())

	buf := &bytes.Buffer{}

	render := func() {
		gr.Mu.RLock()
		defer gr.Mu.RUnlock()
		s.renderGameSse(buf, r, gr.Game)
		_, err := fmt.Fprintf(w, "data: %s\n\n", strings.Replace(buf.String(), "\n", "", -1))
		if err != nil {
			s.logger.Printf("failed to write to SSE output: %v", err)
		}
		buf.Reset()
	}

	for {
		select {
		case <-notify:
			render()
			flusher.Flush()
			keepAliveTicker.Reset(sseKeepAliveInterval)
		case <-keepAliveTicker.C:
			// Send an empty SSE comment to keep connection alive
			_, err := fmt.Fprint(w, ":\n\n")
			if err != nil {
				s.logger.Printf("failed to write to SSE output: %v", err)
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
