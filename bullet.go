package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type BulletTypeEnum uint

const (
	BulletTypeStandard BulletTypeEnum = iota
	BulletTypeFast
)

const (
	BULLET_WIDTH  = 8
	BULLET_HEIGHT = 8
)

type Bullet struct {
	Position
	Rotation    float64
	Bullet_type BulletTypeEnum

	grace_period int
	num_bounces  int
	velocity     float64
}

type BulletHit struct {
	Player string
}

type BulletManager struct {
	bullets []Bullet

	network_manager *NetworkManager
	asset_manager   *AssetManager
	index           uint
}

func (bm *BulletManager) Shoot(bullet Bullet) {
	if bm.network_manager == nil ||
		bm.network_manager.client == nil ||
		!bm.network_manager.client.isConnected() {

		bm.AddBullet(bullet)
		return
	}

	err := bm.network_manager.client.Send(PacketTypeBulletShoot, bullet)
	if err != nil {
		log.Panic(err)
	}
}

func InitBulletManager(nm *NetworkManager, am *AssetManager) *BulletManager {
	if nm == nil || am == nil {
		// this could be solved by not passing pointers
		// but that's not cool
		log.Fatal("bullet manager dependencies not initialized")
	}
	bm := BulletManager{}
	bm.network_manager = nm

	bm.asset_manager = am

	return &bm
}

func (am *AssetManager) GetSpriteFromBulletTypeEnum(bullet_type BulletTypeEnum) []*ebiten.Image {
	switch bullet_type {
	case BulletTypeStandard:
		return am.GetSprites("assets/sprites/stacks/bullet.png")
	case BulletTypeFast:
		return am.GetSprites("assets/sprites/stacks/bullet-sniper.png")
	default:
		return am.GetSprites("assets/sprites/stacks/bullet.png")
	}
}

func (bm *BulletManager) AddBullet(bullet Bullet) {
	bullet.grace_period = bm.DetermineGracePeriod(bullet.Bullet_type)
	bullet.num_bounces = bm.DetermineNumBounces(bullet.Bullet_type)
	bullet.velocity = bm.DetermineVelocity(bullet.Bullet_type)
	bm.bullets = append(bm.bullets, bullet)
}

func (bm *BulletManager) DetermineGracePeriod(bullet_type BulletTypeEnum) int {
	switch bullet_type {
	case BulletTypeFast:
		return 15
	default:
		return 30
	}
}

func (bm *BulletManager) DetermineNumBounces(bullet_type BulletTypeEnum) int {
	switch bullet_type {
	case BulletTypeFast:
		return 1
	default:
		return 2
	}
}

func (bm *BulletManager) DetermineVelocity(bullet_type BulletTypeEnum) float64 {
	switch bullet_type {
	case BulletTypeFast:
		return 3
	default:
		return 1.3
	}
}

func (b *Bullet) Update(level *Level) *Bullet {
	x, y := math.Sin(b.Rotation)*b.velocity, math.Cos(b.Rotation)*b.velocity

	b.Position.Y += y
	collided_object := level.CheckObjectCollisionWithDimensions(b.Position, Position{4, 4})
	if collided_object != nil {
		b.Rotation = math.Pi - b.Rotation
		if b.num_bounces == 0 {
			return nil
		}
		b.num_bounces--
		b.Position.Y -= y
	}

	b.Position.X += x
	collided_object = level.CheckObjectCollisionWithDimensions(b.Position, Position{4, 4})
	if collided_object != nil {
		b.Rotation = -b.Rotation
		if b.num_bounces == 0 {
			return nil
		}
		b.num_bounces--
		b.Position.X -= x
	}

	b.grace_period = max(b.grace_period-1, 0)
	return b
}

func (bm *BulletManager) GetDrawData(g *Game) {
	for _, bullet := range bm.bullets {
		x, y := g.camera.GetRelativePosition(bullet.X, bullet.Y)
		g.draw_data = append(g.draw_data,
			DrawData{
				bm.asset_manager.GetSpriteFromBulletTypeEnum(bullet.Bullet_type),
				Position{x, y},
				-bullet.Rotation - g.camera.rotation + math.Pi,
				1,
				Position{0, -TURRET_HEIGHT * 2},
				1,
			})
	}
}

func (bm *BulletManager) Update(level *Level) {
	bullets := []Bullet{}
	for _, bullet := range bm.bullets {
		bullet := bullet.Update(level)
		if bullet != nil {
			bullets = append(bullets, *bullet)
		}
	}
	bm.bullets = bullets
}

func (bm *BulletManager) IsColliding(position, dimension Position) *Bullet {
	for i, bullet := range bm.bullets {
		if bullet.grace_period > 0 {
			continue
		}

		if bullet.X < position.X+dimension.X &&
			bullet.X+BULLET_WIDTH > position.X &&
			bullet.Y < position.Y+dimension.Y &&
			bullet.Y+BULLET_HEIGHT > position.Y {
			return &bm.bullets[i]
		}
	}

	return nil
}
