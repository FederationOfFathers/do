package main

var gameIDs = map[int]map[string]int{}

func mysqlFindGame(platform int, titleID string) (int, error) {
	var id int
	row := db.QueryRow("SELECT id FROM games where platform = ? AND platform_id = ? LIMIT 1", platform, titleID)
	err := row.Scan(&id)
	return id, err
}

func mysqlCreateGame(platform int, titleID, titleName string) (int, error) {
	if _, ok := gameIDs[platform]; ok {
		if id, ok := gameIDs[platform][titleID]; ok {
			return id, nil
		}
	} else {
		gameIDs[platform] = map[string]int{}
	}

	if id, err := mysqlFindGame(platform, titleID); err != nil && id > 0 {
		gameIDs[platform][titleID] = id
		return id, nil
	}

	res, err := createGame.Exec(platform, titleID, titleName)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	if id > 0 {
		gameIDs[platform][titleID] = int(id)
		return gameIDs[platform][titleID], nil
	}

	if id, err := mysqlFindGame(platform, titleID); id > 0 {
		gameIDs[platform][titleID] = id
		return id, err
	} else {
		return id, err
	}
}
