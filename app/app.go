package app

import (
	"github.com/spf13/viper"

	"github.com/ScruffyPants/talk-to-zombies/api"
	"github.com/ScruffyPants/talk-to-zombies/communication"
	"github.com/ScruffyPants/talk-to-zombies/game"
	"github.com/ScruffyPants/talk-to-zombies/player"
)

type app struct {
	communicationService communication.Service
	gameComponent        game.Component
	playerComponent      player.Component
	httpRouter           *api.Router
}

func NewApp() (*app, error) {
	if err := ParseFlags(); err != nil {
		return nil, err
	}

	communicationService := communication.NewCommunicationService()

	playerComponent := player.NewPlayerComponent()

	gameSettings := game.Settings{
		ZombieCoordinateUpdateInterval: viper.GetDuration("zombie.interval"),
	}
	gameComponent, err := game.NewGameComponent(playerComponent, communicationService, gameSettings)
	if err != nil {
		return nil, err
	}

	communicationService.AddListener(gameComponent)

	httpRouter := api.NewRouter(
		api.RouterSettings{
			Address:      viper.GetString("address"),
			PingInterval: viper.GetDuration("ws.pinginterval"),
			PongWait:     viper.GetDuration("ws.pongwait"),
			WriteWait:    viper.GetDuration("ws.writewait"),
		},
		communicationService,
	)

	return &app{
		communicationService: communicationService,
		gameComponent:        gameComponent,
		playerComponent:      playerComponent,
		httpRouter:           httpRouter,
	}, nil
}

func (a *app) Start() {
	a.httpRouter.Start()
}
