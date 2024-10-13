package main

import (
	"image"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

const (
	RENDER_WIDTH = 320
	RENDER_HEIGHT = 240

	SCREEN_WIDTH = 640
	SCREEN_HEIGHT = 480

	SPRITE_SIZE = 16
)

type Position struct {
	X, Y int
}

func DrawStackedSprite(source []*ebiten.Image, screen *ebiten.Image, x, y int, rotation float64) {
	for i, image := range source {
		op := &ebiten.DrawImageOptions{}

		half_size := float64(SPRITE_SIZE / 2)
		// moving by half-size to rotate around the center
		op.GeoM.Translate(-half_size, -half_size)
		op.GeoM.Rotate(rotation)
		// moving back
		op.GeoM.Translate(half_size, half_size)

		op.GeoM.Translate(float64(x), float64(y-i))
		screen.DrawImage(image, op)
	}
}

func SplitSprites(source *ebiten.Image) []*ebiten.Image{
	count := source.Bounds().Dy() / SPRITE_SIZE
	sprites := []*ebiten.Image{}

	for i := 0; i < count; i++ {
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
	tank Tank
}

func (g *Game) Update() error {
	g.tank.rotation += .01
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.tank.Draw(screen)
}

func (g *Game) Layout(screenWidth, screenHeight int) (renderWidth, renderHeight int) {
	return RENDER_WIDTH, RENDER_HEIGHT
}

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("gotanks")

	img, _, err := ebitenutil.NewImageFromFile("assets/sprites/tank.png")
	if err != nil {
		log.Fatal(err)
	}

	tank := Tank{
		Position: Position{ 200, 200 },
		sprites: SplitSprites(img),
	}

	game := &Game{
		tank: tank,
	}

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
