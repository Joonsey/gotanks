package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
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

// NOTE it's going to be annoing with gob not writing
// 'nil' values i.e (0, 0.0, and worst of all 'false')

// perhaps look for some way to deal with this in the future.
// but for now we just try to avoid values being 0

// a minimal struct for representing a tank.
// this is used for sending over the network
type TankMinimal struct {
	Position
	Rotation        float64
	Turret_rotation float64
	Life            int
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

func (t *Tank) GetDrawData(screen *ebiten.Image, g *Game, camera Camera, clr color.Color) {
	x, y := camera.GetRelativePosition(t.X, t.Y)

	if t.Alive() {
		g.context.draw_data = append(g.context.draw_data, DrawData{
			sprites:   t.sprites,
			position:  Position{x, y},
			rotation:  t.Rotation - camera.rotation,
			intensity: 1,
			opacity:   1},
		)
		g.context.draw_data = append(g.context.draw_data, DrawData{
			sprites:   t.turret.sprites,
			position:  Position{x, y + 1},
			rotation:  *t.turret.rotation,
			intensity: 1,
			offset:    Position{0, -TURRET_HEIGHT},
			opacity:   1},
		)
		if int(g.time*100)%TRACK_INTERVAL == 0 {
			g.context.tracks = append(g.context.tracks, Track{t.Position, t.Rotation, TRACK_LIFETIME})
		}
		radius := 20
		radi_sprite := ebiten.NewImage(radius, radius)
		vector.StrokeCircle(radi_sprite, float32(radius)/2, float32(radius)/2, float32(radius)/2, 2, clr, true)
		g.context.draw_data = append(g.context.draw_data, DrawData{
			sprites:   []*ebiten.Image{radi_sprite},
			position:  Position{x, y - 1},
			rotation:  t.Rotation,
			intensity: 1,
			offset:    Position{0, 1},
			opacity:   1})

	} else {
		g.context.draw_data = append(g.context.draw_data, DrawData{
			sprites:   t.dead_sprites,
			position:  Position{x, y},
			rotation:  t.Rotation - camera.rotation,
			intensity: 1,
			opacity:   1},
		)
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

func (t *TankMinimal) TryAddSmoke(g *Game) {
	if int(g.time*100)%13 == 0 {
		offset := 4.
		pos := t.Position
		pos.Y -= offset
		pos.X -= offset
		g.pm.AddParticle(
			Particle{
				particle_type: ParticleTypeSmoke,
				Position:      pos,
				velocity:      0.2,
				sprites:       g.am.GetSprites("assets/sprites/stacks/particle-cube-template.png"),
			})
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
	} else {
		t.TryAddSmoke(g)
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
		if int(g.time*100)%UPDATE_INTERVAL == 0 {
			go g.nm.client.Send(PacketTypeUpdateCurrentPlayer, t.TankMinimal)
		}
	}

	for _, player := range g.context.player_updates {
		g.gm.ApplyForce(player.Tank.X, player.Tank.Y)
	}
}

func (t *Tank) Respawn(spawn Position) {
	t.Position = spawn

	// it can not be 0 as gob will fail if it's 0
	t.Rotation = 0.001
	t.Life = 10
}

func (t *Tank) Hit(hit BulletHit) {
	if t.Alive() {
		t.Kill()
	}
}
