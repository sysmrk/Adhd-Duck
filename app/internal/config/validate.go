package config

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"
)

// Validate checks explicit values before services or goroutines start. The
// getters retain their defaults for absent values and legacy library callers.
func Validate() error {
	schema := map[string]string{
		"GC_BATCH":                        "Int",
		"GC_ENABLED":                      "Bool",
		"GC_HIGH_UTILIZATION":             "Float",
		"GC_INTERVAL_S":                   "Duration",
		"GC_LOW_UTILIZATION":              "Float",
		"GC_ROLLBACK_KEEP":                "Int",
		"GC_SETTLE_S":                     "Duration",
		"GC_UNUSED_FOR_S":                 "Duration",
		"IMAGE_REPAIR_ENABLED":            "Bool",
		"IMAGE_REPAIR_INTERVAL_S":         "Duration",
		"INVENTORY_EVENTS_ENABLED":        "Bool",
		"LAYER_INVENTORY_ENABLED":         "Bool",
		"LAYER_RESYNC_INTERVAL_S":         "Duration",
		"LAYER_SCAN_INTERVAL_S":           "Duration",
		"METRICS_EVENT_TTL_S":             "Duration",
		"METRICS_LABEL_REGISTRY":          "Bool",
		"MIRROR_ENABLED":                  "Bool",
		"ORIGIN_CHECK_DELAY_S":            "Duration",
		"ORIGIN_CHECK_INTERVAL_S":         "Duration",
		"PREHEAT_ATTRIBUTION_INTERVAL_S":  "Duration",
		"PREHEAT_ATTRIBUTION_RETENTION_S": "Duration",
		"PROPAGATE_ENABLED":               "Bool",
		"PROPAGATE_INTERVAL_S":            "Duration",
		"PROPAGATE_MAX_CONCURRENT":        "Int",
		"PROPAGATE_MAX_UTILIZATION":       "Float",
		"PROPAGATE_PER_SOURCE":            "Int",
		"PROPAGATE_SEED_TIMEOUT_FACTOR":   "Float",
		"PROPAGATE_SEED_TIMEOUT_MAX_S":    "Duration",
		"PROPAGATE_SEED_TIMEOUT_MIN_S":    "Duration",
		"PROPAGATE_WINDOW_S":              "Duration",
		"RANK_BY_LAYERS":                  "Bool",
		"RANK_INTERVAL_S":                 "Duration",
		"RANK_PREFER_IMAGE_LOCALITY":      "Bool",
		"RANK_RESOLVE_CACHE_S":            "Duration",
		"RANK_RESOLVE_TIMEOUT_S":          "Duration",
		"RANK_TOP_N":                      "Int",
		"REPORT_INTERVAL_S":               "Duration",
		"RESCUE_BACKOFF_MAX_S":            "Duration",
		"RESCUE_ENABLED":                  "Bool",
		"RESCUE_INTERVAL_S":               "Duration",
		"RESCUE_MAX_CONCURRENT":           "Int",
		"RESCUE_MAX_SOURCES":              "Int",
		"RESCUE_NODE_MAX_CONCURRENT":      "Int",
		"RESCUE_PIN_TTL_S":                "Duration",
		"RESCUE_RESYNC_INTERVAL_S":        "Duration",
		"RESCUE_RETRY_AFTER_S":            "Duration",
		"RESCUE_TIMEOUT_S":                "Duration",
		"SPREAD_BACKOFF_MAX_S":            "Duration",
		"SPREAD_ENABLED":                  "Bool",
		"SPREAD_MAX_REGISTRY_PULLS":       "Int",
		"SPREAD_RACE_PER_BLOB":            "Int",
		"SPREAD_RETRY_AFTER_S":            "Duration",
		"SPREAD_TRANSFER_TIMEOUT_S":       "Duration",
		"TARGET_TTL_S":                    "Duration",
		"WORKER_STALE_AFTER_S":            "Duration",
	}
	zero := map[string]bool{"PROPAGATE_WINDOW_S": true, "ORIGIN_CHECK_INTERVAL_S": true, "ORIGIN_CHECK_DELAY_S": true, "METRICS_EVENT_TTL_S": true, "PREHEAT_ATTRIBUTION_INTERVAL_S": true, "RANK_RESOLVE_CACHE_S": true, "GC_UNUSED_FOR_S": true, "GC_ROLLBACK_KEEP": true, "GC_SETTLE_S": true}
	for key, kind := range schema {
		v, ok := os.LookupEnv(key)
		if !ok || v == "" {
			continue
		}
		switch kind {
		case "Bool":
			if _, err := strconv.ParseBool(v); err != nil {
				return fmt.Errorf("%s must be a boolean", key)
			}
		case "Int", "Duration":
			n, err := strconv.ParseInt(v, 10, 64)
			if err != nil || n < 0 || (n == 0 && !zero[key]) {
				return fmt.Errorf("%s must be a valid %s integer", key, map[bool]string{true: "nonnegative", false: "positive"}[zero[key]])
			}
			if kind == "Duration" && n > int64((time.Duration(1<<63-1)-time.Minute)/time.Second) {
				return fmt.Errorf("%s duration overflows", key)
			}
			if kind == "Int" && uint64(n) > uint64(^uint(0)>>1) {
				return fmt.Errorf("%s integer overflows", key)
			}
		case "Float":
			n, err := strconv.ParseFloat(v, 64)
			if err != nil || math.IsNaN(n) || math.IsInf(n, 0) || n <= 0 {
				return fmt.Errorf("%s must be finite and positive", key)
			}
			if key != "PROPAGATE_SEED_TIMEOUT_FACTOR" && n > 1 {
				return fmt.Errorf("%s must be at most 1", key)
			}
		}
	}
	if Float("GC_LOW_UTILIZATION", .60) >= Float("GC_HIGH_UTILIZATION", .70) {
		return fmt.Errorf("GC_LOW_UTILIZATION must be less than GC_HIGH_UTILIZATION")
	}
	if Duration("PROPAGATE_SEED_TIMEOUT_MIN_S", 120) > Duration("PROPAGATE_SEED_TIMEOUT_MAX_S", 600) {
		return fmt.Errorf("seed timeout minimum exceeds maximum")
	}
	if Duration("RESCUE_RETRY_AFTER_S", 120) > Duration("RESCUE_BACKOFF_MAX_S", 3600) {
		return fmt.Errorf("rescue retry exceeds backoff maximum")
	}
	if Duration("SPREAD_RETRY_AFTER_S", 15) > Duration("SPREAD_BACKOFF_MAX_S", 600) {
		return fmt.Errorf("spread retry exceeds backoff maximum")
	}
	return nil
}
