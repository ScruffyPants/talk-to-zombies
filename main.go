package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"

	"github.com/ScruffyPants/talk-to-zombies/app"
)

func main() {
	service, err := app.NewApp()
	if err != nil {
		panic(err)
	}

	service.Start()

	quit := make(chan os.Signal, 1)
	listenForExit(quit)

	os.Exit(0)
}

func listenForExit(quit chan os.Signal) {
	signal.Notify(quit, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	interrupt := <-quit
	logrus.Infof("OS Interrupt: (%s) - Gracefully shutting down", interrupt.String())

	return
}
