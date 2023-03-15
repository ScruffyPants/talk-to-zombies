package app

import (
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func ParseFlags() error {
	pflag.Duration("zombie.interval", 2*time.Second, "Zombie coordinate update interval")
	pflag.String("address", ":8082", "HTTP server address")
	pflag.Duration("ws.pinginterval", 10*time.Second, "Ping interval for websocket connections")
	pflag.Duration("ws.pongwait", 20*time.Second, "Pong wait for websocket connections")
	pflag.Duration("ws.writewait", 20*time.Second, "Write wait for websocket connections")
	pflag.String("config", "config.local", "Name of the config file")

	configName, err := pflag.CommandLine.GetString("config")
	if err != nil {
		logrus.Errorf(err.Error())
	}

	pflag.Parse()

	viper.SetConfigName(configName)
	viper.AddConfigPath("./")
	if err = viper.ReadInConfig(); err != nil {
		logrus.Warnf("Failed to load configs: %s", err.Error())
	} else {
		logrus.Infof("Loaded configs from '%s'", viper.ConfigFileUsed())
	}

	// This option enables ENV vars to be set as flag values
	// f.x. DETECT_URL ==> --detect.url
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err = viper.BindPFlags(pflag.CommandLine); err != nil {
		return err
	}

	return nil
}
