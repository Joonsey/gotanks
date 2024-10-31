package main

import (
	"flag"
	"log"
	"gotanks"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	SCREEN_WIDTH  = 640
	SCREEN_HEIGHT = 480
)

func main() {
	ebiten.SetWindowSize(SCREEN_WIDTH, SCREEN_HEIGHT)
	ebiten.SetWindowTitle("gotanks")

	start_server := flag.Bool("server", false, "start server")
	force_new_id := flag.Bool("f", false, "force new id")
	flag.Parse()

	g := game.GameInit()

	if g.SaveIsFresh() || *force_new_id {
		g.GenerateNewPlayerId()
	}

	if *start_server {
		g.HostServer()
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
