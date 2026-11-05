package main

import (
	"context"
	"log"

	"github.com/openbao/openbao/sdk/v2/logical"
	"github.com/openbao/openbao/sdk/v2/plugin"
)

// factory constructs the node-local signing engine.
func factory(ctx context.Context, cfg *logical.BackendConfig) (logical.Backend, error) {
	b := newBackend()
	if err := b.Setup(ctx, cfg); err != nil {
		return nil, err
	}
	return b, nil
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cleanup, err := installEmbeddedCore()
	if err != nil {
		return err
	}
	defer cleanup()
	return plugin.ServeMultiplex(&plugin.ServeOpts{BackendFactoryFunc: factory})
}
