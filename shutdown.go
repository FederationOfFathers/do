package main

import (
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func awaitSignal() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	sig := <-sigc
	logger.Info("Caught Signal", zap.String("signal", sig.String()))
	logger.Info("Stopping Consumer")
	nsqC.Stop()
	logger.Info("Stopping crontab")
	crontab.Stop()
	logger.Info("Waiting for cron jobs to flush")
	activeCronJobs.Wait()
	logger.Info("Waiting for consumer to flush")
	<-nsqC.StopChan
	logger.Info("Everything stopped and accounted for. Shutting down")
}
