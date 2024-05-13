package room

import (
	"context"
	"fmt"
	"github.com/djcrock/prospect/internal/game"
	"github.com/djcrock/prospect/internal/util"
	"sync"
)

const randomIdRetries = 100
const gameIdLength = 12
const playerIdLength = 12

type listener chan<- struct{}

type Room struct {
	Mu        sync.RWMutex
	Game      *game.Game
	listeners map[listener]bool
	// TODO: Can this field be removed? What is it actually doing?
	playerIds map[string]bool

	register   chan listener
	unregister chan listener
}

func NewRoom(game *game.Game) *Room {
	r := &Room{
		Game:       game,
		listeners:  make(map[listener]bool),
		playerIds:  make(map[string]bool),
		register:   make(chan listener),
		unregister: make(chan listener),
	}
	for i := range r.Game.Players {
		r.EnsurePlayer(r.Game.Players[i].Id)
	}

	return r
}

func (r *Room) start() {
	go func() {
		for {
			select {
			case l := <-r.register:
				r.listeners[l] = true
			case l := <-r.unregister:
				delete(r.listeners, l)
			}
		}
	}()
}

func (r *Room) EnsurePlayer(existingPlayerId string) string {
	r.Mu.Lock()
	defer r.Mu.Unlock()
	// Allow players to re-use playerIds from existing cookies
	// (e.g. in the event of a server restart).
	if existingPlayerId != "" {
		r.playerIds[existingPlayerId] = true
		return existingPlayerId
	}
	for range randomIdRetries {
		playerId := util.RandomString(playerIdLength)
		if !r.playerIds[playerId] {
			r.playerIds[playerId] = true
			return playerId
		}
	}
	panic(fmt.Sprintf("Failed to generate a unique playerId after %d iterations", randomIdRetries))
}

func (r *Room) Listen(ctx context.Context) <-chan struct{} {
	// The notify channel is buffered to allow many listeners to be notified concurrently.
	// If the notify channel buffer is full, that means the listener still hasn't reacted
	// to the previous notification and does not need to be notified again.
	notify := make(chan struct{}, 1)

	r.register <- notify

	go func() {
		<-ctx.Done()
		r.unregister <- notify
	}()

	return notify
}

func (r *Room) Notify() {
	for l := range r.listeners {
		select {
		case l <- struct{}{}:
		default:
			// Client already has a pending notification; skip it.
		}
	}
}
