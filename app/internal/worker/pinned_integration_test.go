//go:build integration

package worker

import (
	"adhdduck/internal/blobship"
	"adhdduck/internal/model"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestContainerd_RejectsMismatchedPinnedFinalize(t *testing.T) {
	ctx := context.Background()
	c := testClient(t)
	store := NewContainerdStore(c, testNS(t, "pin-regression"), "overlayfs")
	img := buildImage(t, testImage, baseLayer, appLayer)
	if err := store.Import(ctx, bytes.NewReader(img.archive), "linux/amd64"); err != nil {
		t.Fatal(err)
	}
	f := model.SpreadFinalize{Job: "pin-regression", Image: "registry.example.com/team/pin@sha256:" + strings.Repeat("0", 64), TopType: blobship.MediaTypeOCIManifest, TopDig: img.manifest, TopSize: int64(len(img.blobs[img.manifest])), Metadata: map[string][]byte{}}
	layers := map[string]bool{}
	for _, d := range img.layers {
		layers[d] = true
	}
	for d, b := range img.blobs {
		if !layers[d] {
			f.Metadata[d] = b
		}
	}
	sp := NewSpread(store, nil, spreadToken, "pin-regression", "linux/amd64", "", time.Minute)
	mux := http.NewServeMux()
	sp.Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()
	resp := postSpread(t, srv, "/spread/finalize", f)
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("pinned mismatch accepted: %d", resp.StatusCode)
	}
	if _, _, err := store.Resolve(ctx, f.Image); err == nil {
		t.Fatal("mismatched name registered")
	}
	f.Image = "registry.example.com/team/pin@" + img.manifest
	resp = postSpread(t, srv, "/spread/finalize", f)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("matching pinned finalize failed: %d", resp.StatusCode)
	}
	if _, got, err := store.Resolve(ctx, f.Image); err != nil || got != img.manifest {
		t.Fatal("valid pinned image not registered", got, err)
	}
}
