package game

import (
	"fmt"
	"gotanks/shared"
	"log"
	"math"
	"sync"
)

type BulletTypeEnum uint8

const (
	BulletTypeStandard BulletTypeEnum = iota + 1
	BulletTypeFast
	BulletTypeEnd
)

const (
	BULLET_WIDTH  = 8
	BULLET_HEIGHT = 8
)

type Bullet struct {
	Position
	ID          string
	Rotation    float64
	Bullet_type BulletTypeEnum

	grace_period int
	Num_bounces  int
	Velocity     float64
}

type BulletHit struct {
	Player    string
	Bullet_ID string
}

type BulletManager struct {
	mutex   sync.RWMutex
	bullets map[string]*Bullet

	network_manager  *NetworkManager
	asset_manager    *AssetManager
	particle_manager *ParticleManager
	index            uint
}

func (bm *BulletManager) NewBulletId() string {
	bm.index++
	if bm.network_manager == nil ||
		bm.network_manager.client == nil ||
		!bm.network_manager.client.isConnected() {
		return fmt.Sprintf("0:%d", bm.index)
	}

	return fmt.Sprintf("%x:%d", *bm.network_manager.client.Auth, bm.index)
}

func (bm *BulletManager) Shoot(bullet Bullet) {
	bullet.ID = bm.NewBulletId()
	if bm.network_manager == nil ||
		bm.network_manager.client == nil ||
		!bm.network_manager.client.isConnected() {

		bm.AddBullet(bullet)
		return
	}
	err := bm.network_manager.client.Send(shared.PacketTypeBulletShoot, bullet)
	if err != nil {
		log.Panic(err)
	}
}

func (bm *BulletManager) Reset() {
	for k := range bm.bullets {
		delete(bm.bullets, k)
	}
}

func InitBulletManager(nm *NetworkManager, am *AssetManager, pm *ParticleManager) *BulletManager {
	if nm == nil || am == nil || pm == nil {
		// this could be solved by not passing pointers
		// but that's not cool
		log.Fatal("bullet manager dependencies not initialized")
	}
	bm := BulletManager{}
	bm.network_manager = nm
	bm.asset_manager = am
	bm.particle_manager = pm

	bm.bullets = make(map[string]*Bullet)

	return &bm
}

func (am *AssetManager) GetSpriteFromBulletTypeEnum(bullet_type BulletTypeEnum) string {
	switch bullet_type {
	case BulletTypeStandard:
		return "assets/sprites/stacks/bullet.png"
	case BulletTypeFast:
		return "assets/sprites/stacks/bullet-sniper.png"
	default:
		return "assets/sprites/stacks/bullet.png"
	}
}

func (bm *BulletManager) AddBullet(bullet Bullet) {
	bullet.grace_period = bm.DetermineGracePeriod(bullet.Bullet_type)

	if bm.particle_manager != nil {
		bm.particle_manager.AddParticle(
			Particle{
				particle_type: ParticleTypeGunSmoke,
				Rotation:      bullet.Rotation,
				Position:      bullet.Position,
				velocity:      2,
				offset:        Position{0, -TURRET_HEIGHT * 2},
				max_t:         25,
			})
	}
	bm.mutex.Lock()
	bm.bullets[bullet.ID] = &bullet
	bm.mutex.Unlock()
}

func (bm *BulletManager) DetermineGracePeriod(bullet_type BulletTypeEnum) int {
	switch bullet_type {
	case BulletTypeFast:
		return 15
	default:
		return 30
	}
}

func DetermineNumBounces(bullet_type BulletTypeEnum) int {
	switch bullet_type {
	case BulletTypeFast:
		return 1
	default:
		return 2
	}
}

func DetermineVelocity(bullet_type BulletTypeEnum) float64 {
	switch bullet_type {
	case BulletTypeFast:
		return 3
	default:
		return 1.3
	}
}

func DetermineBaseMagSize(bullet_type BulletTypeEnum) int {
	switch bullet_type {
	case BulletTypeStandard:
		return 2
	default:
		return 1
	}
}

func DetermineBaseReloadSpeed(bullet_type BulletTypeEnum) float64 {
	switch bullet_type {
	case BulletTypeFast:
		return 3
	default:
		return 2
	}
}

func DetermineBulletName(bullet_type BulletTypeEnum) string {
	switch bullet_type {
	case BulletTypeFast:
		return "sniper"
	case BulletTypeStandard:
		return "standard"
	default:
		return "missing!"
	}
}

func DetermineBulletDesc(bullet_type BulletTypeEnum) string {
	switch bullet_type {
	case BulletTypeFast:
		return "Fast travelling bullet,\n with longer reload time"
	case BulletTypeStandard:
		return "Standard-issue bullet, Pew Pew!"
	default:
		return "missing!"
	}
}

func DetermineBulletStats(bullet_type BulletTypeEnum) string {
	switch bullet_type {
	default:
		return fmt.Sprintf("\n - Magazine: %d\n - Reload speed: %.1fs\n - Bullet speed: %.1f",
			DetermineBaseMagSize(bullet_type), DetermineBaseReloadSpeed(bullet_type), DetermineVelocity(bullet_type))
	}
}

func (b *Bullet) Update(level *Level, game *Game) *Bullet {
	x, y := math.Sin(b.Rotation)*b.Velocity, math.Cos(b.Rotation)*b.Velocity

	b.Position.Y += y
	collided_object := level.CheckObjectCollisionWithDimensions(b.Position, Position{4, 4})
	if collided_object != nil {
		b.Rotation = math.Pi - b.Rotation
		if b.Num_bounces == 0 {
			return nil
		}
		b.Num_bounces--
		b.Position.Y -= y
	}

	b.Position.X += x
	collided_object = level.CheckObjectCollisionWithDimensions(b.Position, Position{4, 4})
	if collided_object != nil {
		b.Rotation = -b.Rotation
		if b.Num_bounces == 0 {
			return nil
		}
		b.Num_bounces--
		b.Position.X -= x
	}

	if game != nil {
		level.gm.ApplyForce(b.X, b.Y)
	}
	b.grace_period = max(b.grace_period-1, 0)
	return b
}

func (bm *BulletManager) GetDrawData(g *Game) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	for _, bullet := range bm.bullets {
		x, y := g.camera.GetRelativePosition(bullet.X, bullet.Y)
		g.context.draw_data = append(g.context.draw_data,
			DrawData{
				path:      g.am.GetSpriteFromBulletTypeEnum(bullet.Bullet_type),
				position:  Position{x, y},
				rotation:  -bullet.Rotation - g.camera.rotation + math.Pi,
				intensity: 1,
				offset:    Position{0, -TURRET_HEIGHT * 2},
				opacity:   1,
			})
		g.context.draw_data = append(g.context.draw_data,
			DrawData{
				path:      g.am.GetSpriteFromBulletTypeEnum(bullet.Bullet_type),
				position:  Position{x, y - 4},
				rotation:  -bullet.Rotation - g.camera.rotation + math.Pi,
				intensity: 0,
				offset:    Position{0, (-TURRET_HEIGHT * 2) + 8},
				opacity:   .3,
			})
	}
}

func (bm *BulletManager) Update(level *Level, g *Game) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()
	for key, bullet := range bm.bullets {
		bullet := bullet.Update(level, g)
		if bullet == nil {
			delete(bm.bullets, key)
		}
	}
}

func (bm *BulletManager) IsColliding(position, dimension Position) *Bullet {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	for _, bullet := range bm.bullets {
		if bullet.grace_period > 0 {
			continue
		}

		if bullet.X < position.X+dimension.X &&
			bullet.X+BULLET_WIDTH > position.X &&
			bullet.Y < position.Y+dimension.Y &&
			bullet.Y+BULLET_HEIGHT > position.Y {
			return bullet
		}
	}

	return nil
}
