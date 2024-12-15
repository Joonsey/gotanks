package game

import (
	"fmt"
	"log"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/lafriks/go-tiled"
)

const (
	LEVEL_CONST_GROUND = "ground"
	LEVEL_CONST_STACKS = "stacks"
)

type LevelEnum int

type Level struct {
	tiled_map tiled.Map
	am        *AssetManager

	spawns     []tiled.Object
	collisions []tiled.Object
}

func (l *Level) GetCollisions(object_group *tiled.ObjectGroup) {
	for _, object := range object_group.Objects {
		l.collisions = append(l.collisions, *object)
	}
}

func (l *Level) getSpawns(object_group *tiled.ObjectGroup) {
	for _, object := range object_group.Objects {
		l.spawns = append(l.spawns, *object)
	}
}

func (l *Level) GetSpawnPositions() []Position {
	spawns := []Position{}
	for _, value := range l.spawns {
		spawns = append(spawns, Position{value.X, value.Y})
	}

	return spawns
}

func (l *Level) CheckObjectCollisionWithDimensions(position Position, dimension Position) *tiled.Object {
	for _, object := range l.collisions {
		if object.X < position.X+dimension.X &&
			object.X+object.Width > position.X &&
			object.Y < position.Y+dimension.Y &&
			object.Y+object.Height > position.Y {
			return &object
		}
	}

	return nil
}

func (l *Level) CheckObjectCollision(position Position) *tiled.Object {
	for _, object := range l.collisions {
		if object.X < position.X+SPRITE_SIZE &&
			object.X+object.Width > position.X &&
			object.Y < position.Y+SPRITE_SIZE &&
			object.Y+object.Height > position.Y {
			return &object
		}
	}

	return nil
}

func (l *Level) MakeGrass(object_group *tiled.ObjectGroup, gm *GrassManager) {
	grass_sprites := []*ebiten.Image{}
	for i := range 6 {
		grass, _, err := ebitenutil.NewImageFromFile(fmt.Sprintf("assets/sprites/grass_%d.png", i))
		if err != nil {
			log.Fatal(err)
		}
		grass_sprites = append(grass_sprites, grass)
	}

	gm.Reset()

	multiple := 3
	i := 0
	for _, object := range object_group.Objects {
		for y := range int(object.Height) / multiple {
			for x := range int(object.Width) / multiple {
				entropy := rand.Intn(100)
				position := Position{
					float64(multiple*x) + object.X,
					float64(multiple*y) + object.Y,
				}

				gm.AddGrass(Grass{
					Position: position,
					rotation: 0,
					sprite:   grass_sprites[(i*entropy)%6],
				})
				i++
			}
		}
	}
}

func loadLevel(map_path string, am *AssetManager, gm *GrassManager) Level {
	// level should have owner ship of grass
	// global grass manager should not exist
	// TODO
	game_map, err := tiled.LoadFile(map_path)
	if err != nil {
		log.Fatal(err)
	}

	level := Level{tiled_map: *game_map, am: am}
	for _, object_group := range level.tiled_map.ObjectGroups {
		// Loop through ob in the object group
		switch object_group.Name {
		case "water":
			//
		case "collisions":
			level.GetCollisions(object_group)
		case "spawn":
			level.getSpawns(object_group)
		case "grass":
			if gm != nil {
				level.MakeGrass(object_group, gm)
			}
		}
	}

	if err != nil {
		log.Fatal(err)
	}

	return level
}

func (l *Level) GetDrawData(screen *ebiten.Image, g *Game, camera Camera) {
	for _, layer := range l.tiled_map.Layers {
		// we figure out how to treat the objects from the name of the layer
		switch layer.Name {
		case LEVEL_CONST_GROUND:
			for i, tile := range layer.Tiles {
				if tile.Nil {
					continue
				}

				i_x := float64(i % l.tiled_map.Width)
				i_y := float64(i / l.tiled_map.Width)

				rel_x, rel_y := camera.GetRelativePosition(i_x*SPRITE_SIZE, i_y*SPRITE_SIZE)
				// we offset the 'real' position by the entire size of the level
				// to ensure it's rendered first
				// we then render it at the negative offset such that it's drawn where we intend, just in a doctored order
				offset := float64(l.tiled_map.Width * SPRITE_SIZE)
				rel_x -= offset
				rel_y -= offset
				sprites := l.am.stacked_map[tile.GetTileRect()]
				g.context.draw_data = append(g.context.draw_data, DrawData{
					path:      sprites,
					position:  Position{rel_x, rel_y},
					rotation:  -camera.rotation,
					intensity: 1,
					offset:    Position{offset, offset},
					opacity:   1,
				})
			}
		case LEVEL_CONST_STACKS:
			for i, tile := range layer.Tiles {
				if tile.Nil {
					continue
				}

				i_x := float64(i % l.tiled_map.Width)
				i_y := float64(i / l.tiled_map.Width)

				rel_x, rel_y := camera.GetRelativePosition(i_x*SPRITE_SIZE, i_y*SPRITE_SIZE)
				sprites := l.am.stacked_map[tile.GetTileRect()]
				g.context.draw_data = append(g.context.draw_data, DrawData{
					path:      sprites,
					position:  Position{rel_x, rel_y},
					rotation:  -camera.rotation,
					intensity: 1,
					opacity:   1,
				})
			}
		}
	}
}
