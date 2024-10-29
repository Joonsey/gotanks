package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/tursodatabase/go-libsql"
)

type Player struct {
	Player_ID  string    `db:"player_id"`
	Username   string    `db:"username"`
	Created_at time.Time `db:"created_at"`
	Updated_at time.Time `db:"updated_at"`
}

type Match struct {
	Match_ID   string         `db:"match_id"`
	Start_time time.Time      `db:"start_time"`
	End_time   time.Time      `db:"end_time"`
	Winner_ID  sql.NullString `db:"winner_id"`
}

type Round struct {
	Round_ID  string         `db:"round_id"`
	Match_ID  string         `db:"match_id"`
	Winner_ID sql.NullString `db:"winner_id"`
	Level     LevelEnum      `db:"level"`
}

type KillEvent struct {
	Kill_ID   string    `db:"kill_id"`
	Round_ID  string    `db:"match_id"`
	Killer_ID string    `db:"killer_id"`
	Victim_ID string    `db:"victim_id"`
	Timestamp time.Time `db:"time_stamp"`
}

type ServerStats struct {
	Matches    []*Match
	KillEvents []*KillEvent
	Rounds     []*Round
}

type ServerSyncManager struct {
	stats     ServerStats
	connector *libsql.Connector

	temp_dir  string
	db_handle *sql.DB
}

func InitStatsManager() *ServerSyncManager {
	sm := ServerSyncManager{}
	err := godotenv.Load(".env")
	if err != nil {
		log.Panicf("Error loading .env file: %s", err)
	}

	db_url := os.Getenv("TURSO_DATABASE_URL")
	auth_token := os.Getenv("TURSO_AUTH_TOKEN")
	dir, err := os.MkdirTemp("", "libsql-*")
	if err != nil {
		log.Panicf("Error creating temporary directory:", err)
	}

	sm.temp_dir = dir
	db_path := filepath.Join(dir, "local-server.save")

	connector, err := libsql.NewEmbeddedReplicaConnector(db_path, db_url,
		libsql.WithAuthToken(auth_token),
		libsql.WithSyncInterval(time.Minute),
	)
	if err != nil {
		fmt.Println("Error creating connector:", err)
		os.Exit(1)
	}

	sm.connector = connector
	sm.db_handle = sql.OpenDB(sm.connector)
	return &sm
}

func (sm *ServerSyncManager) DeInit() {
	log.Println("succesfully de-initing the sync manager")
	sm.connector.Close()
	sm.db_handle.Close()
	os.RemoveAll(sm.temp_dir)
}

func NewPlayer(addr string) Player {
	p := Player{}
	p.Player_ID = addr
	p.Created_at = time.Now()
	p.Updated_at = time.Now()
	p.Username = "test_user"

	return p
}

func NewKillEvent(round_id, victim_id, killer_id string) KillEvent {
	k := KillEvent{}
	k.Kill_ID = uuid.NewString()
	k.Killer_ID = killer_id
	k.Round_ID = round_id
	k.Victim_ID = victim_id
	k.Timestamp = time.Now()
	return k
}

// TODO ALL SQL EXEC
// should be goroutines

// TODO do not forget. Please... :(

func NewMatch(sm *ServerSyncManager) Match {
	m := Match{}
	m.Match_ID = uuid.NewString()
	m.Start_time = time.Now()
	_, err := sm.db_handle.Exec("INSERT OR REPLACE INTO matches (match_id, start_time, end_time, winner_id) VALUES (?, ?, ?, ?)",
		m.Match_ID,
		m.Start_time.Format(time.RFC3339),
		sql.NullString{},
		m.Winner_ID)
	if err != nil {
		log.Panic(err)
	}
	return m
}

func NewRound(m Match, level LevelEnum, sm *ServerSyncManager) Round {
	r := Round{}
	r.Round_ID = uuid.NewString()
	r.Match_ID = m.Match_ID
	r.Level = level
	_, err := sm.db_handle.Exec("INSERT OR REPLACE INTO rounds (round_id, match_id, winner_id, level) VALUES (?, ?, ?, ?)",
		r.Round_ID,
		r.Match_ID,
		sql.NullString{},
		r.Level,
	)
	if err != nil {
		log.Panic(err)
	}

	return r
}

func (k *KillEvent) Sync(sm *ServerSyncManager) {
	log.Println(k)
	_, err := sm.db_handle.Exec("INSERT OR REPLACE INTO kill_events (kill_id, time_stamp, round_id, killer_id, victim_id) VALUES (?, ?, ?, ?, ?)",
		k.Kill_ID,
		k.Timestamp.Format(time.RFC3339),
		k.Round_ID,
		k.Killer_ID,
		k.Victim_ID,
	)
	if err != nil {
		log.Panic(err)
	}

}

func (p *Player) Update(sm *ServerSyncManager) {
	p.Updated_at = time.Now()

	_, err := sm.db_handle.Exec("INSERT OR REPLACE INTO players (player_id, created_at, updated_at, username) VALUES (?, ?, ?, ?)",
		p.Player_ID,
		p.Created_at.Format(time.RFC3339),
		p.Updated_at.Format(time.RFC3339),
		p.Username,
	)
	if err != nil {
		log.Panic(err)
	}
}

func (r *Round) CompleteRound(sm *ServerSyncManager) {
	if !r.Winner_ID.Valid {
		log.Panic("winner id can not be null")
	}

	_, err := sm.db_handle.Exec("INSERT OR REPLACE INTO rounds (round_id, match_id, winner_id, level) VALUES (?, ?, ?, ?)",
		r.Round_ID,
		r.Match_ID,
		r.Winner_ID,
		r.Level,
	)
	if err != nil {
		log.Panic(err)
	}
}

func (m *Match) CompleteMatch(sm *ServerSyncManager) {
	if !m.Winner_ID.Valid {
		log.Panic("winner id can not be null")
	}
	m.End_time = time.Now()
	_, err := sm.db_handle.Exec("INSERT OR REPLACE INTO matches (match_id, start_time, end_time, winner_id) VALUES (?, ?, ?, ?)",
		m.Match_ID,
		//match.Session_ID,
		m.Start_time.Format(time.RFC3339),
		m.End_time.Format(time.RFC3339),
		m.Winner_ID)
	if err != nil {
		log.Panic(err)
	}
}
