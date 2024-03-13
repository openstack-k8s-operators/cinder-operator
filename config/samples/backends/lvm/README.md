# Cinder LVM Backend Samples

Two LVM examples are provided using different transport protocols:

- LVM using iSCSI
- LVM using NVMe-TCP

The LVM is a special case, because the stored data is actually on the OCP node
and not an external storage system. This has several implications:

- OCP nodes will reboot after the samples manifests are applied because they
  are creating `MachineConfig` objects to start daemons and create the LVM
  backing file.

- To prevent issues with exported volumes the cinder-operator will
  automatically use the host network when using the LVM backend, and the
  cinder backend needs to be configured to use the host's VLAN IP address. This
  also means that the cinder-volume service doesn't need any
  `networkAttachments`.  The config sample has the IP address used by the CRC
  deployment using `install_yamls`, so any other OCP cluster will need to
  change this.

- We need to mark one of our OCP nodes with a specific label so that:

  - The LVM `MachineConfig` that creates a file based LVM Volume Group and
    ensures it is mounted on boot only really runs in one node (even if the
    systemd unit is deployed in all the nodes).

  - The cinder-volume service for the LVM backend can only run on the node that
    has the LVM Volume Group.

To mark the first ready node as the lvm node we can run:

```bash
$ oc label node \
  $(oc get nodes --field-selector spec.unschedulable=false -l node-role.kubernetes.io/worker -o jsonpath="{.items[0].metadata.name}") \
  openstack.org/cinder-lvm=
```

If we have a specific node in mind we can just do:

```bash
$ oc label node <nodename> openstack.org/cinder-lvm=
```

Or if we are running a single node OCP cluster we can run:
```bash
$ oc label node --all openstack.org/cinder-lvm=
```

To see how this label is being used please look into the following files:

- `./iscsi/backend.yaml`
- `./nvme-tcp/backend.yaml`
- `../bases/lvm/lvm.yaml`
