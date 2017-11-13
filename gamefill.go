package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/FederationOfFathers/xboxapi"
	"go.uber.org/zap"
	gomail "gopkg.in/gomail.v2"
)

const (
	platformUnknown = iota
	platformXbox360
	platformXboxOne
	platformWindows
	platformIOS
	platformAndroid
	platformMobile
	platformGearVR
	platformKindle
)

var cdnPutKey = os.Getenv("CDN_PUT_KEY")

func init() {
	handlers[1]["checkGames"] = doCheckGames
	crontab.AddFunc("@every 8s", cronwrap("queueFillGames", queueFillGames))
	crontab.AddFunc("@every 3600s", cronwrap("doPopulateMissingGamesMeta", doPopulateMissingGamesMeta))
}

func doPopulateMissingGamesMeta(cronID int, name string) {
	fillGamesCheck.Exec()
}

func devicesToPlatform(devices []string) int {
	var is = map[string]bool{}
	for _, device := range devices {
		is[device] = true
	}
	if is, _ := is["XboxOne"]; is {
		return platformXboxOne
	}
	if is, _ := is["Xbox360"]; is {
		return platformXbox360
	}
	if is, _ := is["Win32"]; is {
		return platformWindows
	}
	if is, _ := is["PC"]; is {
		return platformWindows
	}
	if is, _ := is["iOS"]; is {
		return platformIOS
	}
	if is, _ := is["Android"]; is {
		return platformAndroid
	}
	if is, _ := is["Mobile"]; is {
		return platformMobile
	}
	if is, _ := is["Gear VR"]; is {
		return platformGearVR
	}
	if is, _ := is["Kindle"]; is {
		return platformKindle
	}
	if is, _ := is["Kindle Fire"]; is {
		return platformKindle
	}
	return platformUnknown
}

func resolveConsole(title *xboxapi.TilehubTitle, log *zap.Logger) (int, error) {
	if kind := devicesToPlatform(title.Devices); kind != platformUnknown {
		return kind, nil
	}
	details, err := getXboxTitleByString(title.TitleID)
	if err != nil {
		return platformUnknown, err
	}
	if kind := devicesToPlatform(details.Devices()); kind != platformUnknown {
		return kind, nil
	}
	if len(details.Items) > 0 {
		if details.Items[0].MediaGroup == "GameType" {
			if details.Items[0].MediaItemType == "DGame" || details.Items[0].MediaItemType == "DGameDemo" {
				return platformXboxOne, nil
			}
		}
	}
	return platformUnknown, nil
}

func doFillGame(title *xboxapi.TilehubTitle, platform int) error {
	var log = logger.With(
		zap.String("function", "resolveConsole"),
		zap.Int("platform", platform),
		zap.String("title", title.Name),
		zap.String("id", title.TitleID))
	log.Debug("filling")
	var gameID int
	var gameName string
	var gameImage string
	row := getGameInfo.QueryRow(platform, title.TitleID)
	if err := row.Scan(&gameID, &gameName, &gameImage); err != nil {
		log.Error("error scanning", zap.Error(err))
		return err
	}
	if gameImage == "" && title.DisplayImage != "" {
		imageKey := fmt.Sprintf("game-image-%d", gameID)
		putURL := fmt.Sprintf("http://dashboard.fofgaming.com/api/v0/cdn/%s", imageKey)
		req, err := http.NewRequest("PUT", putURL, strings.NewReader(title.DisplayImage))
		if err != nil {
			log.Error("error forming put request", zap.Error(err))
			return err
		}
		req.Header.Set("Access-Key", cdnPutKey)
		rsp, err := http.DefaultClient.Do(req)
		if rsp != nil && rsp.Body != nil {
			defer rsp.Body.Close()
		}
		if err != nil {
			log.Error("error fetching", zap.Error(err))
			return err
		}
		if rsp.StatusCode != 200 {
			log.Error("error fetching", zap.Int("statusCode", rsp.StatusCode), zap.String("status", rsp.Status))
			return fmt.Errorf(rsp.Status)
		}
		if _, err := setGameInfo.Exec(imageKey, gameID); err != nil {
			log.Error("error updating", zap.Error(err))
			return err
		}
		log.Debug(putURL)
	}
	return nil
}

func doCheckGames(job json.RawMessage) error {
	var start = time.Now()
	var log = logger.With(zap.String("type", "handler"), zap.String("handler", "doCheckGames"))
	var user *memberXboxInfo
	if err := json.Unmarshal(job, &user); err != nil {
		log.Error("Error unmarshalling", zap.Error(err))
		return err
	}
	if _, err := doStmt(setMemberMeta, user.ID, "_games_last_check", timeBuf()); err != nil {
		log.Error("Error setting _games_last_check", zap.Error(err))
		return err
	}
	log = log.With(zap.String("member", user.Name), zap.String("xuid", user.XUID))
	apiRes, err := xbl.TileHub(user.XUID)
	if err != nil {
		log.Error("Error checking TileHub", zap.String("username", user.Name), zap.String("xuid", user.XUID), zap.Error(err))
		return err
	}
	var examined = 0
	var new = 0
	var added = 0

	if len(apiRes.Titles) < 1 {
		log.Error("no titles returned for user titlehub-achievement-list", zap.String("username", user.Name), zap.String("xuid", user.XUID))
	}

	for _, title := range apiRes.Titles {
		examined++
		resolved, err := resolveConsole(title, log)
		if err != nil {
			log.Error("error resolving", zap.String("id", title.TitleID), zap.Error(err))
			continue
		}
		if resolved == platformUnknown {
			m := gomail.NewMessage()
			m.SetHeader("From", "do@fofgaming.com")
			m.SetHeader("To", "apokalyptik@apokalyptik.com")
			m.SetHeader("Subject", "Unexpected game device")
			buf, _ := json.MarshalIndent(map[string]interface{}{
				"user":  user,
				"title": title,
			}, "", "\t")
			m.SetBody("text/plain", string(buf))
			d := gomail.Dialer{Host: "localhost", Port: 587}
			if err := d.DialAndSend(m); err != nil {
				log.Error("Error sending email notice...")
			}
			log.Error("Unexpected game device", zap.String("title", title.Name), zap.String("id", title.TitleID))
			continue
		}
		gameID, err := mysqlCreateGame(resolved, title.TitleID, title.Name)
		if err != nil {
			log.Error("error creating game", zap.String("title", title.Name), zap.String("id", title.TitleID), zap.Error(err))
			return err
		}
		res, err := ownGame.Exec(user.ID, gameID, title.TitleHistory.LastTimePlayed.Time())
		if err != nil {
			log.Info(string(title.TitleHistory.LastTimePlayed), zap.Time("parsed", title.TitleHistory.LastTimePlayed.Time()))
			log.Error("error owning game", zap.String("title", title.Name), zap.String("id", title.TitleID), zap.Error(err))
			return err
		}
		if id, err := res.LastInsertId(); err != nil && id > 0 {
			log.Debug(
				"owning",
				zap.String("title", title.Name),
				zap.String("platform_id", title.TitleID),
				zap.Int("platform", resolved),
				zap.Int("local_id", gameID),
				zap.Int64("relationship", id),
			)
			added++
		} else {
			log.Debug(
				"updating",
				zap.String("title", title.Name),
				zap.String("platform_id", title.TitleID),
				zap.Int("platform", resolved),
				zap.Int("local_id", gameID),
			)
		}
	}
	log.Info("run complete", zap.Int("games-created", new), zap.Int("added", added), zap.Int("examined", examined), zap.Duration("took", time.Now().Sub(start)))
	return nil
}

func queueFillGames(cronID int, name string) {
	var log = logger.With(zap.String("type", "cron"), zap.Int("id", cronID), zap.String("name", name))
	var data = memberXboxInfo{}
	log.Debug("findNeedGameFill", zap.Int64("seen", agoTs(month)), zap.ByteString("_games_last_check", agoBytes(time.Hour)))
	row := findNeedGameFill.QueryRow(agoTs(month), agoBytes(time.Hour))
	if row == nil {
		log.Debug("no users need games filled")
		return
	}
	if err := row.Scan(&data.ID, &data.XBL, &data.Name, &data.XUID, &data.LastCheck); err != nil {
		if err == sql.ErrNoRows {
			log.Debug("no users need games filled")
		} else {
			log.Error("error scanning row", zap.Error(err))
		}
		return
	}
	if _, err := doStmt(setMemberMeta, data.ID, "_games_last_check", timeBuf()); err != nil {
		log.Error("Error setting _games_last_check", zap.Error(err))
		return
	}
	enqueuev1("checkGames", data)
	logger.Info("queued", zap.String("username", data.Name), zap.String("xuid", data.XUID), zap.Int("userid", data.ID))
}
