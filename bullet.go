package main

import (
	"log"
	"math"
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
	index           uint
}

func (bm *BulletManager) Shoot(bullet Bullet) {
	if bm.network_manager == nil ||
		!bm.network_manager.client.isConnected() {

		bm.AddBullet(bullet)
		return
	}

	err := bm.network_manager.client.Send(PacketTypeBulletShoot, bullet)
	if err == nil {
		log.Fatal(err)
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

	return &bm
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
				g.tank.sprites, // TODO update sprite
				Position{x, y},
				bullet.rotation - g.camera.rotation,
				1,
				Position{},
				1,
			})
	}
}

func (bm *BulletManager) Update(g* Game) {
	for i, bullet := range bm.bullets {
		collided_object := g.level.CheckObjectCollision(bullet.Position)
		if collided_object == nil {
			bm.bullets[i].Update(.5)
		}
	}
}
