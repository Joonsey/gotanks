package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/lafriks/go-tiled"
)

const (
	LEVEL_CONST_GROUND = "ground"
	LEVEL_CONST_STACKS = "stacks"
)
const waterShaderSrc = `// Ebiten Shader
package main

var Time float
var Resolution vec2
var Camera vec2
var CameraRotation float

// Random function for generating pseudo-random values
func rand(st vec2) float {
    return fract(sin(dot(st.xy, vec2(12.9898, 78.233))) * 43758.5453123)
}

// Interpolation function (linear)
func mix(x float, y float, a float) float {
    return x * (1.0 - a) + y * a
}

// 2D Noise function
func noise(st vec2) float {
    var i = floor(st)
    var f = fract(st)

    // Four corners of the cell
    var a = rand(i)
    var b = rand(i + vec2(1.0, 0.0))
    var c = rand(i + vec2(0.0, 1.0))
    var d = rand(i + vec2(1.0, 1.0))

    // Interpolation along x and y axis
    var u = f * f * (3.0 - 2.0 * f) // Smooth interpolation
    return mix(a, b, u.x) + (c - a) * u.y * (1.0 - u.x) + (d - b) * u.x * u.y
}

func rotateUV(uv vec2, center vec2, angle float) vec2 {
    var s = sin(angle)
    var c = cos(angle)

    // Translate UV to origin, rotate, and translate back
    var rotatedUV = vec2(
        uv.x * c - uv.y * s,
        uv.x * s + uv.y * c,
    )

    return rotatedUV
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
    // Normalize pixel coordinates
    var uv = position.xy / Resolution

    // Apply time-based animation
    uv.x += Time * 0.02
    uv.y += Time * 0.05

	uv += Camera / Resolution

	uv = rotateUV(uv, vec2(0.5, 0.5), CameraRotation)

    // Increase the frequency of the noise pattern for smaller "waves"
    var frequency = 15.0
    var noiseValue = noise(uv * frequency) + sin(Time) * 0.1

    // Base water color
    var baseColor = vec4(0.0, 0.4, 0.8, 1.0)

    // Simple shading based on noise
	if noiseValue > 0.7 {
        baseColor.rgb = vec3(0.2, 0.9, 1.0) // Very light shade
	} else if noiseValue > 0.2 {
        baseColor.rgb = vec3(0.0, 0.6, 1.0) // Lighter shade
    } else {
        baseColor.rgb = vec3(0.0, 0.3, 0.6) // Darker shade
    }
    return baseColor
}
`


type Level struct {
	tiled_map tiled.Map
	am        *AssetManager

	spawns     []tiled.Object
	collisions []tiled.Object

	water_polygons [][]ebiten.Vertex
	water_shader   *ebiten.Shader
}

func (l *Level) GetWaterPolygon(object_group *tiled.ObjectGroup) {
	for _, object := range object_group.Objects {
		// Extract the vertices from the polygon
		var vertices []ebiten.Vertex

		for _, polygons := range object.Polygons {
			for _, point := range *polygons.Points {
				vertices = append(vertices, ebiten.Vertex{
					DstX: float32(object.X + point.X),
					DstY: float32(object.Y + point.Y),
					SrcX: 0, // Texture coordinates not needed for plain shader
					SrcY: 0,
				})
			}
		}

		l.water_polygons = append(l.water_polygons, vertices)
	}
}

func (l *Level) GetCollisions(object_group *tiled.ObjectGroup) {
	for _, object := range object_group.Objects {
		l.collisions = append(l.collisions, *object)
	}
}

func (l *Level) GetSpawns(object_group *tiled.ObjectGroup) {
	for _, object := range object_group.Objects {
		l.spawns = append(l.spawns, *object)
	}
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


func loadLevel(map_path string, am *AssetManager) Level {
	game_map, err := tiled.LoadFile(map_path)
	if err != nil {
		log.Fatal(err)
	}

	level := Level{tiled_map: *game_map, am: am}
    for _, object_group := range level.tiled_map.ObjectGroups {
        // Loop through ob in the object group
		switch object_group.Name {
		case "water":
			level.GetWaterPolygon(object_group)
		case "collisions":
			level.GetCollisions(object_group)
		case "spawn":
			level.GetSpawns(object_group)
		}
	}

	water_shader, err := ebiten.NewShader([]byte(waterShaderSrc))
	if err != nil {
		log.Fatal(err)
	}
	level.water_shader = water_shader

	return level
}

func (l* Level) drawWater(screen *ebiten.Image, g* Game, camera Camera) {
	opts := &ebiten.DrawTrianglesShaderOptions{}
    opts.Uniforms = map[string]interface{}{
        "Time":       g.time,
        "Camera":     []float32{float32(camera.Offset.X), float32(camera.Offset.Y)},
        "CameraRotation": float32(camera.rotation),
        "Resolution": []float32{float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy())},
    }

	for _, polygon := range l.water_polygons {
		path := vector.Path{}

		cam_x, cam_y := camera.Offset.X, camera.Offset.Y
		translated_x, translated_y := float64(polygon[0].DstX) - cam_x, float64(polygon[0].DstY) - cam_y
		render_x := translated_x * math.Cos(camera.rotation) + translated_y * math.Sin(camera.rotation)
		render_y := -translated_x * math.Sin(camera.rotation) + translated_y * math.Cos(camera.rotation)
		path.MoveTo(float32(render_x), float32(render_y))
		for _, p := range polygon[1:] {
			translated_x, translated_y := float64(p.DstX) - cam_x, float64(p.DstY) - cam_y
			render_x := translated_x * math.Cos(camera.rotation) + translated_y * math.Sin(camera.rotation)
			render_y := -translated_x * math.Sin(camera.rotation) + translated_y * math.Cos(camera.rotation)
			path.LineTo(float32(render_x), float32(render_y))
		}
		path.Close()

		vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
		screen.DrawTrianglesShader(vs, is, l.water_shader, opts)
	}

}

func (l *Level) Draw(screen *ebiten.Image, g* Game, camera Camera) {
	// this is currently strange. depricating for now
	// l.drawWater(screen, g, camera)
	for _, layer := range l.tiled_map.Layers {
		// we figure out how to treat the objects from the name of the layer
		switch layer.Name {
		case LEVEL_CONST_GROUND, LEVEL_CONST_STACKS:
			for i, tile := range layer.Tiles {
				if tile.Nil {
					continue
				}
				x := float64(i % l.tiled_map.Width)
				y := float64(i / l.tiled_map.Width)

				sprite := l.am.stacked_map[tile.GetTileRect()]
				translated_x := x * SPRITE_SIZE - camera.Offset.X
				translated_y := y * SPRITE_SIZE - camera.Offset.Y

				render_x := translated_x * math.Cos(camera.rotation) + translated_y * math.Sin(camera.rotation)
				render_y := -translated_x * math.Sin(camera.rotation) + translated_y * math.Cos(camera.rotation)

				DrawStackedSprite(sprite, screen, render_x, render_y, -camera.rotation)
			}
		}
	}

}
