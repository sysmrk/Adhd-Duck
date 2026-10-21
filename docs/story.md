# Project history

ADHD Duck is a fork of Angry Duck, originally developed by hatam-abolghasemi.
The upstream project introduced preheating after registry pushes, layer-based
peer distribution, rescue for failing pulls, local mirroring, image repair
and image garbage collection. Its original development story is available
[upstream](https://github.com/hatam-abolghasemi/Angry-Duck/blob/5102a56b8037bce1b249d2e3bbfcd32e55f66d8f/docs/story.md).

This fork started from upstream version 1.8.15 and first validated the existing
behavior on a single-node K3s cluster in WSL. It then added targeted fixes for
request and configuration validation, pinned digest integrity, seed capacity,
controller receipt-time liveness and GC telemetry. These fixes include unit
and real-containerd regression tests.

The project was subsequently renamed ADHD Duck. Source comments and repository
documentation are maintained in English. The upstream license and attribution
remain intact. Multi-node production behavior and new architectural features
require their own tests; the rename does not establish those guarantees.
