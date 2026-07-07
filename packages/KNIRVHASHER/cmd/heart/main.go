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