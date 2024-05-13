package game

import (
	"errors"
	"slices"
)

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

func (g *Game) Start() error {
	if !g.IsLobby() {
		return errors.New("game already started")
	}
	if !g.HasEnoughPlayers() {
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
			drawIndex := g.Rand.IntN(len(deck))
			drawnCard := deck[drawIndex]
			// Remove the drawnCard from the decl by replacing it with the last
			// card in the deck and reducing the length of the deck by one.
			deck[drawIndex] = deck[len(deck)-1]
			deck = deck[:len(deck)-1]

			// 50/50 chance of the card's orientation being flipped
			if g.Rand.IntN(2) == 0 {
				drawnCard = drawnCard.Flip()
			}
			p.Hand[handIndex] = drawnCard
		}
	}
}

func (g *Game) DecideHandOrientation(playerId string, flip bool) error {
	player, err := g.GetPlayerIndex(playerId)
	if err != nil {
		return err
	}
	p := &g.Players[player]
	if p.HasDecidedHandOrientation {
		return errors.New("player already has selected orientation")
	}

	p.HasDecidedHandOrientation = true

	if flip {
		for i := range p.Hand {
			p.Hand[i] = p.Hand[i].Flip()
		}
	}

	return nil
}

func (g *Game) Prospect(playerId string, left, flip bool, position int) error {
	player, err := g.GetPlayerIndex(playerId)
	if err != nil {
		return err
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
		card = card.Flip()
	}

	p.Hand = slices.Insert(p.Hand, position, card)
	g.Players[g.LastPlayerToPresent].ProspectTokens++

	if p.CanProspectAndPresent && g.CanPlayerPresent(playerId) {
		p.IsDecidingPresent = true
		return nil
	}

	g.nextTurn()

	return nil
}

func (g *Game) Present(playerId string, start, end int) error {
	player, err := g.GetPlayerIndex(playerId)
	if err != nil {
		return err
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
func (g *Game) Pass(playerId string) error {
	player, err := g.GetPlayerIndex(playerId)
	if err != nil {
		return err
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
