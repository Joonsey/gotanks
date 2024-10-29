package main

import (
	"fmt"
	"image"
	"log"
	"math"
	"sort"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
	GameStatePlaying GameStateEnum = iota
	GameStateWaiting
	GameStateStarting
)

type Game struct {
	tank   Tank
	level  Level
	am     *AssetManager
	gm     *GrassManager
	nm     *NetworkManager
	bm     *BulletManager
	pm     *ParticleManager
	camera Camera
	time   float64

	draw_data []DrawData
	tracks    []Track

	player_updates []PlayerUpdate

	// TODO refactor
	new_level_time time.Time
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

	for i := count - 1; i > 0; i-- {
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
	t := g.new_level_time.Sub(time.Now())
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

func (g *Game) Update() error {
	g.tank.Update(g)
	g.camera.Update(g.GetTargetCameraPosition())
	g.gm.Update(g)
	g.bm.Update(&g.level, g)
	g.pm.Update(g)
	g.time += 0.01

	tracks := []Track{}
	for _, track := range g.tracks {
		track.lifetime--
		if track.lifetime >= 0 {
			tracks = append(tracks, track)
		}
	}

	g.tracks = tracks

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
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

	for _, track := range g.tracks {
		x, y := g.camera.GetRelativePosition(track.X, track.Y)
		offset := float64(8)
		opacity := float32(track.lifetime) / float32(TRACK_LIFETIME)
		g.draw_data = append(g.draw_data, DrawData{
			sprites:   g.tank.track_sprites,
			position:  Position{x, y - offset},
			rotation:  track.rotation - g.camera.rotation,
			intensity: 1,
			offset:    Position{0, offset},
			opacity:   opacity})
	}

	sort.Slice(g.draw_data, func(i, j int) bool {
		i_obj := g.draw_data[i]
		j_obj := g.draw_data[j]
		// Compare the transformed Y values
		return i_obj.position.Y < j_obj.position.Y
	})

	for _, data := range g.draw_data {
		DrawStackedSpriteDrawData(screen, data)
	}

	g.draw_data = []DrawData{}

}

func (g *Game) Layout(screenWidth, screenHeight int) (renderWidth, renderHeight int) {
	return RENDER_WIDTH, RENDER_HEIGHT
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("gotanks")

	img, _, err := ebitenutil.NewImageFromFile("assets/sprites/stacks/tank.png")
	if err != nil {
		log.Fatal(err)
	}

	dead_tank_img, _, err := ebitenutil.NewImageFromFile("assets/sprites/stacks/tank-broken.png")
	if err != nil {
		log.Fatal(err)
	}

	turret_img, _, err := ebitenutil.NewImageFromFile("assets/sprites/stacks/tank-barrel.png")
	if err != nil {
		log.Fatal(err)
	}

	track_img, _, err := ebitenutil.NewImageFromFile("assets/sprites/tracks.png")
	if err != nil {
		log.Fatal(err)
	}

	tank := Tank{
		TankMinimal:   TankMinimal{Position: Position{}, Life: 10, Rotation: 0.001},
		sprites:       SplitSprites(img),
		track_sprites: []*ebiten.Image{track_img},
		dead_sprites:  SplitSprites(dead_tank_img),
		turret: Turret{
			sprites: SplitSprites(turret_img),
		},
	}

	game := &Game{tank: tank}
	// this needs to be after game is constructed
	// go does something funny when we ask for a pointer to Game
	// and actually gives Game a copy of tank, not the same tank instance
	game.tank.turret.rotation = &game.tank.Turret_rotation

	game.am = &AssetManager{}
	game.am.Init("temp.json")

	game.camera.rotation = -46 * math.Pi / 180

	game.nm = InitNetworkManager()
	game.pm = InitParticleManager(game.am)
	game.bm = InitBulletManager(game.nm, game.am, game.pm)

	game.gm = &GrassManager{}
	game.level = loadLevel("assets/tiled/level_1.tmx", game.am, game.gm)

	temp_spawn_obj := game.level.spawns[0]
	game.tank.Position = Position{temp_spawn_obj.X, temp_spawn_obj.Y}

	go StartServer()

	go game.nm.client.Listen()
	go game.nm.client.Loop(game)

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
