package game

type Game struct {
	Round               int
	CurrentPlayer       int
	LastPlayerToPresent int
	Deck                []Card
	Presentation        []Card
	Players             []Player
}

type Player struct {
	Name                  string
	Hand                  []Card
	Points                int
	ProspectTokens        int
	ScorePile             int
	CanProspectAndPresent bool
}

type Card [2]int
