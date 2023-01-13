# cinder-operator
// TODO(user): Add simple overview of use/purpose

## Description
// TODO(user): An in-depth paragraph about your project and overview of use

## Getting Started
You’ll need a Kubernetes cluster to run against.  Our recommendation for the time being is to use
[OpenShift Local](https://access.redhat.com/documentation/en-us/red_hat_openshift_local/2.2/html/getting_started_guide/installation_gsg) (formerly known as CRC / Code Ready Containers).
We have [companion development tools](https://github.com/openstack-k8s-operators/install_yamls/blob/master/devsetup/README.md) available that will install OpenShift Local for you.

### Running on the cluster
1. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

2. Build and push your image to the location specified by `IMG`:

```sh
make docker-build docker-push IMG=<some-registry>/cinder-operator:tag
```

3. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=<some-registry>/cinder-operator:tag
```

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Undeploy controller
UnDeploy the controller to the cluster:

```sh
make undeploy
```

### Configure Cinder with Ceph backend

The Cinder services can be configured to interact with an external Ceph cluster.
In particular, the `customServiceConfig` parameter must be used, for each defined
`cinder-volume` and `cinder-backup` instance, to override the `enabled_backends`
parameter and inject the Ceph related parameters.
The `ceph.conf` and the `client keyring` must exist as secrets, and can be
mounted by the cinder pods using the `extraMounts` feature.

Create a secret by generating the following file and then apply it using the `oc`
cli.

---
apiVersion: v1
kind: Secret
metadata:
  name: ceph-client-conf
  namespace: openstack
stringData:
  ceph.client.openstack.keyring: |
    [client.openstack]
        key = <secret key>
        caps mgr = "allow *"
        caps mon = "profile rbd"
        caps osd = "profile rbd pool=images"
  ceph.conf: |
    [global]
    fsid = 7a1719e8-9c59-49e2-ae2b-d7eb08c695d4
    mon_host = 10.1.1.2,10.1.1.3,10.1.1.4


Add the following to the spec of the Cinder CR and then apply it using the `oc`
cli.

```
  extraMounts:
    - name: v1
      region: r1
      extraVol:
        - propagation:
          - CinderVolume
          - CinderBackup
          volumes:
          - name: ceph
            projected:
              sources:
              - secret:
                  name: ceph-client-conf
          mounts:
          - name: ceph
            mountPath: "/etc/ceph"
            readOnly: true
```

The following represents an example of the entire Cinder object that can be used
to trigger the Cinder service deployment, and enable the Cinder backend that
points to an external Ceph cluster.


```
apiVersion: cinder.openstack.org/v1beta1
kind: Cinder
metadata:
  name: cinder
  namespace: openstack
spec:
  serviceUser: cinder
  databaseInstance: openstack
  databaseUser: cinder
  cinderAPI:
    replicas: 1
    containerImage: quay.io/tripleowallabycentos9/openstack-cinder-api:current-tripleo
  cinderScheduler:
    replicas: 1
    containerImage: quay.io/tripleowallabycentos9/openstack-cinder-scheduler:current-tripleo
  cinderBackup:
    replicas: 1
    containerImage: quay.io/tripleowallabycentos9/openstack-cinder-backup:current-tripleo
    customServiceConfig: |
      [DEFAULT]
      backup_driver = cinder.backup.drivers.ceph.CephBackupDriver
      backup_ceph_pool = backups
      backup_ceph_user = admin
  secret: cinder-secret
  cinderVolumes:
    volume1:
      containerImage: quay.io/tripleowallabycentos9/openstack-cinder-volume:current-tripleo
      replicas: 1
      customServiceConfig: |
        [DEFAULT]
        enabled_backends=ceph
        [ceph]
        volume_backend_name=ceph
        volume_driver=cinder.volume.drivers.rbd.RBDDriver
        rbd_ceph_conf=/etc/ceph/ceph.conf
        rbd_user=admin
        rbd_pool=volumes
        rbd_flatten_volume_from_snapshot=False
        rbd_secret_uuid=<Ceph_FSID>
  extraMounts:
    - name: cephfiles
      region: r1
      extraVol:
      - propagation:
        - CinderVolume
        - CinderBackup
        extraVolType: Ceph
        volumes:
        - name: ceph
          projected:
            sources:
            - secret:
                name: ceph-client-files
        mounts:
        - name: ceph
          mountPath: "/etc/ceph"
          readOnly: true
```

When the service is up and running, it's possible to interact with the Cinder
API and create the Ceph `cinder type` backend which is associated with the Ceph
tier specified in the config file.

## Example: configure Cinder with additional networks

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
    networkAttachents:
    - storage
  cinderVolume:
    volume1:
      ...
      networkAttachents:
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

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

### How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/)
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
