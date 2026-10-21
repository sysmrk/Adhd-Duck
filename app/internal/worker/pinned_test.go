package worker

import (
	"angryduck/internal/blobship"
	"angryduck/internal/model"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRescueRejectsPinnedPlanMismatch(t *testing.T) {
	image := "registry.example.com/team/app@sha256:" + strings.Repeat("0", 64)
	store := blobship.NewMemStore()
	store.AddTestImage(testImage, "base", "app")
	// Use a genuine plan but rename it to the requested mismatched pin.
	plan, err := blobship.BuildPlan(context.Background(), store, testImage, "linux/amd64")
	if err != nil {
		t.Fatal(err)
	}
	plan.Image = image
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { json.NewEncoder(w).Encode(plan) }))
	defer peer.Close()
	rescue := NewRescue(blobship.NewMemStore(), nil, testToken, "dst", "linux/amd64", 2)
	if _, err := rescue.fetchPlan(context.Background(), image, model.RescueSource{NodeID: "src", Address: strings.TrimPrefix(peer.URL, "http://")}); err == nil {
		t.Fatal("accepted mismatched plan")
	}
}
