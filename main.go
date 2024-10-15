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
}

func (g *Game) GetTargetCameraPosition() Position {
	targetX := g.tank.X - RENDER_WIDTH/2
	targetX = max(0, targetX)
	targetX = min(float64(g.level.tiled_map.Width*SPRITE_SIZE - RENDER_WIDTH), targetX)

	targetY := g.tank.Y - RENDER_HEIGHT/2
	targetY = max(0, targetY)
	targetY = min(float64(g.level.tiled_map.Height*SPRITE_SIZE - RENDER_HEIGHT), targetY)

	// TODO should perhaps offset by relative mouse position to give the illusion of 'zoom' or 'focus'

	return Position{targetX, targetY}
}

func (g *Game) Update() error {
	g.tank.Update()
	g.camera.Update(g.GetTargetCameraPosition())
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.level.Draw(screen, g.camera)
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

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
