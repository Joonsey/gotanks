package main

import (
	"flag"
	"gotanks"
	"log"
	"net/http"
	_ "net/http/pprof"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	ebiten.SetWindowSize(game.SCREEN_WIDTH, game.SCREEN_HEIGHT)
	ebiten.SetWindowTitle("gotanks")

	start_server := flag.Bool("server", false, "start server")
	force_new_id := flag.Bool("f", false, "force new id")
	profiler := flag.Bool("p", false, "start profiler")
	mediator_addr := flag.String("mediator", game.MEDIATOR_ADDR, "mediator server address")

	flag.Parse()

	g := game.GameInit(*mediator_addr)

	if g.SaveIsFresh() || *force_new_id {
		g.GenerateNewPlayerId()
	}

	if *start_server {
		g.HostServer()
	}

	if *profiler {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
