package game

import (
	"testing"
)

func TestGetDeck(t *testing.T) {
	t.Run("2 players - too few", func(t *testing.T) {
		_, err := GetDeck(2)
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("6 players - too many", func(t *testing.T) {
		_, err := GetDeck(6)
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("5 players - 45 card deck", func(t *testing.T) {
		deck, err := GetDeck(5)
		if err != nil {
			t.Fatalf("no error expected, got %v", err)
		}
		if len(deck) != 45 {
			t.Fatalf("expected 45 card deck, got %v", len(deck))
		}
	})
	t.Run("4 players - 44 card deck without 9/10", func(t *testing.T) {
		deck, err := GetDeck(4)
		if err != nil {
			t.Fatalf("no error expected, got %v", err)
		}
		if len(deck) != 44 {
			t.Fatalf("expected 44 card deck, got %v", len(deck))
		}
		for _, card := range deck {
			if card == (Card{9, 10}) || card == (Card{10, 9}) {
				t.Fatalf("deck contains invalid card: %v", card)
			}
		}
	})
	t.Run("3 players - 36 card deck with no 10s", func(t *testing.T) {
		deck, err := GetDeck(3)
		if err != nil {
			t.Fatalf("no error expected, got %v", err)
		}
		if len(deck) != 36 {
			t.Fatalf("expected 36 card deck, got %v", len(deck))
		}
		for _, card := range deck {
			if card[0] == 10 || card[1] == 10 {
				t.Fatalf("deck contains invalid card: %v", card)
			}
		}
	})
}
