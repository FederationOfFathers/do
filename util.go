package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/gommon/log"
)

func cdnImage(url, cdnID string) error {
	putURL := fmt.Sprintf("http://dashboard.fofgaming.com/api/v0/cdn/%s", cdnID)
	req, err := http.NewRequest("PUT", putURL, strings.NewReader(url))
	if err != nil {
		return err
	}
	req.Header.Set("Access-Key", cdnPutKey)
	rsp, err := http.DefaultClient.Do(req)
	if rsp != nil && rsp.Body != nil {
		defer rsp.Body.Close()
	}
	if err != nil {
		return err
	}
	if rsp.StatusCode != 200 {
		return fmt.Errorf("Error putting image to CDN http %s", rsp.Status)
	}
	log.Debug(putURL)
	return nil
}

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
