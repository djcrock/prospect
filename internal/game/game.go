package game

import "errors"

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

func GetDeck(players int) ([]Card, error) {
	if players < 3 {
		return nil, errors.New("not enough players")
	}
	if players > 5 {
		return nil, errors.New("too many players")
	}
	if players == 3 {
		// Omit all cards containing 10 (the first 9 cards)
		cardsToRemove := 9
		result := make([]Card, len(baseDeck)-cardsToRemove)
		copy(result, baseDeck[cardsToRemove:])
		return result, nil
	}
	if players == 4 {
		// Remove the 10/9 card (the first card)
		cardsToRemove := 1
		result := make([]Card, len(baseDeck)-cardsToRemove)
		copy(result, baseDeck[cardsToRemove:])
		return result, nil
	}
	result := make([]Card, len(baseDeck))
	copy(result, baseDeck)
	return result, nil
}
