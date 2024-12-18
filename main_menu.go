package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (g *Game) DrawMainMenu(screen *ebiten.Image) {
	g.DrawStripes(screen)

	textOp := text.DrawOptions{}
	msg := "  join game"
	if g.context.current_selection == 0 {
		msg = "* join game"
	}
	fontSize := 8.
	textOp.GeoM.Translate(RENDER_WIDTH/2, RENDER_HEIGHT/2+fontSize*3)
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)

	textOp = text.DrawOptions{}
	msg = "  host"
	if g.context.current_selection == 1 {
		msg = "* host"
	}
	textOp.GeoM.Translate(RENDER_WIDTH/2, RENDER_HEIGHT/2+fontSize*5)
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
}

func (g *Game) UpdateMainMenu() error {
	g.context.background_time++

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.context.current_selection++
		if g.context.current_selection >= 2 {
			g.context.current_selection = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.context.current_selection--
		if g.context.current_selection < 0 {
			g.context.current_selection = 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.context.current_selection == 0 {
			g.context.current_state = GameStateServerPicking
		}
		if g.context.current_selection == 1 {
			g.HostServer()
		}
	}
	return nil
}
