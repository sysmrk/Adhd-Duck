# Production deployment gate

This branch is a hardening candidate, not a certification of production readiness.
Supported target: Linux containerd with overlayfs. CRI-O/OpenShift is not supported.

Before upgrading, create `adhdduck-rescue-token` and `adhdduck-webhook-token`
Secrets with a `token` key of at least 32 random characters. Missing tokens now
stop startup; existing unauthenticated deployments must migrate first.

Snapshot export and receive are disabled by default. Keep
`discard_unpacked_layers = false` on every node. `ALLOW_SNAPSHOT_TRANSFER=true`
is an explicit acceptance of unverifiable snapshot content from trusted peers.

The chart enables ingress isolation. Configure `networkPolicy.webhookPeers`
for your ingress controller and `networkPolicy.monitoringPeers` for Prometheus.
These are Kubernetes NetworkPolicyPeer lists; use namespaceSelector and
podSelector together to avoid allowing an entire namespace. Test CNI node-to-pod
access to mirror port 18082. Egress remains unrestricted for registries, DNS,
node-exporter and the Kubernetes API; apply site-specific egress restrictions.

Configure `ingress.tls` with hosts and secretName. Internal peer traffic still
uses HTTP: encrypted node-to-node CNI transport is a deployment prerequisite.
The shared token trusts every worker with every node's image content. This is
not suitable for mutually untrusted tenants. Worker socket access remains
root-equivalent; use a dedicated, tightly controlled namespace and image publisher.

Start with GC_ENABLED=false on a small node pool. Verify disk measurement for
the actual containerd filesystem before enabling cleanup. Keep kubelet GC enabled.
Test registry outage with digest-pinned images and IfNotPresent, worker/controller
restarts, failed transfers, disk pressure, concurrent pulls, secret rotation,
mirror configuration cleanup on uninstall and rollback to the previous version.
Record results and recovery time before expanding the rollout.

Remaining release gates: dependency vulnerability triage; encrypted transport
validation; multi-node fault and load tests; image/chart provenance bound to
published digests. The standalone provenance workflow covers binaries only.
