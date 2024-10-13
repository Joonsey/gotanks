package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	SPEED          = 2
	ROTATION_SPEED = .04
)

type Tank struct {
	Position
	sprites  []*ebiten.Image
	rotation float64
}

func (t *Tank) Draw(screen *ebiten.Image) {
	// we offset by 90 deg, because we are 'facing' to the right
	// as default degrees 0 is facing right, but drawn facing upward
	rotation := t.rotation + 90*math.Pi/180
	DrawStackedSprite(t.sprites, screen, t.X, t.Y, rotation)
}

func (t *Tank) Update() {
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		x := math.Cos(t.rotation)
		y := math.Sin(t.rotation)

		t.X += x * SPEED
		t.Y += y * SPEED
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		x := math.Cos(t.rotation)
		y := math.Sin(t.rotation)

		t.X += x * -SPEED
		t.Y += y * -SPEED
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		t.rotation -= ROTATION_SPEED
	}

	if ebiten.IsKeyPressed(ebiten.KeyD) {
		t.rotation += ROTATION_SPEED
	}
}
