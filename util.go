package main

import (
	"bytes"
	"time"
)

func agoBytes(ago time.Duration) []byte {
	return timeBytes(time.Now().Add(0 - ago))
}

func timeBytes(t time.Time) []byte {
	buf, _ := t.MarshalJSON()
	return bytes.Replace(buf, []byte("\""), []byte(""), -1)
}

func agoTs(ago time.Duration) int64 {
	return time.Now().Add(0 - ago).Unix()
}

func timeBuf() []byte {
	buf, _ := time.Now().MarshalJSON()
	return bytes.Replace(buf, []byte("\""), []byte(""), -1)
}
