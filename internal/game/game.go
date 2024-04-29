package game

import (
	"cmp"
	"errors"
	"slices"
)

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

func (g *Game) AddPlayer(id, name string) error {
	if g.IsFull() {
		return errors.New("game is full")
	}
	if p := g.GetPlayerById(id); p != nil {
		return errors.New("player already exists")
	}

	g.Players = append(g.Players, Player{Id: id, Name: name})

	return nil
}

func (g *Game) RemovePlayer(id string) {
	if g.Round > 0 {
		return
	}
	i, err := g.GetPlayerIndex(id)
	if err != nil {
		return
	}
	g.Players = slices.Delete(g.Players, i, i+1)
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

func (g *Game) Start() error {
	if len(g.Players) < 3 {
		return errors.New("not enough players")
	}
	g.startRound()
	return nil
}

func (g *Game) startRound() {
	deck := GetDeck(len(g.Players))
	g.CurrentPlayer = g.Round
	g.Round++
	cardsPerPlayer := len(deck) / len(g.Players)
	for i := range g.Players {
		p := &g.Players[i]
		p.HasDecidedHandOrientation = false
		p.CanProspectAndPresent = true
		p.Hand = make([]Card, cardsPerPlayer)
		for handIndex := range cardsPerPlayer {
			drawIndex := g.rand.IntN(len(deck))
			drawnCard := deck[drawIndex]
			// Remove the drawnCard from the decl by replacing it with the last
			// card in the deck and reducing the length of the deck by one.
			deck[drawIndex] = deck[len(deck)-1]
			deck = deck[:len(deck)-1]

			// 50/50 chance of the card's orientation being flipped
			if g.rand.IntN(2) == 0 {
				drawnCard[0], drawnCard[1] = drawnCard[1], drawnCard[0]
			}
			p.Hand[handIndex] = drawnCard
		}
	}
}

func (g *Game) HavePlayersDecidedHandOrientation() bool {
	for i := range g.Players {
		if !g.Players[i].HasDecidedHandOrientation {
			return false
		}
	}
	return true
}

func (g *Game) DecideHandOrientation(player int, flip bool) error {
	if player < 0 || player >= len(g.Players) {
		return errors.New("player out of range")
	}
	p := &g.Players[player]
	if p.HasDecidedHandOrientation {
		return errors.New("player already has selected orientation")
	}

	p.HasDecidedHandOrientation = true

	if flip {
		for i := range p.Hand {
			p.Hand[i][0], p.Hand[i][1] = p.Hand[i][1], p.Hand[i][0]
		}
	}

	return nil
}

func (g *Game) Prospect(player int, left, flip bool, position int) error {
	if player < 0 || player >= len(g.Players) {
		return errors.New("player out of range")
	}
	if player != g.CurrentPlayer {
		return errors.New("not your turn")
	}
	if len(g.Presentation) == 0 {
		return errors.New("nothing to prospect")
	}
	p := &g.Players[g.CurrentPlayer]
	if position > len(p.Hand) {
		return errors.New("position out of range")
	}

	var card Card
	if left {
		card = g.Presentation[0]
		g.Presentation = g.Presentation[1:]
	} else {
		card = g.Presentation[len(g.Presentation)-1]
		g.Presentation = g.Presentation[:len(g.Presentation)-1]
	}
	if flip {
		card[0], card[1] = card[1], card[0]
	}

	p.Hand = slices.Insert(p.Hand, position, card)
	g.Players[g.LastPlayerToPresent].ProspectTokens++

	if p.CanProspectAndPresent && g.CanPlayerPresent(player) {
		p.IsDecidingPresent = true
		return nil
	}

	g.nextTurn()

	return nil
}

func (g *Game) Present(player, start, end int) error {
	if player < 0 || player >= len(g.Players) {
		return errors.New("player out of range")
	}
	if player != g.CurrentPlayer {
		return errors.New("not your turn")
	}
	if !g.HavePlayersDecidedHandOrientation() {
		return errors.New("waiting for players to select hand orientation")
	}
	p := &g.Players[g.CurrentPlayer]
	if start < 0 || start >= len(p.Hand) {
		return errors.New("start is out of range")
	}
	if end < start || end > len(p.Hand) {
		return errors.New("end is out of range")
	}

	newPresentation := append([]Card(nil), p.Hand[start:end]...)
	if !IsValidPresentation(newPresentation) {
		return errors.New("invalid presentation")
	}
	if ComparePresentations(newPresentation, g.Presentation) <= 0 {
		return errors.New("new presentation does not beat existing presentation")
	}

	p.ScorePile += len(g.Presentation)
	g.LastPlayerToPresent = g.CurrentPlayer
	g.Presentation = newPresentation

	// Remove the presented cards from the Player's hand
	p.Hand = slices.Delete(p.Hand, start, end)

	// If the Player did a ProspectAndPresent, consume that opportunity
	if p.IsDecidingPresent {
		p.CanProspectAndPresent = false
		p.IsDecidingPresent = false
	}

	if len(p.Hand) == 0 {
		g.endRound()
		return nil
	}

	g.nextTurn()
	return nil
}

// Pass on the opportunity to Present after Prospect.
func (g *Game) Pass(player int) error {
	if player < 0 || player >= len(g.Players) {
		return errors.New("player out of range")
	}
	if player != g.CurrentPlayer {
		return errors.New("not your turn")
	}
	p := &g.Players[g.CurrentPlayer]
	if !p.IsDecidingPresent {
		return errors.New("must prospect or present")
	}
	p.IsDecidingPresent = false
	g.nextTurn()
	return nil
}

func (g *Game) nextTurn() {
	g.CurrentPlayer = (g.CurrentPlayer + 1) % len(g.Players)
	if g.CurrentPlayer == g.LastPlayerToPresent {
		g.endRound()
	}
}

func (g *Game) endRound() {
	for i := range g.Players {
		p := &g.Players[i]
		p.Points += p.ScorePile
		p.ScorePile = 0
		p.Points += p.ProspectTokens
		p.ProspectTokens = 0
		if i != g.LastPlayerToPresent {
			p.Points -= len(p.Hand)
		}
		p.Hand = nil
	}

	if g.IsGameOver() {
		return
	}
	g.startRound()
}

func (g *Game) IsEmpty() bool {
	return len(g.Players) == 0
}

func (g *Game) IsFull() bool {
	return len(g.Players) >= maxPlayers
}

func (g *Game) IsGameOver() bool {
	return g.Round == len(g.Players)
}

func (g *Game) CanPlayerPresent(player int) bool {
	return len(GetPlayablePresentations(g.Players[player].Hand, g.Presentation)) > 0
}

// GetPlayablePresentations determines which presentations could be played from
// the given hand that would beat the provided presentation. Results are sorted
// from least to most valuable.
func GetPlayablePresentations(hand, presentation []Card) [][]Card {
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
			}
		}
	}
	slices.SortFunc(validPresentations, ComparePresentations)
	return validPresentations
}

func getPresentationVals(presentation []Card) []int {
	vals := make([]int, len(presentation))
	for i := range presentation {
		vals[i] = presentation[i][0]
	}
	return vals
}

func IsValidPresentation(presentation []Card) bool {
	if len(presentation) == 0 {
		return false
	}
	if len(presentation) == 1 {
		return true
	}
	vals := getPresentationVals(presentation)

	isAlike := true
	isAscendingRun := true
	isDescendingRun := true
	for i := 1; i < len(vals); i++ {
		if vals[i] != vals[i-1] {
			isAlike = false
		}
		if vals[i] != vals[i-1]+1 {
			isAscendingRun = false
		}
		if vals[i] != vals[i-1]-1 {
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
	for _, aVal := range getPresentationVals(a) {
		if aMax != 0 && aVal != aMax {
			aSame = false
		}
		if aVal > aMax {
			aMax = aVal
		}
	}

	bMax := 0
	bSame := true
	for _, bVal := range getPresentationVals(b) {
		if bMax != 0 && bVal != bMax {
			bSame = false
		}
		if bVal > bMax {
			bMax = bVal
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
