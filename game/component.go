package game

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	cmap "github.com/orcaman/concurrent-map/v2"
	"github.com/sirupsen/logrus"
	"github.com/teris-io/shortid"

	"github.com/ScruffyPants/talk-to-zombies/communication"
	"github.com/ScruffyPants/talk-to-zombies/player"
)

type Component interface{}

type component struct {
	playerComponent      player.Component
	communicationService communication.Service

	gameSettings      Settings
	gameInstanceStore cmap.ConcurrentMap[string, *gameInstance]
	shortIDGenerator  *shortid.Shortid
}

func NewGameComponent(
	playerComponent player.Component,
	communicationService communication.Service,
	gameSettings Settings) (*component, error) {
	shortIDGenerator, err := shortid.New(1, shortid.DefaultABC, rand.Uint64())
	if err != nil {
		return nil, err
	}

	return &component{
		playerComponent:      playerComponent,
		communicationService: communicationService,

		gameSettings:      gameSettings,
		gameInstanceStore: cmap.New[*gameInstance](),
		shortIDGenerator:  shortIDGenerator,
	}, nil
}

func (c *component) OnMessage(ctx context.Context, connectionID string, message communication.Message) {
	playerByConnectionID, err := c.playerComponent.GetPlayerByConnectionID(connectionID)
	if err != nil && !errors.Is(err, player.ErrPlayerNotFound) {
		c.sendErrorToConnection(ctx, connectionID, fmt.Errorf("error getting player by connection ID: %w", err))
		return
	}
	isInGame := !errors.Is(err, player.ErrPlayerNotFound)

	switch strings.ToLower(message.Type) {
	case "start":
		c.handleStart(ctx, connectionID, isInGame, message.Arguments)
	case "shoot":
		c.handleShoot(ctx, playerByConnectionID, isInGame, message.Arguments)
	case "join":
		c.handleJoin(ctx, connectionID, isInGame, message.Arguments)
	default:
		c.sendMessageToConnection(ctx, connectionID, fmt.Sprintf("command %s is not supported", message.Type))
		return
	}
}

func (c *component) OnDisconnect(ctx context.Context, connectionID string) {
	playerByConnectionID, err := c.playerComponent.GetPlayerByConnectionID(connectionID)
	if err != nil && !errors.Is(err, player.ErrPlayerNotFound) {
		c.sendErrorToConnection(ctx, connectionID, fmt.Errorf("error getting player by connection ID: %w", err))
		return
	}

	defer c.playerComponent.DeletePlayerByConnectionID(connectionID)

	if len(playerByConnectionID.GameID) == 0 {
		return
	}

	gamePlayers, err := c.playerComponent.GetPlayersByGameID(playerByConnectionID.GameID)
	if err != nil {
		c.sendErrorToConnection(ctx, connectionID, fmt.Errorf("error getting players by game id: %w", err))
		return
	}

	if len(gamePlayers) == 0 {
		instance, ok := c.gameInstanceStore.Get(playerByConnectionID.GameID)
		if !ok {
			return
		}

		instance.stop()
		c.gameInstanceStore.Pop(playerByConnectionID.GameID)
	}
}

func (c *component) handleStart(ctx context.Context, connectionID string, isInGame bool, arguments []string) {
	if isInGame {
		c.sendMessageToConnection(ctx, connectionID, "already in a game")
		return
	}

	if len(arguments) != 1 {
		c.sendMessageToConnection(ctx, connectionID, "start command requires one argument: START {player}")
		return
	}

	gameID, err := c.shortIDGenerator.Generate()
	if err != nil {
		logrus.WithContext(ctx).Errorf("error generating short id: %s", err.Error())
		return
	}

	p := player.Player{
		ConnectionID: connectionID,
		Username:     arguments[0],
		GameID:       gameID,
	}

	if _, err = c.playerComponent.NewPlayer(p); err != nil {
		c.sendErrorToConnection(ctx, connectionID, fmt.Errorf("error creating new player: %w", err))
		return
	}

	instance := newGameInstance(gameID, c.gameSettings, c.playerComponent, c.communicationService)
	c.listenToGameOverSignal(instance)

	c.gameInstanceStore.Set(gameID, instance)

	c.sendMessageToConnection(ctx, connectionID, fmt.Sprintf("GAME %s", gameID))
}

func (c *component) handleShoot(ctx context.Context, playerByConnectionID player.Player, isInGame bool, arguments []string) {
	if !isInGame {
		c.sendMessageToConnection(ctx, playerByConnectionID.ConnectionID, "must be in a game")
		return
	}

	instance, ok := c.gameInstanceStore.Get(playerByConnectionID.GameID)
	if !ok {
		c.playerComponent.DeletePlayerByConnectionID(playerByConnectionID.ConnectionID)
		return // TODO: maybe handle this more gracefully
	}

	if len(arguments) != 2 {
		c.sendMessageToConnection(ctx, playerByConnectionID.ConnectionID, "shoot command requires two arguments: SHOOT {x} {y}")
		return
	}

	x, err := strconv.Atoi(arguments[0])
	if err != nil {
		c.sendMessageToConnection(ctx, playerByConnectionID.ConnectionID, "coordinate must be an integer")
		return
	}

	y, err := strconv.Atoi(arguments[1])
	if err != nil {
		c.sendMessageToConnection(ctx, playerByConnectionID.ConnectionID, "coordinate must be an integer")
		return
	}

	instance.handleUserShot(x, y, playerByConnectionID)
}

func (c *component) handleJoin(ctx context.Context, connectionID string, isInGame bool, arguments []string) {
	if isInGame {
		c.sendMessageToConnection(ctx, connectionID, "already in game")
		return
	}

	if len(arguments) != 2 {
		c.sendMessageToConnection(ctx, connectionID, "join command requires two arguments: JOIN {game ID} {username}")
		return
	}

	instance, ok := c.gameInstanceStore.Get(arguments[0])
	if !ok {
		c.sendMessageToConnection(ctx, connectionID, "game not found")
		return
	}

	if _, err := c.playerComponent.NewPlayer(player.Player{
		ConnectionID: connectionID,
		Username:     arguments[1],
		GameID:       instance.id,
	}); err != nil {
		c.sendErrorToConnection(ctx, connectionID, fmt.Errorf("error creating user instance: %w", err))
		return
	}
}

func (c *component) sendErrorToConnection(ctx context.Context, connectionID string, err error) {
	c.sendMessageToConnection(ctx, connectionID, err.Error())
	logrus.WithContext(ctx).Error(err)
}

func (c *component) sendMessageToConnection(ctx context.Context, connectionID string, message string) {
	if err := c.communicationService.SendMessageToConnection(connectionID, []byte(message)); err != nil {
		logrus.WithContext(ctx).Errorf("error sending a message to connection: %s", err.Error())
	}
}

func (c *component) listenToGameOverSignal(instance *gameInstance) {
	go func() {
		for <-instance.gameOverChan {
			c.gameInstanceStore.Pop(instance.id)
		}
	}()
}
