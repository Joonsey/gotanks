package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Tank struct {
	Position
	sprites  []*ebiten.Image
	rotation float64
}

func (t *Tank) Draw(screen *ebiten.Image) {
	DrawStackedSprite(t.sprites, screen, t.X, t.Y, t.rotation)
}
