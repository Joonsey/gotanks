package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type Camera struct {
	Offset   Position
	rotation float64
}

// does nothing
const RotationStep = 0.0174533

func (c *Camera) Update(target_pos Position) {
	coefficient := 10.0
	c.Offset.X += (target_pos.X - c.Offset.X) / coefficient
	c.Offset.Y += (target_pos.Y - c.Offset.Y) / coefficient

	if inpututil.IsKeyJustPressed(ebiten.KeyQ) {
		c.rotation -= 1.0 * (math.Pi / 180)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyE) {
		c.rotation += 1.0 * (math.Pi) / 180
	}

	// TODO fix camera rendering bug
	// causing the black tiny artifacts
	c.rotation = snapToInterval(c.rotation, RotationStep)

}

func snapToInterval(value, step float64) float64 {
	return math.Round(value/step) * step
}

func (c *Camera) GetCameraDrawOptions() *ebiten.DrawImageOptions {
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(-c.Offset.X, -c.Offset.Y)

	return &op
}

func (c *Camera) GetRelativePosition(abs_x, abs_y float64) (float64, float64) {

	translated_x := abs_x - c.Offset.X
	translated_y := abs_y - c.Offset.Y

	rel_x := translated_x*math.Cos(c.rotation) + translated_y*math.Sin(c.rotation)
	rel_y := -translated_x*math.Sin(c.rotation) + translated_y*math.Cos(c.rotation)

	return rel_x, rel_y
}
