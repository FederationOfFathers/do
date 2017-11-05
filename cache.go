package main

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/peterbourgon/diskv"
	"go.uber.org/zap"
)

var cachePath = "./disk-cache"
var cache *diskv.Diskv

func init() {
	flag.StringVar(&cachePath, "cache", cachePath, "path to the directory to use as a disk cache for the xboxapi calls")
}

func cacheGet(key string, into interface{}) (bool, error) {
	log := logger.With(zap.String("module", "cache"), zap.String("op", "get"), zap.String("key", key))
	buf, err := cache.Read(key)
	if err != nil {
		return false, err
	}
	expiry, _ := binary.Varint(buf[:binary.MaxVarintLen64])
	if expiry <= time.Now().Unix() {
		return false, nil
	}
	err = json.Unmarshal(buf[binary.MaxVarintLen64:], &into)
	if err != nil {
		log.Error("error unmarshalling", zap.Error(err))
	}
	return true, err
}

func cacheSet(key string, ttl time.Duration, from interface{}) error {
	log := logger.With(zap.String("module", "cache"), zap.String("op", "set"), zap.String("key", key), zap.Duration("ttl", ttl))
	var expiry = make([]byte, binary.MaxVarintLen64)
	binary.PutVarint(expiry, time.Now().Add(ttl).Unix())
	buf, err := json.Marshal(from)
	if err != nil {
		log.Error("error marshalling", zap.Error(err))
		return err
	}
	err = cache.Write(key, append(expiry, buf...))
	if err != nil {
		log.Error("error setting cache", zap.Error(err))
	}
	return err
}

func initDiskCache() {
	cache = diskv.New(diskv.Options{
		BasePath:     cachePath,
		CacheSizeMax: 1024 * 1024,
		Transform: func(s string) []string {
			h := md5.New()
			fmt.Fprintf(h, s)
			b := h.Sum(nil)
			return strings.Split(fmt.Sprintf("%0x/%0x/%0x", b[0:1], b[1:2], b[2:3]), "/")
		},
	})
}
