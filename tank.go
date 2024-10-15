package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	SPEED          = 2
	ROTATION_SPEED = .04
)

type Turret struct {
	sprites  []*ebiten.Image
	rotation float64
}

type Tank struct {
	Position
	sprites  []*ebiten.Image
	rotation float64
	turret   Turret
}

func (t *Tank) Draw(screen *ebiten.Image, camera Camera) {
	x, y := t.X-camera.Offset.X, t.Y-camera.Offset.Y
	DrawStackedSprite(t.sprites, screen, x, y, t.rotation)
	DrawStackedSprite(t.turret.sprites, screen, x, y-3, t.turret.rotation)
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

	x, y := ebiten.CursorPosition()
	t.turret.rotation = -math.Atan2(t.X-float64(x), t.Y-float64(y))
}
