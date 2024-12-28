package game

import (
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

func (g *Game) UpdateGameplay() error {
	g.tank.Update(g)
	g.camera.Update(g.GetTargetCameraPosition())
	level := g.CurrentLevel()
	level.gm.Update(g)
	g.bm.Update(level, g)
	g.pm.Update(g)
	g.time += 0.01

	tracks := []Track{}
	for _, track := range g.context.tracks {
		track.lifetime--
		if track.lifetime >= 0 {
			tracks = append(tracks, track)
		}
	}

	g.context.tracks = tracks

	return nil
}

func (g *Game) DrawUI(screen *ebiten.Image) {
	for count, player := range g.context.player_updates {
		g.DrawPlayerUI(screen, player, len(g.context.player_updates), g.nm.client.wins[player.ID], count, g.am.new_level_font)
	}

	if g.nm.client.server_state == ServerGameStateStartingNewRound {
		defer g.DrawNewLevelTimer(screen)
	}
	if g.nm.client.server_state == ServerGameStateGameOver {
		defer g.DrawGameOver(screen)
	}

	g.DrawAmmo(screen)
}

func (g *Game) DrawGameplay(screen *ebiten.Image) {
	level := g.CurrentLevel()
	level.GetDrawData(screen, g, g.camera)
	g.tank.GetDrawData(screen, g, g.camera, PLAYER_COLOR)
	g.bm.GetDrawData(g)
	g.pm.GetDrawData(g)
	if g.nm.client.isConnected() {
		g.nm.GetDrawData(g)
		defer g.DrawUI(screen)
	}

	for _, track := range g.context.tracks {
		x, y := g.camera.GetRelativePosition(track.X, track.Y)
		offset := float64(8)
		opacity := float32(track.lifetime) / float32(TRACK_LIFETIME)
		g.context.draw_data = append(g.context.draw_data, DrawData{
			sprite:    g.tank.track_sprite,
			position:  Position{x, y - offset},
			rotation:  track.rotation - g.camera.rotation,
			intensity: 1,
			offset:    Position{0, offset},
			opacity:   opacity})
	}

	sort.Slice(g.context.draw_data, func(i, j int) bool {
		i_obj := g.context.draw_data[i]
		j_obj := g.context.draw_data[j]
		// Compare the transformed Y values
		return i_obj.position.Y < j_obj.position.Y
	})

	for _, data := range g.context.draw_data {
		g.DrawStackedSpriteDrawData(screen, data)
	}

	g.context.draw_data = []DrawData{}

}
