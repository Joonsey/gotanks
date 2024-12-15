package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

type Grass struct {
	Position
	rotation float64
	sprite   *ebiten.Image
}

type GrassManager struct {
	stiffness int
	sway      float64
	t         float64
	reversed  bool

	grass []Grass
}

func InitializeGrassManager() {}

func (gm *GrassManager) AddGrass(grass Grass) {
	gm.grass = append(gm.grass, grass)
}

func (gm *GrassManager) Reset() {
	gm.grass = []Grass{}
}

func (g *Grass) GetDrawData(camera Camera, gm *GrassManager) DrawData {
	x, y := camera.GetRelativePosition(g.X, g.Y)
	rotation := gm.sway + math.Cos(g.Y)
	intensity := max(1/1-float32(math.Sqrt(g.rotation*g.rotation)), 0.8)
	return DrawData{
		sprite:    g.sprite,
		position:  Position{x, y},
		rotation:  g.rotation + rotation,
		intensity: intensity,
		offset:    Position{},
		opacity:   1,
	}
}

func (g *Grass) Update() {
	reset_factor := 0.1
	if g.rotation > 0 {
		g.rotation = max(g.rotation-reset_factor, 0)
	} else if g.rotation < 0 {
		g.rotation = min(g.rotation+reset_factor, 0)
	}
}

func (g *Grass) ApplyForce(x, y float64) {
	//distance := math.Sqrt(x*x + y*y)

	direction := math.Atan2(y, x)

	g.rotation = direction

}

func (gm *GrassManager) GetDrawData(g *Game) {
	for _, grass := range gm.grass {
		g.context.draw_data = append(g.context.draw_data, grass.GetDrawData(g.camera, gm))
	}
}

func (gm *GrassManager) Update(g *Game) {
	gm.t += 0.01
	gm.sway = math.Sin(gm.t)
	for i := range gm.grass {
		gm.grass[i].Update()
	}
}

func (gm *GrassManager) ApplyForce(x, y float64) {
	for i, grass := range gm.grass {
		dis_x, dis_y := x-grass.X, y-grass.Y
		distance := math.Sqrt(dis_x*dis_x + dis_y*dis_y)

		if distance < 10 {
			gm.grass[i].ApplyForce(dis_x, dis_y)
		}
	}
}
