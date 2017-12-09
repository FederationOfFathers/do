package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"strings"
	"sync"

	"github.com/FederationOfFathers/xboxapi"
	"go.uber.org/zap"
)

var doCheevoFill = false

//                  gid     aid id
var cheevoMap = map[int]map[int]int{}
var cheevoMapLock sync.RWMutex

func init() {
	flag.BoolVar(&doCheevoFill, "cheevos", doCheevoFill, "dev -- fill achievements")
}

func initCheevoFill() {
	if !development || doCheevoFill {
		logger.Debug("Doing Cheevo Fill")
		handlers[1]["cheevos"] = doFillCheevos
		crontab.AddFunc("@every 10s", cronwrap("queueFillCheevos", queueFillCheevos))
	} else {
		logger.Debug("Skipping Cheevo Fill")
	}
}

func doFillCheevos(job json.RawMessage) error {
	var log = logger.With(zap.String("type", "handler"), zap.String("handler", "doFillCheevos"))
	var fillFor int
	if err := json.Unmarshal(job, &fillFor); err != nil {
		return err
	}
	log.Debug("got game to fill for", zap.Int("game_id", fillFor))

	var xuid string
	var titleID json.Number
	var memberID int

	row := getGameXuidAndID.QueryRow(fillFor)
	if err := row.Scan(&titleID, &xuid, &memberID); err != nil {
		if err != sql.ErrNoRows {
			log.Error("Unable to find titleID and xuid", zap.Int("GameID", fillFor), zap.Error(err))
			return err
		}
		log.Info("No rows found for titleID and xuid", zap.Int("GameID", fillFor))
		return nil
	}

	titleID64, err := titleID.Int64()
	if err != nil {
		log.Error("unable to convert titleID to int64", zap.String("titleID", titleID.String()), zap.Error(err))
		return err
	}

	list, err := xbl.Achievements(xuid, int(titleID64))
	if err != nil {
		log.Error("unable to get cheevo list", zap.String("xuid", xuid), zap.String("titleID", titleID.String()))
		return err
	}

	log.Debug("found", zap.Int("GameID", fillFor), zap.String("xuid", xuid), zap.String("titleID", titleID.String()))

	for _, entry := range list {
		aid, err := cheevo(entry)
		if err != nil {
			log.Error("Error fetching cheevo id from db", zap.Error(err), zap.String("image", entry.Image))
			return err
		}
		if !entry.Unlocked {
			continue
		}
		if _, err := ownGameCheevo.Exec(memberID, aid); err != nil {
			log.Error("Error owning achievement", zap.Int("memberID", memberID), zap.Int("aid", aid))
			return err
		}
	}
	if _, err := db.Exec("UPDATE membergames SET cheevos = NOW() WHERE id = ? limit 1", fillFor); err != nil {
		log.Error("error updating cheevos", zap.Error(err))
	}
	return nil
}

func queueFillCheevos(cronID int, name string) {
	var log = logger.With(zap.String("type", "cron"), zap.Int("id", cronID), zap.String("name", name))
	var fillFor int
	row := findCheevoFill.QueryRow()
	if err := row.Scan(&fillFor); err != nil {
		if err == sql.ErrNoRows {
			log.Debug("No cheevos to fill")
			return
		}
		log.Error("Error querying for cheevos to fill", zap.Error(err))
		return
	}
	log.Debug("found game to fill for", zap.Int("game_id", fillFor))
	enqueuev1("cheevos", fillFor)
	log.Info("queued", zap.Int("game_id", fillFor))
	if _, err := db.Exec("UPDATE membergames SET cheevos_checked = NOW() WHERE id = ? limit 1", fillFor); err != nil {
		log.Error("error updating cheevos_checked", zap.Error(err))
	}
}

func cheevo(a *xboxapi.Achievement) (int, error) {
	cheevoMapLock.RLock()
	// Thread cache
	if _, ok := cheevoMap[a.TitleID]; ok {
		if id, ok := cheevoMap[a.TitleID][a.ID]; ok {
			cheevoMapLock.RUnlock()
			return id, nil
		}
		cheevoMapLock.RUnlock()
	} else {
		cheevoMapLock.RUnlock()
		cheevoMapLock.Lock()
		cheevoMap[a.TitleID] = map[int]int{}
		cheevoMapLock.Unlock()
	}

	// Check the table
	var aid int
	row := getGameCheevo.QueryRow(a.TitleID, a.ID)
	err := row.Scan(&aid)
	if err == nil {
		cheevoMapLock.Lock()
		cheevoMap[a.TitleID][a.ID] = aid
		cheevoMapLock.Unlock()
		return aid, nil
	}
	if err != sql.ErrNoRows {
		return 0, err
	}

	// Ok, Fine, Insert...
	cheevoMapLock.Lock()
	res, err := putGameCheevo.Exec(a.TitleID, a.ID, strings.TrimSpace(a.Name), strings.TrimSpace(a.Description)) // , strings.TrimSpace(a.Image))
	cheevoMapLock.Unlock()
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return int(id), err
	}
	logger.Debug("found new cheevo", zap.Int("id", a.ID), zap.String("name", a.Name), zap.Int64("ourID", id))
	cheevoMapLock.Lock()
	cheevoMap[a.TitleID][a.ID] = int(id)
	cheevoMapLock.Unlock()
	return int(id), err
}
