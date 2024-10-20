package main

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	SPEED          = 2
	ROTATION_SPEED = .04
	TRACK_LIFETIME = 64
	TRACK_INTERVAL = 8
)

type Turret struct {
	sprites  []*ebiten.Image
	rotation float64
}

type Track struct {
	Position
	rotation float64
	lifetime int
}

type Tank struct {
	Position
	sprites  []*ebiten.Image
	rotation float64
	turret   Turret

	track_sprites []*ebiten.Image
}

func (t *Tank) GetDrawData(screen *ebiten.Image, g *Game, camera Camera) {
	x, y := camera.GetRelativePosition(t.X, t.Y)

	g.draw_data = append(g.draw_data, DrawData{t.sprites, Position{x, y}, t.rotation - camera.rotation, Position{}})
	g.draw_data = append(g.draw_data, DrawData{t.turret.sprites, Position{x, y + 1}, t.turret.rotation, Position{0, -4}})
	if int(g.time * 100) % TRACK_INTERVAL == 0 {
		g.tracks = append(g.tracks, Track{t.Position, t.rotation, TRACK_LIFETIME})
	}
}

func (t *Tank) DrawTurret(screen *ebiten.Image, camera Camera) {
	x, y := camera.GetRelativePosition(t.X, t.Y)
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

	g.gm.ApplyForce(t.X, t.Y)

	x, y := ebiten.CursorPosition()

	rel_x, rel_y := g.camera.GetRelativePosition(t.X, t.Y)

	t.turret.rotation = -math.Atan2(rel_x-float64(x), rel_y-float64(y))
}
