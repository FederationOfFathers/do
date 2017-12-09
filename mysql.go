package main

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
)

var db *sql.DB
var cfgMysqlURI = os.Getenv("MYSQL")

var (
	ownGame          *sql.Stmt
	createGame       *sql.Stmt
	setMemberMeta    *sql.Stmt
	findNeedXUID     *sql.Stmt
	findNeedGameFill *sql.Stmt
	fillXUID         *sql.Stmt
	fillXUIDCheck    *sql.Stmt
	fillGamesCheck   *sql.Stmt
	getGameInfo      *sql.Stmt
	setGameInfo      *sql.Stmt
	getGameXuidAndID *sql.Stmt
	putGameCheevo    *sql.Stmt
	getGameCheevo    *sql.Stmt
	ownGameCheevo    *sql.Stmt
	findCheevoFill   *sql.Stmt
)

func mustPrepare(query string) *sql.Stmt {
	rval, err := db.Prepare(query)
	if err != nil {
		logger.Fatal("Error preparing query", zap.String("query", query), zap.Error(err))
	}
	return rval
}

func initQueries() {
	findCheevoFill = mustPrepare(strings.Join([]string{
		"SELECT mg.id",
		"FROM `membergames` mg",
		"JOIN games g ON ( mg.game = g.id )",
		"WHERE",
		"`cheevos` < `played`",
		"AND `cheevos_checked` < `played`",
		fmt.Sprintf("AND g.platform IN(%d,%d)", platformXbox360, platformXboxOne),
		"ORDER BY PLAYED ASC LIMIT 1",
	}, " "))
	ownGameCheevo = mustPrepare("INSERT IGNORE INTO member_xbl_cheevos (member,cheevo,unlocked,unlocked_at) VALUES(?,?,1,NOW())")
	getGameCheevo = mustPrepare(
		"SELECT id FROM games_xbl_cheevos WHERE game_id = ? AND game_aid = ? LIMIT 1",
	)
	putGameCheevo = mustPrepare(strings.Join([]string{
		"INSERT INTO games_xbl_cheevos",
		"(game_id,game_aid,name,description,image)",
		"VALUES",
		"(?,?,?,?,?)",
	}, " "))

	getGameXuidAndID = mustPrepare(strings.Join([]string{
		"SELECT",
		"  g.platform_id AS titleID,",
		"  mm.meta_value AS xuid,",
		"  mg.member as memberID",
		"FROM",
		"  membergames mg",
		"  JOIN games g ON ( mg.game = g.id )",
		"  JOIN membermeta mm ON ( mg.member = mm.member_id AND mm.meta_key = 'xuid')",
		"WHERE",
		"  mg.id = ?",
		fmt.Sprintf("  AND g.platform IN(%d,%d)", platformXbox360, platformXboxOne),
		"LIMIT 1",
	}, " "))

	setMemberMeta = mustPrepare(strings.Join([]string{
		"INSERT INTO membermeta (member_ID,meta_key,meta_value) VALUES(?,?,?)",
		"ON DUPLICATE KEY UPDATE meta_value=VALUES(meta_value)",
	}, " "))

	createGame = mustPrepare("INSERT IGNORE INTO games (platform,platform_id,name) VALUES(?,?,?)")
	ownGame = mustPrepare(strings.Join([]string{
		"INSERT INTO membergames (member,game,played)",
		"VALUES(?,?,?)",
		"ON DUPLICATE KEY UPDATE played=VALUES(played)",
	}, " "))

	fillXUID = mustPrepare(`INSERT IGNORE INTO membermeta (member_id,meta_key,meta_value) SELECT id,"xuid","0" FROM members`)
	fillXUIDCheck = mustPrepare(`INSERT IGNORE INTO membermeta (member_id,meta_key,meta_value) SELECT id,"_xuid_last_check","0" FROM members`)
	fillGamesCheck = mustPrepare(`INSERT IGNORE INTO membermeta (member_id,meta_key,meta_value) SELECT id,"_games_last_check","0" FROM members`)
	findNeedXUID = mustPrepare(strings.Join([]string{
		"SELECT",
		"	m.id, m.xbl, m.name, mm.meta_value AS xuid, mmm.meta_value AS xuid_check",
		"FROM",
		"	members m",
		"	LEFT JOIN membermeta mm ON ( m.id = mm.member_ID AND mm.meta_key = 'xuid' )",
		"	LEFT JOIN membermeta mmm ON ( m.id = mmm.member_ID AND mmm.meta_key = '_xuid_last_check' )",
		"WHERE",
		"	xbl NOT IN('','**DISABLED**')",
		"	AND seen > ?",
		"	AND mm.meta_value IN('','0')",
		"	AND ( mmm.meta_value < ? OR mmm.meta_value IS NULL )",
		"ORDER BY",
		"	mmm.meta_value DESC",
		"LIMIT 1",
	}, " "))
	findNeedGameFill = mustPrepare(strings.Join([]string{
		"SELECT",
		"	m.id, m.xbl, m.name, mm.meta_value AS xuid, mmm.meta_value AS lastcheck",
		"FROM",
		"	members m",
		"	LEFT JOIN membermeta mm ON ( m.id = mm.member_ID AND mm.meta_key = 'xuid' )",
		"	LEFT JOIN membermeta mmm ON ( m.id = mmm.member_ID AND mmm.meta_key = '_games_last_check' )",
		"WHERE",
		"	mm.meta_value NOT IN('','**DISABLED**')",
		"	AND seen > ?",
		"HAVING",
		"	mm.meta_value != '0' AND ( mmm.meta_value = '' OR mmm.meta_value < ? OR mmm.meta_value IN('','0',NULL) )",
		"ORDER BY",
		"	m.id ASC",
		"LIMIT 1",
	}, " "))

	getGameInfo = mustPrepare("SELECT id,name,image FROM games WHERE platform = ? AND platform_id = ? LIMIT 1")
	setGameInfo = mustPrepare("UPDATE games SET image = ? WHERE id = ? LIMIT 1")
}

func initExecQueries() {
	fillXUID.Exec()
	fillXUIDCheck.Exec()
	fillGamesCheck.Exec()
}

func initMySQL() {
	m, err := sql.Open("mysql", cfgMysqlURI)
	if err != nil {
		logger.Fatal("Error connecting to MySQL", zap.Error(err))
	}
	if err := m.Ping(); err != nil {
		logger.Fatal("Error pinging MySQL", zap.Error(err))
	}
	db = m
}

func doStmt(s *sql.Stmt, args ...interface{}) (time.Duration, error) {
	start := time.Now()
	_, err := s.Exec(args...)
	return time.Now().Sub(start), err
}
