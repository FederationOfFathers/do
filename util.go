package main

import (
	"time"
)

func agoBytes(ago time.Duration) []byte {
	return timeBytes(time.Now().Add(0 - ago))
}

func timeBytes(t time.Time) []byte {
	b, _ := t.MarshalJSON()
	return b
}

func agoTs(ago time.Duration) int64 {
	return time.Now().Add(0 - ago).Unix()
}

func timeBuf() []byte {
	buf, _ := time.Now().MarshalJSON()
	return buf
}
