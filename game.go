package game

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	RENDER_WIDTH  = 500
	RENDER_HEIGHT = 380

	SCREEN_WIDTH  = 1280
	SCREEN_HEIGHT = 960

	SPRITE_SIZE = 16
)

type Position struct {
	X, Y float64
}

type DrawData struct {
	sprite *ebiten.Image
	// will override sprite stack caching
	// should only be set IF:
	//		- is not sprite stack
	//		- have to respect draw order
	//		- is only one image
	path      string
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
	GameStateTankLoadout
)

type GameContext struct {
	// gameplay
	draw_data      []DrawData
	tracks         []Track
	player_updates []PlayerUpdate

	// TODO refactor
	new_level_time time.Time
	game_over_time time.Time

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

const (
	alpha = 200
)

var player_palette = []color.Color{
	color.RGBA{R: 255, G: 85, B: 85, A: alpha},
	color.RGBA{R: 255, G: 85, B: 255, A: alpha},
	color.RGBA{R: 85, G: 255, B: 85, A: alpha},
	color.RGBA{R: 255, G: 255, B: 85, A: alpha},
}

var PLAYER_COLOR = color.RGBA{R: 85, G: 85, B: 255, A: alpha}

func (g *Game) DrawStackedSpriteDrawData(screen *ebiten.Image, data DrawData) {
	if data.r != 0 || data.g != 0 || data.b != 0 {
		if data.sprite != nil {
			op := &ebiten.DrawImageOptions{}
			x := data.position.X + data.offset.X
			y := data.position.Y + data.offset.Y
			op.GeoM.Translate(-float64(data.sprite.Bounds().Dx())/2, -float64(data.sprite.Bounds().Dy())/2)
			op.GeoM.Rotate(data.rotation - 90*math.Pi / 180)
			op.GeoM.Translate(x, y)
			scale := ebiten.ColorScale{}
			scale.SetR(data.r * data.intensity)
			scale.SetG(data.g * data.intensity)
			scale.SetB(data.b * data.intensity)
			op.ColorScale.ScaleAlpha(data.opacity)
			op.ColorScale.ScaleWithColorScale(scale)

			screen.DrawImage(data.sprite, op)
		} else {
			g.DrawStackedSpriteWithColor(
				data.path,
				screen,
				data.position.X+data.offset.X,
				data.position.Y+data.offset.Y,
				data.rotation,
				data.r*data.intensity,
				data.g*data.intensity,
				data.b*data.intensity,
				data.opacity,
			)
		}
	} else {
		if data.sprite != nil {
			op := &ebiten.DrawImageOptions{}
			x := data.position.X + data.offset.X
			y := data.position.Y + data.offset.Y
			op.GeoM.Translate(-float64(data.sprite.Bounds().Dx())/2, -float64(data.sprite.Bounds().Dy())/2)
			op.GeoM.Rotate(data.rotation - 90*math.Pi / 180)
			op.GeoM.Translate(x, y)
			scale := ebiten.ColorScale{}
			scale.SetR(data.intensity)
			scale.SetG(data.intensity)
			scale.SetB(data.intensity)
			op.ColorScale.ScaleAlpha(data.opacity)
			op.ColorScale.ScaleWithColorScale(scale)

			screen.DrawImage(data.sprite, op)
		} else {
			g.DrawStackedSprite(
				data.path,
				screen,
				data.position.X+data.offset.X,
				data.position.Y+data.offset.Y,
				data.rotation,
				data.intensity,
				data.opacity,
			)
		}
	}
}

func (g *Game) DrawStackedSprite(path string, screen *ebiten.Image, x, y, rotation float64, intensity, opacity float32) {
	g.DrawStackedSpriteWithColor(path, screen, x, y, rotation, intensity, intensity, intensity, opacity)
}

func (game *Game) DrawStackedSpriteWithColor(path string, screen *ebiten.Image, x, y, rotation float64, r, g, b, opacity float32) {
	game.am.DrawRotatedSprite(screen, path, x, y, rotation, r, g, b, opacity)
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

// TODO move to UI drawing
// TODO refactor
func (g *Game) DrawGameOver(screen *ebiten.Image) {
	textOp := text.DrawOptions{}
	t := g.context.game_over_time.Sub(time.Now())
	msg := fmt.Sprintf("Back to lobby in %.2f.", max(t.Seconds(), 0))
	fontSize := 8.
	textOp.GeoM.Translate(RENDER_WIDTH, RENDER_HEIGHT)
	textOp.GeoM.Translate(-float64(len(msg))*fontSize, -fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
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

func (g *Game) UpdateTankLoadout() error {
	if g.nm.client.isConnected() {
		g.nm.client.KeepAlive(g)
	}

	loader_type := g.tank.Get(LoaderMask)
	bullet_type := g.tank.Get(BulletMask)
	barrel_type := g.tank.Get(BarrelMask)
	track_type := g.tank.Get(TracksMask)

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.context.current_selection++
		if g.context.current_selection >= 5 {
			g.context.current_selection = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.context.current_selection--
		if g.context.current_selection < 0 {
			g.context.current_selection = 4
		}
	}

	incr_key := ebiten.KeyD
	switch g.context.current_selection {
	case 0:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.context.current_selection = 0
			if g.nm.client.isConnected() {
				g.context.current_state = GameStateLobby
			} else {
				g.context.current_state = GameStateMainMenu
			}
		}
	case 1:
		if inpututil.IsKeyJustPressed(incr_key) {
			g.tank.Set(LoaderMask, max((loader_type+1)%LoaderEnd, 1))
		}
	case 2:
		if inpututil.IsKeyJustPressed(incr_key) {
			g.tank.Set(BulletMask, max((bullet_type+1)%uint8(BulletTypeEnd), 1))
		}
	case 3:
		if inpututil.IsKeyJustPressed(incr_key) {
			g.tank.Set(BarrelMask, max((barrel_type+1)%BarrelEnd, 1))
		}
	case 4:
		if inpututil.IsKeyJustPressed(incr_key) {
			g.tank.Set(TracksMask, max((track_type+1)%TracksEnd, 1))
		}
	}

	// not great, but works
	g.tank.Reset()
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

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.context.current_selection = 0
		g.context.current_state = GameStateTankLoadout
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
			g.nm.client.Connect(server)
			g.context.current_server = &server
			g.context.current_state = GameStateLobby
		}
	}
	return nil
}

func (g *Game) HostServer() {
	name := CreateServerName()
	go StartServer(name)
	g.context.current_state = GameStateLobby
	g.context.current_server = &AvailableServer{Ip: "127.0.0.1", Port: SERVERPORT, Name: name, Player_count: 0, Max_players: 2}
	g.nm.client.Connect(*g.context.current_server)
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

func (g *Game) DrawPlayerUI(screen *ebiten.Image, player PlayerUpdate, num_players int, wins int, count int, font *text.GoTextFaceSource) {
	clr := player_palette[count%len(player_palette)]
	if g.nm.client.isSelf(player.ID) {
		clr = PLAYER_COLOR
	}
	width := RENDER_WIDTH / num_players
	fontSize := 8.
	vector.DrawFilledRect(screen, float32(width*count), 0, float32(width), float32(fontSize)*2, clr, true)

	textOp := text.DrawOptions{}
	name := player.ID
	if len(player.ID) > 6 {
		name = player.ID[0:6]
	}
	msg := fmt.Sprintf("%-6s | %d", name, wins)

	textOp.GeoM.Translate(float64(width*count+(width/2)), fontSize*2)
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, -fontSize*1.5)

	text.Draw(screen, msg, &text.GoTextFace{Source: font, Size: fontSize}, &textOp)
}

func (g *Game) DrawAmmo(screen *ebiten.Image) {
	bullet_sprites := g.am.GetSpriteFromBulletTypeEnum(BulletTypeEnum(g.tank.Get(BulletMask)))

	rel_x, rel_y := g.camera.GetRelativePosition(g.tank.X, g.tank.Y)

	x_offset := float64(-35)

	for i := range g.tank.MaxBulletsInMagazine {
		if i >= g.tank.BulletsInMagazine {
			g.am.DrawRotatedSprite(screen, bullet_sprites, rel_x+x_offset, rel_y-10*float64(i), math.Pi/2, .5, .5, .5, 1)
		} else {
			g.am.DrawRotatedSprite(screen, bullet_sprites, rel_x+x_offset, rel_y-10*float64(i), math.Pi/2, 1, 1, 1, 1)
		}
	}
}

func (g *Game) DrawUI(screen *ebiten.Image) {
	for count, player := range g.context.player_updates {
		g.DrawPlayerUI(screen, player, len(g.context.player_updates), g.nm.client.wins[player.ID], count, g.am.new_level_font)
	}

	g.DrawAmmo(screen)
}

func (g *Game) Update() error {
	var err error = nil
	switch g.context.current_state {
	case GameStatePlaying:
		err = g.UpdateGameplay()
	case GameStateLobby:
		err = g.UpdateLobby()
	case GameStateTankLoadout:
		err = g.UpdateTankLoadout()
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
	g.tank.GetDrawData(screen, g, g.camera, PLAYER_COLOR)
	g.gm.GetDrawData(g)
	g.bm.GetDrawData(g)
	g.pm.GetDrawData(g)
	if g.nm.client.isConnected() {
		g.nm.GetDrawData(g)
		defer g.DrawUI(screen)

		if g.nm.client.server_state == ServerGameStateStartingNewRound {
			defer g.DrawNewLevelTimer(screen)
		}
		if g.nm.client.server_state == ServerGameStateGameOver {
			defer g.DrawGameOver(screen)
		}
	}

	for _, track := range g.context.tracks {
		x, y := g.camera.GetRelativePosition(track.X, track.Y)
		offset := float64(8)
		opacity := float32(track.lifetime) / float32(TRACK_LIFETIME)
		g.context.draw_data = append(g.context.draw_data, DrawData{
			path:      g.tank.track_sprites_path,
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
		g.DrawStackedSpriteDrawData(screen, data)
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

	// TODO draw time until start when all are ready
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

func (g *Game) DrawTankLoadout(screen *ebiten.Image) {
	for i := range 4 {
		textOp := text.DrawOptions{}
		var msg string = ""
		switch i {
		case 0:
			msg = fmt.Sprintf("loadout: %s\n", DetermineLoaderName(g.tank.Get(LoaderMask)))
		case 1:
			msg = fmt.Sprintf("bullet: %s\n", DetermineBulletName(BulletTypeEnum(g.tank.Get(BulletMask))))
		case 2:
			msg = fmt.Sprintf("TEMP: bullet: %s\n", DetermineBulletName(BulletTypeEnum(g.tank.Get(BarrelMask))))
		case 3:
			msg = fmt.Sprintf("TEMP: bullet: %s\n", DetermineBulletName(BulletTypeEnum(g.tank.Get(TracksMask))))
		}

		if i+1 == g.context.current_selection {
			msg = fmt.Sprintf("* %s", msg)
		} else {
			msg = fmt.Sprintf("  %s", msg)
		}
		fontSize := 8.
		textOp.GeoM.Translate(RENDER_WIDTH/2, float64(i)*fontSize+200)
		textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
		text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
	}

	msg := "  back to menu"
	if g.nm.client.isConnected() {
		msg = "  back to lobby"
	}
	if g.context.current_selection == 0 {
		msg = fmt.Sprintf("* %s", msg)
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
	case GameStateTankLoadout:
		g.DrawTankLoadout(screen)
	}
}

func (g *Game) Layout(screenWidth, screenHeight int) (renderWidth, renderHeight int) {
	return RENDER_WIDTH, RENDER_HEIGHT
}

func (g *Game) SaveIsFresh() bool {
	return g.sm.IsFresh()
}

func (g *Game) GenerateNewPlayerId() {
	g.sm.data.Player_ID = uuid.New()
	g.sm.Save()
}

func GameInit() *Game {
	am := &AssetManager{}
	am.Init("temp.json")

	game := Game{}
	game.am = am

	tank := Tank{
		TankMinimal:        TankMinimal{Position: Position{}, Life: 10, Rotation: 0.001},
		sprites_path:       "assets/sprites/stacks/tank.png",
		track_sprites_path: "assets/sprites/tracks.png",
		dead_sprites_path:  "assets/sprites/stacks/tank-broken.png",
		turret: Turret{
			sprites_path: "assets/sprites/stacks/tank-barrel.png",
		},
	}

	tank.Component.Set(LoaderMask, LoaderAutoloader)
	tank.Component.Set(BarrelMask, BarrelHeavy)
	tank.Component.Set(BulletMask, uint8(BulletTypeStandard))
	tank.Component.Set(TracksMask, 1)

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

	go game.nm.client.Listen()
	go game.nm.client.Loop(&game)

	return &game
}
