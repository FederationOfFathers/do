package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/FederationOfFathers/xboxapi"
)

var cfgXboxAPI = os.Getenv("XBOXAPI")
var xbl *xboxapi.Client

func init() {
	xbl = xboxapi.New(&xboxapi.Config{
		APIKey:   cfgXboxAPI,
		Language: "en-US",
	})
}

func getXboxTitleByInt(id int) (*xboxapi.Title, error) {
	var rval *xboxapi.Title
	cacheKey := fmt.Sprintf("xbox-title-by-int-%d", id)
	ok, err := cacheGet(cacheKey, &rval)
	if ok && err == nil {
		return rval, err
	}
	hex := fmt.Sprintf("%x", id)
	rval, err = xbl.GameDetailsHex(hex)
	cacheSet(cacheKey, time.Hour*24*30, rval)
	return rval, err
}

func getXboxTitleByString(id string) (*xboxapi.Title, error) {
	intID, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}
	return getXboxTitleByInt(intID)
}
