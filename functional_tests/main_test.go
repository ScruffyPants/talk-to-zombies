package functional_tests

import (
	"github.com/ScruffyPants/talk-to-zombies/app"
	"github.com/spf13/viper"
	"testing"
)

const testWSAddress = ":8082"

func TestMain(m *testing.M) {
	viper.Set("ws.address", testWSAddress)

	service, err := app.NewApp()
	if err != nil {
		panic(err)
	}
	service.Start()

	m.Run()
}
