package main

import (
	"github.com/hajimehoshi/ebiten/v2"
)

type Grass struct {
	Id int
	Position
	Rotation float64
}

func normalize(val, amt, target float64) float64 {
    if val > target+amt {
        val -= amt
    } else if val < target-amt {
        val += amt
    } else {
        val = target
    }
    return val
}

type GrassManager struct {
	tileSize int
	stiffness int
	maxUnique int
	grassAsset GrassAsset

	grass []Grass
}

func InitializeGrassManager() {}

func (gm* GrassManager) Draw(screen *ebiten.Image) {
	for _, grass := range gm.grass {
		op := ebiten.DrawImageOptions{}
		op.GeoM.Rotate(grass.Rotation)
		op.GeoM.Translate(float64(grass.X), float64(grass.Y))
		screen.DrawImage(gm.grassAsset.Sprites[grass.Id], &op)
	}
}

type GrassAsset struct {
	Sprites []*ebiten.Image
}
