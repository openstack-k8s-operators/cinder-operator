# Hacking

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
