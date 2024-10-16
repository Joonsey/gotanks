package main

import (
	"log"

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

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
    // Normalize pixel coordinates
    var uv = position.xy / Resolution

    // Apply time-based animation
    uv.x += Time * 0.02
    uv.y += Time * 0.05

	uv += Camera / Resolution

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

	water_polygons [][]ebiten.Vertex
	water_shader   *ebiten.Shader
}

func (l *Level) GetWaterPolygon() {
    // Loop through the object layers in the map
    for _, layer := range l.tiled_map.ObjectGroups {
        // Loop through ob in the object group
        if layer.Name == "water" {
			for _, object := range layer.Objects {
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
    }
}

func loadLevel(map_path string, am *AssetManager) Level {
	game_map, err := tiled.LoadFile(map_path)
	if err != nil {
		log.Fatal(err)
	}

	level := Level{tiled_map: *game_map, am: am}
	level.GetWaterPolygon()

	water_shader, err := ebiten.NewShader([]byte(waterShaderSrc))
	if err != nil {
		log.Fatal(err)
	}
	level.water_shader = water_shader

	return level
}

func (l *Level) Draw(screen *ebiten.Image, g* Game, camera Camera) {
	opts := &ebiten.DrawTrianglesShaderOptions{}
    opts.Uniforms = map[string]interface{}{
        "Time":       g.time,
        "Camera":     []float32{float32(camera.Offset.X), float32(camera.Offset.Y)},
        "Resolution": []float32{float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy())},
    }

	for _, polygon := range l.water_polygons {
		path := vector.Path{}
		cam_x, cam_y := float32(camera.Offset.X), float32(camera.Offset.Y)
		path.MoveTo(float32(polygon[0].DstX) - cam_x, float32(polygon[0].DstY) - cam_y)
		for _, p := range polygon[1:] {
			path.LineTo(float32(p.DstX) - cam_x, float32(p.DstY) - cam_y)
		}
		path.Close()

		vs, is := path.AppendVerticesAndIndicesForFilling(nil, nil)
		screen.DrawTrianglesShader(vs, is, l.water_shader, opts)
	}

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

				// TODO implement potential rotation (?)
				// this implementation is naive, need something more dynamic
				//render_x := (x - y) * (SPRITE_SIZE / 2)
				//render_y := (x + y) * (SPRITE_SIZE / 2)
				//DrawStackedSprite(sprite, screen, render_x-camera.Offset.X, render_y-camera.Offset.Y, 45*math.Pi / 180)

				sprite := l.am.stacked_map[tile.GetTileRect()]
				render_x := x * SPRITE_SIZE
				render_y := y * SPRITE_SIZE
				DrawStackedSprite(sprite, screen, render_x-camera.Offset.X, render_y-camera.Offset.Y, 0)
			}
		}
	}

}
