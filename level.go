package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/lafriks/go-tiled"
)

const (
	LEVEL_CONST_STACKS = "stacks"
)

type Level struct {
	tiled_map tiled.Map
	am        *AssetManager
}

func loadLevel(map_path string, am *AssetManager) Level {
	game_map, err := tiled.LoadFile(map_path)
	if err != nil {
		log.Fatal(err)
	}

	level := Level{tiled_map: *game_map, am: am}

	return level
}

func (l *Level) Draw(screen *ebiten.Image) {
	for _, layer := range l.tiled_map.Layers {
		if layer.Name == LEVEL_CONST_STACKS {
			for i, tile := range layer.Tiles {
				if tile.Nil {
					continue
				}
				x := float64(i % l.tiled_map.Width)
				y := float64(i / l.tiled_map.Width)
				sprite := l.am.stacked_map[tile.GetTileRect()]
				DrawStackedSprite(sprite, screen, x*SPRITE_SIZE, y*SPRITE_SIZE, 0)
			}
		}
	}
}
