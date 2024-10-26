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

type Bullet struct {
	Position
	rotation    float64
	bullet_type BulletTypeEnum
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
	bm.bullets = append(bm.bullets, bullet)
}

func (b *Bullet) Update(velocity float64) {
	x, y := math.Sin(b.rotation), math.Cos(b.rotation)

	b.X += x * velocity
	b.Y += y * velocity
}

func (bm *BulletManager) GetDrawData(g *Game) {
	for _, bullet := range bm.bullets {
		x, y := g.camera.GetRelativePosition(bullet.X, bullet.Y)
		g.draw_data = append(g.draw_data,
			DrawData{
				bm.asset_manager.GetSpriteFromBulletTypeEnum(bullet.bullet_type),
				Position{x, y},
				-bullet.rotation - g.camera.rotation + math.Pi,
				1,
				Position{0, -TURRET_HEIGHT * 2},
				1,
			})
	}
}

func (bm *BulletManager) Update(g *Game) {
	for i, bullet := range bm.bullets {
		bullet.Position.X += 4
		bullet.Position.Y += 4
		collided_object := g.level.CheckObjectCollisionWithDimensions(bullet.Position, Position{4, 4})
		if collided_object == nil {
			bm.bullets[i].Update(.5)
		}
	}
}
