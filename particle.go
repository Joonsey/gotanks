package main

import (
	"math"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

type ParticleTypeEnum int

const (
	ParticleTypeDebris ParticleTypeEnum = iota
	ParticleTypeSmoke
)

type Particle struct {
	Position
	Rotation float64
	Offset_z float64
	sprites  []*ebiten.Image
	velocity float64

	current_t float64
	max_t     float64
	intensity float64
	opacity   float64
	offset    Position

	seed     int64
	variance int

	particle_type ParticleTypeEnum
}

type ParticleManager struct {
	stiffness int

	particles []*Particle
}

func InitParticleManager() *ParticleManager {
	pm := ParticleManager{}
	pm.particles = []*Particle{}

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
	particle.seed = time.Now().UnixMilli()

	switch particle.particle_type {
	case ParticleTypeSmoke:
		particle.variance = 4
	}
	pm.particles = append(pm.particles, &particle)
}

func (pm *ParticleManager) Reset() {
	pm.particles = []*Particle{}
}

func (p *Particle) GetDrawData(camera Camera) DrawData {
	x, y := camera.GetRelativePosition(p.X, p.Y)
	return DrawData{
		p.sprites,
		Position{x, y - p.Offset_z},
		p.Rotation - camera.rotation,
		float32(p.intensity),
		p.offset,
		float32(p.opacity),
	}
}

func (p *Particle) GetDrawShadowData(camera Camera) DrawData {
	x, y := camera.GetRelativePosition(p.X, p.Y)
	return DrawData{
		p.sprites,
		Position{x, y - 20},
		p.Rotation - camera.rotation,
		0.2,
		Position{0, 20},
		0.25,
	}
}

func calculateY(current_t, max_t, y_end float64) float64 {
	return (y_end / 2) * (1 + math.Sin((2*math.Pi*current_t/max_t)-(math.Pi/2)))
}

func CalculateSmokeOffsetX(p Particle) float64 {
	return float64(p.seed%int64(p.variance)) - (float64(p.variance) / 2) + (math.Sin(p.current_t/(p.max_t/2))*p.current_t)/30

}

func (p *Particle) Update(level *Level) {
	switch p.particle_type {
	case ParticleTypeDebris:
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
		p.intensity = 1 - p.current_t/p.max_t
	case ParticleTypeSmoke:
		p.Offset_z += p.velocity
		p.offset.X = CalculateSmokeOffsetX(*p)

		p.intensity = p.current_t / p.max_t
		//p.opacity = p.current_t /.5 / p.max_t
	}
	p.current_t++
}

func (pm *ParticleManager) GetDrawData(g *Game) {
	for _, particle := range pm.particles {
		g.draw_data = append(g.draw_data, particle.GetDrawData(g.camera))
		g.draw_data = append(g.draw_data, particle.GetDrawShadowData(g.camera))
	}
}

func (pm *ParticleManager) Update(g *Game) {
	particles := []*Particle{}
	for _, particle := range pm.particles {
		if particle.particle_type == ParticleTypeDebris {
			if int(particle.current_t*100)%14 == 0 {
				position := particle.Position
				position.Y -= particle.Offset_z
				pm.AddParticle(
					Particle{
						particle_type: ParticleTypeSmoke,
						Position:      position,
						velocity:      0.1,
						sprites:       particle.sprites,
						current_t:     30,
						max_t:         80,
						variance:      12,
					},
				)
			}
		}
		particle.Update(&g.level)
		if particle.Offset_z <= 8 {
			g.gm.ApplyForce(particle.X, particle.Y)
		}
	}

	for _, particle := range pm.particles {
		if particle.current_t < particle.max_t {
			particles = append(particles, particle)
		}
	}

	pm.particles = particles
}
