package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	SERVERPORT  = 7707
	BUFFER_SIZE = 2048

	// update_interval = fps / desired ticks per second
	// 3 = 60/20
	UPDATE_INTERVAL = 3
)

type Client struct {
	conn   *net.UDPConn
	target *net.UDPAddr

	packet_channel chan PacketData
	is_connected   bool
	ID             string
	server_state   ServerGameStateEnum
}

type NetworkManager struct {
	client *Client
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

	// TODO do something better
	nm.client.ID = strings.Split(nm.client.conn.LocalAddr().String(), "[::]:")[1]

	nm.client.target = &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: SERVERPORT}
	return &nm
}

func (c *Client) isSelf(id string) bool {
	player_port := strings.Split(id, ":")[1]
	return c.ID == player_port
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

func (nm *NetworkManager) GetDrawData(g *Game) {
	if !g.nm.client.isConnected() {
		log.Panic("tried to get draw data without being connected")
		return
	}

	for _, player := range g.player_updates {
		if nm.client.isSelf(player.ID) {
			continue
		}

		t := player.Tank

		x, y := g.camera.GetRelativePosition(t.X, t.Y)
		if t.Alive() {
			g.draw_data = append(g.draw_data,
				DrawData{
					sprites:   g.tank.sprites,
					position:  Position{x, y},
					rotation:  t.Rotation - g.camera.rotation,
					intensity: 1,
					opacity:   1},
			)
			g.draw_data = append(g.draw_data,
				DrawData{
					sprites:   g.tank.turret.sprites,
					position:  Position{x, y + 1},
					rotation:  t.Turret_rotation,
					intensity: 1,
					offset:    Position{0, -TURRET_HEIGHT},
					opacity:   1},
			)
			if int(g.time*100)%TRACK_INTERVAL == 0 {
				g.tracks = append(g.tracks, Track{t.Position, t.Rotation, TRACK_LIFETIME})
			}
		} else {
			// TODO extrapolate dead sprites data
			dead_sprites := g.tank.dead_sprites
			g.draw_data = append(g.draw_data, DrawData{
				sprites:   dead_sprites,
				position:  Position{x, y},
				rotation:  t.Rotation - g.camera.rotation,
				intensity: 1,
				opacity:   1},
			)

			// not sure how i feel about this living in a draw call
			t.TryAddSmoke(g)

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
	case PacketTypeUpdatePlayers:
		err := dec.Decode(&game.player_updates)
		if err != nil {
			log.Panic("error decoding player updates", err)
		}
	case PacketTypePlayerHit:
		hit := BulletHit{}
		err := dec.Decode(&hit)
		if err != nil {
			log.Panic("error decoding bullet", err)
		}
		if c.isSelf(hit.Player) {
			game.tank.Hit(hit)
		}
		particle_sprite := game.am.GetSprites("assets/sprites/stacks/particle-cube-template.png")
		bullet := game.bm.bullets[hit.Bullet_ID]

		seed := time.Now().Unix()
		particle_count := float64(seed % 5)
		for i := range int(particle_count) + 1 {
			// TODO seed this so it can be reasonably consistent across clients
			n := rand.Float64() + 1
			game.pm.AddParticle(
				Particle{Position: bullet.Position,
					Rotation:      bullet.Rotation + (float64(i)/particle_count)*1.5,
					sprites:       particle_sprite,
					velocity:      n * .4,
					particle_type: ParticleTypeDebrisFromTank,
					max_t:         60 * n,
				})
		}

		delete(game.bm.bullets, hit.Bullet_ID)
	case PacketTypeNewLevel:
		event := NewLevelEvent{Spawns: map[string]Position{}}
		err := dec.Decode(&event)
		if err != nil {
			log.Panic("error decoding spawn map", err)
		}

		spawn, ok := event.Spawns[c.ID]
		if !ok {
			log.Panic("could not find spawn in spawn map ", event.Spawns)
		}

		game.new_level_time = event.Timestamp
		go func() {
			time.Sleep(event.Timestamp.Sub(time.Now()))
			game.tank.Respawn(spawn)
			game.bm.Reset()
		}()
	case PacketTypeServerStateChanged:
		err := dec.Decode(&c.server_state)
		if err != nil {
			log.Panic("error decoding new server state")
		}
	}
}
