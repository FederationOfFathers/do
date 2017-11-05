package main

import (
	"encoding/json"
)

func enqueuev1(name string, data interface{}) error {
	rawData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(job1{
		Name: name,
		Data: rawData,
	})
	if err != nil {
		return err
	}
	job, err := json.Marshal(jobEnvelope{
		Version: 1,
		Raw:     raw,
	})
	if err != nil {
		return err
	}
	return nsqP.Publish(nsqTopic, job)
}
