package models

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/guiperry/knirv-sdk-go/models"
	"github.com/guiperry/knirv-sdk-go/models/frame"
)

func TestManifestIsServedAndReadable(t *testing.T) {
	r := NewRegistry()
	mux := http.NewServeMux()
	r.Mount(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	c := sdk.NewClient(srv.URL, nil)
	if _, err := c.CapabilityManifest(context.Background()); err == nil {
		t.Fatal("manifest served before it was configured")
	}

	r.SetManifest(frame.Default(), func() frame.LiveFacts { return frame.LiveFacts{Model: "qwen", CtxSize: 8192} })
	mux2 := http.NewServeMux()
	r.Mount(mux2)
	srv2 := httptest.NewServer(mux2)
	defer srv2.Close()
	m, err := sdk.NewClient(srv2.URL, nil).CapabilityManifest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	reg := frame.Default()
	if len(m.Stages) != len(reg.StageIDs()) || m.Facts.CtxSize != 8192 {
		t.Fatalf("%+v", m.Facts)
	}
	want, _ := reg.Build("heart.chat", m.Facts)
	if m.FrameHashes["heart.chat"] != want.Hash {
		t.Fatal("served hash differs from a local build: the two callers would disagree")
	}
}
