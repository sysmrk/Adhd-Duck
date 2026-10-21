# Input and configuration validation

Both commands validate explicit numeric and boolean settings after loading
the environment and before starting services. Intervals and counts must be
positive, except documented disable/window/cache/GC settings that allow zero.
Thresholds must be finite, greater than zero and at most one; GC requires
`LOW < HIGH`. Seed timeout minimum must not exceed maximum; retry periods
must not exceed their corresponding backoff maximum. Overflow is rejected.

HTTP JSON endpoints accept one object only. Unknown or duplicate fields and trailing JSON
are rejected with HTTP 400. Malformed image references, malformed SHA-256
digests, utilization outside `[0,1]`, excessive fields and oversized lists
are rejected before state is updated or work starts. HTTP 202 still means
accepted, not completed. API clients must use the current documented fields.

The report endpoint accepts at most 8 MiB, 32,768 images/repos each and
131,072 entries per layer array. Image references are at most 512 bytes.
General strings are at most 4,096 bytes, error/reason fields 64 KiB, rescue
source arrays 64 entries and metadata maps 4,096 entries. JSON nesting is
limited to 32 levels. Byte-encoded metadata retains the existing finalize
limit. Large collections and strings are checked before allocating typed
inventory slices. Oversized HTTP bodies return 413 rather than being silently
truncated. Pull, cancel and webhook bodies allow 64 KiB; blob exports and
finalize retain their larger endpoint-specific caps.

Image names pinned by digest must match the root descriptor returned by a
rescue peer or supplied in finalize. Hash checking of every metadata blob
continues to apply. Valid tag references and valid matching pins still work.

Worker freshness uses controller receipt time; the worker timestamp does not
extend or shorten liveness. `PROPAGATE_MAX_UTILIZATION` also limits fallback
seed selection (default 0.70). When fallback has no eligible node with capacity,
preheat returns 503 and the ranker sends no pull order.

These changes do not implement fail-closed authentication, a shared worker
resource budget, or atomic coordination between GC and image imports. Those
remain separate improvements. Create valid token Secrets for deployments.
