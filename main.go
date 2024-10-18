package main

import (
	"image"
	"log"
	"math"

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

func DrawStackedSprite(source []*ebiten.Image, screen *ebiten.Image, x, y, rotation float64) {
	for i, image := range source {
		op := &ebiten.DrawImageOptions{}

		half_size := float64(SPRITE_SIZE / 2)
		// moving by half-size to rotate around the center
		op.GeoM.Translate(-half_size, -half_size)
		op.GeoM.Rotate(rotation - 90*math.Pi/180)
		// moving back
		op.GeoM.Translate(half_size, half_size)

		op.GeoM.Translate(x, y-float64(i))
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
	camera Camera
	time   float64
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
	g.time += 0.01

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.level.Draw(screen, g, g.camera)
	g.tank.Draw(screen, g.camera)
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

	tank := Tank{
		Position: Position{0, 0},
		sprites:  SplitSprites(img),
		turret: Turret{
			sprites: SplitSprites(turret_img),
		},
	}

	game := &Game{tank: tank}

	game.am = &AssetManager{}
	game.am.Init("temp.json")
	game.level = loadLevel("assets/tiled/level_1.tmx", game.am)

	temp_spawn_obj := game.level.spawns[0]
	game.tank.Position = Position{temp_spawn_obj.X, temp_spawn_obj.Y}

	game.camera.rotation = -46*math.Pi/180

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
