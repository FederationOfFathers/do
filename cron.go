package main

import (
	"sync"
	"time"

	"github.com/robfig/cron"
	"go.uber.org/zap"
)

var activeCronJobs = sync.WaitGroup{}
var crontab = cron.New()
var lockCron = sync.Mutex{}
var cronID = 0

func initCrontab() {
	crontab.Start()
}

func cronwrap(name string, job func(int, string)) func() {
	return func() {
		activeCronJobs.Add(1)
		defer activeCronJobs.Done()
		lockCron.Lock()
		cronID++
		var id = cronID
		lockCron.Unlock()
		logger.Debug("starting cron job", zap.Int("id", id), zap.String("name", name))
		start := time.Now()
		job(cronID, name)
		logger.Debug("finished cron job", zap.Int("id", id), zap.String("name", name), zap.Duration("took", time.Now().Sub(start)))
	}
}
