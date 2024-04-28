package game

import (
	"fmt"
	"math/rand/v2"
	"testing"
)

func TestGetDeck(t *testing.T) {
	t.Run("5 players - 45 card deck", func(t *testing.T) {
		deck := GetDeck(5)
		if len(deck) != 45 {
			t.Fatalf("expected 45 card deck, got %v", len(deck))
		}
	})
	t.Run("4 players - 44 card deck without 9/10", func(t *testing.T) {
		deck := GetDeck(4)
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
		deck := GetDeck(3)
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

func TestComparePresentations(t *testing.T) {
	type args struct {
		a []Card
		b []Card
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{"nothing equals nothing", args{[]Card{}, []Card{}}, 0},
		{"something beats nothing", args{[]Card{{1, 2}}, []Card{}}, 1},
		{"like beats run", args{[]Card{{1, 2}, {1, 3}, {1, 4}}, []Card{{1, 2}, {2, 3}, {3, 4}}}, 1},
		{"more beats fewer", args{[]Card{{1, 2}, {2, 3}, {3, 4}}, []Card{{1, 2}, {1, 3}}}, 1},
		{"high like beats low like", args{[]Card{{2, 1}, {2, 3}}, []Card{{1, 4}, {1, 5}}}, 1},
		{"high run beats low run", args{[]Card{{7, 1}, {8, 2}}, []Card{{1, 2}, {2, 3}}}, 1},
		{"equal runs draw", args{[]Card{{7, 1}, {8, 2}}, []Card{{7, 2}, {8, 3}}}, 0},
		{"equal likes draw", args{[]Card{{6, 1}, {6, 2}}, []Card{{6, 2}, {6, 3}}}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ComparePresentations(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("ComparePresentations(%v, %v) = %v, want %v", tt.args.a, tt.args.b, got, tt.want)
			}
			if got := ComparePresentations(tt.args.b, tt.args.a); got != -tt.want {
				t.Errorf("flipped ComparePresentations(%v, %v) = %v, want %v", tt.args.b, tt.args.a, got, -tt.want)
			}
		})
	}
}

func TestGame_AddPlayer(t *testing.T) {
	g := &Game{}
	maxPlayers := 5
	for i := range maxPlayers {
		err := g.AddPlayer(fmt.Sprintf("Player %d", i))
		if err != nil {
			t.Fatalf("unexpected error adding player %d: %v", i, err)
		}
	}
	err := g.AddPlayer("Excess Player")
	if err == nil {
		t.Fatalf("expected error adding player excess player")
	}
}

func TestGame_Start(t *testing.T) {
	g := &Game{rand: rand.New(rand.NewPCG(1, 2))}
	err := g.Start()
	if err == nil {
		t.Fatalf("expected error starting game with 0 players")
	}

	err = g.AddPlayer("Player 0")
	if err != nil {
		t.Fatalf("unexpected error adding player 0: %v", err)
	}
	err = g.Start()
	if err == nil {
		t.Fatalf("expected error starting game with 1 player")
	}

	err = g.AddPlayer("Player 1")
	if err != nil {
		t.Fatalf("unexpected error adding player 1: %v", err)
	}
	err = g.Start()
	if err == nil {
		t.Fatalf("expected error starting game with 2 players")
	}

	err = g.AddPlayer("Player 2")
	if err != nil {
		t.Fatalf("unexpected error adding player 2: %v", err)
	}
	err = g.Start()
	if err != nil {
		t.Fatalf("unexpected error starting game with 3 players: %v", err)
	}
}

func TestGame_DecideHandOrientation(t *testing.T) {
	t.Run("no flip", func(t *testing.T) {
		g := &Game{Round: 1, Players: []Player{
			{Hand: []Card{{1, 2}, {3, 4}}},
		}}
		err := g.DecideHandOrientation(0, false)
		if err != nil {
			t.Fatalf("unexpected error deciding hand orientation: %v", err)
		}
		assertCardSlicesEqual(t, []Card{{1, 2}, {3, 4}}, g.Players[0].Hand)
		if !g.HavePlayersDecidedHandOrientation() {
			t.Fatalf("expected players to have decided hand orientation")
		}
	})

	t.Run("flip", func(t *testing.T) {
		g := &Game{Round: 1, Players: []Player{
			{Hand: []Card{{1, 2}, {3, 4}}},
		}}
		err := g.DecideHandOrientation(0, true)
		if err != nil {
			t.Fatalf("unexpected error deciding hand orientation: %v", err)
		}
		assertCardSlicesEqual(t, []Card{{2, 1}, {4, 3}}, g.Players[0].Hand)
		if !g.HavePlayersDecidedHandOrientation() {
			t.Fatalf("expected players to have decided hand orientation")
		}
	})
}

func TestGame_Present(t *testing.T) {
	t.Run("round ending presentation", func(t *testing.T) {
		g := &Game{
			Round: 1,
			Players: []Player{
				{Hand: []Card{{1, 2}, {2, 3}}, Points: 1, ScorePile: 2, ProspectTokens: 3, HasDecidedHandOrientation: true},
				{Hand: []Card{{5, 6}, {7, 8}}, HasDecidedHandOrientation: true},
			},
			rand: rand.New(rand.NewPCG(1, 2)),
		}

		err := g.Present(0, 0, 2)
		if err != nil {
			t.Fatalf("unexpected error presenting for player 0: %v", err)
		}
		if g.Players[0].Points != 6 {
			t.Fatalf("expected 6 points for player 0, got %v", g.Players[0].Points)
		}
		if g.Players[1].Points != -2 {
			t.Fatalf("expected -2 points for player 1, got %v", g.Players[1].Points)
		}
		if g.Round != 2 {
			t.Fatalf("expected game to enter round 2, got %v", g.Round)
		}
		if g.CurrentPlayer != 1 {
			t.Fatalf("expected current player to be 1, got %v", g.CurrentPlayer)
		}

	})
}

func TestIsValidPresentation(t *testing.T) {
	tests := []struct {
		name         string
		presentation []Card
		want         bool
	}{
		{"empty is invalid", []Card{}, false},
		{"non-sequential cards are invalid", []Card{{1, 2}, {2, 4}, {5, 6}}, false},
		{"single card is valid", []Card{{1, 2}}, true},
		{"like cards valid", []Card{{1, 2}, {1, 4}, {1, 3}}, true},
		{"ascending cards valid", []Card{{1, 2}, {2, 4}, {3, 6}}, true},
		{"descending cards valid", []Card{{6, 2}, {5, 4}, {4, 6}}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidPresentation(tt.presentation); got != tt.want {
				t.Errorf("IsValidPresentation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGameplay(t *testing.T) {
	g := &Game{rand: rand.New(rand.NewPCG(1, 2))}
	for i := range 3 {
		err := g.AddPlayer(fmt.Sprintf("Player %d", i))
		if err != nil {
			t.Fatalf("unexpected error adding player %d: %v", i, err)
		}
	}
	err := g.Start()
	if err != nil {
		t.Fatalf("unexpected error starting game: %v", err)
	}

	err = g.DecideHandOrientation(0, false)
	if err != nil {
		t.Fatalf("unexpected error decideing hand orientation: %v", err)
	}
	err = g.DecideHandOrientation(0, true)
	if err == nil {
		t.Fatalf("expected error decideing hand orientation for second time")
	}
	err = g.DecideHandOrientation(1, true)
	if err != nil {
		t.Fatalf("unexpected error decideing hand orientation: %v", err)
	}
	err = g.DecideHandOrientation(2, true)
	if err != nil {
		t.Fatalf("unexpected error decideing hand orientation: %v", err)
	}

	assertCardSlicesEqual(t, []Card{{3, 5}, {1, 2}, {1, 9}, {5, 7}, {1, 6}, {1, 7}, {5, 2}, {8, 4}, {3, 4}, {7, 9}, {5, 4}, {3, 9}}, g.Players[0].Hand)
	assertCardSlicesEqual(t, []Card{{8, 7}, {1, 4}, {6, 5}, {8, 3}, {5, 8}, {5, 1}, {1, 3}, {9, 2}, {4, 2}, {7, 2}, {6, 3}, {4, 7}}, g.Players[1].Hand)
	assertCardSlicesEqual(t, []Card{{1, 8}, {6, 8}, {4, 6}, {6, 7}, {6, 2}, {3, 7}, {9, 5}, {2, 3}, {9, 6}, {4, 9}, {8, 2}, {8, 9}}, g.Players[2].Hand)

	err = g.Present(0, 3, 4)
	if err != nil {
		t.Fatalf("unexpected error presenting for player 0: %v", err)
	}
	if g.Players[0].ScorePile != 0 {
		t.Fatalf("expected player 1 ScorePile to be 0, got %d", g.Players[0].ScorePile)
	}
	assertCardSlicesEqual(t, []Card{{5, 7}}, g.Presentation)
	assertCardSlicesEqual(t, []Card{{3, 5}, {1, 2}, {1, 9}, {1, 6}, {1, 7}, {5, 2}, {8, 4}, {3, 4}, {7, 9}, {5, 4}, {3, 9}}, g.Players[0].Hand)

	err = g.Present(1, 0, 1)
	if err != nil {
		t.Fatalf("unexpected error presenting for player 1: %v", err)
	}
	if g.Players[1].ScorePile != 1 {
		t.Fatalf("expected player 1 ScorePile to be 1, got %d", g.Players[1].ScorePile)
	}
	assertCardSlicesEqual(t, []Card{{8, 7}}, g.Presentation)
	assertCardSlicesEqual(t, []Card{{1, 4}, {6, 5}, {8, 3}, {5, 8}, {5, 1}, {1, 3}, {9, 2}, {4, 2}, {7, 2}, {6, 3}, {4, 7}}, g.Players[1].Hand)

	err = g.Present(2, 6, 7)
	if err != nil {
		t.Fatalf("unexpected error presenting for player 2: %v", err)
	}
	if g.Players[2].ScorePile != 1 {
		t.Fatalf("expected player 2 ScorePile to be 1, got %d", g.Players[2].ScorePile)
	}
	assertCardSlicesEqual(t, []Card{{9, 5}}, g.Presentation)
	assertCardSlicesEqual(t, []Card{{1, 8}, {6, 8}, {4, 6}, {6, 7}, {6, 2}, {3, 7}, {2, 3}, {9, 6}, {4, 9}, {8, 2}, {8, 9}}, g.Players[2].Hand)

	err = g.Prospect(0, true, true, 5)
	if err != nil {
		t.Fatalf("unexpected error prospecting player 0: %v", err)
	}
	if g.Players[2].ProspectTokens != 1 {
		t.Fatalf("expected player 2 ProspectTokens to be 1, got %d", g.Players[2].ProspectTokens)
	}
	assertCardSlicesEqual(t, []Card{}, g.Presentation)
	assertCardSlicesEqual(t, []Card{{3, 5}, {1, 2}, {1, 9}, {1, 6}, {1, 7}, {5, 9}, {5, 2}, {8, 4}, {3, 4}, {7, 9}, {5, 4}, {3, 9}}, g.Players[0].Hand)

	err = g.Present(1, 5, 6)
	if err != nil {
		t.Fatalf("unexpected error presenting for player 1: %v", err)
	}
	assertCardSlicesEqual(t, []Card{{1, 3}}, g.Presentation)
	assertCardSlicesEqual(t, []Card{{1, 4}, {6, 5}, {8, 3}, {5, 8}, {5, 1}, {9, 2}, {4, 2}, {7, 2}, {6, 3}, {4, 7}}, g.Players[1].Hand)
}

func assertCardSlicesEqual(t *testing.T, a, b []Card) {
	if len(a) != len(b) {
		t.Fatalf("expected %d cards, got %d: %v, %v", len(a), len(b), a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("array element mismatch at index %d: expected %v cards, got %v: %v, %v", i, a[i], b[i], a, b)

		}
	}
}
