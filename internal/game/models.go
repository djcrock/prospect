package game

import "math/rand/v2"

type Game struct {
	Round               int
	CurrentPlayer       int
	LastPlayerToPresent int
	Presentation        []Card
	Players             []Player

	rand *rand.Rand
}

type Player struct {
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
