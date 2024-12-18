package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func (g *Game) UpdateTankLoadout() error {
	g.context.background_time++

	if g.nm.client.isConnected() {
		g.nm.client.KeepAlive(g)
	}

	loader_type := g.tank.Get(LoaderMask)
	bullet_type := g.tank.Get(BulletMask)
	barrel_type := g.tank.Get(BarrelMask)
	track_type := g.tank.Get(TracksMask)

	if inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.context.current_selection++
		if g.context.current_selection >= 5 {
			g.context.current_selection = 0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.context.current_selection--
		if g.context.current_selection < 0 {
			g.context.current_selection = 4
		}
	}

	incr_key := ebiten.KeyD
	decr_key := ebiten.KeyA
	switch g.context.current_selection {
	case 0:
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.context.current_selection = 0
			if g.nm.client.isConnected() {
				g.context.current_state = GameStateLobby
			} else {
				g.context.current_state = GameStateMainMenu
			}
		}
	case 1:
		if inpututil.IsKeyJustPressed(incr_key) {
			g.tank.Set(LoaderMask, max((loader_type+1)%LoaderEnd, 1))
		} else if inpututil.IsKeyJustPressed(decr_key) {
			if loader_type <= 1 {
				g.tank.Set(LoaderMask, LoaderEnd-1)
			} else {
				g.tank.Set(LoaderMask, loader_type-1)
			}
		}
	case 2:
		if inpututil.IsKeyJustPressed(incr_key) {
			g.tank.Set(BulletMask, max((bullet_type+1)%uint8(BulletTypeEnd), 1))
		} else if inpututil.IsKeyJustPressed(decr_key) {
			if bullet_type <= 1 {
				g.tank.Set(BulletMask, uint8(BulletTypeEnd)-1)
			} else {
				g.tank.Set(BulletMask, bullet_type-1)
			}
		}
	case 3:
		if inpututil.IsKeyJustPressed(incr_key) {
			g.tank.Set(BarrelMask, max((barrel_type+1)%BarrelEnd, 1))
		} else if inpututil.IsKeyJustPressed(decr_key) {
			if barrel_type <= 1 {
				g.tank.Set(BarrelMask, BarrelEnd-1)
			} else {
				g.tank.Set(BarrelMask, barrel_type-1)
			}
		}
	case 4:
		if inpututil.IsKeyJustPressed(incr_key) {
			g.tank.Set(TracksMask, max((track_type+1)%TracksEnd, 1))
		} else if inpututil.IsKeyJustPressed(decr_key) {
			if track_type <= 1 {
				g.tank.Set(TracksMask, TracksEnd-1)
			} else {
				g.tank.Set(TracksMask, track_type-1)
			}
		}
	}

	// not great, but works
	g.tank.Reset()
	return nil
}

func (g *Game) DrawTankLoadoutInfoScreen(screen *ebiten.Image, mask uint32) {

	y_offset := float32(100)
	x_padding := float32(200)
	x_offset := float32(0)
	bg_clr := color.RGBA{R: 0, G: 0, B: 0, A: 200}
	vector.DrawFilledRect(screen, RENDER_WIDTH/2+x_offset-x_padding, y_offset, x_padding*2, 150, bg_clr, true)
	vector.StrokeRect(screen, RENDER_WIDTH/2+x_offset-x_padding, y_offset, x_padding*2, 150, 2, STRIPE_COLOR, true)

	text_padding := 8.
	textOp := text.DrawOptions{}
	fontSize := 8.
	textOp.GeoM.Translate(float64(RENDER_WIDTH/2+x_offset-x_padding)+text_padding, float64(y_offset)+text_padding)
	textOp.LineSpacing = fontSize

	t := g.tank.Get(mask)

	msg := "Bullet: standard\n\nDescription: Standard-issue slower traveling,\nbut higher base magasine size.\n\nStats: "
	switch mask {
	case LoaderMask:
		msg = fmt.Sprintf("%s: %s\n\nDescription: %s\n\nStats: %s",
			"Loader", DetermineLoaderName(t), DetermineLoaderDesc(t), DetermineLoaderStats(t))
	case BulletMask:
		bt := BulletTypeEnum(t)
		msg = fmt.Sprintf("%s: %s\n\nDescription: %s\n\nStats: %s",
			"Bullet", DetermineBulletName(bt), DetermineBulletDesc(bt), DetermineBulletStats(bt))
	case BarrelMask:
		msg = fmt.Sprintf("%s: %s\n\nDescription: %s\n\nStats: %s",
			"Barrel", DetermineBarrelName(t), DetermineBarrelDesc(t), DetermineBarrelStats(t))
	case TracksMask:
		msg = fmt.Sprintf("%s: %s\n\nDescription: %s\n\nStats: %s",
			"Tracks", "TODO", "TODO", "TODO")
	default:
		return
	}

	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)

}
func (g *Game) DrawTankLoadout(screen *ebiten.Image) {
	g.DrawStripes(screen)

	for i := range 4 {
		textOp := text.DrawOptions{}
		var msg string = ""
		var mask uint32 = 0
		switch i {
		case 0:
			mask = LoaderMask
			msg = fmt.Sprintf("loader: < %s >", DetermineLoaderName(g.tank.Get(mask)))
		case 1:
			mask = BulletMask
			msg = fmt.Sprintf("bullet: < %s >", DetermineBulletName(BulletTypeEnum(g.tank.Get(mask))))
		case 2:
			mask = BarrelMask
			msg = fmt.Sprintf("barrel: < %s >", DetermineBarrelName(g.tank.Get(mask)))
		case 3:
			mask = TracksMask
			msg = fmt.Sprintf("tracks: < %s >", DetermineBulletName(BulletTypeEnum(g.tank.Get(TracksMask))))
		}

		if i+1 == g.context.current_selection {
			g.DrawTankLoadoutInfoScreen(screen, mask)

			msg = fmt.Sprintf("* %s", msg)
		} else {
			msg = fmt.Sprintf("  %s", msg)
		}
		fontSize := 8.
		margin := 2
		textOp.GeoM.Translate(RENDER_WIDTH/2, float64(i)*fontSize+250)
		textOp.GeoM.Translate(-200, fontSize+float64(margin))
		text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
	}

	msg := "back to menu"
	if g.nm.client.isConnected() {
		msg = "back to lobby"
	}
	if g.context.current_selection == 0 {
		msg = fmt.Sprintf("* %s", msg)
	} else {
		msg = fmt.Sprintf("  %s", msg)
	}
	fontSize := 8.
	textOp := text.DrawOptions{}
	textOp.GeoM.Translate(RENDER_WIDTH/2, RENDER_HEIGHT-(fontSize*3))
	textOp.GeoM.Translate(-float64(len(msg)/2)*fontSize, fontSize)
	text.Draw(screen, msg, &text.GoTextFace{Source: g.am.new_level_font, Size: fontSize}, &textOp)
}
