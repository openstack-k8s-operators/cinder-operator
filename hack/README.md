# Hacking

### Testing PRs

To facilitate testing PRs we have the `hack/checkout_pr.sh` script that will
checkout the specified cinder PR into local branch pr#, then recursively check
the dependencies using the PRs commit messages and then replace modules in our
go workspace to use those dependencies.

Example to get the PR#65 from cinder-operator:

```sh
$ hack/checkout_pr.sh 65
Cleaning go.work
Fetching PR 65 on upstream/pr65
Checking PR dependecies
Setting dependencies: lib-common=88
Source for lib-common PR#88 is github.com/fmount/lib-common@extra_volumes
Checking the go mod version for branch @extra_volumes
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/common=github.com/fmount/lib-common/modules/common@v0.0.0-20221123175721-3e11759d254f
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/database=github.com/fmount/lib-common/modules/database@v0.0.0-20221123175721-3e11759d254f
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/storage=github.com/fmount/lib-common/modules/storage@v0.0.0-20221123175721-3e11759d254f
```

This script leverages scripts `hack/cleandeps.py` that cleans existing
dependencies, `hack/showdeps.py` that shows dependencies for a given PR, and
`hack/setdeps.py` that sets the go workspace replaces.

### Ceph cluster

As describe in the [Getting Started Guide](../README.md#getting-started), the
`dev/create-ceph.sh` script can help us create a *toy* Ceph cluster we can use
for development.

### LVM backend

Similar to the script that creates a *toy* Ceph backend there is also a script
called `dev/create-lvm.sh` that create an LVM Cinder VG, within the CRC VM,
that can be used by the Cinder LVM backend driver.

### Helpers

If we source `hack/dev/helpers.sh` we'll get a couple of helper functions:

- `crc_login`: To login to the OpenShift cluster.
- `crc_ssh`: To SSH to the OpenShift VM or to run SSH commands in it.

### SSH OpenShift VM

We can SSH into the OpenShift VM multiple ways: Using `oc debug`, using `ssh`,
or using the `virsh console`.

With `oc debug`:

```sh
$ oc get node
NAME                 STATUS   ROLES           AGE   VERSION
crc-p9hmx-master-0   Ready    master,worker   26d   v1.24.0+4f0dd4d

$ oc debug node/crc-p9hmx-master-0

sh-4.4# chroot /host
```

To use `ssh` we can do:

```sh
$ ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ~/.crc/machines/crc/id_ecdsa core@`crc ip`

[core@crc-p9hmx-master-0 ~]$
```

Or we can use the helper function defined before to just do `crc_ssh`.

### Containers in VM

The OpenShift VM runs CoreOS and uses [Cri-O](https://cri-o.io/) as the
container runtime, so once we are inside the container we need to use `crictl`
to interact with the containers:

```sh
[core@crc-p9hmx-master-0 ~]$ sudo crictl ps
```

And its configuration files are under `/etc/containers`.
