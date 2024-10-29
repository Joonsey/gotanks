package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type ServerGameStateEnum int

const (
	ServerGameStateStartingNewMatch ServerGameStateEnum = iota
	ServerGameStatePlaying
	ServerGameStateWaiting
	ServerGameStateStartingNewRound
	ServerGameStateGoingBackToLobby
)

const (
	NEW_LEVEL_INTERVAL_S  = 3
	STATE_CHANGE_GRACE_MS = 500

	WIN_THRESHOLD = 3
	DRAW_CONST    = "DRAW"
)

type ConnectedPlayer struct {
	tank TankMinimal
	addr *net.UDPAddr
}

type PlayerUpdate struct {
	Tank TankMinimal
	ID   string
}

type ConnectedPlayers struct {
	sync.RWMutex
	m map[string]ConnectedPlayer
}

type NewRoundEvent struct {
	Spawns    map[string]Position
	Timestamp time.Time
	Level     LevelEnum
	Winner    string
}

type NewMatchEvent struct {
	Timestamp time.Time
}

type ServerStats struct {
	Matches     []*Match
	KillEvents  []*KillEvent
	DeathEvents []*DeathEvent
	Rounds      []*Round

	winner string
}

type Server struct {
	conn                    *net.UDPConn
	accepts_new_connections bool
	update_count            int

	packet_channel    chan PacketData
	connected_players ConnectedPlayers

	bm    BulletManager
	level Level
	state ServerGameStateEnum
	stats ServerStats

	wait_time time.Time

	// temporary until db
	round_id int
	match_id int
}

func StartServer() {
	server := Server{}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: SERVERPORT})
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()

	server.conn = conn

	server.packet_channel = make(chan PacketData)
	server.connected_players.m = make(map[string]ConnectedPlayer)

	server.accepts_new_connections = true
	server.level = loadLevel("assets/tiled/level_1.tmx", nil, nil)
	server.bm.bullets = make(map[string]*Bullet)

	go server.Listen()
	go server.StartHandlingPackets()

	// this is blocking
	server.StartServerLogic()

}

func (s *Server) Listen() {
	log.Println("server is listening")
	buf := make([]byte, BUFFER_SIZE)
	for {
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			log.Panic("error reading from connection:", err)
		}

		packet, data, err := DeserializePacket(buf[:n])
		if err != nil {
			log.Panic("error reading from connection:", err)
		}

		packet_data := PacketData{packet, data, *addr}
		s.packet_channel <- packet_data
	}
}

func (s *Server) Broadcast(packet Packet, data interface{}) {
	s.connected_players.RLock()
	defer s.connected_players.RUnlock()

	raw_data, err := SerializePacket(packet, data)
	if err != nil {
		log.Panic(err)
	}

	for _, value := range s.connected_players.m {
		s.conn.WriteToUDP(raw_data, value.addr)
	}
}

func (s *Server) HandlePacket(packet_data PacketData) {
	dec := gob.NewDecoder(bytes.NewReader(packet_data.Data))
	switch packet_data.Packet.PacketType {
	case PacketTypeBulletShoot:
		bullet := Bullet{}
		err := dec.Decode(&bullet)
		if err != nil {
			log.Panic("error decoding bullet", err)
		}

		s.Broadcast(packet_data.Packet, bullet)
		s.bm.AddBullet(bullet)
	case PacketTypeUpdateCurrentPlayer:
		s.connected_players.Lock()
		player := s.connected_players.m[packet_data.Addr.String()]
		err := dec.Decode(&player.tank)
		if err != nil {
			log.Panic("error decoding player", err)
		}
		s.connected_players.m[packet_data.Addr.String()] = player
		s.connected_players.Unlock()
	}
}

func (s *Server) GetSpawnMap() map[string]Position {
	s.connected_players.RLock()
	defer s.connected_players.RUnlock()

	spawn_map := make(map[string]Position)
	spawns := s.level.GetSpawnPositions()

	i := 0
	for key := range s.connected_players.m {
		// until revision of 'id'
		player_port := strings.Split(key, ":")[1]
		spawn_map[player_port] = spawns[i%len(spawns)]
		i++
	}

	return spawn_map
}

func (s *Server) UpdateServerLogic() {
	duration, err := time.ParseDuration("16ms")
	defer time.Sleep(duration)
	if err != nil {
		log.Panic(err)
	}

	if s.update_count%UPDATE_INTERVAL == 0 {
		packet := Packet{PacketType: PacketTypeUpdatePlayers}

		players := []PlayerUpdate{}
		s.connected_players.RLock()
		for key, value := range s.connected_players.m {
			players = append(players, PlayerUpdate{Tank: value.tank, ID: key})
		}
		s.connected_players.RUnlock()
		s.Broadcast(packet, players)
	}

	s.bm.Update(&s.level, nil)

	s.connected_players.RLock()
	for key, value := range s.connected_players.m {
		if !value.tank.Alive() {
			continue
		}

		bullet_hit := s.bm.IsColliding(value.tank.Position, Position{16, 16})
		if bullet_hit != nil {
			packet := Packet{PacketType: PacketTypePlayerHit}
			data := BulletHit{Player: key, Bullet_ID: bullet_hit.ID}
			s.connected_players.RUnlock()
			s.Broadcast(packet, data)
			s.connected_players.RLock()

			delete(s.bm.bullets, bullet_hit.ID)
		}
	}
	s.connected_players.RUnlock()

	prior_state := s.state
	new_state := s.CheckServerState()
	if prior_state != new_state {
		packet := Packet{PacketType: PacketTypeServerStateChanged}
		s.Broadcast(packet, new_state)
	}

	s.update_count++
}

func (s *Server) StartServerLogic() {
	for {
		s.UpdateServerLogic()
	}
}

func (s *Server) GetAlivePlayers() (alive []ConnectedPlayer, total_count int) {
	s.connected_players.Lock()
	defer s.connected_players.Unlock()

	alive_player := []ConnectedPlayer{}
	for _, value := range s.connected_players.m {
		if value.tank.Alive() {
			alive_player = append(alive_player, value)
		}
	}

	return alive_player, len(s.connected_players.m)
}

func (s *Server) DetermineNextLevel() LevelEnum {
	return 1
}

func (s *Server) GetHighestWinCount() (top_player string, highest_wins int) {
	match := s.GetCurrentMatch()

	wins := make(map[string]int)
	if match == nil {
		for _, round := range s.stats.Rounds {
			wins[round.Winner_ID]++
		}
	} else {
		for _, round := range s.stats.Rounds {
			if round.Match_ID == match.Match_ID {
				wins[round.Winner_ID]++
			}
		}
	}

	for player_id, wins := range wins {
		if wins > highest_wins {
			top_player = player_id
			highest_wins = wins
		}
	}

	return top_player, highest_wins
}

func (s *Server) GetCurrentMatch() *Match {
	if len(s.stats.Matches) == 0 {
		return nil
	}
	return s.stats.Matches[len(s.stats.Matches)-1]
}

func (s *Server) StartNewMatch() *Match {
	match := Match{}
	match.Match_ID = fmt.Sprintf("%d", s.match_id)
	s.match_id++

	match.StartTime = time.Now()

	return &match
}

func (s *Server) StartNewRound() *Round {
	current_match := s.GetCurrentMatch()
	if current_match == nil {
		log.Panic("can not start a round before a match")
	}
	round := Round{}
	round.Round_ID = fmt.Sprintf("%d", s.round_id)
	s.round_id++
	round.Match_ID = current_match.Match_ID
	round.Level = s.DetermineNextLevel()

	return &round
}

func (s *Server) CheckServerState() ServerGameStateEnum {
	alive, total := s.GetAlivePlayers()
	after_grace_period := time.Now().After(s.wait_time)
	new_state := s.state
	switch s.state {
	case ServerGameStatePlaying:
		if after_grace_period && len(alive) <= 1 && total > 1 {
			current_round := s.stats.Rounds[len(s.stats.Rounds)-1]

			winner_id := DRAW_CONST
			if len(alive) > 0 {
				winner_id = alive[0].addr.String()
				current_round.Winner_ID = winner_id
			}

			top_player, highest_wins := s.GetHighestWinCount()
			if highest_wins >= WIN_THRESHOLD {
				packet := Packet{PacketType: PacketTypeNewMatch}
				wait_time := time.Now().Add(time.Second * NEW_LEVEL_INTERVAL_S)
				event := NewMatchEvent{
					Timestamp: wait_time,
				}

				match := s.GetCurrentMatch()
				match.Winner_ID = top_player
				match.EndTime = time.Now()

				// should go back to lobby
				// workaround for now
				new_state = ServerGameStateStartingNewMatch
				s.wait_time = wait_time
				s.Broadcast(packet, event)
			} else {
				packet := Packet{PacketType: PacketTypeNewRound}
				spawns := s.GetSpawnMap()
				wait_time := time.Now().Add(time.Second * NEW_LEVEL_INTERVAL_S)
				event := NewRoundEvent{
					Spawns:    spawns,
					Timestamp: wait_time,
					Level:     s.DetermineNextLevel(),
					Winner:    winner_id,
				}

				s.wait_time = wait_time
				s.Broadcast(packet, event)
				new_state = ServerGameStateStartingNewRound

			}
			s.bm.Reset()
		}
	case ServerGameStateStartingNewMatch:
		if after_grace_period {
			s.stats.Matches = append(s.stats.Matches, s.StartNewMatch())
			new_state = ServerGameStateStartingNewRound
			s.wait_time = time.Now().Add(time.Millisecond * STATE_CHANGE_GRACE_MS)
			packet := Packet{PacketType: PacketTypeNewRound}
			spawns := s.GetSpawnMap()
			wait_time := time.Now().Add(time.Second * NEW_LEVEL_INTERVAL_S)
			event := NewRoundEvent{
				Spawns:    spawns,
				Timestamp: wait_time,
				Level:     s.DetermineNextLevel(),
			}
			// we omitt the field Winner here
			// not very clean but it is what it is

			s.wait_time = wait_time
			s.Broadcast(packet, event)
			new_state = ServerGameStateStartingNewRound
			s.bm.Reset()
		}
	case ServerGameStateStartingNewRound:
		// adding an extra buffer to let people alive themselves
		if after_grace_period && total > 0 {
			s.stats.Rounds = append(s.stats.Rounds, s.StartNewRound())
			new_state = ServerGameStatePlaying
			s.wait_time = time.Now().Add(time.Millisecond * STATE_CHANGE_GRACE_MS)
			s.bm.Reset()
		}
	}

	s.state = new_state
	return s.state
}

func (s *Server) AuthorizePacket(packet_data PacketData) error {
	s.connected_players.Lock()
	defer s.connected_players.Unlock()

	for key, _ := range s.connected_players.m {
		if key == packet_data.Addr.String() {
			// authorized, and no erros
			return nil
		}
	}

	if s.accepts_new_connections {
		s.connected_players.m[packet_data.Addr.String()] = ConnectedPlayer{addr: &packet_data.Addr}

		// added and authorized
		log.Println("accepted new connection")
		return nil
	}

	return errors.New("not authorized")
}

func (s *Server) StartHandlingPackets() {
	for {
		select {
		case packet_data := <-s.packet_channel:
			err := s.AuthorizePacket(packet_data)
			if err != nil {
				log.Println("authorization error: ", err)
				continue
			}
			s.HandlePacket(packet_data)
		}
	}
}
