# ADHD Duck versus Angry Duck

ADHD Duck is an Apache-2.0 fork, not an independently invented distribution
engine. Credit for preheat, layer-aware ranking, peer blob distribution, rescue,
mirror, image repair, GC, metrics and the original integration tests belongs to
Angry Duck and its contributors. These are inherited features.

Comparison reference: Angry Duck main at
[`8eda727e4cf2279dd44b89498665ad88e99e9c06`](https://github.com/hatam-abolghasemi/Angry-Duck/tree/8eda727e4cf2279dd44b89498665ad88e99e9c06),
read on 2026-11-09. This is a snapshot comparison; upstream can change later.
The common ancestor in Git is `5102a56b8037bce1b249d2e3bbfcd32e55f66d8f`.

| Area | Upstream reference | ADHD Duck 1.9.0 | Practical consequence |
|---|---|---|---|
| Image distribution and rescue | Implemented | Inherited | No claim that the fork invented or accelerates these algorithms. |
| Peer network transport | HTTP and shared bearer token | TLS 1.3, verified dedicated CA/service SAN, client certificates plus existing tokens | Reduces network interception, tampering and credential exposure between pods. |
| Missing shared/webhook token | Optional Secrets, documented unauthenticated behavior | Required Secrets; executables stop if unusable | An incomplete install fails visibly instead of exposing control operations. |
| AUTH_MODE=warn | Available migration option | Refused by executables | Production cannot accidentally remain in permissive migration mode. |
| Snapshot transfer | Last-resort fallback with documented unverifiable content | Export and receive disabled by default; explicit opt-in | Loses fallback availability where blobs were discarded, in exchange for reducing image-store poisoning risk. |
| Status inventory | Open API endpoint | Peer client certificate plus shared token | Reduces disclosure of private image inventory. |
| Network policy | Operator-provided | Chart includes default ingress isolation and configurable peers | Less accidental reachability; site-specific egress remains an operator responsibility. |
| Controller URL | Namespace embedded in defaults | Derived from Helm release namespace | Avoids broken worker reports when installing outside the default namespace. |
| Image selection | Repository and tag | Optional immutable digest for both components | Makes deployments reproducible and resistant to mutable-tag replacement. |
| Rollout scope | Broad worker tolerations | Linux/amd64 selection, no blanket toleration, configurable nodeSelector | Supports a small first rollout without silently scheduling on control-plane taints. |
| GC default | Enabled | Disabled until disk layout is checked | Avoids automatic deletion based on the wrong filesystem measurement. |
| Optional integration defaults | Ingress and ServiceMonitor enabled | Both disabled until configured | A basic install needs neither an ingress controller nor monitoring CRDs. |
| Release gate | Build/publish workflow | Unit/race/vet, vulnerability and real-containerd integration gates; image SBOM/provenance and attestations | Improves release traceability and catches regressions before publication. |
| Containerd client | Legacy 1.x client | v2.4.2 client, refreshed dependencies and Go toolchain | Removes legacy client dependency findings; daemon updates remain the platform team's responsibility. |
| Beginner documentation | Feature and deployment guides | End-to-end guide including certificates, secrets, expected output, errors and rollback | Fewer hidden assumptions for new operators. |

## What is not claimed

There is no benchmark proving ADHD Duck is faster, uses less memory or scales
further than this upstream reference. Security defaults can add TLS CPU overhead.
This release does not add CRI-O/OpenShift or Windows support, multi-controller
leader election, independent tenant isolation or zero-downtime credential rotation.
The node socket still grants root-equivalent access. A trusted worker can access
private image content within the same trust domain. These are explicit design
limits, not problems solved by changing a project name.

## How to reproduce the source comparison

```bash
git remote add upstream https://github.com/hatam-abolghasemi/Angry-Duck.git
git fetch upstream
git show 8eda727e4cf2279dd44b89498665ad88e99e9c06:app/internal/sharedtoken/auth.go
git show 8eda727e4cf2279dd44b89498665ad88e99e9c06:helm/angryduck/values.yaml
git diff 5102a56b8037bce1b249d2e3bbfcd32e55f66d8f..main -- app helm .github docs
```

The diff includes the rename and documentation changes. Inspect behavioral
changes and tests; line counts alone do not measure improvement.
