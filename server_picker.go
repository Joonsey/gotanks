package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
)

func (g *Game) UpdateServerPicking() error {
	g.context.background_time++

	g.context.available_servers = g.nm.client.GetServerList(g)

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.context.current_selection++
		if g.context.current_selection >= len(g.context.available_servers)+1 {
			g.context.current_selection = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.context.current_selection--
		if g.context.current_selection < 0 {
			g.context.current_selection = len(g.context.available_servers)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if g.context.current_selection == 0 {
			g.context.current_state = GameStateMainMenu
			g.context.current_server = nil
		} else if g.context.current_selection < len(g.context.available_servers)+1 {
			server := g.context.available_servers[g.context.current_selection-1]
			g.nm.Connect(server)
			g.context.current_server = &server
			g.context.current_state = GameStateLobby
		}
	}
	return nil
}

func (g *Game) DrawServerPicking(screen *ebiten.Image) {
	g.DrawStripes(screen)
	for i, server := range g.context.available_servers {
		textOp := text.DrawOptions{}
		msg := fmt.Sprintf("  %-10s| %d/%d", server.Name, server.Player_count, server.Max_players)
		if i+1 == g.context.current_selection {
			msg = fmt.Sprintf("* %-10s| %d/%d", server.Name, server.Player_count, server.Max_players)
		}
		fontSize := 8.
		textOp.GeoM.Translate(RENDER_WIDTH/2, float64(i)*fontSize)
		textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
		text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
	}

	msg := "  back to menu"
	if g.context.current_selection == 0 {
		msg = "* back to menu"
	}
	fontSize := 8.
	textOp := text.DrawOptions{}
	textOp.GeoM.Translate(RENDER_WIDTH/2, RENDER_HEIGHT-(fontSize*3))
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
}
