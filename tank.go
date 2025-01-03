package game

import (
	"gotanks/shared"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	SPEED          = 2
	ROTATION_SPEED = .04
	TRACK_LIFETIME = 80
	TRACK_INTERVAL = 3
	TURRET_HEIGHT  = 4

	// we need this because GOB fails to decode values which are '0'
	// so we need to do some tech to fix it
	TANK_DEAD_VALUE = -1
)

type Turret struct {
	sprites_path string
	rotation     *float64
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
	Component
	Rotation        float64
	Turret_rotation float64
	Life            int
}

type Tank struct {
	TankMinimal
	sprites_path string
	turret       Turret

	track_sprite      *ebiten.Image
	dead_sprites_path string

	ReloadTime           float64 // Time remaining for reload
	BulletsInMagazine    int
	MaxBulletsInMagazine int // Based on bullet configuration
	IsReloading          bool
}

func (t *TankMinimal) Alive() bool {
	return t.Life != -1
}

func (t *TankMinimal) Kill() {
	t.Life = -1
}

const RADIUS = 40

var RADI_SPRITE = ebiten.NewImage(RADIUS, RADIUS)

var polygonImage *ebiten.Image = ebiten.NewImage(1, 1)

func (t *Tank) GetDrawData(screen *ebiten.Image, g *Game, camera Camera, clr color.Color) {
	x, y := camera.GetRelativePosition(t.X, t.Y)
	RADI_SPRITE.Clear()
	radius := RADIUS

	if t.Alive() {
		g.context.draw_data = append(g.context.draw_data, DrawData{
			path:      t.sprites_path,
			position:  Position{x, y},
			rotation:  t.Rotation - camera.rotation,
			intensity: 1,
			opacity:   1},
		)
		g.context.draw_data = append(g.context.draw_data, DrawData{
			path:      t.turret.sprites_path,
			position:  Position{x, y + 1},
			rotation:  *t.turret.rotation,
			intensity: 1,
			offset:    Position{0, -TURRET_HEIGHT},
			opacity:   1},
		)
		if int(g.time*100)%TRACK_INTERVAL == 0 {
			g.context.tracks = append(g.context.tracks, Track{t.Position, t.Rotation, TRACK_LIFETIME})
		}
		vector.StrokeCircle(RADI_SPRITE, float32(radius)/2, float32(radius)/2, float32(radius)/4, 2, clr, true)
		rotationRad := t.Rotation

		offset := -math.Pi + camera.rotation

		// Top point calculation (rotation - 90 degrees)
		topX := math.Cos(rotationRad-math.Pi/2-offset) * 16
		topY := math.Sin(rotationRad-math.Pi/2-offset) * 16

		// Left point calculation (rotation degrees)
		leftX := math.Cos(rotationRad-offset) * 10
		leftY := math.Sin(rotationRad-offset) * 10

		// Right point calculation (rotation - 180 degrees)
		rightX := math.Cos(rotationRad-math.Pi-offset) * 10
		rightY := math.Sin(rotationRad-math.Pi-offset) * 10

		points := []Position{
			{
				X: topX + float64(radius/2),
				Y: topY + float64(radius/2),
			},
			{
				X: leftX + float64(radius/2),
				Y: leftY + float64(radius/2),
			},
			{
				X: rightX + float64(radius/2),
				Y: rightY + float64(radius/2),
			},
		}

		path := vector.Path{}
		path.MoveTo(float32(points[0].X), float32(points[0].Y))
		for _, p := range points[1:] {
			path.LineTo(float32(p.X), float32(p.Y))
		}
		path.Close()

		vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
		polygonImage.Fill(PLAYER_COLOR)

		RADI_SPRITE.DrawTriangles(vs, is, polygonImage, &ebiten.DrawTrianglesOptions{})

	} else {
		g.context.draw_data = append(g.context.draw_data, DrawData{
			path:      t.dead_sprites_path,
			position:  Position{x, y},
			rotation:  t.Rotation - camera.rotation,
			intensity: 1,
			opacity:   1},
		)
		vector.DrawFilledCircle(RADI_SPRITE, float32(radius)/2, float32(radius)/2, float32(radius)/4, color.RGBA{R: 0, G: 0, B: 0, A: 128}, true)
	}
	g.context.draw_data = append(g.context.draw_data, DrawData{
		sprite:    RADI_SPRITE,
		position:  Position{x, y - 1},
		rotation:  0,
		intensity: 1,
		offset:    Position{0, 1},
		opacity:   1})

}

func (t *Tank) TryMove(g *Game, speed float64) {
	initial_position := t.Position
	level := g.CurrentLevel()
	x := math.Cos(t.Rotation)
	y := math.Sin(t.Rotation)

	t.X += x * speed
	collided_object := level.CheckObjectCollision(t.Position)
	if collided_object != nil {
		t.Position.X = initial_position.X
	}

	t.Y += y * speed
	collided_object = level.CheckObjectCollision(t.Position)
	if collided_object != nil {
		t.Position.Y = initial_position.Y
	}

}

func (t *TankMinimal) TryAddSmoke(g *Game) {
	if int(g.time*100)%9 == 0 {
		offset := 4.
		pos := t.Position
		pos.Y -= offset
		pos.X -= offset
		g.pm.AddParticle(
			Particle{
				particle_type: ParticleTypeSmoke,
				Position:      pos,
				velocity:      0.2,
				sprite_path:   "assets/sprites/stacks/particle-cube-template.png",
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

	g.CurrentLevel().gm.ApplyForce(t.X, t.Y)

	loader_type := t.Get(LoaderMask)
	bullet_type := StandardBulletTypeEnum(t.Get(BulletMask))
	barrel_type := t.Get(BarrelMask)

	base_reload_speed := DetermineBaseReloadSpeed(bullet_type)
	reload_speed_multiplier := DetermineReloadSpeedMultiplier(loader_type)
	effective_reload_speed := base_reload_speed * reload_speed_multiplier

	switch loader_type {
	case LoaderAutoloader:
		// Autoloader logic: reload multiple bullets at once but takes longer
		if t.IsReloading {
			t.ReloadTime = t.ReloadTime - (0.06)
			if t.ReloadTime <= 0 {
				// Refill the magazine
				t.BulletsInMagazine = t.MaxBulletsInMagazine
				t.IsReloading = false
			}
		} else if t.BulletsInMagazine <= 0 {
			t.StartReload(effective_reload_speed) // Example: 3 seconds to reload with autoloader
		}

	case LoaderFastReload:
		// Fast reload logic: faster single-bullet reload
		if t.IsReloading {
			t.ReloadTime = t.ReloadTime - (0.06)
			if t.ReloadTime <= 0 {
				// Reload one bullet
				t.BulletsInMagazine++
				t.IsReloading = false
			}
		} else if t.BulletsInMagazine < t.MaxBulletsInMagazine {
			t.StartReload(effective_reload_speed) // Example: 1 second per bullet
		}

	case LoaderManualReload:
		// Manual reload logic: requires skill-check
		if t.IsReloading {
			// Simulate skill-check mechanism (placeholder logic)
			t.ReloadTime = t.ReloadTime - (0.06)
			if t.ReloadTime <= 0 {
				t.BulletsInMagazine++
				t.IsReloading = false
			}
		} else if t.BulletsInMagazine < t.MaxBulletsInMagazine {
			t.StartReload(effective_reload_speed) // Example: base reload time for manual reload
		}
	}

	x, y := ebiten.CursorPosition()

	rel_x, rel_y := g.camera.GetRelativePosition(t.X, t.Y)
	rel_rotation := -math.Atan2(rel_x-float64(x), rel_y-float64(y))
	t.Turret_rotation = rel_rotation

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButton0) && t.Alive() && t.Fire() {
		bullet_pos := t.Position

		// offsetting bullet position
		// this is nessecary because we are shooting from the barrel
		// not the base
		bullet_pos.Y += TURRET_HEIGHT

		bullet := StandardBullet{}
		bullet.Position = bullet_pos
		bullet.Rotation = -rel_rotation + -g.camera.rotation + math.Pi
		bullet.Bullet_type = bullet_type

		// TODO consider moving this initialization to server side
		// for 'security' / 'integrity'
		bullet.Num_bounces = DetermineNumBounces(bullet.Bullet_type) + DetermineAdditionalBounces(barrel_type)
		bullet.Velocity = DetermineVelocity(bullet.Bullet_type) * DetermineVelocityMultiplier(barrel_type)
		g.bm.Shoot(bullet)
	}

	if g.nm.client.isConnected() {
		if int(g.time*100)%UPDATE_INTERVAL == 0 {
			go g.nm.client.Send(shared.PacketTypeUpdateCurrentPlayer, t.TankMinimal)
		}
	}

	for _, player := range g.context.player_updates {
		g.CurrentLevel().gm.ApplyForce(player.Tank.X, player.Tank.Y)
	}
}

func (t *Tank) StartReload(reloadTime float64) {
	t.IsReloading = true
	t.ReloadTime = reloadTime
}

func (t *Tank) Fire() bool {
	loader_type := t.Get(LoaderMask)
	if t.BulletsInMagazine <= 0 {
		return false
	}

	switch loader_type {
	default:
		t.BulletsInMagazine--
		return true
	}
}

func (t *Tank) Reset() {
	loaderType := t.Get(LoaderMask)
	bulletType := StandardBulletTypeEnum(t.Get(BulletMask))

	t.MaxBulletsInMagazine = DetermineBaseMagSize(bulletType) * int(math.Floor(DetermineMaxMagMultiplier(loaderType)))
	t.Life = 10
	t.BulletsInMagazine = t.MaxBulletsInMagazine
	t.ReloadTime = 0
	t.IsReloading = false
}

func (t *Tank) Respawn(spawn Position) {
	t.Position = spawn

	// it can not be 0 as gob will fail if it's 0
	t.Rotation = 0.001

	t.Reset()
}

func (t *Tank) Hit(hit BulletHit) {
	if t.Alive() {
		t.Kill()
	}
}
