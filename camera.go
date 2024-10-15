package main

import "github.com/hajimehoshi/ebiten/v2"

type Camera struct {
	Offset Position
}

func (c *Camera) Update(target_pos Position) {
	coefficient := 10.0
	c.Offset.X += (target_pos.X - c.Offset.X) / coefficient
	c.Offset.Y += (target_pos.Y - c.Offset.Y) / coefficient
}

func (c *Camera) GetCameraDrawOptions() *ebiten.DrawImageOptions {
	op := ebiten.DrawImageOptions{}
	op.GeoM.Translate(-c.Offset.X, -c.Offset.Y)

	return &op
}
