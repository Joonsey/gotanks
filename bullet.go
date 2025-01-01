package game

import (
	"fmt"
	"gotanks/shared"
	"log"
	"math"
	"sync"
)

type StandardBulletTypeEnum uint8

const (
	StandardBulletTypeStandard StandardBulletTypeEnum = iota + 1
	StandardBulletTypeFast
	StandardBulletTypeEnd
)

const (
	BULLET_WIDTH  = 8
	BULLET_HEIGHT = 8
)

type Bullet interface {
	Update(level *Level, g *Game) Bullet
	GetDrawData(g *Game)
	GetId() string
	IsColliding(position, dimension Position) bool
}

type StandardBullet struct {
	Position
	ID          string
	Rotation    float64
	Bullet_type StandardBulletTypeEnum

	Num_bounces int
	Velocity    float64

	grace_period int
}

type BulletHit struct {
	Player    string
	Bullet_ID string
}

type BulletManager struct {
	Observer
	mutex   sync.RWMutex
	bullets map[string]Bullet

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
	//bullet.ID = bm.NewBulletId()
	// TODO
	// deprecate
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

	bm.bullets = make(map[string]Bullet)

	return &bm
}

func (am *AssetManager) GetSpriteFromBulletTypeEnum(bullet_type StandardBulletTypeEnum) string {
	switch bullet_type {
	case StandardBulletTypeStandard:
		return "assets/sprites/stacks/bullet.png"
	case StandardBulletTypeFast:
		return "assets/sprites/stacks/bullet-sniper.png"
	default:
		return "assets/sprites/stacks/bullet.png"
	}
}

func (bm *BulletManager) AddBullet(bullet Bullet) {
	bm.mutex.Lock()
	bm.bullets[bullet.GetId()] = bullet
	bm.mutex.Unlock()
}

func (bm *BulletManager) OnEvent(event Event) {
	switch event.Name {
	case EventBulletFired:
		bm.AddBullet(event.Data.(Bullet))
	case EventPlayerHit:
	}
}

func (bm *BulletManager) DetermineGracePeriod(bullet_type StandardBulletTypeEnum) int {
	switch bullet_type {
	case StandardBulletTypeFast:
		return 15
	default:
		return 30
	}
}

func DetermineNumBounces(bullet_type StandardBulletTypeEnum) int {
	switch bullet_type {
	case StandardBulletTypeFast:
		return 1
	default:
		return 2
	}
}

func DetermineVelocity(bullet_type StandardBulletTypeEnum) float64 {
	switch bullet_type {
	case StandardBulletTypeFast:
		return 3.3
	default:
		return 2.3
	}
}

func DetermineBaseMagSize(bullet_type StandardBulletTypeEnum) int {
	switch bullet_type {
	case StandardBulletTypeStandard:
		return 4
	default:
		return 2
	}
}

func DetermineBaseReloadSpeed(bullet_type StandardBulletTypeEnum) float64 {
	switch bullet_type {
	case StandardBulletTypeFast:
		return 3
	default:
		return 2
	}
}

func DetermineBulletName(bullet_type StandardBulletTypeEnum) string {
	switch bullet_type {
	case StandardBulletTypeFast:
		return "sniper"
	case StandardBulletTypeStandard:
		return "standard"
	default:
		return "missing!"
	}
}

func DetermineBulletDesc(bullet_type StandardBulletTypeEnum) string {
	switch bullet_type {
	case StandardBulletTypeFast:
		return "Fast travelling bullet,\n with longer reload time"
	case StandardBulletTypeStandard:
		return "Standard-issue bullet, Pew Pew!"
	default:
		return "missing!"
	}
}

func DetermineBulletStats(bullet_type StandardBulletTypeEnum) string {
	switch bullet_type {
	default:
		return fmt.Sprintf("\n - Magazine: %d\n - Reload speed: %.1fs\n - Bullet speed: %.1f",
			DetermineBaseMagSize(bullet_type), DetermineBaseReloadSpeed(bullet_type), DetermineVelocity(bullet_type))
	}
}

func (b StandardBullet) GetId() string {
	return b.ID
}

func (b StandardBullet) GetDrawData(g *Game) {
	x, y := g.camera.GetRelativePosition(b.X, b.Y)
	g.context.draw_data = append(g.context.draw_data,
		DrawData{
			path:      g.am.GetSpriteFromBulletTypeEnum(b.Bullet_type),
			position:  Position{x, y},
			rotation:  -b.Rotation - g.camera.rotation + math.Pi,
			intensity: 1,
			offset:    Position{0, -TURRET_HEIGHT * 2},
			opacity:   1,
		})
	g.context.draw_data = append(g.context.draw_data,
		DrawData{
			path:      g.am.GetSpriteFromBulletTypeEnum(b.Bullet_type),
			position:  Position{x, y - 4},
			rotation:  -b.Rotation - g.camera.rotation + math.Pi,
			intensity: 0,
			offset:    Position{0, (-TURRET_HEIGHT * 2) + 8},
			opacity:   .3,
		})
}

func (b StandardBullet) Update(level *Level, game *Game) Bullet {
	x, y := math.Sin(b.Rotation)*b.Velocity, math.Cos(b.Rotation)*b.Velocity

	b.Position.Y += y
	collided_object := level.CheckObjectCollisionWithDimensions(b.Position, Position{4, 4})
	if collided_object != nil {
		b.Rotation = math.Pi - b.Rotation
		if game != nil {
			game.pm.AddParticle(
				Particle{
					particle_type: ParticleTypeGunSmoke,
					Position:      b.Position,
					Rotation:      b.Rotation,
					velocity:      2,
					sprite_path:   "assets/sprites/stacks/particle-cube-template.png",
					max_t:         15,
				})
		}
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
		if game != nil {
			game.pm.AddParticle(
				Particle{
					particle_type: ParticleTypeGunSmoke,
					Position:      b.Position,
					Rotation:      b.Rotation,
					velocity:      2,
					sprite_path:   "assets/sprites/stacks/particle-cube-template.png",
					max_t:         15,
				})
		}
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
		bullet.GetDrawData(g)
	}
}

func (bm *BulletManager) Update(level *Level, g *Game) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()
	for key, bullet := range bm.bullets {
		bullet := bullet.Update(level, g)
		if bullet == nil {
			delete(bm.bullets, key)
		} else {
			bm.bullets[key] = bullet
		}
	}
}

func (bullet StandardBullet) IsColliding(position, dimension Position) bool {
	if bullet.grace_period > 0 {
		return false
	}

	if bullet.X < position.X+dimension.X &&
		bullet.X+BULLET_WIDTH > position.X &&
		bullet.Y < position.Y+dimension.Y &&
		bullet.Y+BULLET_HEIGHT > position.Y {
		return true
	}

	return false
}

func (bm *BulletManager) IsColliding(position, dimension Position) *Bullet {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()
	for _, bullet := range bm.bullets {
		hit := bullet.IsColliding(position, dimension)
		if hit {
			return &bullet
		}
	}

	return nil
}
