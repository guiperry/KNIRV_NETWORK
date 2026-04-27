package main

import (
	"flag"
	"log"

	"knirvhasher/pkg/hashing/transformer"
)

func main() {
	addr := flag.String("addr", ":8090", "HEARTService listen address")
	useHashNetwork := flag.Bool("hashnetwork", false, "Enable optional HashNetwork fast path")
	useCerebras := flag.Bool("cerebras", false, "Enable optional Cerebras WSE2 path")
	flag.Parse()

	cfg := transformer.DefaultHEARTConfig(*useHashNetwork, *useCerebras)
	svc, err := transformer.NewHEARTServiceWithConfig(cfg)
	if err != nil {
		log.Fatalf("heart init: %v", err)
	}
	log.Printf("HEARTService listening on %s", *addr)
	log.Fatal(svc.ListenAndServe(*addr))
}