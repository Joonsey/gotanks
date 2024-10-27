package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"net"
)

const (
	SERVERPORT  = 7707
	BUFFER_SIZE = 2048
)

type Client struct {
	conn   *net.UDPConn
	target *net.UDPAddr

	packet_channel chan PacketData
	is_connected   bool
}

type NetworkManager struct {
	client *Client

	tanks []TankMinimal
}

func (c *Client) isConnected() bool {
	return c.is_connected
}

func InitNetworkManager() *NetworkManager {
	nm := NetworkManager{}
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Fatal(err)
	}

	nm.client = &Client{}
	nm.client.packet_channel = make(chan PacketData)
	nm.client.conn = conn

	nm.client.target = &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: SERVERPORT}
	return &nm
}

func (c *Client) Send(packet_type PacketType, data interface{}) error {
	if !c.isConnected() {
		return errors.New("tried to send without being connected")
	}
	packet := Packet{}
	packet.PacketType = packet_type
	data_bytes, err := SerializePacket(packet, data)

	if err != nil {
		return err
	}

	_, err = c.conn.WriteToUDP(data_bytes, c.target)
	return err
}

func (c *Client) Listen() {
	if c.isConnected() {
		log.Panic("attempted to 'Listen' while already connected")
	}

	c.is_connected = true
	buf := make([]byte, BUFFER_SIZE)
	for {
		n, addr, err := c.conn.ReadFromUDP(buf)
		if err != nil {
			log.Panic("error reading from connection:", err)
		}

		packet, data, err := DeserializePacket(buf[:n])
		if err != nil {
			log.Panic("error reading from connection:", err)
		}

		packet_data := PacketData{packet, data, *addr}
		c.packet_channel <- packet_data
	}
}

func (c *Client) Loop(game *Game) {
	for {
		select {
		case packet_data := <-c.packet_channel:
			c.HandlePacket(packet_data, game)
		}
	}
}

func (c *Client) HandlePacket(packet_data PacketData, game *Game) {
	dec := gob.NewDecoder(bytes.NewReader(packet_data.Data))
	switch packet_data.Packet.PacketType {
	case PacketTypeBulletShoot:
		bullet := Bullet{}
		err := dec.Decode(&bullet)
		if err != nil {
			log.Panic("error decoding bullet", err)
		}

		game.bm.AddBullet(bullet)
	}
}
