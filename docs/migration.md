# Migrating from Angry Duck to ADHD Duck

The rename covers source imports, binaries, image names, Kubernetes objects,
Secret references, state paths, metrics, the Helm chart and dashboard. This
is not a drop-in in-place rename of a running Helm release.

| Component | Previous identifier | New identifier |
|---|---|---|
| Repository | `hatam-abolghasemi/Angry-Duck` | `sysmrk/Adhd-Duck` |
| Source module/import prefix | `angryduck` | `adhdduck` |
| Binaries | `angryduck-controller`, `angryduck-worker` | `adhdduck-controller`, `adhdduck-worker` |
| Helm chart directory | `helm/angryduck` | `helm/adhdduck` |
| Default namespace/resource prefix | `angryduck` | `adhdduck` |
| Image repository prefix | `ghcr.io/hatam-abolghasemi/angry-duck-` | `ghcr.io/sysmrk/adhd-duck-` |
| Node state directory | `/var/lib/angryduck` | `/var/lib/adhdduck` |
| Metric prefix | `angryduck_` | `adhdduck_` |
| Integration-test environment prefix | `ANGRYDUCK_` | `ADHDDUCK_` |

Application settings such as `CONTROLLER_URL`, `RESCUE_TOKEN_PATH` and
`PROPAGATE_MAX_UTILIZATION` retain their setting names. Default paths and
addresses containing the project prefix change. Inspect your overrides.

## Safe deployment sequence

1. Build and validate the fork's controller and worker images, or publish a
   release first. The new registry/chart paths are targets, not evidence that
   artifacts have already been published.
2. Save the existing manifests, configuration, Secrets and node state. Keep
   token values out of logs and version control.
3. Stop the old workers before starting the new ones. Never run two image-GC
   workers against the same node runtime during migration.
4. Inspect the old mirror `hosts.toml` files. Graceful shutdown removes files
   managed by the old worker. If any remain, remove only files with the old
   ownership marker after the old worker has stopped; preserve administrator
   supplied registry configuration.
5. Preserve usage and rescue state by copying the old state into the new node
   state directory with the same ownership and permissions, where needed.
   Account for old leases/pins and their expiry before declaring cleanup done.
6. Create the renamed token/registry Secrets securely in the new namespace,
   adapt runtime mounts for the node distribution, and deploy the new chart.
7. Check health, authenticated reports, full layer inventory, preheat and
   mirror reads. Update metrics queries, alerts and Grafana to the new prefix.
8. Keep the previous manifests and images available for rollback. Stop the
   new workers before restoring the old deployment.

The rebranding commit changes repository code and examples. It does not itself
execute this cluster migration, rotate tokens, or publish a release. Existing
test deployments continue to use their current names until migrated.

Historical release notes retain upstream names where they describe upstream
artifacts. Attribution links intentionally continue to point to Angry Duck.
