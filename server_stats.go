package main

import "time"

type Player struct {
	Player_ID  string    `db:"player_id"`
	Username   string    `db:"username"`
	Created_at time.Time `db:"created_at"`
	Updated_at time.Time `db:"updated_at"`
}

type Match struct {
	Match_ID   string    `db:"match_id"`
	Start_time time.Time `db:"start_time"`
	End_time   time.Time `db:"end_time"`
	Winner_ID  string    `db:"winner_id"`
}

type Round struct {
	Round_ID  string    `db:"round_id"`
	Match_ID  string    `db:"match_id"`
	Winner_ID string    `db:"winner_id"`
	Level     LevelEnum `db:"level"`
}

type KillEvent struct {
	Kill_ID   string    `db:"kill_id"`
	Round_ID  string    `db:"match_id"`
	Killer_ID string    `db:"killer_id"`
	Victim_ID string    `db:"victim_id"`
	Timestamp time.Time `db:"time_stamp"`
}
