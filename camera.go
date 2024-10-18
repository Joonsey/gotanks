package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Camera struct {
	Offset   Position
	rotation float64
}

func (c *Camera) Update(target_pos Position) {
	coefficient := 10.0
	c.Offset.X += (target_pos.X - c.Offset.X) / coefficient
	c.Offset.Y += (target_pos.Y - c.Offset.Y) / coefficient

	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		c.rotation -= 0.01
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		c.rotation += 0.01
	}
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
