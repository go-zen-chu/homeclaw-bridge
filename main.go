package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-zen-chu/homeclaw-bridge/internal/bridge"
	"github.com/go-zen-chu/homeclaw-bridge/internal/server"
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}

	b := bridge.New()
	s := server.New(b)

	log.Printf("starting homeclaw-bridge on %s", addr)
	if err := http.ListenAndServe(addr, s); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
