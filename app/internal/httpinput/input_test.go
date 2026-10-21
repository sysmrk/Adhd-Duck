package httpinput

import (
	"adhdduck/internal/model"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeRejectsBadAndOversizedRequests(t *testing.T) {
	for _, tc := range []struct {
		body   string
		limit  int64
		status int
	}{
		{`{"image":"busybox"} {}`, 1024, 400}, {`{"image":"busybox","typo":true}`, 1024, 400},
		{`{"image":"busybox","image":"other"}`, 1024, 400},
		{`{"image":"https://registry.test/app"}`, 1024, 400}, {`{"image":"app@sha256:abc"}`, 1024, 400},
		{`{"image":"` + strings.Repeat("a", 2000) + `"}`, 1024, 413}, {`{"image":"` + strings.Repeat("a", 513) + `"}`, 2048, 400},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/pull", strings.NewReader(tc.body))
		var order model.PullOrder
		err := Decode(w, r, tc.limit, &order)
		if err == nil {
			t.Fatal("accepted bad body")
		}
		Error(w, "invalid order", err)
		if w.Code != tc.status {
			t.Fatalf("got %d want %d", w.Code, tc.status)
		}
	}
}

func TestDecodeRejectsDeepAndLargeCollectionsBeforeTypedAllocation(t *testing.T) {
	for _, body := range []string{
		`{"images":[` + strings.TrimSuffix(strings.Repeat(`"a",`, 32769), ",") + `]}`,
		strings.Repeat("[", 34) + "0" + strings.Repeat("]", 34),
	} {
		var rep model.WorkerReport
		if Decode(httptest.NewRecorder(), httptest.NewRequest("POST", "/report", strings.NewReader(body)), ReportLimit, &rep) == nil {
			t.Fatal("accepted pathological shape")
		}
		if rep.Images != nil {
			t.Fatal("allocated typed inventory before rejecting shape")
		}
	}
}

func TestDecodeReportBoundsAndValidInventory(t *testing.T) {
	for _, body := range []string{
		`{"node_id":"n","address":"n:1","images":["` + strings.Repeat("a", 2<<20) + `"]}`,
		`{"node_id":"n","address":"n:1","utilization":1.1}`,
		`{"node_id":"n","address":"n:1","layers":{"epoch":"e","seq":1,"add_blobs":["sha256:bad"]}}`,
	} {
		var rep model.WorkerReport
		if Decode(httptest.NewRecorder(), httptest.NewRequest("POST", "/report", strings.NewReader(body)), ReportLimit, &rep) == nil {
			t.Fatal("accepted invalid report")
		}
	}
	var rep model.WorkerReport
	body := `{"node_id":"n","address":"n:1","images":[` + strings.TrimSuffix(strings.Repeat(`"registry.test/team/app:v1",`, 10000), ",") + `],"utilization":0.5}`
	if err := Decode(httptest.NewRecorder(), httptest.NewRequest("POST", "/report", strings.NewReader(body)), ReportLimit, &rep); err != nil {
		t.Fatal(err)
	}
	if len(rep.Images) != 10000 {
		t.Fatal("inventory truncated")
	}
}
