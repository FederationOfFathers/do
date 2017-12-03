package main

import (
	"database/sql"
	"encoding/json"
	"strings"
)

var dbStateSet *sql.Stmt
var dbStateGet *sql.Stmt

func initState() {
	dbStateSet = mustPrepare("INSERT INTO `do_state` (`k`,`v`) VALUES(?,?) ON DUPLICATE KEY UPDATE `v`=VALUES(`v`)")
	dbStateGet = mustPrepare("SELECT `v` FROM `do_state` WHERE `k` = ? LIMIT 1")
}

func stateSet(key string, value interface{}) error {
	buf, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = dbStateSet.Exec(key, buf)
	return err
}

func getState(key string, value interface{}) error {
	var buf string
	row := dbStateGet.QueryRow(key)
	err := row.Scan(&buf)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}
	return json.NewDecoder(strings.NewReader(buf)).Decode(&value)
}
