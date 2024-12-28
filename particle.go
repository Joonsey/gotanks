package game

import (
	"image"
	"image/color"
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type ParticleTypeEnum int

const (
	ParticleTypeDebrisFromTank ParticleTypeEnum = iota
	ParticleTypeSmoke
	ParticleTypeGunSmoke
	ParticleTypeTest
	ParticleTypeDonut
)

type Particle struct {
	Position
	Rotation    float64
	Offset_z    float64
	sprite_path string
	velocity    float64

	current_t float64
	max_t     float64
	intensity float64
	opacity   float64
	offset    Position

	seed     int64
	variance int
	shadow   bool

	particle_type ParticleTypeEnum

	r, g, b float32
}

type ParticleManager struct {
	am *AssetManager

	particles []*Particle
}

var PARTICLE_SPRITE_BUFFER = ebiten.NewImage(RENDER_WIDTH, RENDER_HEIGHT)

func InitParticleManager(am *AssetManager) *ParticleManager {
	pm := ParticleManager{}
	pm.particles = []*Particle{}

	pm.am = am
	return &pm
}

func (pm *ParticleManager) AddParticle(particle Particle) {
	if particle.intensity == 0 {
		particle.intensity = 1
	}
	if particle.opacity == 0 {
		particle.opacity = 1
	}

	if particle.max_t == 0 {
		particle.max_t = 120
	}

	if particle.sprite_path == "" {
		particle.sprite_path = "assets/sprites/stacks/particle-cube-template.png"
	}

	particle.seed = time.Now().UnixMilli()
	particle.shadow = true

	switch particle.particle_type {
	case ParticleTypeDebrisFromTank:
		particle.r = 1
		particle.g = .5
		particle.b = 0
	case ParticleTypeSmoke:
		particle.variance = 4
	case ParticleTypeDonut:
		particle.shadow = true

		// offset y - shadow margin
		particle.Offset_z = -16
		particle.offset.Y = -20

	}
	pm.particles = append(pm.particles, &particle)
}

func (pm *ParticleManager) Reset() {
	pm.particles = []*Particle{}
}

func (p *Particle) GetDrawData(camera Camera) DrawData {
	x, y := camera.GetRelativePosition(p.X, p.Y)
	switch p.particle_type {
	case ParticleTypeDonut:
		radius := max(p.velocity*p.current_t, 1)
		margin := 10
		PARTICLE_SPRITE_BUFFER.Clear()

		f32_radius := float32(radius)
		f32_margin := float32(margin)
		inner_color := color.RGBA{R: 255, G: 85, B: 85, A: 255}
		outer_color := color.White
		vector.StrokeCircle(PARTICLE_SPRITE_BUFFER, f32_radius+f32_margin, f32_radius+f32_margin, f32_radius, 6, outer_color, false)
		vector.StrokeCircle(PARTICLE_SPRITE_BUFFER, f32_radius+f32_margin, f32_radius+f32_margin+1, f32_radius, 6, outer_color, false)
		vector.StrokeCircle(PARTICLE_SPRITE_BUFFER, f32_radius+f32_margin, f32_radius+f32_margin, f32_radius, 4, inner_color, false)

		return DrawData{
			sprite:    PARTICLE_SPRITE_BUFFER.SubImage(image.Rect(0, 0, int(f32_radius+f32_margin)*2, int(f32_radius+f32_margin)*2)).(*ebiten.Image),
			position:  Position{x, y - p.Offset_z},
			rotation:  90 * math.Pi / 180,
			intensity: float32(p.intensity),
			offset:    p.offset,
			opacity:   float32(p.opacity),
			r:         p.r,
			g:         p.g,
			b:         p.b,
		}
	default:
		return DrawData{
			path:      p.sprite_path,
			position:  Position{x, y - p.Offset_z},
			rotation:  p.Rotation - camera.rotation,
			intensity: float32(p.intensity),
			offset:    p.offset,
			opacity:   float32(p.opacity),
			r:         p.r,
			g:         p.g,
			b:         p.b,
		}
	}
}

func interpolateColor(r_1, g_1, b_1, r_2, g_2, b_2, t float32) (float32, float32, float32) {
	r := r_1 + t*(r_2-r_1)
	g := g_1 + t*(g_2-g_1)
	b := b_1 + t*(b_2-b_1)
	return r, g, b
}

func exponentialDecay(start_value, decay_rate, time float64) float64 {
	return start_value * math.Exp(-decay_rate*time)
}

func (p *Particle) GetDrawShadowData(camera Camera) DrawData {
	x, y := camera.GetRelativePosition(p.X, p.Y)
	switch p.particle_type {
	case ParticleTypeDonut:
		radius := max(p.velocity*p.current_t, 1)
		margin := 10

		f32_radius := float32(radius)
		f32_margin := float32(margin)
		return DrawData{
			sprite:    PARTICLE_SPRITE_BUFFER.SubImage(image.Rect(0, 0, int(f32_radius+f32_margin)*2, int(f32_radius+f32_margin)*2)).(*ebiten.Image),
			position:  Position{x, y - 20},
			rotation:  90 * math.Pi / 180,
			intensity: 0.2,
			offset:    Position{0, 20},
			opacity:   0.25,
		}
	default:
		return DrawData{
			path:      p.sprite_path,
			position:  Position{x, y - 20},
			rotation:  p.Rotation - camera.rotation,
			intensity: 0.2,
			offset:    Position{0, 20},
			opacity:   0.25,
		}
	}
}

func calculateY(current_t, max_t, y_end float64) float64 {
	return (y_end / 2) * (1 + math.Sin((2*math.Pi*current_t/max_t)-(math.Pi/2)))
}

func calculateSmokeOffsetX(p Particle) float64 {
	return float64(p.seed%int64(p.variance)) - (float64(p.variance) / 2) + (math.Sin(p.current_t/(p.max_t/2))*p.current_t)/30

}

func (p *Particle) Update(game *Game) {
	level := game.CurrentLevel()
	switch p.particle_type {
	case ParticleTypeDebrisFromTank:
		x, y := math.Sin(p.Rotation)*p.velocity, math.Cos(p.Rotation)*p.velocity

		p.Position.Y += y
		collided_object := level.CheckObjectCollisionWithDimensions(p.Position, Position{4, 4})
		if collided_object != nil {
			p.Rotation = math.Pi - p.Rotation
			p.Position.Y -= y
		}

		p.Position.X += x
		collided_object = level.CheckObjectCollisionWithDimensions(p.Position, Position{4, 4})
		if collided_object != nil {
			p.Rotation = -p.Rotation
			p.Position.X -= x
		}

		// TODO better math here
		if p.current_t >= p.max_t/2 {
			p.Offset_z = calculateY(p.current_t, p.max_t/2, 5)
		} else {
			p.Offset_z = calculateY(p.current_t, p.max_t/2, 20)
		}
		p.r, p.g, p.b = interpolateColor(1, 0.14, 0, 1, 1, 0, (float32(p.current_t) / float32(p.max_t)))
	case ParticleTypeSmoke:
		p.Offset_z += p.velocity
		p.offset.X = calculateSmokeOffsetX(*p)

		p.intensity = p.current_t / p.max_t
	case ParticleTypeGunSmoke:
		x, y := math.Sin(p.Rotation)*p.velocity, math.Cos(p.Rotation)*p.velocity
		velocity := exponentialDecay(p.velocity, 5, p.current_t/p.max_t)
		p.Position.X += velocity * x
		p.Position.Y += velocity * y
	case ParticleTypeTest:
		p.Position = game.tank.Position
		p.current_t--
	case ParticleTypeDonut:
		lifetime_progress := p.current_t / p.max_t
		if lifetime_progress <= 0.8 {
			p.opacity = 1.0
		} else {
			fade_progress := (lifetime_progress - 0.8) / 0.2
			p.opacity = 1.0 - fade_progress
		}
	}

	p.current_t++
}

func (pm *ParticleManager) GetDrawData(g *Game) {
	for _, particle := range pm.particles {
		g.context.draw_data = append(g.context.draw_data, particle.GetDrawData(g.camera))
		if particle.shadow {
			g.context.draw_data = append(g.context.draw_data, particle.GetDrawShadowData(g.camera))
		}
	}
}

func (pm *ParticleManager) Update(g *Game) {
	particles := []*Particle{}
	for _, particle := range pm.particles {
		if particle.particle_type == ParticleTypeDebrisFromTank {
			if int(particle.current_t*100)%14 == 0 {
				position := particle.Position
				position.Y -= particle.Offset_z
				pm.AddParticle(
					Particle{
						particle_type: ParticleTypeSmoke,
						Position:      position,
						velocity:      0.1,
						sprite_path:   particle.sprite_path,
						current_t:     30,
						max_t:         80,
						variance:      12,
					},
				)
			}
		}
		if particle.particle_type == ParticleTypeGunSmoke {
			if int(particle.current_t*100)%8 == 0 {
				position := particle.Position
				position.Y -= particle.Offset_z
				pm.AddParticle(
					Particle{
						particle_type: ParticleTypeSmoke,
						Position:      position,
						velocity:      0.1,
						sprite_path:   particle.sprite_path,
						max_t:         15,
						variance:      320,
						offset:        particle.offset,
					},
				)
			}
		}
		particle.Update(g)
		if particle.Offset_z <= 8 {
			g.CurrentLevel().gm.ApplyForce(particle.X, particle.Y)
		}
	}

	for _, particle := range pm.particles {
		if particle.current_t < particle.max_t {
			particles = append(particles, particle)
		}
	}

	pm.particles = particles
}
