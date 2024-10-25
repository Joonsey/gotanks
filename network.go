package main

import (
	"errors"
	"log"
	"net"
)

const (
	SERVERPORT = 7707
)

type Client struct {
	conn   *net.UDPConn
	target *net.UDPAddr
}

type NetworkManager struct {
	client *Client
}

func (c *Client) isConnected() bool {
	return false
}

func InitNetworkManager() *NetworkManager {
	nm := NetworkManager{}
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		log.Fatal(err)
	}

	nm.client = &Client{}
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

	if err == nil {
		return err
	}

	_, err = c.conn.WriteToUDP(data_bytes, c.target)
	return err
}
