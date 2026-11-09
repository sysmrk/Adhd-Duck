# Security model, version 1.9.0

## Trust domain and runtime access

Deploy only inside one trusted administrative domain. Workers run as root with
ALL capabilities dropped, a read-only root filesystem, RuntimeDefault seccomp,
no host PID namespace and no privilege escalation. They nevertheless mount the
read-write containerd socket, which grants root-equivalent node control. No
container securityContext can make that socket safe for untrusted code.
Limit who can publish worker images, patch the DaemonSet or read its Secrets.
Controller RBAC watches pods; worker RBAC reads its own DaemonSet for shutdown.
Workers also mount containerd content/configuration and persistent rescue state.

## Transport and authentication

Both executables refuse startup without a dedicated peer CA, leaf certificate
and key. Every controller/worker outbound client verifies TLS 1.3 and the SAN
`adhdduck.internal`, supplies a client certificate, disables environment proxies
and rejects redirects. This applies to reports, pull orders, rescue, propagation,
spread, repair, holder lookups and peer mirror content requests. Registry and
Kubernetes clients use their own separate TLS trust and credentials.

API servers verify presented client certificates. All paths except `/healthz`
and `/webhook/preheat` require a verified client certificate. Health exposes only
`ok` so HTTPS kubelet probes need no client certificate. The webhook requires
CI's separate bearer token. Existing shared-token guards remain on control and
content endpoints; `/status` requires that token too. `/metrics` requires a peer
client certificate. Missing/weak tokens stop startup, and AUTH_MODE=warn is
refused. TLS material is loaded once at startup: rotation requires restarts.

The chart shares a peer leaf and shared token across the installation. It does
not authenticate each node as an independent principal. Only issue certificates
from a dedicated CA controlled for this service. Never use the general corporate
CA as a blanket authorization boundary. Anyone holding the peer certificate and
shared token has access to private image content within this trust domain.

The node-local containerd mirror on port 18082 is an exception: HTTP with a
random per-pod bearer in root-only hosts.toml. It is accessed by the same node's
containerd; cross-node mirror content uses the TLS worker API. Do not expose
18082 through a Service, Ingress or remote node forwarding. Validate local-node
reachability and CNI rules before enabling the mirror.

## Content integrity

Blobs are checked against their expected digest and size before import.
A digest is an integrity address, not a publisher signature or an authorization
policy. Enforce approved image sources and image signing through your platform's
admission controls. Those controls are outside this component.

Snapshot contents cannot be bound to their original registry blob digest.
Snapshot export and receive are disabled by default in the executable. Setting
ALLOW_SNAPSHOT_TRANSFER=true accepts the risk of malicious snapshot content from
a credential holder. Keep discard_unpacked_layers=false and retain verified
blobs instead. Never enable the fallback in a mutually untrusted environment.

## Deployment controls

The chart includes ingress isolation for the APIs. Explicitly allow monitoring
and ingress-controller peers. Restrict egress at your site to DNS, the API server,
selected registries, node-exporter and other ADHD Duck pods. The project cannot
infer your registry, API or DNS addresses. Ingress is disabled by default; when
using ingress-nginx the chart verifies HTTPS backend certificates. Configure
front-end TLS separately. Install admission exceptions narrowly for hostPath/root.

GC is disabled by default because utilization currently measures the root mount
through node-exporter. Only enable after checking the actual containerd disk.
Keep kubelet's own GC and capacity protections. Certificate renewal, credential
rotation, registry access and monitoring are ongoing operator responsibilities.

## Reporting

Use the repository's private security advisory form, not a public issue, for
undisclosed vulnerabilities. Provide version, prerequisite access, impact and a
minimal reproducer. See [installation](getting-started.md) and
[production validation](production.md) for operational checks.
