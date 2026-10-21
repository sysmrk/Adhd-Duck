package controller

import (
	"angryduck/internal/model"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestReceiptTimeControlsFreshness(t *testing.T) {
	r := NewRegistry(20*time.Millisecond, time.Minute)
	for _, stamp := range []time.Time{time.Now().Add(24 * time.Hour), time.Now().Add(-24 * time.Hour)} {
		r.Update(model.WorkerReport{NodeID: "n", Address: "n:1", Timestamp: stamp})
		if r.FreshCount() != 1 {
			t.Fatal("new report must be fresh regardless of worker clock")
		}
		if delta := time.Since(r.Snapshot().Workers[0].LastSeen); delta < 0 || delta > time.Second {
			t.Fatal("LastSeen is not receipt time", delta)
		}
	}
	time.Sleep(30 * time.Millisecond)
	if r.FreshCount() != 0 {
		t.Fatal("future worker clock kept node fresh")
	}
}

func TestFallbackCapacityAndNoCapacityResponse(t *testing.T) {
	reg := NewRegistry(time.Minute, time.Minute)
	reg.Update(model.WorkerReport{NodeID: "full", Address: "127.0.0.1:1", Utilization: .99})
	rank := NewRanker(reg, 1, time.Minute, nil, false, false)
	if got := rank.OrderNow("registry.test/app:1"); len(got) != 0 {
		t.Fatal("ordered full node", got)
	}
	server := NewServer(reg, rank)
	w := httptest.NewRecorder()
	server.Handler().ServeHTTP(w, httptest.NewRequest("POST", "/webhook/preheat", strings.NewReader(`{"image":"busybox"}`)))
	if w.Code != 503 {
		t.Fatalf("got %d: %s", w.Code, w.Body.String())
	}
}

func TestReportRejectsLargeFieldAndTrailingJSON(t *testing.T) {
	reg := NewRegistry(time.Minute, time.Minute)
	server := NewServer(reg, NewRanker(reg, 1, time.Minute, nil, false, false))
	for _, body := range []string{`{"node_id":"n","address":"n:1","images":["` + strings.Repeat("a", 2<<20) + `"]}`, `{"node_id":"n","address":"n:1"} {}`} {
		w := httptest.NewRecorder()
		server.Handler().ServeHTTP(w, httptest.NewRequest("POST", "/report", strings.NewReader(body)))
		if w.Code != 400 || reg.FreshCount() != 0 {
			t.Fatal("invalid report retained", w.Code)
		}
	}
}
