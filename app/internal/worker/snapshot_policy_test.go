package worker

import (
	"adhdduck/internal/blobship"
	"adhdduck/internal/model"
	"context"
	"net/http/httptest"
	"testing"
)

func TestSnapshotPolicyRejectsBeforeStoreAccess(t *testing.T) {
	rs := &Rescue{DisableSnapshotTransfer: true}
	if _, err := rs.receiveSnapshot(context.Background(), blobship.Plan{}, blobship.Layer{}, model.RescueSource{}); err == nil {
		t.Fatal("snapshot receive accepted")
	}
	w := httptest.NewRecorder()
	rs.HandleSnapshotExport(w, httptest.NewRequest("POST", "/snapshots/export", nil))
	if w.Code != 403 {
		t.Fatalf("export status %d", w.Code)
	}
}
