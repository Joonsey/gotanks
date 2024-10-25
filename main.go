package main

import (
	"image"
	"log"
	"math"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
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
}

func DrawStackedSpriteDrawData(screen *ebiten.Image, data DrawData) {
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

func DrawStackedSprite(source []*ebiten.Image, screen *ebiten.Image, x, y, rotation float64, intensity, opacity float32) {
	for i, image := range source {
		op := &ebiten.DrawImageOptions{}

		half_size := float64(SPRITE_SIZE / 2)
		// moving by half-size to rotate around the center
		op.GeoM.Translate(-half_size, -half_size)
		op.GeoM.Rotate(rotation - 90*math.Pi/180)
		// moving back
		op.GeoM.Translate(half_size, half_size)

		op.GeoM.Translate(x, y-float64(i))
		scale := ebiten.ColorScale{}
		scale.SetR(intensity)
		scale.SetG(intensity)
		scale.SetB(intensity)
		op.ColorScale.ScaleAlpha(opacity)
		op.ColorScale.ScaleWithColorScale(scale)
		screen.DrawImage(image, op)
	}
}

func SplitSprites(source *ebiten.Image) []*ebiten.Image {
	count := source.Bounds().Dy() / SPRITE_SIZE
	sprites := []*ebiten.Image{}

	for i := count - 1; i > 0; i-- {
		rect := image.Rectangle{}
		rect.Min.X = 0
		rect.Max.X = 16
		rect.Min.Y = i * SPRITE_SIZE
		rect.Max.Y = (1 + i) * SPRITE_SIZE
		sprites = append(sprites, ebiten.NewImageFromImage(source.SubImage(rect)))
	}

	return sprites
}

type Game struct {
	tank   Tank
	level  Level
	am     *AssetManager
	gm     *GrassManager
	camera Camera
	time   float64

	draw_data []DrawData
	tracks    []Track
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
	g.gm.GetDrawData(screen, g)

	for _, track := range g.tracks {
		x, y := g.camera.GetRelativePosition(track.X, track.Y)
		offset := float64(8)
		opacity := float32(track.lifetime) / float32(TRACK_LIFETIME)
		g.draw_data = append(g.draw_data, DrawData{g.tank.track_sprites, Position{x, y - offset}, track.rotation - g.camera.rotation, 1, Position{0, offset}, opacity})
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

	turret_img, _, err := ebitenutil.NewImageFromFile("assets/sprites/stacks/tank-barrel.png")
	if err != nil {
		log.Fatal(err)
	}

	track_img, _, err := ebitenutil.NewImageFromFile("assets/sprites/tracks.png")
	if err != nil {
		log.Fatal(err)
	}

	tank := Tank{
		Position:      Position{0, 0},
		sprites:       SplitSprites(img),
		track_sprites: []*ebiten.Image{track_img},
		turret: Turret{
			sprites: SplitSprites(turret_img),
		},
	}

	game := &Game{tank: tank}

	game.am = &AssetManager{}
	game.am.Init("temp.json")

	game.camera.rotation = -46 * math.Pi / 180

	game.gm = &GrassManager{}
	game.level = loadLevel("assets/tiled/level_1.tmx", game.am, game.gm)

	temp_spawn_obj := game.level.spawns[0]
	game.tank.Position = Position{temp_spawn_obj.X, temp_spawn_obj.Y}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
