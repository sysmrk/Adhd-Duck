# Install and operate ADHD Duck, step by step

This guide targets a Kubernetes administrator who is new to this project.
Commands run on your administration machine unless a step explicitly says
**on every node**. Do not run node commands inside an application container.

## 1. Understand what you are installing

A Deployment runs one controller. A DaemonSet runs one worker on each selected
node. CI sends the controller an image name after pushing it. The controller
orders workers to fetch and share layers. Workers need the containerd socket:
this grants root-equivalent node control, despite dropped Linux capabilities.
Install only with your platform team's authorization in a dedicated namespace.
Preheat, spreading, rescue, mirroring and image cleanup come from Angry Duck;
see [the comparison](upstream-comparison.md) for this fork's changes.

Supported scope: Linux/amd64, containerd 2.4 with overlayfs and Kubernetes
1.30 or later. The automated integration matrix exercises containerd 2.4.2 and legacy
1.7.36. The client library is containerd v2.4.2. Other runtime versions require
qualification in your cluster; do not use an end-of-life daemon in production. CRI-O/OpenShift and Windows nodes are not supported. Version ranges
are compatibility claims, not permission to run an unsupported vendor release.

You need: kubectl, Helm 3, Git, OpenSSL, and Docker with Buildx on the build
machine. You also need a registry you can push to and every node can pull from.
An existing Kubernetes cluster and node-exporter on each node are prerequisites;
this project does not install Kubernetes or node-exporter.

## 2. Check the cluster and node runtime

```bash
kubectl config current-context
kubectl get nodes -o wide
kubectl get nodes -o custom-columns=NAME:.metadata.name,RUNTIME:.status.nodeInfo.containerRuntimeVersion,ARCH:.status.nodeInfo.architecture
kubectl auth can-i create daemonsets -n adhdduck
```

Stop if the context is not the intended cluster, the runtime is CRI-O, or the
nodes are not amd64. On **every selected node**, check:

```bash
sudo test -S /run/containerd/containerd.sock
sudo test -d /var/lib/containerd/io.containerd.content.v1.content
sudo containerd config dump | grep -E 'snapshotter|discard_unpacked_layers|config_path|root ='
```

The chart currently mounts these exact paths. Distributions such as k3s can
use different paths and generated config: do not apply this chart unchanged
there. Keep `discard_unpacked_layers = false`. The mirror needs containerd's
registry `config_path` to include `/etc/containerd/certs.d`. Have your platform
team manage node configuration and service restarts using its normal rollout
process; never blindly overwrite an existing containerd configuration.

## 3. Choose published artifacts or build from source

### A. Install the published 1.9.0 release (recommended)

The two images and OCI chart have been verified anonymously accessible on GHCR.
No Docker build machine or image push account is needed for this path. Clone the
repository to obtain the TLS helper and the pinned production values:

```bash
git clone https://github.com/sysmrk/Adhd-Duck.git
cd Adhd-Duck
helm show chart oci://ghcr.io/sysmrk/charts/adhdduck --version 1.9.0
```

Continue with steps 4 and 5 to create the namespace, token Secrets and TLS Secret.
Create registry-pull only if the application images you preheat are private.
Label the initial nodes as shown in step 6. Then install with immutable images:

```bash
helm template adhdduck oci://ghcr.io/sysmrk/charts/adhdduck --version 1.9.0 \
  -n adhdduck -f helm/adhdduck/values-production.yaml > .local/rendered.yaml
kubectl apply --dry-run=server -f .local/rendered.yaml
helm upgrade --install adhdduck oci://ghcr.io/sysmrk/charts/adhdduck --version 1.9.0 \
  -n adhdduck -f helm/adhdduck/values-production.yaml --wait --timeout 5m
```

Continue at step 7. Before changing settings, copy values-production.yaml to
.local/values.yaml and use that file for subsequent upgrades. The sample values
are conservative starting resources, not a throughput guarantee. See
[release digests and evidence](releases/1.9.0.md).

### B. Build your own images (optional)


```bash
git clone https://github.com/sysmrk/Adhd-Duck.git
cd Adhd-Duck
git rev-parse HEAD
```

Record that commit for rollback and traceability. Substitute your actual registry
and project below. Never use the documentation's example hostname unchanged.
`docker login` prompts for credentials; do not put a password in shell history.

```bash
export REGISTRY=registry.example.com/platform
export VERSION=1.9.0
docker login registry.example.com
docker buildx build --platform linux/amd64 --provenance=mode=max --sbom=true \
  -f app/Dockerfile.controller -t "$REGISTRY/adhd-duck-controller:$VERSION" --push app
docker buildx build --platform linux/amd64 --provenance=mode=max --sbom=true \
  -f app/Dockerfile.worker -t "$REGISTRY/adhd-duck-worker:$VERSION" --push app
```

For centrally published images, use only a successful release and verify its
attestation with `gh attestation verify oci://IMAGE@sha256:DIGEST --repo sysmrk/Adhd-Duck`.
The presence of a tag or a README badge does not prove an image was published.
For production, record the two registry digests and configure them in step 6.

## 4. Create namespace, credentials and tokens

```bash
kubectl create namespace adhdduck --dry-run=client -o yaml | kubectl apply -f -
mkdir -p .local/tokens
chmod 700 .local .local/tokens
umask 077
openssl rand -hex 32 > .local/tokens/shared
openssl rand -hex 32 > .local/tokens/webhook
kubectl -n adhdduck create secret generic adhdduck-rescue-token \
  --from-file=token=.local/tokens/shared --dry-run=client -o yaml | kubectl apply -f -
kubectl -n adhdduck create secret generic adhdduck-webhook-token \
  --from-file=token=.local/tokens/webhook --dry-run=client -o yaml | kubectl apply -f -
```

Run token generation only on first installation. Regenerating these files is a
rotation and requires coordinating all pods and CI. A missing or short token
now stops startup. Do not use `AUTH_MODE=warn`.

For a private registry, use a dedicated **pull-only robot account**. Prepare a
Docker config file for that account in `.local/registry-config.json`, then:

```bash
kubectl -n adhdduck create secret generic registry-pull \
  --type=kubernetes.io/dockerconfigjson \
  --from-file=.dockerconfigjson=.local/registry-config.json \
  --dry-run=client -o yaml | kubectl apply -f -
```

This Secret serves both application-image pulls by ADHD Duck and, when selected
under `imagePullSecrets`, pulling ADHD Duck's own images. If different registry
accounts are needed, create a separate image pull Secret. A workload's own
`imagePullSecrets` are not automatically inherited by the workers.

## 5. Create TLS credentials

Prefer certificates from a dedicated enterprise CA. The peer leaf must have
DNS SAN `adhdduck.internal`, serverAuth and clientAuth extended usages, and a
key matching the certificate. This SAN is intentionally verified even when
connecting to a pod IP. Never set `insecureSkipVerify`.

For an initial isolated installation, the included script creates a dedicated
CA and a 90-day leaf. It refuses to overwrite existing keys:

```bash
./scripts/create-peer-tls.sh
kubectl -n adhdduck create secret generic adhdduck-peer-tls \
  --from-file=ca.crt=.local/peer-tls/ca.crt \
  --from-file=tls.crt=.local/peer-tls/tls.crt \
  --from-file=tls.key=.local/peer-tls/tls.key \
  --dry-run=client -o yaml | kubectl apply -f -
```

Keep `ca.key` offline. Never upload it to Kubernetes or Git. Track certificate
expiry and renew before the 90-day deadline. The chart shares a leaf and token
within one trusted administrative domain; it does not isolate mutually hostile
workers or tenants. Compromise of a worker requires rotating both credentials.

## 6. Configure a small initial rollout

Choose two or more worker nodes and label them. Substitute existing node names:

```bash
kubectl label node worker-01 adhdduck-enabled=true
kubectl label node worker-02 adhdduck-enabled=true
```

Create `.local/values.yaml`:

```yaml
controller:
  image:
    repository: registry.example.com/platform/adhd-duck-controller
    tag: "1.9.0"
    digest: "" # Set sha256:... from the registry for production.
worker:
  image:
    repository: registry.example.com/platform/adhd-duck-worker
    tag: "1.9.0"
    digest: "" # Set the WORKER digest, not the controller digest.
  nodeSelector:
    adhdduck-enabled: "true"
  tolerations: [] # Do not deploy on tainted control-plane nodes.
imagePullSecrets:
  - name: registry-pull
ingress:
  enabled: false
serviceMonitor:
  enabled: false
config:
  GC_ENABLED: "false"
  MIRROR_ENABLED: "false"
  ALLOW_SNAPSHOT_TRANSFER: "false"
```

Start with GC and the local containerd mirror off. Preheat and peer transfers
still work. No Ingress or monitoring CRDs are required for this first install.

```bash
helm lint ./helm/adhdduck -f .local/values.yaml
helm template adhdduck ./helm/adhdduck -n adhdduck -f .local/values.yaml > .local/rendered.yaml
kubectl apply --dry-run=server -f .local/rendered.yaml
helm upgrade --install adhdduck ./helm/adhdduck -n adhdduck \
  -f .local/values.yaml --wait --timeout 5m
kubectl -n adhdduck rollout status deployment/adhdduck-controller
kubectl -n adhdduck rollout status daemonset/adhdduck-worker
kubectl -n adhdduck get pods -o wide
```

Server dry-run must pass before installation. Admission errors about hostPath or
root require an explicit, narrowly scoped platform exception; do not disable
admission controls cluster-wide.

## 7. Verify TLS and send the first preheat

In terminal A:

```bash
kubectl -n adhdduck port-forward service/adhdduck-controller 8080:8080
```

In terminal B, from the repository directory:

```bash
curl --noproxy '*' --resolve adhdduck.internal:8080:127.0.0.1 \
  --cacert .local/peer-tls/ca.crt https://adhdduck.internal:8080/healthz
```

Expected: `ok`. `/status` without a client certificate must return 401.
Retrieve status with both credentials:

```bash
curl --noproxy '*' --resolve adhdduck.internal:8080:127.0.0.1 \
  --cacert .local/peer-tls/ca.crt --cert .local/peer-tls/tls.crt \
  --key .local/peer-tls/tls.key \
  -H "Authorization: Bearer $(cat .local/tokens/shared)" \
  https://adhdduck.internal:8080/status
```

Push an application image first. Replace the example reference with that image:

```bash
curl --fail-with-body --noproxy '*' \
  --resolve adhdduck.internal:8080:127.0.0.1 --cacert .local/peer-tls/ca.crt \
  -H "Authorization: Bearer $(cat .local/tokens/webhook)" \
  -H 'Content-Type: application/json' \
  -d '{"image":"registry.example.com/apps/demo:build-001"}' \
  https://adhdduck.internal:8080/webhook/preheat
kubectl -n adhdduck logs deployment/adhdduck-controller --tail=100
kubectl -n adhdduck logs -l app.kubernetes.io/component=worker --tail=100 --prefix
```

Expected: HTTP 202 with `accepted:true` and worker progress in logs. HTTP 503
means there are no fresh/eligible workers; solve that before integrating CI.
Use unique tags or image digests and `imagePullPolicy: IfNotPresent` for outage
rescue. `Always` can still require registry resolution and block startup.

## 8. Connect CI through an HTTPS webhook

Expose only `/webhook/preheat`, never `/status` or worker APIs. For ingress-nginx,
set `ingress.enabled:true`, an actual hostname and `ingress.tls` with a valid
public/internal front-end TLS Secret. The chart verifies backend TLS using the
peer Secret and the service SAN. Configure `networkPolicy.webhookPeers` with
namespaceSelector AND podSelector matching your ingress controller. Inspect its
real namespace and pod labels; do not copy an arbitrary label example blindly.
Other ingress implementations require equivalent verified HTTPS backend settings.

Store the webhook token as a masked/protected CI variable. After pushing an image:

```bash
curl --fail --silent --show-error --max-time 10 \
  -H "Authorization: Bearer ${ADHDDUCK_WEBHOOK_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d "{\"image\":\"${IMAGE}\"}" \
  https://duck.your-company.example/webhook/preheat || true
```

The `|| true` keeps preheating best-effort; it must not make deployment depend
on ADHD Duck. In a validation job omit it so failures are visible.

## 9. Enable optional features deliberately

Mirror: first confirm every selected node's containerd config_path. Set
`MIRROR_ENABLED:"true"` and `MIRROR_REGISTRIES` to your real registry host.
The worker writes only its marked `hosts.toml` files; it leaves independently
managed files unchanged. Node-to-local-worker traffic on port 18082 remains
HTTP with a per-pod token stored root-only in hosts.toml. It stays on that node;
peer-to-peer content traffic on 18081 uses mutual TLS. Test your CNI's local
node traffic and policy behavior before enabling it.

GC: keep disabled if containerd data is on a separate filesystem. Current disk
measurement uses node-exporter's `/` mountpoint. Validate disk usage and kubelet
GC interaction on your node layout; then enable `GC_ENABLED:"true"`. Start on
the small pool and observe deletions and rollback image retention. Never disable
kubelet GC as part of this installation.

Monitoring: install Prometheus Operator first. Set `serviceMonitor.enabled:true`,
`networkPolicy.monitoringPeers` and `serviceMonitor.tlsConfig` with CA, client
certificate and key Secret selectors and `serverName: adhdduck.internal`.
Prometheus must be authorized to read these Secrets in this namespace. The shared
peer credential grants trust-domain access; restrict who administers Prometheus.
Import `grafana/adhdduck-dashboard.json` and alert on unavailable workers, stalled
transfers, GC errors, memory pressure and TLS certificate expiration.

## 10. Troubleshoot common failures

| Symptom | Check and remedy |
|---|---|
| Pending / FailedMount | `kubectl -n adhdduck describe pod POD`; check TLS/token Secrets, node paths and node labels. |
| ImagePullBackOff on ADHD Duck itself | Check image repository/digest, registry credentials and connectivity. The worker cannot rescue its own missing image. |
| CrashLoopBackOff | Read logs; missing tokens, invalid TLS files and invalid config now stop startup intentionally. |
| x509 errors | Check CA chain, SAN, certificate expiry and node clock. Reissue correctly; do not bypass verification. |
| 401 from status | Supply a valid client certificate AND the shared bearer token. |
| Webhook 401 | Supply the CI webhook token, not the shared worker token. |
| Webhook 503 | Inspect worker readiness, network policies, reports and free disk. |
| Mirror not used | Check config_path, existing hosts.toml ownership and `MIRROR_ENABLED`. |
| Snapshot transfer disabled | Retain source blobs; do not turn on unsafe fallback just to hide the error. |
| Disk remains full | Images may not be the cause; inspect logs/volumes and actual image filesystem. |

## 11. Upgrade, rotate credentials and roll back

Before upgrade save the current values, image digests and chart revision:

```bash
helm -n adhdduck history adhdduck
helm -n adhdduck get values adhdduck > .local/previous-values.yaml
```

Build/publish the new images, update the two digests, lint/render/server-dry-run,
then run the same `helm upgrade --install` command. Wait for both rollouts and
repeat step 7. If unsuccessful, `helm -n adhdduck rollback adhdduck REVISION
--wait --timeout 5m` restores the selected Helm revision. Keep its image artifacts
and credentials available. A rollback across this release's TLS boundary also
requires the old configuration and corresponding chart.

Certificates and tokens are loaded at startup. Replace the Secret, restart the
controller and worker DaemonSet, update CI's webhook token if changed, and repeat
positive AND negative authentication checks. Coordinate a maintenance window:
mixed credentials temporarily interrupt peer transfers. Ordinary registry pulls
must remain available. For CA rotation, first distribute a bundle containing old
and new CA, restart all peers, then rotate leaf certificates and restart, finally
remove the old CA and restart again. Restart monitoring/ingress clients as needed.

Uninstall: first set `MIRROR_ENABLED:"false"`, upgrade, wait for all workers and
verify marked hosts.toml files are gone on each node; then `helm uninstall adhdduck
-n adhdduck`. Inspect unreachable/offline nodes manually before reusing them.
Do not delete shared containerd directories. Remove Secrets only after you no
longer need rollback. Never delete Kubernetes workloads to test recovery in production.
