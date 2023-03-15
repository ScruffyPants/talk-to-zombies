package game

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/ScruffyPants/talk-to-zombies/communication"
	"github.com/ScruffyPants/talk-to-zombies/player"
)

type Settings struct {
	ZombieCoordinateUpdateInterval time.Duration
}

type gameInstance struct {
	id           string
	zombieList   []zombie
	zombieTicker *time.Ticker
	gameOverChan chan bool
	settings     Settings

	playerComponent      player.Component
	communicationService communication.Service
}

func newGameInstance(gameID string,
	settings Settings,
	playerComponent player.Component,
	communicationService communication.Service) *gameInstance {

	instance := &gameInstance{
		id:           gameID,
		zombieList:   []zombie{newZombie()},
		gameOverChan: make(chan bool),
		settings:     settings,

		playerComponent:      playerComponent,
		communicationService: communicationService,
	}

	instance.startZombieTicker()

	return instance
}

func (i *gameInstance) startZombieTicker() {
	i.zombieTicker = time.NewTicker(i.settings.ZombieCoordinateUpdateInterval)

	go func() {
		for {
			_, ok := <-i.zombieTicker.C
			if !ok {
				return
			}

			i.handleZombieUpdate()
		}
	}()
}

func (i *gameInstance) handleUserShot(x, y int, player player.Player) {
	for j := range i.zombieList {
		if i.zombieList[j].x == x && i.zombieList[j].y == y {
			shotHitMessage := fmt.Sprintf("BOOM %s 1 %s", player.Username, i.zombieList[j].name)
			i.broadcastToAllPlayers(shotHitMessage)
			i.stop()
			return
		}
	}

	shotMissedMessage := fmt.Sprintf("BOOM %s 0", player.Username)
	if err := i.communicationService.SendMessageToConnection(player.ConnectionID, []byte(shotMissedMessage)); err != nil {
		logrus.Errorf("error sending message to connection: %s", err)
	}
}

func (i *gameInstance) handleZombieUpdate() {
	for j := range i.zombieList {
		switch rand.Intn(4) {
		case 0:
			if i.zombieList[j].x > 0 {
				i.zombieList[j].x--
			} else {
				i.zombieList[j].x++
			}
		case 1:
			if i.zombieList[j].x < 10 {
				i.zombieList[j].x++
			} else {
				i.zombieList[j].x--
			}
		default:
			if i.zombieList[j].y < 30 {
				i.zombieList[j].y++
			} else {
				i.zombieList[j].y--
			}
		}

		zombieCoordinatesMessage := fmt.Sprintf("WALK %s %d %d", i.zombieList[j].name, i.zombieList[j].x, i.zombieList[j].y)
		i.broadcastToAllPlayers(zombieCoordinatesMessage)

		if i.zombieList[j].y >= 30 {
			// TODO: broadcast zombie has reached player?
			i.stop()
		}
	}
}

func (i *gameInstance) stop() {
	close(i.gameOverChan)
	i.zombieTicker.Stop()
}

func (i *gameInstance) broadcastToAllPlayers(message string) {
	players, err := i.playerComponent.GetPlayersByGameID(i.id)
	if err != nil {
		logrus.Errorf("error trying to get players by gameID (%s): %s", i.id, err.Error())
		return
	}

	for _, p := range players {
		go func() {
			if err = i.communicationService.SendMessageToConnection(p.ConnectionID, []byte(message)); err != nil {
				logrus.Errorf("error sending message to connection: %s", err)
			}
		}()
	}
}
