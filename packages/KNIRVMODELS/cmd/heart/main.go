package main

import (
	"flag"
	"log"

	"knirvmodels/pkg/hashing/transformer"
)

func main() {
	// Dev tool only: the daemon serves HEART at /models/heart on its unix socket. The
	// default is loopback and not 8090, which is KNIRVSERVER's port.
	addr := flag.String("addr", "127.0.0.1:8097", "HEARTService listen address")
	useHashNetwork := flag.Bool("hashnetwork", false, "Enable optional HashNetwork fast path")
	useCerebras := flag.Bool("cerebras", false, "Enable optional Cerebras WSE2 path")
	externalGenURL := flag.String("external-generator-url", "",
		"HTTP endpoint for external LLM WASM generation (bootstrap mode). "+
			"When set, WASM generation uses this endpoint until the internal "+
			"Gorgonite model is fully trained.")
	flag.Parse()

	cfg := transformer.DefaultHEARTConfig(*useHashNetwork, *useCerebras)
	if *externalGenURL != "" {
		cfg.ExternalGeneratorURL = *externalGenURL
		log.Printf("HEARTService: using external LLM generator at %s (bootstrap mode)", *externalGenURL)
	}

	svc, err := transformer.NewHEARTServiceWithConfig(cfg)
	if err != nil {
		log.Fatalf("heart init: %v", err)
	}
	log.Printf("HEARTService listening on %s", *addr)
	log.Fatal(svc.ListenAndServe(*addr))
}