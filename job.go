package main

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

var errNoHandler = fmt.Errorf("No handler for job")
var errUnsupportedVersion = fmt.Errorf("Error unsupported job version")

type jobEnvelope struct {
	Version int             `json:"v"`
	Raw     json.RawMessage `json:"r"`
}

type job1 struct {
	Name string          `json:"n"`
	Data json.RawMessage `json:"d"`
}

func doJob(data []byte) error {
	var env *jobEnvelope
	err := json.Unmarshal(data, &env)
	if err != nil {
		logger.Error("Error parsing job envelope", zap.ByteString("data", data), zap.Error(err))
		return err
	}
	switch env.Version {
	case 1:
		var job *job1
		if err := json.Unmarshal(env.Raw, &job); err != nil {
			return err
		}
		if handler, ok := handlers[env.Version][job.Name]; ok {
			return handler(job.Data)
		}
		logger.Debug("no handler for job", zap.ByteString("job", data))
		if development {
			return nil
		}
		return errNoHandler
	default:
		return errUnsupportedVersion
	}
}
