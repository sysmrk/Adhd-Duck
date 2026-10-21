package config

import "testing"

func TestValidateRejectsUnsafeSettings(t *testing.T) {
	for _, tc := range []struct{ k, v string }{
		{"RANK_INTERVAL_S", "0"}, {"REPORT_INTERVAL_S", "-1"}, {"RANK_TOP_N", "-5"},
		{"GC_HIGH_UTILIZATION", "NaN"}, {"GC_LOW_UTILIZATION", "+Inf"}, {"PROPAGATE_MAX_UTILIZATION", "1.1"},
		{"PROPAGATE_SEED_TIMEOUT_FACTOR", "-1"}, {"RESCUE_ENABLED", "perhaps"}, {"GC_BATCH", "no"},
		{"RESCUE_TIMEOUT_S", "9223372036854775807"}, {"GC_LOW_UTILIZATION", "0.9"},
		{"PROPAGATE_SEED_TIMEOUT_MIN_S", "601"}, {"SPREAD_RETRY_AFTER_S", "601"},
	} {
		t.Run(tc.k+tc.v, func(t *testing.T) {
			t.Setenv(tc.k, tc.v)
			if Validate() == nil {
				t.Fatal("accepted invalid setting")
			}
		})
	}
}

func TestValidateAllowsDocumentedZeroSettings(t *testing.T) {
	for _, key := range []string{"PROPAGATE_WINDOW_S", "ORIGIN_CHECK_INTERVAL_S", "ORIGIN_CHECK_DELAY_S", "METRICS_EVENT_TTL_S", "PREHEAT_ATTRIBUTION_INTERVAL_S", "RANK_RESOLVE_CACHE_S", "GC_UNUSED_FOR_S", "GC_ROLLBACK_KEEP", "GC_SETTLE_S"} {
		t.Setenv(key, "0")
	}
	if err := Validate(); err != nil {
		t.Fatal(err)
	}
}
