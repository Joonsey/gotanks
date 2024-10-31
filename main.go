package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

const (
	RENDER_WIDTH  = 320
	RENDER_HEIGHT = 240

	SCREEN_WIDTH  = 640
	SCREEN_HEIGHT = 480

	SPRITE_SIZE = 16
)

type Position struct {
	X, Y float64
}

type DrawData struct {
	sprites   []*ebiten.Image
	position  Position
	rotation  float64
	intensity float32
	offset    Position
	opacity   float32

	r, g, b float32
}

// not to be confused with ServerGameStateEnum
type GameStateEnum int

const (
	GameStateLobby GameStateEnum = iota
	GameStatePlaying
	GameStateServerPicking
	GameStateMainMenu
)

type GameContext struct {
	// gameplay
	draw_data      []DrawData
	tracks         []Track
	player_updates []PlayerUpdate

	// TODO refactor
	new_level_time time.Time

	available_servers []AvailableServer
	current_state     GameStateEnum
	current_selection int
	isReady           bool

	current_server *AvailableServer
}

type Game struct {
	tank   Tank
	level  Level
	am     *AssetManager
	gm     *GrassManager
	nm     *NetworkManager
	bm     *BulletManager
	pm     *ParticleManager
	sm     *SaveManager
	camera Camera
	time   float64

	context GameContext
}

func DrawStackedSpriteDrawData(screen *ebiten.Image, data DrawData) {
	if data.r != 0 || data.g != 0 || data.b != 0 {
		DrawStackedSpriteWithColor(
			data.sprites,
			screen,
			data.position.X+data.offset.X,
			data.position.Y+data.offset.Y,
			data.rotation,
			data.r*data.intensity,
			data.g*data.intensity,
			data.b*data.intensity,
			data.opacity,
		)
	} else {
		DrawStackedSprite(
			data.sprites,
			screen,
			data.position.X+data.offset.X,
			data.position.Y+data.offset.Y,
			data.rotation,
			data.intensity,
			data.opacity,
		)
	}
}

func DrawStackedSprite(source []*ebiten.Image, screen *ebiten.Image, x, y, rotation float64, intensity, opacity float32) {
	DrawStackedSpriteWithColor(source, screen, x, y, rotation, intensity, intensity, intensity, opacity)
}

func DrawStackedSpriteWithColor(source []*ebiten.Image, screen *ebiten.Image, x, y, rotation float64, r, g, b, opacity float32) {
	half_size := float64(source[0].Bounds().Dx()) / 2

	for i, image := range source {
		op := &ebiten.DrawImageOptions{}

		// moving by half-size to rotate around the center
		op.GeoM.Translate(-half_size, -half_size)
		op.GeoM.Rotate(rotation - 90*math.Pi/180)
		// moving back
		op.GeoM.Translate(half_size, half_size)

		op.GeoM.Translate(x, y-float64(i))
		scale := ebiten.ColorScale{}
		scale.SetR(r)
		scale.SetG(g)
		scale.SetB(b)
		op.ColorScale.ScaleAlpha(opacity)
		op.ColorScale.ScaleWithColorScale(scale)
		screen.DrawImage(image, op)
	}
}

func SplitSprites(source *ebiten.Image) []*ebiten.Image {
	width := source.Bounds().Dx()
	count := source.Bounds().Dy() / width
	sprites := []*ebiten.Image{}

	for i := count - 1; i >= 0; i-- {
		rect := image.Rectangle{}
		rect.Min.X = 0
		rect.Max.X = width
		rect.Min.Y = i * width
		rect.Max.Y = (1 + i) * width
		sprites = append(sprites, ebiten.NewImageFromImage(source.SubImage(rect)))
	}

	return sprites
}

// TODO refactor
func (g *Game) DrawNewLevelTimer(screen *ebiten.Image) {
	textOp := text.DrawOptions{}
	t := g.context.new_level_time.Sub(time.Now())
	msg := fmt.Sprintf("New round in %.2f!", max(t.Seconds(), 0))
	fontSize := 8.
	textOp.GeoM.Translate(RENDER_WIDTH/2, RENDER_HEIGHT-fontSize*4)
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
}

func (g *Game) GetTargetCameraPosition() Position {
	// TODO should perhaps offset by relative mouse position to give the illusion of 'zoom' or 'focus'
	targetX := float64(RENDER_WIDTH) / 2
	targetY := float64(RENDER_HEIGHT) / 2

	// Step 2: Apply rotation to the camera's offset
	rotated_x := targetX*math.Cos(g.camera.rotation) - targetY*math.Sin(g.camera.rotation)
	rotated_y := targetX*math.Sin(g.camera.rotation) + targetY*math.Cos(g.camera.rotation)

	// Step 3: Calculate the final camera position by adding the rotated offset to the tank's position
	return Position{
		X: g.tank.X - rotated_x,
		Y: g.tank.Y - rotated_y,
	}
}

func (g *Game) UpdateGameplay() error {
	g.tank.Update(g)
	g.camera.Update(g.GetTargetCameraPosition())
	g.gm.Update(g)
	g.bm.Update(&g.level, g)
	g.pm.Update(g)
	g.time += 0.01

	tracks := []Track{}
	for _, track := range g.context.tracks {
		track.lifetime--
		if track.lifetime >= 0 {
			tracks = append(tracks, track)
		}
	}

	g.context.tracks = tracks

	return nil
}

func (g *Game) UpdateLobby() error {
	g.nm.client.KeepAlive(g)

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.nm.client.Send(PacketTypeClientToggleReady, []byte{})
	}

	if !g.nm.client.isConnected() {
		g.context.current_selection = 0
		g.context.current_state = GameStateServerPicking
	}

	return nil
}

func (g *Game) UpdateServerPicking() error {
	g.context.available_servers = g.nm.client.GetServerList(g)

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.context.current_selection++
		if g.context.current_selection >= len(g.context.available_servers)+1 {
			g.context.current_selection = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.context.current_selection--
		if g.context.current_selection < 0 {
			g.context.current_selection = len(g.context.available_servers)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.context.current_selection == 0 {
			g.context.current_state = GameStateMainMenu
			g.context.current_server = nil
		} else if g.context.current_selection < len(g.context.available_servers)+1 {
			server := g.context.available_servers[g.context.current_selection-1]
			g.nm.client.Connect(server.Ip, server.Port)
			g.context.current_server = &server
			g.context.current_state = GameStateLobby
		}
	}
	return nil
}

func (g *Game) HostServer() {
	go StartServer()
	g.nm.client.Connect("127.0.0.1", SERVERPORT)
	g.context.current_state = GameStateLobby
	g.context.current_server = &AvailableServer{Ip: "127.0.0.1", Port: SERVERPORT, Name: "Localhost", Player_count: 0, Max_players: 2}
}

func (g *Game) UpdateMainMenu() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.context.current_selection++
		if g.context.current_selection >= 2 {
			g.context.current_selection = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.context.current_selection--
		if g.context.current_selection < 0 {
			g.context.current_selection = 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.context.current_selection == 0 {
			g.context.current_state = GameStateServerPicking
		}
		if g.context.current_selection == 1 {
			g.HostServer()
		}
	}
	return nil
}

func (g *Game) Update() error {
	var err error = nil
	switch g.context.current_state {
	case GameStatePlaying:
		err = g.UpdateGameplay()
	case GameStateLobby:
		err = g.UpdateLobby()
	case GameStateMainMenu:
		err = g.UpdateMainMenu()
	case GameStateServerPicking:
		err = g.UpdateServerPicking()
	default:
		err = errors.New("invalid state")
	}
	return err
}

func (g *Game) DrawGameplay(screen *ebiten.Image) {
	g.level.GetDrawData(screen, g, g.camera)
	g.tank.GetDrawData(screen, g, g.camera)
	g.gm.GetDrawData(g)
	g.bm.GetDrawData(g)
	g.pm.GetDrawData(g)
	if g.nm.client.isConnected() {
		g.nm.GetDrawData(g)

		if g.nm.client.server_state == ServerGameStateStartingNewRound {
			defer g.DrawNewLevelTimer(screen)
		}

	}

	for _, track := range g.context.tracks {
		x, y := g.camera.GetRelativePosition(track.X, track.Y)
		offset := float64(8)
		opacity := float32(track.lifetime) / float32(TRACK_LIFETIME)
		g.context.draw_data = append(g.context.draw_data, DrawData{
			sprites:   g.tank.track_sprites,
			position:  Position{x, y - offset},
			rotation:  track.rotation - g.camera.rotation,
			intensity: 1,
			offset:    Position{0, offset},
			opacity:   opacity})
	}

	sort.Slice(g.context.draw_data, func(i, j int) bool {
		i_obj := g.context.draw_data[i]
		j_obj := g.context.draw_data[j]
		// Compare the transformed Y values
		return i_obj.position.Y < j_obj.position.Y
	})

	for _, data := range g.context.draw_data {
		DrawStackedSpriteDrawData(screen, data)
	}

	g.context.draw_data = []DrawData{}

}

func PlayerReadyString(n uint) string {
	if NetBoolify(n) {
		return "ready!"
	}
	return "not ready!"
}

func (g *Game) DrawLobby(screen *ebiten.Image) {
	if g.context.current_server == nil {
		log.Panic("current server is nil")
	}
	fontSize := 8.

	textOp := text.DrawOptions{}
	msg := fmt.Sprintf("server name: '%s'", g.context.current_server.Name)
	textOp.GeoM.Translate(RENDER_WIDTH/2, float64(1)*fontSize)
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)

	for i, player := range g.context.player_updates {
		textOp := text.DrawOptions{}
		msg := fmt.Sprintf("%s is %s", player.ID[0:6], PlayerReadyString(player.Ready))
		textOp.GeoM.Translate(RENDER_WIDTH/2, float64(i+3)*fontSize)
		textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
		text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
	}
}

func (g *Game) DrawMainMenu(screen *ebiten.Image) {
	textOp := text.DrawOptions{}
	msg := "  join game"
	if g.context.current_selection == 0 {
		msg = "* join game"
	}
	fontSize := 8.
	textOp.GeoM.Translate(RENDER_WIDTH/2, RENDER_HEIGHT/2+fontSize*3)
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)

	textOp = text.DrawOptions{}
	msg = "  host"
	if g.context.current_selection == 1 {
		msg = "* host"
	}
	textOp.GeoM.Translate(RENDER_WIDTH/2, RENDER_HEIGHT/2+fontSize*5)
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
}

func (g *Game) DrawServerPicking(screen *ebiten.Image) {
	for i, server := range g.context.available_servers {
		textOp := text.DrawOptions{}
		msg := fmt.Sprintf("  %-10s| %d/%d", server.Name, server.Player_count, server.Max_players)
		if i+1 == g.context.current_selection {
			msg = fmt.Sprintf("* %-10s| %d/%d", server.Name, server.Player_count, server.Max_players)
		}
		fontSize := 8.
		textOp.GeoM.Translate(RENDER_WIDTH/2, float64(i)*fontSize)
		textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
		text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
	}

	msg := "  back to menu"
	if g.context.current_selection == 0 {
		msg = "* back to menu"
	}
	fontSize := 8.
	textOp := text.DrawOptions{}
	textOp.GeoM.Translate(RENDER_WIDTH/2, RENDER_HEIGHT-(fontSize*3))
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
}

func (g *Game) Draw(screen *ebiten.Image) {
	switch g.context.current_state {
	case GameStatePlaying:
		g.DrawGameplay(screen)
	case GameStateLobby:
		g.DrawLobby(screen)
	case GameStateMainMenu:
		g.DrawMainMenu(screen)
	case GameStateServerPicking:
		g.DrawServerPicking(screen)
	}
}

func (g *Game) Layout(screenWidth, screenHeight int) (renderWidth, renderHeight int) {
	return RENDER_WIDTH, RENDER_HEIGHT
}

func GameInit() *Game {
	am := &AssetManager{}
	am.Init("temp.json")

	game := Game{}
	game.am = am

	tank_sprite := am.GetSprites("assets/sprites/stacks/tank.png")
	tank_broken_sprite := am.GetSprites("assets/sprites/stacks/tank-broken.png")
	tank_barrel_sprite := am.GetSprites("assets/sprites/stacks/tank-barrel.png")
	track_sprite := am.GetSprites("assets/sprites/tracks.png")

	tank := Tank{
		TankMinimal:   TankMinimal{Position: Position{}, Life: 10, Rotation: 0.001},
		sprites:       tank_sprite,
		track_sprites: track_sprite,
		dead_sprites:  tank_broken_sprite,
		turret: Turret{
			sprites: tank_barrel_sprite,
		},
	}

	game.tank = tank
	game.tank.turret.rotation = &game.tank.Turret_rotation

	game.camera.rotation = -46 * math.Pi / 180

	game.sm = InitSaveManager()
	game.nm = InitNetworkManager()
	game.pm = InitParticleManager(game.am)
	game.bm = InitBulletManager(game.nm, game.am, game.pm)

	game.nm.client.Auth = &game.sm.data.Player_ID

	game.gm = &GrassManager{}
	game.level = loadLevel("assets/tiled/level_1.tmx", game.am, game.gm)

	temp_spawn_obj := game.level.spawns[0]
	game.tank.Position = Position{temp_spawn_obj.X, temp_spawn_obj.Y}

	game.context.current_state = GameStateMainMenu
	return &game
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("gotanks")

	start_server := flag.Bool("server", false, "start server")
	force_new_id := flag.Bool("f", false, "force new id")
	flag.Parse()

	game := GameInit()

	if game.sm.IsFresh() || *force_new_id {
		game.sm.data.Player_ID = uuid.New()
		game.sm.Save()
	}

	if *start_server {
		game.HostServer()
	}

	go game.nm.client.Listen()
	go game.nm.client.Loop(game)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
