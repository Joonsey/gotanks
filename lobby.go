package game

import (
	"fmt"
	"gotanks/shared"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (g *Game) DrawLobby(screen *ebiten.Image) {
	g.DrawStripes(screen)
	if g.context.current_server == nil {
		log.Panic("current server is nil")
	}
	fontSize := 8.

	textOp := text.DrawOptions{}
	msg := fmt.Sprintf("server: '%s'", g.context.current_server.Name)
	textOp.GeoM.Translate(1, 1)
	font_face := &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}

	text.Draw(screen, msg, font_face, &textOp)

	textOp = text.DrawOptions{}
	msg = "[TAB] loadout"
	textOp.GeoM.Translate(1, RENDER_HEIGHT-(fontSize+1))
	text.Draw(screen, msg, font_face, &textOp)

	textOp = text.DrawOptions{}
	msg = "[R]   ready/not ready"
	textOp.GeoM.Translate(1, RENDER_HEIGHT-(fontSize+1)*2)
	text.Draw(screen, msg, font_face, &textOp)

	for i := range 4 {
		clr := player_palette[i%len(player_palette)]
		padding := 5
		// padding + player name truncated + spacing + is ready + padding
		width := 8 * (padding + 8 + 1 + 1 + padding)
		// padding + font size + padding
		height := padding + 8 + padding

		margin := 12

		stroke_width := 2.0
		textOp := text.DrawOptions{}
		msg := "waiting for player"
		textOp.GeoM.Translate((RENDER_WIDTH/2)-float64(width/2), (RENDER_HEIGHT/2)+float64(i)*float64(fontSize+float64(margin*2)+stroke_width))
		textOp.GeoM.Translate(fontSize, float64(padding))
		textOp.ColorScale.ScaleWithColor(MISSING_PLAYER_COLOR)
		if i >= len(g.context.player_updates) {
			clr = MISSING_PLAYER_COLOR

			vector.StrokeRect(screen, (RENDER_WIDTH/2)-float32(width/2), (RENDER_HEIGHT/2)+float32(i)*float32(fontSize+float64(margin*2)+stroke_width), float32(width), float32(height), float32(stroke_width), clr, true)
			text.Draw(screen, msg, font_face, &textOp)

		} else {
			player := g.context.player_updates[i]

			if g.nm.client.isSelf(player.ID) {
				clr = PLAYER_COLOR
			}

			vector.StrokeRect(screen, (RENDER_WIDTH/2)-float32(width/2), (RENDER_HEIGHT/2)+float32(i)*float32(fontSize+float64(margin*2)+stroke_width), float32(width), float32(height), float32(stroke_width), clr, true)
			msg = fmt.Sprintf("%s %s", player.ID[0:8], PlayerReadyString(player.Ready))
			textOp.ColorScale.Reset()
			text.Draw(screen, msg, font_face, &textOp)
		}
	}

	// TODO draw time until start when all are ready
}

func (g *Game) UpdateLobby() error {
	g.context.background_time++

	g.nm.client.KeepAlive(g)

	if inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.nm.client.Send(shared.PacketTypeClientToggleReady, []byte{})
	}

	if !g.nm.client.isConnected() {
		g.context.current_selection = 0
		g.context.current_state = GameStateServerPicking
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.context.current_selection = 0
		g.context.current_state = GameStateTankLoadout
	}

	return nil
}
