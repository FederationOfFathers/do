package main

import (
	"flag"
	"time"
)

const day = time.Hour * 24
const month = day * 30
const week = day * 7
const year = day * 365

func main() {
	flag.Parse()
	initLogger()
	initDiskCache()
	initMySQL()
	initState()
	initQueries()
	initExecQueries()
	initProducer()
	initConsumer()
	initCrontab()
	initUserFill()
	initGameFill()
	initCheevoFill()
	awaitSignal()
}
