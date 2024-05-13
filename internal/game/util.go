package game

import (
	"cmp"
	"errors"
	"slices"
)

const minPlayers = 3
const maxPlayers = 5

var baseDeck = makeBaseDeck()

func makeBaseDeck() []Card {
	var baseDeck []Card
	for top := 10; top >= 1; top-- {
		for bottom := top - 1; bottom >= 1; bottom-- {
			baseDeck = append(baseDeck, Card{top, bottom})
		}
	}
	return baseDeck
}

func GetDeck(players int) []Card {
	cardsToRemove := 0
	if players == 3 {
		// Omit all cards containing 10 (the first 9 cards)
		cardsToRemove = 9
	}
	if players == 4 {
		// Remove the 10/9 card (the first card)
		cardsToRemove = 1
	}
	result := make([]Card, len(baseDeck)-cardsToRemove)
	copy(result, baseDeck[cardsToRemove:])
	return result
}

func (g *Game) GetPlayerById(id string) *Player {
	i, err := g.GetPlayerIndex(id)
	if err != nil {
		return nil
	}
	return &g.Players[i]
}

func (g *Game) GetPlayerIndex(id string) (int, error) {
	for i := range g.Players {
		if g.Players[i].Id == id {
			return i, nil
		}
	}

	return -1, errors.New("player not found")
}

func (g *Game) GetCurrentPlayer() *Player {
	return &g.Players[g.CurrentPlayer]
}

func (g *Game) IsEmpty() bool {
	return len(g.Players) == 0
}

func (g *Game) HasEnoughPlayers() bool {
	return len(g.Players) >= minPlayers
}

func (g *Game) IsFull() bool {
	return len(g.Players) >= maxPlayers
}

func (g *Game) IsLobby() bool {
	return g.Round == 0
}

func (g *Game) IsGameOver() bool {
	return g.Round == len(g.Players)
}

func (g *Game) HavePlayersDecidedHandOrientation() bool {
	for i := range g.Players {
		if !g.Players[i].HasDecidedHandOrientation {
			return false
		}
	}
	return true
}

func (g *Game) CanPlayerPresent(playerId string) bool {
	player, err := g.GetPlayerIndex(playerId)
	if err != nil {
		return false
	}
	return len(getPlayablePresentations(g.Players[player].Hand, g.Presentation)) > 0
}

func (g *Game) PlayablePresentations(playerId string) [][]Card {
	player, err := g.GetPlayerIndex(playerId)
	if err != nil {
		return nil
	}
	return getPlayablePresentations(g.Players[player].Hand, g.Presentation)
}

// getPlayablePresentations determines which presentations could be played from
// the given hand that would beat the provided presentation. Results are sorted
// from least to most valuable.
func getPlayablePresentations(hand, presentation []Card) [][]Card {
	validPresentations := GetValidPresentations(hand)
	for i := range validPresentations {
		if ComparePresentations(validPresentations[i], presentation) > 0 {
			return validPresentations[i:]
		}
	}

	return nil
}

// GetValidPresentations identifies the groups of cards in a hand that could be
// presented together. The results are sorted from least to most valuable.
func GetValidPresentations(hand []Card) [][]Card {
	var validPresentations [][]Card
	for i := range hand {
		for j := range hand[i:] {
			presentation := hand[i : i+j+1]
			if IsValidPresentation(presentation) {
				validPresentations = append(validPresentations, presentation)
			} else {
				// Adding more cards to an invalid presentation won't make it valid
				break
			}
		}
	}
	slices.SortFunc(validPresentations, ComparePresentations)
	return validPresentations
}

func IsValidPresentation(presentation []Card) bool {
	if len(presentation) == 0 {
		return false
	}
	if len(presentation) == 1 {
		return true
	}

	isAlike := true
	isAscendingRun := true
	isDescendingRun := true
	for i := 1; i < len(presentation); i++ {
		if presentation[i][0] != presentation[i-1][0] {
			isAlike = false
		}
		if presentation[i][0] != presentation[i-1][0]+1 {
			isAscendingRun = false
		}
		if presentation[i][0] != presentation[i-1][0]-1 {
			isDescendingRun = false
		}
		if !isAlike && !isAscendingRun && !isDescendingRun {
			return false
		}
	}

	return isAlike || isAscendingRun || isDescendingRun
}

// ComparePresentations compares the value of two presentations (assumed to be
// valid), and returns 1 if the value of a is greater, -1 if the value of b is
// greater, and 0 if they have the same value. This function is usable with the
// slices.SortFunc function to rank presentations.
func ComparePresentations(a, b []Card) int {
	if len(a) != len(b) {
		return cmp.Compare(len(a), len(b))
	}

	aMax := 0
	aSame := true
	for i := range a {
		if aMax != 0 && a[i][0] != aMax {
			aSame = false
		}
		if a[i][0] > aMax {
			aMax = a[i][0]
		}
	}

	bMax := 0
	bSame := true
	for i := range b {
		if bMax != 0 && b[i][0] != bMax {
			bSame = false
		}
		if b[i][0] > bMax {
			bMax = b[i][0]
		}
	}

	if aSame && !bSame {
		return 1
	}
	if !aSame && bSame {
		return -1
	}

	return cmp.Compare(aMax, bMax)
}
