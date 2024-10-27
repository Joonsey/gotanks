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
}

type ConnectedPlayers struct {
	sync.RWMutex
	m map[*net.UDPAddr]ConnectedPlayer
}

type Server struct {
	conn *net.UDPConn
	accepts_new_connections bool

	packet_channel chan PacketData
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
	server.connected_players.m = make(map[*net.UDPAddr]ConnectedPlayer)

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

	for key, _ := range s.connected_players.m {
		s.conn.WriteToUDP(raw_data, key)
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

		log.Println(bullet)
		s.Broadcast(packet_data.Packet, bullet)
		s.bm.AddBullet(bullet)
	}
}

func (s *Server) UpdateServerLogic() {
	duration, err := time.ParseDuration("16ms")
	if err != nil {
		log.Panic(err)
	}

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
		if key == &packet_data.Addr {
			// authorized, and no erros
			return nil
		}
	}

	if s.accepts_new_connections {
		s.connected_players.m[&packet_data.Addr] = ConnectedPlayer{}

		// added and authorized
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

