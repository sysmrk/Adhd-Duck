# Production scope and acceptance record

Version 1.9.0 is built for one trusted administrative domain, Linux/amd64,
containerd/overlayfs and Kubernetes. CRI-O/OpenShift, Windows workers, multi-tenant
peer isolation and multiple active controllers are outside the supported scope.
The controller is one replica; state rebuilds from worker reports after restart.
Registry pulls remain the fallback; preheating must not gate deployments.

## Automated release gates

- Formatting, vet, race-enabled unit tests and TLS authentication regression tests.
- Real-containerd integration tests in GitHub Actions, including image import,
  repair, pins, snapshots and concurrent transfers; run on the release tag too.
- govulncheck on reachable code; a finding blocks publication, with no ignore list.
- Helm lint, namespace/TLS render checks and configurable node scope.
- Image SBOM/provenance, signed build attestations bound to published image digests.

A workflow definition is not a successful test result. Inspect the actual Actions
run for your commit and the successful Release run for your tag. No production
cluster was accessed during development of this revision. Automated passing
checks qualify the tested build; they cannot certify a different CNI, disk layout
or organization-specific traffic pattern.

## Site acceptance before enabling on all nodes

Use the [step-by-step guide](getting-started.md). Record the cluster version,
containerd version/config, CNI version, node filesystem, image digests, chart
values, start/end time, observed result and recovery time for every case:

| Case | Required outcome |
|---|---|
| Registry healthy, new image | Webhook 202, workers receive verified layers, workloads start normally. |
| Unauthorized API request | Missing/wrong token or client certificate cannot change state or read private content. |
| Registry unavailable | Previously cached image with IfNotPresent starts; unavailable content fails visibly. |
| Controller restart | Worker reports rebuild inventory and new preheats resume. |
| Worker killed mid-transfer | Retries recover without publishing an incomplete image or deleting a running image. |
| Rollout and uninstall with mirror | Only owned configuration is removed; direct registry pulls continue. |
| Disk pressure | No in-use image removed; actual image filesystem measured; kubelet protections remain enabled. |
| Certificate/token rotation | New credentials work, old credentials rejected after rollout; interruption bounded. |
| Representative peak workload | Record p95 rollout/transfer latency, registry bytes, network, CPU, memory and errors. |
| Helm rollback | Previous qualified image/chart and matching credentials restore service. |

Do not claim a universal throughput number without running that workload.
GC starts disabled because the current utilization source measures `/` through
node-exporter. Leave it off for a separate image filesystem until a suitable
measurement implementation is provided. Keep snapshots disabled and retain blobs.

Review [security boundaries](security.md) and [upstream differences](upstream-comparison.md).
