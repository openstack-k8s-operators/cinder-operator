# Cinder LVM Backend Samples

Two LVM examples are provided using different transport protocols:

- LVM using iSCSI
- LVM using NVMe-TCP

The LVM is a special case, because the stored data is actually on the OCP node
and not an external storage system. This has several implications:

- OCP nodes will reboot after the samples manifests are applied because they
  are creating `MachineConfig` objects to start daemons and create the LVM
  backing file.

- Network configuration on the OCP host needs to be different, as it needs a
  macvlan interface on top of the storage VLAN interface. This can be easily
  done by modifying the operators deployment step running:
  `NETWORK_STORAGE_MACVLAN=true make openstack`. Other option is to manually
  modify the `nncp` of the OCP node.

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

To see how this label is being used please look into the following files:

- `./iscsi/backend.yaml`
- `./nvme-tcp/backend.yaml`
- `../bases/lvm/lvm.yaml`
