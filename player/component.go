package player

import (
	"fmt"
	"github.com/google/uuid"

	cmap "github.com/orcaman/concurrent-map/v2"
)

var (
	ErrPlayerNotFound = fmt.Errorf("player not found")
)

type Component interface {
	NewPlayer(player Player) (string, error)
	GetPlayerByConnectionID(connectionID string) (Player, error)
	GetPlayersByGameID(gameID string) ([]Player, error)
	DeletePlayerByConnectionID(connectionID string)
}

type component struct {
	playerStore cmap.ConcurrentMap[string, Player]
}

func NewPlayerComponent() *component {
	return &component{playerStore: cmap.New[Player]()}
}

func (c *component) NewPlayer(player Player) (string, error) {
	userID := uuid.NewString()

	c.playerStore.Set(userID, player)

	return userID, nil
}

func (c *component) GetPlayerByConnectionID(connectionID string) (Player, error) {
	var player Player
	var found bool

	c.playerStore.IterCb(func(_ string, p Player) {
		if p.ConnectionID == connectionID {
			player = p
			found = true
			return
		}
	})

	if !found {
		return Player{}, ErrPlayerNotFound
	}

	return player, nil
}

func (c *component) GetPlayersByGameID(gameID string) ([]Player, error) {
	var players []Player

	c.playerStore.IterCb(func(_ string, p Player) {
		if p.GameID == gameID {
			players = append(players, p)
		}
	})

	return players, nil
}

func (c *component) DeletePlayerByConnectionID(connectionID string) {
	var id string
	c.playerStore.IterCb(func(userID string, p Player) {
		if p.ConnectionID == connectionID {
			id = userID
		}
	})

	if id != "" {
		c.playerStore.Pop(id)
		return
	}
}
