# Validation evidence

Date: 2026-11-09. Functional/security revision:
`210378d4108964a3d19b3febb394d04f2ff7241f`.

[GitHub Actions run 42572391534](https://github.com/sysmrk/Adhd-Duck/actions/runs/42572391534)
completed successfully:

- Formatting, go vet (normal and integration build tags), build, reachable-code
  vulnerability scan, race-enabled unit tests.
- Real-containerd integration tests on 1.7.36 and 2.4.2.
- Controller and worker container builds.
- Helm lint and security-contract rendering tests.

Local checks also passed with Go 1.27.3: race-enabled unit tests, vet including
integration build tags, govulncheck (`No vulnerabilities found`), actionlint,
Helm security-contract tests and a running controller binary tested over HTTPS.
The binary smoke verified health success, rejection of status without a client
certificate, rejection without a token, acceptance with both credentials, and
rejection of an unauthenticated webhook. TLS tests reject untrusted server CAs
and credential-bearing redirects.

Subsequent supply-chain workflow changes pin base image digests and action
commits and add an OS vulnerability gate before publication. Their own Actions
run and the final Release run must be checked separately; this record does not
predeclare their success. Published artifact attestations refer to the actual
release source commit and digest.

No organization-specific production cluster, CNI, registry outage under real
load, or disk layout was available here. Complete the site acceptance table in
[production.md](production.md) before expanding to all production nodes. Passing
a unit test cannot replace that evidence.

The pinned supply-chain and OS scanning pipeline passed on main in
[CI run 42573138923](https://github.com/sysmrk/Adhd-Duck/actions/runs/42573138923).
[Release run 42572888654, attempt 2](https://github.com/sysmrk/Adhd-Duck/actions/runs/42572888654)
completed successfully and published v1.9.0, images, chart and attestations.
One intermediate CI attempt failed while downloading upstream archives (HTTP
504); it did not fail a program test. Subsequent runs passed; bounded download
retries are now configured. The initial release-page API call was denied after
workflow drift on main; aligning the approved workflow baseline allowed the
unchanged release source to be published on retry.

Release maintainers: freeze .github/workflows changes while publishing a version.
GitHub's release API requires additional workflow permission when the target's
workflow files differ from the current default branch; GITHUB_TOKEN cannot grant
that permission. Do not compensate by changing an existing version's source or
retagging its artifacts. Fix the baseline or publish the original source with an
authorized maintainer credential instead.
