# Configure Cinder with additional networks

The Cinder spec can be used to configure Cinder to have the pods
being attached to additional networks to e.g. connect to a Ceph
RBD server on a dedicated storage network.

Create a network-attachement-definition which then can be referenced
from the Cinder CR.

```
---
apiVersion: k8s.cni.cncf.io/v1
kind: NetworkAttachmentDefinition
metadata:
  name: storage
  namespace: openstack
spec:
  config: |
    {
      "cniVersion": "0.3.1",
      "name": "storage",
      "type": "macvlan",
      "master": "enp7s0.21",
      "ipam": {
        "type": "whereabouts",
        "range": "172.18.0.0/24",
        "range_start": "172.18.0.50",
        "range_end": "172.18.0.100"
      }
    }
```

The following represents an example of Cinder resource that can be used
to trigger the service deployment, and have the Cinder Volume and Cinder Backup
service pods attached to the storage network using the above NetworkAttachmentDefinition.

```
apiVersion: cinder.openstack.org/v1beta1
kind: Cinder
metadata:
  name: cinder
spec:
  ...
  cinderBackup:
    ...
    networkAttachments:
    - storage
  cinderVolume:
    volume1:
      ...
      networkAttachments:
      - storage
...
```

When the service is up and running, it will now have an additional nic
configured for the storage network:

```
# oc rsh cinder-volume-volume1-0
Defaulted container "cinder-volume" out of: cinder-volume, probe, init (init)
sh-5.1# ip a
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
3: eth0@if334: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1450 qdisc noqueue state UP group default
    link/ether 0a:58:0a:82:01:44 brd ff:ff:ff:ff:ff:ff link-netns 7ea23955-d990-449d-81f9-fd57cb710daa
    inet 10.130.1.68/23 brd 10.130.1.255 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fe80::bc92:5cff:fef3:2e27/64 scope link
       valid_lft forever preferred_lft forever
4: net1@if17: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
    link/ether 26:35:09:03:71:b3 brd ff:ff:ff:ff:ff:ff link-netns 7ea23955-d990-449d-81f9-fd57cb710daa
    inet 172.18.0.20/24 brd 172.18.0.255 scope global net1
       valid_lft forever preferred_lft forever
    inet6 fe80::2435:9ff:fe03:71b3/64 scope link
       valid_lft forever preferred_lft forever
```
