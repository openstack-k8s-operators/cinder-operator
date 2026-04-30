# AGENTS.md - cinder-operator

## Project overview

cinder-operator is a Kubernetes operator that manages
[OpenStack Cinder](https://docs.openstack.org/cinder/latest/) (the block
storage service: volume provisioning, snapshots, backups, and multi-backend
management) on OpenShift/Kubernetes. It is part of the
[openstack-k8s-operators](https://github.com/openstack-k8s-operators) project.

Key Cinder domain concepts: **backends** (Ceph RBD, LVM, NFS, NetApp, Dell,
HPE, Pure Storage), **volume types**, **scheduler** (capacity-weighted
placement), **backup** (volume backup/restore).

Go module: `github.com/openstack-k8s-operators/cinder-operator`
API group: `cinder.openstack.org`
API version: `v1beta1`

## Tech stack

| Layer | Technology |
|-------|------------|
| Language | Go (modules, multi-module workspace via `go.work`) |
| Scaffolding | [Kubebuilder v4](https://book.kubebuilder.io/) + [Operator SDK](https://sdk.operatorframework.io/) |
| CRD generation | controller-gen (DeepCopy, CRDs, RBAC, webhooks) |
| Config management | Kustomize |
| Packaging | OLM bundle |
| Testing | Ginkgo/Gomega + envtest (functional), KUTTL (integration) |
| Linting | golangci-lint (`.golangci.yaml`) |
| CI | Zuul (`zuul.d/`), Prow (`.ci-operator.yaml`), GitHub Actions |

## Custom Resources

| Kind | Purpose |
|------|---------|
| `Cinder` | Top-level CR. Owns the database, keystone service, transport URL, and spawns sub-CRs for each service component. |
| `CinderAPI` | Manages the Cinder API deployment (httpd/WSGI). |
| `CinderScheduler` | Manages the scheduler service deployment. |
| `CinderVolume` | Manages volume service instances (one per backend). |
| `CinderBackup` | Manages the backup service deployment. |

The `Cinder` CR has defaulting and validating admission webhooks.
Sub-CRs are created and owned by the `Cinder` controller -- not intended to
be created directly by users.

## Directory structure

| Directory | Contents |
|-----------|----------|
| `api/v1beta1/` | CRD types (`cinder_types.go`, `cinderapi_types.go`, `cinderbackup_types.go`, `cinderscheduler_types.go`, `cindervolume_types.go`), conditions, webhook markers |
| `cmd/` | `main.go` entry point |
| `internal/controller/` | Reconcilers: `cinder_controller.go`, `cinderapi_controller.go`, `cinderbackup_controller.go`, `cinderscheduler_controller.go`, `cindervolume_controller.go` |
| `internal/cinder/` | Cinder-level resource builders (db-sync, common helpers) |
| `internal/cinderapi/` | CinderAPI resource builders |
| `internal/cinderbackup/` | CinderBackup resource builders |
| `internal/cinderscheduler/` | CinderScheduler resource builders |
| `internal/cindervolume/` | CinderVolume resource builders |
| `internal/webhook/v1beta1/` | Webhook implementation |
| `templates/` | Config files and scripts mounted into pods via `OPERATOR_TEMPLATES` env var |
| `config/crd,rbac,manager,webhook/` | Generated Kubernetes manifests (CRDs, RBAC, deployment, webhooks) |
| `config/samples/` | Example CRs (Kustomize overlays). `backends/{ceph,lvm,nfs,netapp,dell,hpe,pure}` for storage. `httpd-overrides/` for Apache customization. |
| `test/functional/` | envtest-based Ginkgo/Gomega tests |
| `test/kuttl/` | KUTTL integration tests (scaling, TLS scenarios) |
| `hack/` | Helper scripts (CRD schema checker, local webhook runner) |
| `docs/` | Backend guides, networking, probes, scheduler performance, user guide |

## Build commands

After modifying Go code, always run: `make generate manifests fmt vet`.

## Code style guidelines

- Follow standard openstack-k8s-operators conventions and lib-common patterns.
- Use `lib-common` modules for conditions, endpoints, TLS, storage, and other
  cross-cutting concerns rather than re-implementing them.
- CRD types go in `api/v1beta1/`. Controller logic goes in
  `internal/controller/`. Resource-building helpers go in `internal/cinder*`
  packages matching the CR they support.
- Config templates are plain files in `templates/` -- they are mounted at
  runtime via the `OPERATOR_TEMPLATES` environment variable.
- Webhook logic is split between the kubebuilder markers in `api/v1beta1/` and
  the implementation in `internal/webhook/v1beta1/`.

## Testing

- Functional tests use the envtest framework with Ginkgo/Gomega and live in
  `test/functional/`.
- KUTTL integration tests live in `test/kuttl/`.
- Run all functional tests: `make test`.
- When adding a new field or feature, add corresponding test cases in
  `test/functional/` and update `test/functional/cinder_test_data.go` with
  fixture data.

## Key dependencies

- [lib-common](https://github.com/openstack-k8s-operators/lib-common): shared modules for conditions, endpoints, database, TLS, secrets, etc.
- [infra-operator](https://github.com/openstack-k8s-operators/infra-operator): RabbitMQ and topology APIs.
- [mariadb-operator](https://github.com/openstack-k8s-operators/mariadb-operator): database provisioning.
- [keystone-operator](https://github.com/openstack-k8s-operators/keystone-operator): identity service registration.
- [gophercloud](https://github.com/gophercloud/gophercloud): Go OpenStack SDK (indirect).
