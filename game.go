package game

import (
	"errors"
	"fmt"
	"gotanks/shared"
	"image"
	"image/color"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	RENDER_WIDTH  = 500
	RENDER_HEIGHT = 380

	SCREEN_WIDTH  = 1280
	SCREEN_HEIGHT = 960

	SPRITE_SIZE = 16

	AMOUNT_OF_STRIPES = 22
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
	levels         []Level

	// TODO refactor
	new_level_time time.Time
	game_over_time time.Time

	available_servers []shared.AvailableServer
	current_state     GameStateEnum
	current_selection int
	isReady           bool
	background_time   int
	current_level     int

	current_server *shared.AvailableServer
}

type Game struct {
	tank   Tank
	am     *AssetManager
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

var stripe_texture = ebiten.NewImage(SCREEN_WIDTH, SCREEN_HEIGHT)

var player_palette = []color.Color{
	color.RGBA{R: 255, G: 85, B: 85, A: alpha},
	color.RGBA{R: 255, G: 85, B: 255, A: alpha},
	color.RGBA{R: 85, G: 255, B: 85, A: alpha},
	color.RGBA{R: 255, G: 255, B: 85, A: alpha},
}

var STRIPE_COLOR = color.RGBA{R: 0, G: 107, B: 107, A: 255}
var PLAYER_COLOR = color.RGBA{R: 85, G: 85, B: 255, A: alpha}
var MISSING_PLAYER_COLOR = color.RGBA{R: 175, G: 175, B: 175, A: 255}

func (g *Game) CurrentLevel() *Level {
	return &g.context.levels[g.context.current_level]
}

func (g *Game) DrawStackedSpriteDrawData(screen *ebiten.Image, data DrawData) {
	if data.r != 0 || data.g != 0 || data.b != 0 {
		if data.sprite != nil {
			op := &ebiten.DrawImageOptions{}
			x := data.position.X + data.offset.X
			y := data.position.Y + data.offset.Y
			op.GeoM.Translate(-float64(data.sprite.Bounds().Dx())/2, -float64(data.sprite.Bounds().Dy())/2)
			op.GeoM.Rotate(data.rotation - 90*math.Pi/180)
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
			op.GeoM.Rotate(data.rotation - 90*math.Pi/180)
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

func (g *Game) Reset() {
	g.bm.Reset()
	g.pm.Reset()
	g.context.tracks = []Track{}
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

func (g *Game) HostServer() {
	name := CreateServerName()
	go StartServer(name, g.nm.mediator_addr)
	g.context.current_state = GameStateLobby
	g.context.current_server = &shared.AvailableServer{Ip: "127.0.0.1", Port: SERVERPORT, Name: name, Player_count: 0, Max_players: 2}
	g.nm.Connect(*g.context.current_server)
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

func PlayerReadyString(n uint) string {
	if NetBoolify(n) {
		return "R"
	}
	return "X"
}

func (g *Game) DrawStripes(screen *ebiten.Image) {
	screen.Fill(color.RGBA{R: 11, B: 11, G: 11, A: 255})

	offset_x := SCREEN_WIDTH / AMOUNT_OF_STRIPES / 2
	for i := range AMOUNT_OF_STRIPES + 1 {
		op := ebiten.DrawImageOptions{}
		x := offset_x * (i - 1) * 2
		op.GeoM.Translate(float64(x)+float64(g.context.background_time/4%(offset_x*2)), -SCREEN_WIDTH/2)
		op.GeoM.Rotate(45)
		screen.DrawImage(stripe_texture, &op)
	}
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

func (g *Game) InitStripeTexture() {
	vector.DrawFilledRect(stripe_texture, 0, 0, float32(SCREEN_WIDTH/AMOUNT_OF_STRIPES/2), SCREEN_HEIGHT, STRIPE_COLOR, true)
}

func GameInit(mediator_addr string) *Game {
	am := &AssetManager{}
	am.Init("temp.json")

	game := Game{}
	game.am = am

	game.InitStripeTexture()

	tank := Tank{
		TankMinimal:       TankMinimal{Position: Position{}, Life: 10, Rotation: 0.001},
		sprites_path:      "assets/sprites/stacks/tank.png",
		track_sprite:      am.GetSprites("assets/sprites/tracks.png")[0],
		dead_sprites_path: "assets/sprites/stacks/tank-broken.png",
		turret: Turret{
			sprites_path: "assets/sprites/stacks/tank-barrel.png",
		},
	}

	tank.Component.Set(LoaderMask, 1)
	tank.Component.Set(BarrelMask, 1)
	tank.Component.Set(BulletMask, 1)
	tank.Component.Set(TracksMask, 1)

	game.tank = tank
	game.tank.turret.rotation = &game.tank.Turret_rotation

	game.camera.rotation = -46 * math.Pi / 180

	game.sm = InitSaveManager()
	game.nm = InitNetworkManager(mediator_addr)
	game.pm = InitParticleManager(game.am)
	game.bm = InitBulletManager(game.nm, game.am, game.pm)

	game.nm.client.Auth = &game.sm.data.Player_ID

	for i := range LEVEL_COUNT {
		level_path := fmt.Sprintf("assets/tiled/level_%d.tmx", i+1)
		game.context.levels = append(game.context.levels, loadLevel(level_path, game.am, true))
	}

	game.context.current_state = GameStateMainMenu

	go game.nm.client.Listen()
	go game.nm.client.Loop(&game)

	return &game
}
