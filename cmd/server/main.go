package main

import (
	"flag"
	"gotanks"
	"net"
)

func main() {
	mediator_addr := flag.String("mediator", game.MEDIATOR_ADDR, "mediator server address")

	flag.Parse()

	game.StartServer(game.CreateServerName(), &net.UDPAddr{IP: net.ParseIP(*mediator_addr), Port: game.MEDIATOR_PORT})
}
