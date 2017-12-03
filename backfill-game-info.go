package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func init() {
	// handlers[1]["doBackfillGameImage"] = doBackfillGameImage
	// crontab.AddFunc("@every 60s", cronwrap("queueFillGameImages", queueFillGameImages))
}

func doBackfillGameImage(raw json.RawMessage) error {
	logger.Debug("starting", zap.String("function", "doBackfillGameImage"), zap.ByteString("raw", raw))
	var id int
	if err := json.Unmarshal(raw, &id); err != nil {
		logger.Error("unmarshalling", zap.String("function", "doBackfillGameImage"), zap.ByteString("raw", raw), zap.Error(err))
		return err
	}
	var game = &struct {
		id         int
		name       string
		platform   int
		platformID int
		image      string
		hex        string
	}{
		id: id,
	}

	row := db.QueryRow("SELECT name,platform,platform_id,image FROM games WHERE id = ? LIMIT 1", id)
	if err := row.Scan(&game.name, &game.platform, &game.platformID, &game.image); err != nil {
		logger.Error("querying", zap.Error(err))
		return err
	}
	game.hex = fmt.Sprintf("%x", game.platformID)
	var log = logger.With(
		zap.String("function", "doBackfillGameImage"),
		zap.Int("platform", game.platform),
		zap.Int("id", game.id),
		zap.String("name", game.name),
		zap.String("hex", game.hex))
	if game.image != "" {
		return nil
	}
	title, err := xbl.GameDetailsHex(game.hex)
	if err != nil {
		log.Error("fetching title", zap.Error(err))
		return err
	}
	var images = map[string]struct {
		url   string
		width int
	}{}
	var image string
	for _, item := range title.Items {
		for _, img := range item.Images {
			if v, ok := images[img.Purpose]; !ok || v.width < img.Width {
				images[img.Purpose] = struct {
					url   string
					width int
				}{
					url:   img.URL,
					width: img.Width,
				}
			}
		}
	}
	for _, preference := range []string{"BoxArt", "BrandedKeyArt", "Poster"} {
		if img, ok := images[preference]; ok {
			image = img.url
			break
		}
	}
	if image == "" {
		log.Error("no appropriate image found")
		return nil
	}
	imageKey := fmt.Sprintf("game-image-%d", game.id)
	putURL := fmt.Sprintf("http://dashboard.fofgaming.com/api/v0/cdn/%s", imageKey)
	req, err := http.NewRequest("PUT", putURL, strings.NewReader(image))
	if err != nil {
		log.Debug("error forming put request", zap.Error(err))
		return err
	}
	req.Header.Set("Access-Key", cdnPutKey)
	rsp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Debug("error fetching", zap.Error(err))
		return err
	}
	if rsp.StatusCode != 200 {
		log.Debug("error fetching", zap.Int("statusCode", rsp.StatusCode), zap.String("status", rsp.Status))
		return fmt.Errorf(rsp.Status)
	}
	if _, err := setGameInfo.Exec(imageKey, id); err != nil {
		log.Debug("error updating", zap.Error(err))
		return err
	}
	log.Debug(putURL)
	return nil
}

func queueFillGameImages(cronID int, name string) {
	var id int
	var log = logger.With(zap.String("type", "cron"), zap.Int("id", cronID), zap.String("name", name))
	row := db.QueryRow("SELECT id FROM games WHERE image = ''")
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			log.Debug("no games need images filled")
		} else {
			log.Error("error scanning row", zap.Error(err))
		}
		return
	}
	enqueuev1("doBackfillGameImage", id)
	logger.Info("queued", zap.Int("id", id))
}
