package main

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"gotanks"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Host struct {
	Time int64
	game.AvailableServer
}
type HostsMap map[string]Host
const (
	TIMEOUT_MS = 7000
)

func timeoutStaleConnections(keyword_map *HostsMap) {
	for key, value := range *keyword_map {
		if time.Now().UnixMilli()-value.Time > TIMEOUT_MS {
			fmt.Printf("%s:%d user timed out using '%s' connection key\n", value.Ip, value.Port, value.Name)
			delete(*keyword_map, key)
		}
	}
}

func main() {
	server_addr, err := net.ResolveUDPAddr("udp", ":8080")
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	host_map := make(HostsMap)

	conn, err := net.ListenUDP("udp", server_addr)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}

	fmt.Println("mediator listening")
	defer conn.Close()

	packet_channel := make(chan game.PacketData)

	go func() {
		for {
			timeoutStaleConnections(&host_map)
			time.Sleep(time.Second * 1)
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, addr, err := conn.ReadFromUDP(buf)
			if err != nil {
				fmt.Println("error reading", err)
			}

			packet, data, err := game.DeserializePacket(buf[:n])
			if err != nil {
				fmt.Println("error reading", err)
			}

			packet_data := game.PacketData{Packet: packet, Data: data, Addr: *addr}
			packet_channel <- packet_data
		}
	}()

	for {
		select {
		case packet_data := <-packet_channel:
			dec := gob.NewDecoder(bytes.NewReader(packet_data.Data))
			switch packet_data.Packet.PacketType {
			case game.PacketTypeAvailableHosts:
				l := []game.AvailableServer{}
				for _, value := range host_map {
					l = append(l, value.AvailableServer)
				}
				serialized_packet, err := game.SerializePacket(packet_data.Packet, [16]byte{}, l)
				if err != nil {
					fmt.Println("error serializing packet", err)
				}
				sort.Slice(l, func(i, j int) bool {
					return l[i].Name < l[j].Name
				})

				conn.WriteToUDP(serialized_packet, &packet_data.Addr)
			case game.PacketTypeUpdateMediator:
				var server game.AvailableServer
				err := dec.Decode(&server)
				if err != nil {
					log.Panic("error during decoding", err)
				}
				val, ok := host_map[server.Name]
				if !ok {
					log.Panic("error during decoding", err)
				}

				val.Max_players = server.Max_players
				val.Player_count = server.Player_count
				host_map[server.Name] = val

			case game.PacketTypeKeepAlive:
				// refreshing timeout
				for key, value := range host_map {
					if fmt.Sprintf("%s:%d", value.Ip, value.Port) == packet_data.Addr.String() {
						value.Time = time.Now().UnixMilli()
						host_map[key] = value
					}
				}
			case game.PacketTypeMatchConnect:
				var inner_data game.ReconcilliationData
				err := dec.Decode(&inner_data)
				if err != nil {
					log.Panic("error during decoding", err)
				}

				val, ok := host_map[inner_data.Name]
				if !ok {
					fmt.Println("could not find match")
					break
				}
				packet := game.Packet{PacketType: game.PacketTypeMatchConnect}
				serialized_packet, err := game.SerializePacket(packet, [16]byte{}, packet_data.Addr)

				tar_addr := &net.UDPAddr{IP: net.ParseIP(val.Ip), Port: val.Port}
				log.Printf("sending new player at %s to server at %s\n", &packet_data.Addr, tar_addr)
				conn.WriteToUDP(serialized_packet, tar_addr)
			case game.PacketTypeMatchHost:
				var inner_data game.ReconcilliationData
				err := dec.Decode(&inner_data)
				if err != nil {
					log.Panic("error during decoding", err)
				}

				// if already exists
				if host_map[inner_data.Name].Name != "" {
					break
				}

				addr_str := strings.Split(packet_data.Addr.String(), ":")
				port, err:= strconv.Atoi(addr_str[1])
				if err != nil {
					fmt.Println("error parsing addr to int")
				}
				host_map[inner_data.Name] = Host{Time: time.Now().UnixMilli(), AvailableServer: game.AvailableServer{Ip: addr_str[0], Port: port, Name: inner_data.Name}}
				fmt.Println("added new host: ", inner_data)
			case game.PacketTypeMatchStart:
				var inner_data game.ReconcilliationData
				err := dec.Decode(&inner_data)
				if err != nil {
					fmt.Println("error during decoding", err)
				}

				fmt.Printf("%s's server has started, and has been removed from eligible lobbies\n", packet_data.Addr.String())
				delete(host_map, inner_data.Name)
			}
		}
	}
}

