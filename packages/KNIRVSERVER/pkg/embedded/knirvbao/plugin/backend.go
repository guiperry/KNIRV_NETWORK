package main

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/guiperry/knirv-sdk-go/signing"
	"github.com/openbao/openbao/sdk/v2/framework"
	"github.com/openbao/openbao/sdk/v2/logical"
)

func newBackend() *framework.Backend {
	return &framework.Backend{BackendType: logical.TypeLogical, RunningVersion: "v0.1.0", Paths: candidatePaths(&signerService{})}
}

// candidatePaths define the engine's public surface; private key storage has
// no matching route.
func candidatePaths(svc *signerService) []*framework.Path {
	name := framework.GenericNameRegex("name")
	nameField := &framework.FieldSchema{Type: framework.TypeString}
	readPublic := func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
		name := d.Get("name").(string)
		meta, err := readMetadata(ctx, req.Storage, name)
		if err != nil {
			return nil, err
		}
		version := meta.Version
		if v, ok := d.GetOk("version"); ok {
			version = v.(int)
			if version < 1 || version > meta.Version {
				return nil, errors.New("invalid key version")
			}
		}
		pub, err := describeVersion(ctx, req.Storage, name, version)
		if err != nil {
			return nil, err
		}
		return &logical.Response{Data: map[string]any{"version": pub.Version, "type": pub.Type, "address": pub.Address, "public_key": pub.PublicKey}}, nil
	}
	return []*framework.Path{
		{Pattern: "keys/" + name + "/policy", Fields: map[string]*framework.FieldSchema{"name": nameField}, Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
				var policy Policy
				if err := decodeRequest(req.Data, &policy); err != nil {
					return nil, err
				}
				if err := svc.setPolicy(ctx, req.Storage, d.Get("name").(string), policy); err != nil {
					return nil, err
				}
				return &logical.Response{Data: map[string]any{"updated": true}}, nil
			},
			logical.ReadOperation: func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
				p, err := loadPolicy(ctx, req.Storage, d.Get("name").(string))
				if err != nil {
					return nil, err
				}
				return &logical.Response{Data: map[string]any{"type_urls": p.TypeURLs, "chain_ids": p.ChainIDs, "messages": p.Messages, "max_fee": p.MaxFee, "rate_limit": p.RateLimit}}, nil
			},
		}},
		{Pattern: "keys/" + name, Fields: map[string]*framework.FieldSchema{"name": nameField, "version": {Type: framework.TypeInt}, "type": {Type: framework.TypeString}, "private_key": {Type: framework.TypeString, DisplayAttrs: &framework.DisplayAttributes{Sensitive: true}}}, Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
				pub, err := svc.createKey(ctx, req.Storage, d.Get("name").(string), d.Get("type").(string), d.Get("private_key").(string))
				if err != nil {
					return nil, err
				}
				return &logical.Response{Data: map[string]any{"version": pub.Version, "type": pub.Type, "address": pub.Address, "public_key": pub.PublicKey}}, nil
			}, logical.ReadOperation: readPublic,
		}},
		{Pattern: "address/" + name, Fields: map[string]*framework.FieldSchema{"name": nameField, "version": {Type: framework.TypeInt}}, Callbacks: map[logical.Operation]framework.OperationFunc{logical.ReadOperation: readPublic}},
		{Pattern: "rotate/" + name, Fields: map[string]*framework.FieldSchema{"name": nameField}, Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
				pub, err := svc.rotateKey(ctx, req.Storage, d.Get("name").(string))
				if err != nil {
					return nil, err
				}
				return &logical.Response{Data: map[string]any{"version": pub.Version, "type": pub.Type, "address": pub.Address, "public_key": pub.PublicKey}}, nil
			},
		}},
		{Pattern: "sign/direct/" + name, Fields: map[string]*framework.FieldSchema{"name": nameField}, Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
				var input signing.SignRequest
				if err := decodeRequest(req.Data, &input); err != nil {
					return nil, err
				}
				tx, err := svc.direct(ctx, req.Storage, d.Get("name").(string), input, time.Now())
				if err != nil {
					return rejected(), nil
				}
				return responseValue(tx)
			},
		}},
		{Pattern: "sign/message/" + name, Fields: map[string]*framework.FieldSchema{"name": nameField}, Callbacks: map[logical.Operation]framework.OperationFunc{
			logical.UpdateOperation: func(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
				var input signing.MessageEnvelope
				if err := decodeRequest(req.Data, &input); err != nil {
					return nil, err
				}
				msg, err := svc.message(ctx, req.Storage, d.Get("name").(string), input, time.Now())
				if err != nil {
					return rejected(), nil
				}
				return responseValue(msg)
			},
		}},
	}
}

func responseValue(value any) (*logical.Response, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, errors.New("cannot encode response")
	}
	var data map[string]any
	if err := json.Unmarshal(encoded, &data); err != nil {
		return nil, errors.New("cannot encode response")
	}
	return &logical.Response{Data: data}, nil
}

func decodeRequest(data map[string]any, out any) error {
	if unsafeNumber(data) {
		return errors.New("request contains an imprecise number")
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return errors.New("invalid request")
	}
	if err := json.Unmarshal(encoded, out); err != nil {
		return errors.New("invalid request")
	}
	return nil
}

// Some JSON decoders convert numbers to float64 before the plugin sees them.
// Reject values outside the exact integer range instead of signing rounded
// account numbers, fees, timestamps or amounts.
func unsafeNumber(value any) bool {
	switch value := value.(type) {
	case float64:
		return value >= 1<<53 || value <= -(1<<53)
	case map[string]any:
		for _, child := range value {
			if unsafeNumber(child) {
				return true
			}
		}
	case []any:
		for _, child := range value {
			if unsafeNumber(child) {
				return true
			}
		}
	}
	return false
}

func rejected() *logical.Response {
	return &logical.Response{Data: map[string]any{"error": "signing rejected", "data": map[string]any{"audit": "signing_rejected"}}}
}
