package room

import (
	"fmt"
	"github.com/djcrock/prospect/internal/game"
	"github.com/djcrock/prospect/internal/util"
	"math/rand/v2"
	"sync"
)

type Collection struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewCollection() *Collection {
	return &Collection{
		rooms: make(map[string]*Room),
	}
}

func (c *Collection) NewRoom() *Room {
	c.mu.Lock()
	defer c.mu.Unlock()

	for range randomIdRetries {
		gameId := util.RandomString(gameIdLength)
		_, ok := c.rooms[gameId]
		if !ok {
			g := &game.Game{
				Id:   gameId,
				Rand: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
			}
			gameRoom := NewRoom(g)
			c.rooms[gameId] = gameRoom
			gameRoom.start()
			return gameRoom
		}
	}
	panic(fmt.Sprintf("Failed to generate a unique gameId after %d iterations", randomIdRetries))
}

func (c *Collection) GetRoom(gameId string) *Room {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rooms[gameId]
}

func (c *Collection) RemoveRoom(gameId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.rooms, gameId)
}

func (c *Collection) GetRoomForGame(game *game.Game) *Room {
	gameId := game.Id
	c.mu.RLock()
	room, ok := c.rooms[gameId]
	c.mu.RUnlock()
	if ok {
		return room
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Check one more time, in case a room was created while the lock was released
	room, ok = c.rooms[gameId]
	if ok {
		return room
	}

	room = NewRoom(game)
	c.rooms[gameId] = room

	return room
}
