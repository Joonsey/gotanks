package game

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	SERVERPORT  = 7707
	BUFFER_SIZE = 2048

	MEDIATOR_PORT = 8080
	// MEDIATOR_ADDR = "84.215.22.166"
	MEDIATOR_ADDR = "127.0.0.1"

	// update_interval = fps / desired ticks per second
	// 3 = 60/20
	UPDATE_INTERVAL = 3
)

type Client struct {
	conn   *net.UDPConn
	target *net.UDPAddr

	packet_channel chan PacketData
	is_connected   bool
	server_state   ServerGameStateEnum
	wins           map[string]int
	Auth           *[16]byte

	time_last_packet time.Time

	available_servers []AvailableServer
}

type AvailableServer struct {
	Ip   string
	Port int
	Name string

	Player_count int
	Max_players  int
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
	nm.client.wins = make(map[string]int)
	nm.client.conn = conn

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		if nm.client.isConnected() {
			nm.client.Disconnect()
		}
		os.Exit(0)
	}()

	go func() {
		for {
			if nm.client.isConnected() {
				time.Sleep(time.Second * 2)
				t := time.Now().Add(-time.Second * 7)
				if nm.client.time_last_packet.Before(t) {
					log.Println("no response for 5s, considering connection closed")
					nm.client.Disconnect()
				}
			} else {
				time.Sleep(time.Second * 2)
				data_bytes, err := SerializePacket(Packet{PacketType: PacketTypeAvailableHosts}, *nm.client.Auth, []byte{})
				if err != nil {
					log.Println("unable to serialize packet, but we don't break for that reason")
					continue
				}
				nm.client.conn.WriteToUDP(data_bytes, &net.UDPAddr{IP: net.ParseIP(MEDIATOR_ADDR), Port: MEDIATOR_PORT})
			}
		}
	}()
	return &nm
}

func (c *Client) isSelf(id string) bool {
	return AuthToString(*c.Auth) == id
}

func (c *Client) Send(packet_type PacketType, data interface{}) error {
	if !c.isConnected() {
		return errors.New("tried to send without being connected")
	}
	packet := Packet{}
	packet.PacketType = packet_type
	data_bytes, err := SerializePacket(packet, *c.Auth, data)

	if err != nil {
		return err
	}

	_, err = c.conn.WriteToUDP(data_bytes, c.target)
	return err
}

func (c *Client) Listen() {
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

func (c *Client) Connect(server AvailableServer) {
	if c.isConnected() {
		log.Panic("tried to connect while already connected")
	}
	c.target = &net.UDPAddr{IP: net.ParseIP(server.Ip), Port: server.Port}

	data := ReconcilliationData{Name: server.Name}
	data_bytes, _ := SerializePacket(Packet{PacketType: PacketTypeMatchConnect}, *c.Auth, data)
	c.conn.WriteToUDP(data_bytes, &net.UDPAddr{IP: net.ParseIP(MEDIATOR_ADDR), Port: MEDIATOR_PORT})
	c.is_connected = true
}

func (c *Client) Disconnect() {
	if !c.isConnected() {
		log.Panic("tried to disconnect while not connected")
	}
	c.Send(PacketTypeDisconnect, "disconnect")
	c.is_connected = false
	c.target = nil
}

func (c *Client) Loop(game *Game) {
	for {
		select {
		case packet_data := <-c.packet_channel:
			c.HandlePacket(packet_data, game)
			if packet_data.Packet.PacketType != PacketTypeAvailableHosts {
				c.time_last_packet = time.Now()
			}
		}
	}
}

func (c *Client) GetServerList(game *Game) []AvailableServer {
	// TODO
	return c.available_servers
}

func (c *Client) KeepAlive(game *Game) {
	if int(game.time*100)%KEEPALIVE_INTERVAL == 0 {
		c.Send(PacketTypeKeepAlive, []byte{})
	}
}

func (nm *NetworkManager) GetDrawData(g *Game) {
	if !g.nm.client.isConnected() {
		log.Panic("tried to get draw data without being connected")
		return
	}

	for i, player := range g.context.player_updates {
		if nm.client.isSelf(player.ID) {
			continue
		}

		t := player.Tank

		x, y := g.camera.GetRelativePosition(t.X, t.Y)
		if t.Alive() {
			g.context.draw_data = append(g.context.draw_data,
				DrawData{
					sprites:   g.tank.sprites,
					position:  Position{x, y},
					rotation:  t.Rotation - g.camera.rotation,
					intensity: 1,
					opacity:   1},
			)
			g.context.draw_data = append(g.context.draw_data,
				DrawData{
					sprites:   g.tank.turret.sprites,
					position:  Position{x, y + 1},
					rotation:  t.Turret_rotation,
					intensity: 1,
					offset:    Position{0, -TURRET_HEIGHT},
					opacity:   1},
			)
			if int(g.time*100)%TRACK_INTERVAL == 0 {
				g.context.tracks = append(g.context.tracks, Track{t.Position, t.Rotation, TRACK_LIFETIME})
			}
			radius := 20
			radi_sprite := ebiten.NewImage(radius, radius)
			vector.StrokeCircle(radi_sprite, float32(radius)/2, float32(radius)/2, float32(radius)/2, 2, player_palette[i%len(player_palette)], true)
			g.context.draw_data = append(g.context.draw_data, DrawData{
				sprites:   []*ebiten.Image{radi_sprite},
				position:  Position{x, y - 1},
				rotation:  t.Rotation,
				intensity: 1,
				offset:    Position{0, 1},
				opacity:   1},
			)
		} else {
			// TODO extrapolate dead sprites data
			dead_sprites := g.tank.dead_sprites
			g.context.draw_data = append(g.context.draw_data, DrawData{
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
func (c *Client) IncrementWin(winner string) {
	if winner != "" {
		c.wins[winner]++
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
		err := dec.Decode(&game.context.player_updates)
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
		particle_count = float64(seed % 3)
		for i := range int(particle_count) + 1 {
			// TODO seed this so it can be reasonably consistent across clients
			n := rand.Float64() + 1
			game.pm.AddParticle(
				Particle{Position: bullet.Position,
					Rotation:      bullet.Rotation + (float64(i)/particle_count)*1.5,
					sprites:       particle_sprite,
					velocity:      n * .2,
					particle_type: ParticleTypeDebrisFromTank,
					max_t:         45 * n,
				})
		}
		game.pm.AddParticle(
			Particle{Position: bullet.Position,
				Rotation:      0,
				sprites:       particle_sprite,
				velocity:      .6,
				particle_type: ParticleTypeDonut,
				max_t:         55,
			})

		delete(game.bm.bullets, hit.Bullet_ID)
	case PacketTypeNewRound:
		event := NewRoundEvent{Spawns: map[string]Position{}}
		err := dec.Decode(&event)
		if err != nil {
			log.Panic("error decoding new round event", err)
		}

		c.IncrementWin(event.Winner)

		spawn, ok := event.Spawns[AuthToString(*c.Auth)]
		if !ok {
			log.Panic("could not find spawn in spawn map ", event.Spawns)
		}

		game.context.new_level_time = event.Timestamp
		go func() {
			time.Sleep(event.Timestamp.Sub(time.Now()))
			game.context.current_state = GameStatePlaying
			game.tank.Respawn(spawn)
			game.bm.Reset()
			game.pm.Reset()
		}()
	case PacketTypeNewMatch:
		event := NewMatchEvent{}
		err := dec.Decode(&event)
		if err != nil {
			log.Panic("error decoding spawn map", err)
		}
		go func() {
			time.Sleep(event.Timestamp.Sub(time.Now()))
			for k := range c.wins {
				c.wins[k] = 0
			}
			game.bm.Reset()
			game.pm.Reset()
		}()
	case PacketTypeServerStateChanged:
		err := dec.Decode(&c.server_state)
		if err != nil {
			log.Panic("error decoding new server state")
		}
	case PacketTypeBackToLobby:
		game.context.current_state = GameStateLobby
	case PacketTypeGameOver:
		event := NewRoundEvent{Spawns: map[string]Position{}}
		err := dec.Decode(&event)
		if err != nil {
			log.Panic("error decoding game over event", err)
		}

		c.IncrementWin(event.Winner)

		game.context.game_over_time = event.Timestamp
		go func() {
			time.Sleep(event.Timestamp.Sub(time.Now()))
			game.context.current_state = GameStateLobby
			game.bm.Reset()
		}()
	case PacketTypeAvailableHosts:
		err := dec.Decode(&c.available_servers)
		if err != nil {
			log.Panic("error decoding new server state")
		}
	}
}
