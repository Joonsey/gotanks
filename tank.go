package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

const (
	SPEED          = 2
	ROTATION_SPEED = .04
	TRACK_LIFETIME = 64
	TRACK_INTERVAL = 8
	TURRET_HEIGHT  = 4

	// we need this because GOB fails to decode values which are '0'
	// so we need to do some tech to fix it
	TANK_DEAD_VALUE = -1
)

type Turret struct {
	sprites  []*ebiten.Image
	rotation *float64
}

type Track struct {
	Position
	rotation float64
	lifetime int
}

// this is suposed to be an interface for network transfer
// we extend it for 'real' use
type TankMinimal struct {
	Position
	Rotation        float64
	Turret_rotation float64
	Life			int
}

type Tank struct {
	TankMinimal
	sprites []*ebiten.Image
	turret  Turret

	track_sprites []*ebiten.Image
	dead_sprites  []*ebiten.Image
}

func (t *TankMinimal) Alive() bool {
	return t.Life != -1
}

func (t *TankMinimal) Kill() {
	t.Life = -1
}

func (t *Tank) GetDrawData(screen *ebiten.Image, g *Game, camera Camera) {
	x, y := camera.GetRelativePosition(t.X, t.Y)

	if t.Alive() {
		g.draw_data = append(g.draw_data, DrawData{t.sprites, Position{x, y}, t.Rotation - camera.rotation, 1, Position{}, 1})
		g.draw_data = append(g.draw_data, DrawData{t.turret.sprites, Position{x, y + 1}, *t.turret.rotation, 1, Position{0, -TURRET_HEIGHT}, 1})
		if int(g.time*100)%TRACK_INTERVAL == 0 {
			g.tracks = append(g.tracks, Track{t.Position, t.Rotation, TRACK_LIFETIME})
		}

	} else {
		g.draw_data = append(g.draw_data, DrawData{t.dead_sprites, Position{x, y}, t.Rotation - camera.rotation, 1, Position{}, 1})
	}
}

func (t *Tank) DrawTurret(screen *ebiten.Image, camera Camera) {
	x, y := camera.GetRelativePosition(t.X, t.Y)
	DrawStackedSprite(t.turret.sprites, screen, x, y-3, *t.turret.rotation, 1, 1)
}

func (t *Tank) TryMove(g *Game, speed float64) {
	initial_position := t.Position
	x := math.Cos(t.Rotation)
	y := math.Sin(t.Rotation)

	t.X += x * speed
	t.Y += y * speed
	collided_object := g.level.CheckObjectCollision(t.Position)
	if collided_object != nil {
		t.Position = initial_position
	}

}

func (t *Tank) Update(g *Game) {
	if t.Alive() {
		if ebiten.IsKeyPressed(ebiten.KeyW) {
			t.TryMove(g, SPEED)
		}

		if ebiten.IsKeyPressed(ebiten.KeyS) {
			t.TryMove(g, -SPEED)
		}

		if ebiten.IsKeyPressed(ebiten.KeyA) {
			t.Rotation -= ROTATION_SPEED
		}

		if ebiten.IsKeyPressed(ebiten.KeyD) {
			t.Rotation += ROTATION_SPEED
		}
	}

	g.gm.ApplyForce(t.X, t.Y)

	x, y := ebiten.CursorPosition()

	rel_x, rel_y := g.camera.GetRelativePosition(t.X, t.Y)
	rel_rotation := -math.Atan2(rel_x-float64(x), rel_y-float64(y))
	t.Turret_rotation = rel_rotation
	bullet_pos := t.Position

	// offsetting bullet position
	// this is nessecary because we are shooting from the barrel
	// not the base
	bullet_pos.Y += TURRET_HEIGHT

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) && t.Alive() {
		bullet := Bullet{}
		bullet.Position = bullet_pos
		bullet.Rotation = -rel_rotation + -g.camera.rotation + math.Pi
		bullet.Bullet_type = BulletTypeStandard

		g.bm.Shoot(bullet)
	}

	if g.nm.client.isConnected() {
		if int(g.time*100)%16 == 0 {
			go g.nm.client.Send(PacketTypeUpdateCurrentPlayer, t.TankMinimal)
		}
	}
}

func (t *Tank) Hit(hit BulletHit) {
	if t.Alive(){
		t.Kill()
	}
}
