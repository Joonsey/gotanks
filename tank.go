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
	x, y := camera.GetRelativePosition(t.X, t.Y)

	DrawStackedSprite(t.sprites, screen, x, y, t.rotation-camera.rotation)
	DrawStackedSprite(t.turret.sprites, screen, x, y-3, t.turret.rotation)
}

func (t *Tank) TryMove(g *Game, speed float64) {
	initial_position := t.Position
	x := math.Cos(t.rotation)
	y := math.Sin(t.rotation)

	t.X += x * speed
	t.Y += y * speed
	collided_object := g.level.CheckObjectCollision(t.Position)
	if collided_object != nil {
		t.Position = initial_position
	}

}

func (t *Tank) Update(g *Game) {
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		t.TryMove(g, SPEED)
	}

	if ebiten.IsKeyPressed(ebiten.KeyS) {
		t.TryMove(g, -SPEED)
	}

	if ebiten.IsKeyPressed(ebiten.KeyA) {
		t.rotation -= ROTATION_SPEED
	}

	if ebiten.IsKeyPressed(ebiten.KeyD) {
		t.rotation += ROTATION_SPEED
	}

	x, y := ebiten.CursorPosition()

	rel_x, rel_y := g.camera.GetRelativePosition(t.X, t.Y)

	t.turret.rotation = -math.Atan2(rel_x-float64(x), rel_y-float64(y))
}
