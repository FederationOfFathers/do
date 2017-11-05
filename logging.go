package main

import (
	"flag"

	"go.uber.org/zap"
)

var development = false

var logger *zap.Logger

func initLogger() {
	if development {
		logger, _ = zap.NewDevelopment()
	} else {
		logger, _ = zap.NewProduction()
	}

}

func init() {
	flag.BoolVar(&development, "dev", development, "enable development mode (mainly logging changes)")
}
