package game

import (
	"math/rand/v2"
	"sync"
)

type Game struct {
	Mu                  sync.RWMutex
	Id                  string
	Round               int
	CurrentPlayer       int
	LastPlayerToPresent int
	Presentation        []Card
	Players             []Player

	Rand *rand.Rand
}

type Player struct {
	Id                        string
	Name                      string
	Hand                      []Card
	Points                    int
	ProspectTokens            int
	ScorePile                 int
	CanProspectAndPresent     bool
	HasDecidedHandOrientation bool
	IsDecidingPresent         bool
}

type Card [2]int

func (c Card) Flip() Card {
	return Card{c[1], c[0]}
}
