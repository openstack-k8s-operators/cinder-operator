# Cinder Backend Samples

This directory includes a set of Cinder Backend configuration samples that use
the `kustomize` configuration management tool available through the `oc
kustomize` command.

These samples are not meant to serve as deployment recommendations, just as
working examples to serve as reference, and they use a base OpenStack
configuration that is unlikely to match any real deployment.

For each backend there will be a `backend.yaml` file containing an overlay for
the `OpenStackControlPlane` with just the storage related information.

They will usually also have a secrets `yaml` file where we need to replace the
sample credentials with our real storage ones.

Backend pre-requirements will be listed in that same `backend.yaml` file.
These can range from having to replace the storage system's address and
credentials in a different yaml file, to having to create secrets.

Currently available samples are:

- Ceph
- NFS
- LVM using iSCSI
- LVM using NVMe-TCP
- HPE 3PAR iSCSI
- HPE 3PAR FC
- NetApp ONTAP iSCSI
- NetApp ONTAP FC
- NetApp ONTAP NFS
- Pure Storage iSCSI
- Pure Storage FC
- Pure Storage NVMe-RoCE
- Dell PowerMax iSCSI

**NOTE**: These examples are designed to be applied one at time. If you attempt
to add a second backend by applying a second example it will result in the
first backend being removed. It is certainly possible to define a deployment
with multiple backends, but that requires a single CR that defines all
backends.

## Using Examples

Once we have `crc` running making an OpenStack deployment using one of the
example backends is trivial and it's almost the same as any other deployment
with the addition of creating and exporting the manifest to use:

```
$ cd install_yamls
$ make crc_storage openstack
$ oc kustomize ../cinder-operator/config/samples/backends/lvm/iscsi > ~/openstack-deployment.yaml
$ export OPENSTACK_CR=`realpath ~/openstack-deployment.yaml`
$ make openstack_deploy
```

Be aware that the some of the Cinder examples (mostly all but NFS and Ceph)
will reboot your OpenShift cluster because they use `MachineConfig` manifests
(to deploy things like `iscsid` and `multipathd`) that require a reboot to be
applied due to the immutability of CoreOS (the underlying OpenShift operating
system).  This means that the deployment takes longer and the cluster will stop
responding for a bit.

The reboot will only happen the first time the `MachineConfig` is applied.

For real deployments OpenShift administration may not fall under our
responsibilities, so we may need to reach to someone else to apply the
`MachineConfig`, but for our samples we assume we are on a dev environment and
we can do everything ourselves.

If we already have a deployment working we can always use
`oc kustomize lvm/iscsi | oc apply -f -`. from this directory to make the
changes.

Some backends, like 3PAR and Pure, require a custom container because they have
external dependencies. So there is a need to build a specific container. In
RHOSP, vendors will provide certified containers through Red Hat's container
image registry. For illustration and development purposes this repository
provides samples of what a `Dockerfile` would look like for each of the
vendor's. This `Dockerfile` will be present in the vendor's directory (e.g.
`hpe/Dockerfile` and `pure/Dockerfile`), so we would need to build a container,
make it available in a registry, and then provide it to the cinder operator via
the `containerImage` as seen in the samples.

## Ceph example

For Ceph we need an extra step to deploy the Ceph cluster. In our case it will
be a toy cluster sufficient for minor PoC tests deployed with `make ceph
TIMEOUT=90`, instead of a user-supplied external Ceph cluster. So the steps
are:

```
$ cd install_yamls
$ make ceph TIMEOUT=90
$ make crc_storage openstack
$ oc kustomize ../cinder-operator/config/samples/backends/lvm/iscsi > ~/openstack-deployment.yaml
$ export OPENSTACK_CR=`realpath ~/openstack-deployment.yaml`
$ make openstack_deploy
```

## Adding new samples

We are open to PRs adding new samples for other backends.

All new backends will need to use the `bases/openstack` as a resource and
depending on the transport protocol may also require other bases such as
`bases/iscsid`, `bases/multipathd`, or `bases/nvmeof`.

Most backends will require credentials to access the storage, usually there are
2 types of credentials:

- Configuration options in `cinder.conf`
- External files

You can find the right approach to each of them in the `nfs` sample (for
configuration parameters) and the `ceph` sample (for providing files).
