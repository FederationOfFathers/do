package main

import (
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type memberXboxInfo struct {
	ID        int    `json:"id"`
	XBL       string `json:"gt"`
	Name      string `json:"un"`
	XUID      string `json:"x"`
	LastCheck string `json:"l"`
}

func init() {
	handlers[1]["emptyXUID"] = doFillEmptyXUID
	crontab.AddFunc("@every 60s", cronwrap("queueFillEmptyXUID", queueFillEmptyXUID))
	crontab.AddFunc("@every 3600s", cronwrap("doPopulateMissingXUIDMeta", doPopulateMissingXUIDMeta))
}

func doPopulateMissingXUIDMeta(cronID int, name string) {
	fillXUID.Exec()
	fillXUIDCheck.Exec()
}

func doFillEmptyXUID(job json.RawMessage) error {
	var log = logger.With(zap.String("type", "handler"), zap.String("handler", "doFillEmptyXUID"))
	var user *memberXboxInfo
	if err := json.Unmarshal(job, &user); err != nil {
		log.Error("Error unmarshalling", zap.Error(err), zap.ByteString("data", job))
		return err
	}
	timeBuf, _ := time.Now().MarshalJSON()
	if _, err := setMemberMeta.Exec(user.ID, "_xuid_last_check", timeBuf); err != nil {
		log.Error("Errorsetting _xuid_last_check", zap.Error(err))
		return err
	}
	xuidInt, err := xbl.XUID(user.XBL)
	if err != nil {
		log.Error("Error checking xuid", zap.String("username", user.Name), zap.Int("userid", user.ID), zap.Error(err))
		return err
	}
	xuid := strconv.Itoa(xuidInt)
	if _, err := setMemberMeta.Exec(user.ID, "xuid", xuid); err != nil {
		log.Error("Error setting xuid", zap.String("username", user.Name), zap.Int("userid", user.ID), zap.Error(err))
		return err
	}
	log.Info("Set xuid", zap.String("username", user.Name), zap.Int("userid", user.ID), zap.String("xuid", xuid))
	return nil
}

func queueFillEmptyXUID(cronID int, name string) {
	var log = logger.With(zap.String("type", "cron"), zap.Int("id", cronID), zap.String("name", name))
	var data = memberXboxInfo{}
	row := findNeedXUID.QueryRow(agoTs(month), agoBytes(day))
	if row == nil {
		log.Debug("no users requiring xuid check")
		return
	}
	if err := row.Scan(&data.ID, &data.XBL, &data.Name, &data.XUID, &data.LastCheck); err != nil {
		if err == sql.ErrNoRows {
			log.Debug("no users requiring xuid check")
		} else {
			log.Error("Error scanning row", zap.Error(err))
		}
		return
	}
	enqueuev1("emptyXUID", data)
	log.Info("queued", zap.String("username", data.Name), zap.Int("userid", data.ID))
}
