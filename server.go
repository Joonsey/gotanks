package game

import (
	"bytes"
	"database/sql"
	"encoding/gob"
	"errors"
	"fmt"
	"gotanks/shared"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

type ServerGameStateEnum int

const (
	NetBoolTrue  = 1
	NetBoolFalse = 2
)

const (
	ServerGameStateWaitingInLobby ServerGameStateEnum = iota
	ServerGameStatePlaying
	ServerGameStateStartingNewMatch
	ServerGameStateStartingNewRound
	ServerGameStateGoingBackToLobby
	ServerGameStateGameOver
)

// converts uint to bool
// we need this because gob can't decode nil values
func NetBoolify(n uint) bool {
	return n == 1
}

const (
	NEW_LEVEL_INTERVAL_S  = 3
	GAME_OVER_INTERVAL_S  = 5
	STATE_CHANGE_GRACE_MS = 500
	KEEPALIVE_INTERVAL    = 30

	WIN_THRESHOLD = 3
)

type ConnectedPlayer struct {
	tank   TankMinimal
	player Player
	addr   *net.UDPAddr
	ready  uint
}

type PlayerUpdate struct {
	Tank  TankMinimal
	ID    string
	Ready uint
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

func CreateServerName() string {
	names := []string{
		"apple", "banana", "cherry", "date", "elderberry", "fig", "grape", "honeydew",
		"kiwi", "lemon", "mango", "nectarine", "orange", "papaya", "quince", "raspberry",
		"strawberry", "tangerine", "ugli", "vanilla", "watermelon", "xigua", "yam", "zucchini",
		"brick", "walls", "yellow", "red", "blue", "purple", "orange", "funny", "warm", "cold",
	}
	return names[rand.Intn(len(names))]
}

type Server struct {
	conn                    *net.UDPConn
	accepts_new_connections bool
	update_count            int

	packet_channel    chan shared.PacketData
	connected_players ConnectedPlayers

	bm     BulletManager
	levels []Level
	state  ServerGameStateEnum
	sm     *ServerSyncManager
	Name   string

	current_level int

	wait_time time.Time

	mediator_addr *net.UDPAddr
}

func (s *Server) CurrentLevel() *Level {
	return &s.levels[s.current_level]
}

func StartServer(name string, mediator_addr *net.UDPAddr) {
	server := Server{}
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: SERVERPORT})
	if err != nil {
		log.Panic(err)
	}
	defer conn.Close()

	server.conn = conn

	server.packet_channel = make(chan shared.PacketData)
	server.connected_players.m = make(map[string]ConnectedPlayer)

	server.accepts_new_connections = true
	for i := range LEVEL_COUNT {
		level_path := fmt.Sprintf("assets/tiled/level_%d.tmx", i+1)
		server.levels = append(server.levels, loadLevel(level_path, nil, true))
	}
	server.bm.bullets = make(map[string]Bullet)

	server.mediator_addr = mediator_addr

	server.sm = InitStatsManager()
	server.Name = name

	go server.Listen()
	go server.StartHandlingPackets()
	server.TellMediator()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		server.sm.DeInit()
		os.Exit(0)
	}()
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

		packet, data, err := shared.DeserializePacket(buf[:n])
		if err != nil {
			log.Panic("error reading from connection:", err)
		}

		packet_data := shared.PacketData{Packet: packet, Data: data, Addr: *addr}
		s.packet_channel <- packet_data
	}
}

func (s *Server) UpdateMediator() {
	data := shared.AvailableServer{Player_count: len(s.connected_players.m), Max_players: 4, Name: s.Name}
	raw_data, err := shared.SerializePacket(shared.Packet{PacketType: shared.PacketTypeUpdateMediator}, [16]byte{}, data)
	if err != nil {
		log.Panic("failed to serialize packet")
	}
	s.conn.WriteToUDP(raw_data, s.mediator_addr)
}
func (s *Server) KeepAliveMediator() {
	raw_data, err := shared.SerializePacket(shared.Packet{PacketType: shared.PacketTypeKeepAlive}, [16]byte{}, []byte{})
	if err != nil {
		log.Panic("failed to serialize packet")
	}
	s.conn.WriteToUDP(raw_data, s.mediator_addr)
}

func (s *Server) Broadcast(packet shared.Packet, data interface{}) {
	s.connected_players.RLock()
	defer s.connected_players.RUnlock()

	raw_data, err := shared.SerializePacket(packet, [16]byte{}, data)
	if err != nil {
		log.Panic(err)
	}

	for _, value := range s.connected_players.m {
		s.conn.WriteToUDP(raw_data, value.addr)
	}
}

func (s *Server) HandlePacket(packet_data shared.PacketData) {
	dec := gob.NewDecoder(bytes.NewReader(packet_data.Data))
	switch packet_data.Packet.PacketType {
	case shared.PacketTypeBulletShoot:
		bullet := StandardBullet{}
		err := dec.Decode(&bullet)
		if err != nil {
			log.Panic("error decoding bullet", err)
		}

		// TODO
		// bullet factory
		bullet.grace_period = s.bm.DetermineGracePeriod(bullet.Bullet_type)
		bullet.ID = s.bm.NewBulletId()

		s.Broadcast(packet_data.Packet, bullet)
		s.bm.AddBullet(bullet)
	case shared.PacketTypeUpdateCurrentPlayer:
		s.connected_players.Lock()
		player := s.connected_players.m[shared.AuthToString(packet_data.Packet.Auth)]
		err := dec.Decode(&player.tank)
		if err != nil {
			log.Panic("error decoding player", err)
		}
		s.connected_players.m[shared.AuthToString(packet_data.Packet.Auth)] = player
		s.connected_players.Unlock()
	case shared.PacketTypeClientToggleReady:
		s.connected_players.Lock()
		player := s.connected_players.m[shared.AuthToString(packet_data.Packet.Auth)]
		if NetBoolify(player.ready) {
			player.ready = NetBoolFalse
		} else {
			player.ready = NetBoolTrue
		}

		s.connected_players.m[shared.AuthToString(packet_data.Packet.Auth)] = player
		s.connected_players.Unlock()
	case shared.PacketTypeDisconnect:
		s.connected_players.Lock()
		delete(s.connected_players.m, shared.AuthToString(packet_data.Packet.Auth))
		s.connected_players.Unlock()
	case shared.PacketTypeMatchConnect:
		var addr net.UDPAddr
		err := dec.Decode(&addr)
		if err != nil {
			log.Panic("error during decoding", err)
		}
		data_bytes, err := shared.SerializePacket(shared.Packet{PacketType: shared.PacketTypeMatchConnect}, [16]byte{}, []byte{})
		if err != nil {
			log.Panic("error during serializing", err)
		}
		s.conn.WriteToUDP(data_bytes, &addr)
	}
}

func (s *Server) GetSpawnMap() map[string]Position {
	s.connected_players.RLock()
	defer s.connected_players.RUnlock()

	spawn_map := make(map[string]Position)
	spawns := s.CurrentLevel().GetSpawnPositions()

	i := 0
	for key := range s.connected_players.m {
		spawn_map[key] = spawns[i%len(spawns)]
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
		packet := shared.Packet{PacketType: shared.PacketTypeUpdatePlayers}

		players := []PlayerUpdate{}
		s.connected_players.RLock()
		for key, value := range s.connected_players.m {
			players = append(players, PlayerUpdate{Tank: value.tank, ID: key, Ready: value.ready})
		}
		sort.Slice(players, func(i, j int) bool {
			return players[i].ID < players[j].ID
		})
		s.connected_players.RUnlock()
		s.Broadcast(packet, players)
		s.KeepAliveMediator()
		s.UpdateMediator()
	}

	s.bm.Update(s.CurrentLevel(), nil)

	s.connected_players.RLock()
	for key, value := range s.connected_players.m {
		if !value.tank.Alive() {
			continue
		}

		bullet_hit := s.bm.IsColliding(value.tank.Position, Position{16, 16})
		if bullet_hit != nil {
			packet := shared.Packet{PacketType: shared.PacketTypePlayerHit}
			data := BulletHit{Player: key, Bullet_ID: (*bullet_hit).GetId()}
			s.connected_players.RUnlock()
			s.Broadcast(packet, data)
			s.connected_players.RLock()

			// we know that the bullet is supposed be split up this way:
			// 'owner:bullet_id' however, we don't have a solid way to id a user yet
			// so this is currently not 100% working, but will when authorization is complete
			// so TODO authorization...
			delete(s.bm.bullets, (*bullet_hit).GetId())
			if len(s.sm.stats.Rounds) > 0 {
				round_id := s.sm.stats.Rounds[len(s.sm.stats.Rounds)-1].Round_ID
				shooter_id := strings.Split((*bullet_hit).GetId(), ":")[0]
				kill_event := NewKillEvent(round_id, key, shooter_id)
				kill_event.Sync(s.sm)
			}
		}
	}
	s.connected_players.RUnlock()

	prior_state := s.state
	new_state := s.CheckServerState()
	if prior_state != new_state {
		packet := shared.Packet{PacketType: shared.PacketTypeServerStateChanged}
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

func (s *Server) GetReadyPlayers() (ready []ConnectedPlayer, total_count int) {
	s.connected_players.Lock()
	defer s.connected_players.Unlock()

	ready_players := []ConnectedPlayer{}
	for _, value := range s.connected_players.m {
		if NetBoolify(value.ready) {
			ready_players = append(ready_players, value)
		}
	}

	return ready_players, len(s.connected_players.m)
}

func (s *Server) DetermineNextLevel() LevelEnum {
	s.current_level = (s.current_level + 1) % len(s.levels)
	return LevelEnum(s.current_level)
}

func (s *Server) GetHighestWinCount() (top_player string, highest_wins int) {
	match := s.GetCurrentMatch()

	wins := make(map[string]int)
	if match == nil {
		for _, round := range s.sm.stats.Rounds {
			wins[round.Winner_ID.String]++
		}
	} else {
		for _, round := range s.sm.stats.Rounds {
			if round.Match_ID == match.Match_ID {
				wins[round.Winner_ID.String]++
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
	if len(s.sm.stats.Matches) == 0 {
		return nil
	}
	return s.sm.stats.Matches[len(s.sm.stats.Matches)-1]
}

func (s *Server) StartNewMatch() *Match {
	match := NewMatch(s.sm)

	return &match
}

func (s *Server) StartNewRound() *Round {
	current_match := s.GetCurrentMatch()
	if current_match == nil {
		log.Panic("can not start a round before a match")
	}
	round := NewRound(*current_match, LevelEnum(s.current_level), s.sm)

	return &round
}

func (s *Server) TellMediator() {
	data := shared.ReconcilliationData{Name: s.Name}
	raw_data, err := shared.SerializePacket(shared.Packet{PacketType: shared.PacketTypeMatchHost}, [16]byte{}, data)
	if err != nil {
		log.Panic("failed to serialize packet")
	}
	s.conn.WriteToUDP(raw_data, s.mediator_addr)
}

func (s *Server) CheckServerState() ServerGameStateEnum {
	alive, total := s.GetAlivePlayers()
	after_grace_period := time.Now().After(s.wait_time)
	new_state := s.state
	switch s.state {
	case ServerGameStateWaitingInLobby:
		ready, total := s.GetReadyPlayers()
		if after_grace_period && total > 1 && len(ready) == total {
			packet := shared.Packet{PacketType: shared.PacketTypeNewMatch}
			new_state = ServerGameStateStartingNewMatch
			wait_time := time.Now().Add(time.Second * NEW_LEVEL_INTERVAL_S)
			event := NewMatchEvent{
				Timestamp: wait_time,
			}
			s.Broadcast(packet, event)
		}
	case ServerGameStateGameOver:
		if after_grace_period {
			s.wait_time = time.Now().Add(time.Millisecond * STATE_CHANGE_GRACE_MS)
			new_state = ServerGameStateWaitingInLobby
		}
	case ServerGameStatePlaying:
		if after_grace_period && len(alive) <= 1 && total > 1 {
			current_round := s.sm.stats.Rounds[len(s.sm.stats.Rounds)-1]

			winner_id := alive[0].player.Player_ID
			current_round.Winner_ID = sql.NullString{String: winner_id, Valid: true}
			go current_round.CompleteRound(s.sm)

			top_player, highest_wins := s.GetHighestWinCount()
			if highest_wins >= WIN_THRESHOLD {
				match := s.GetCurrentMatch()
				match.Winner_ID = sql.NullString{String: top_player, Valid: true}
				go match.CompleteMatch(s.sm)

				new_state = ServerGameStateGameOver
				wait_time := time.Now().Add(time.Second * GAME_OVER_INTERVAL_S)

				packet := shared.Packet{PacketType: shared.PacketTypeGameOver}
				event := NewRoundEvent{
					Timestamp: wait_time,
					Level:     LevelEnum(s.current_level),
					Winner:    winner_id,
				}
				s.wait_time = wait_time
				s.Broadcast(packet, event)
			} else {
				packet := shared.Packet{PacketType: shared.PacketTypeNewRound}
				spawns := s.GetSpawnMap()
				wait_time := time.Now().Add(time.Second * NEW_LEVEL_INTERVAL_S)
				event := NewRoundEvent{
					Spawns:    spawns,
					Timestamp: wait_time,
					Level:     LevelEnum(s.current_level),
					Winner:    winner_id,
				}

				s.wait_time = wait_time
				s.Broadcast(packet, event)
				new_state = ServerGameStateStartingNewRound

			}
		}
	case ServerGameStateStartingNewMatch:
		if after_grace_period {
			// TODO validate that this is accurate
			// works as intended from a little bit of testing
			s.DetermineNextLevel()

			s.sm.stats.Matches = append(s.sm.stats.Matches, s.StartNewMatch())
			new_state = ServerGameStateStartingNewRound
			s.wait_time = time.Now().Add(time.Millisecond * STATE_CHANGE_GRACE_MS)
			packet := shared.Packet{PacketType: shared.PacketTypeNewRound}
			spawns := s.GetSpawnMap()
			wait_time := time.Now().Add(time.Second * NEW_LEVEL_INTERVAL_S)
			event := NewRoundEvent{
				Spawns:    spawns,
				Timestamp: wait_time,
				Level:     LevelEnum(s.current_level),
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
			s.sm.stats.Rounds = append(s.sm.stats.Rounds, s.StartNewRound())
			new_state = ServerGameStatePlaying
			s.wait_time = time.Now().Add(time.Millisecond * STATE_CHANGE_GRACE_MS)
			s.bm.Reset()

			s.connected_players.Lock()
			for key, value := range s.connected_players.m {
				value.ready = NetBoolFalse
				s.connected_players.m[key] = value
			}
			s.connected_players.Unlock()
		}
	}

	s.state = new_state
	return s.state
}

func (s *Server) AuthorizePacket(packet_data shared.PacketData) error {
	s.connected_players.Lock()
	defer s.connected_players.Unlock()
	// this is the mediator server, typically
	if packet_data.Packet.Auth == [16]byte{} {
		return nil
	}

	auth := shared.AuthToString(packet_data.Packet.Auth)

	for key, _ := range s.connected_players.m {
		if key == auth {
			// authorized, and no erros
			return nil
		}
	}

	if s.accepts_new_connections {
		var player Player
		player_ptr := s.sm.GetPlayer(auth)
		if player_ptr != nil {
			player = *player_ptr
			log.Println("player joined:  ", player.Player_ID)
		} else {
			player = NewPlayer(auth)
			log.Println("made new player:", player.Player_ID)
		}

		go player.Update(s.sm)
		s.connected_players.m[auth] = ConnectedPlayer{addr: &packet_data.Addr, player: player}
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
