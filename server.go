package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"net"
	"sync"
	"time"
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

type Server struct {
	conn                    *net.UDPConn
	accepts_new_connections bool
	update_count            int

	packet_channel    chan PacketData
	connected_players ConnectedPlayers

	bm BulletManager
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

func (s *Server) UpdateServerLogic() {
	duration, err := time.ParseDuration("16ms")
	if err != nil {
		log.Panic(err)
	}

	if s.update_count%16 == 0 {
		packet := Packet{PacketType: PacketTypeUpdatePlayers}

		players := []PlayerUpdate{}
		s.connected_players.RLock()
		for key, value := range s.connected_players.m {
			players = append(players, PlayerUpdate{Tank: value.tank, ID: key})
		}
		s.connected_players.RUnlock()
		s.Broadcast(packet, players)
	}

	s.update_count++
	defer time.Sleep(duration)
}

func (s *Server) StartServerLogic() {
	for {
		s.UpdateServerLogic()
	}
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
