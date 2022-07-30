package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

const (
	W = 80
	H = 24
)

func main() {
	log.Println("Starting termbot")
	bot := NewTerminalBot()
	log.Println("Bot up")

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	log.Println("Shutting down")
	bot.Shutdown()
}
