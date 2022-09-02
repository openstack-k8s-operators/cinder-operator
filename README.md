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

The Cinder spec API can be used to configure and customize the Ceph backend. In
particular, the `customServiceConfig` parameter should be used, for each
defined volume, to override the `enabled_backends` parameter, which must exist
in `cinder.conf` to make the `cinderVolume` pod run. The global `cephBackend`
parameter is used to specify the Ceph client-related "key/value" pairs required
to connect the service with an external Ceph cluster. Multiple external Ceph
clusters are not supported at the moment. The following represents an example
of the Cinder object that can be used to trigger the Cinder service deployment,
and enable the Cinder backend that points to an external Ceph cluster.

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
  secret: cinder-secret
  cinderVolumes:
    volume1:
      containerImage: quay.io/tripleowallabycentos9/openstack-cinder-volume:current-tripleo
      replicas: 1
      customServiceConfig: |
        [DEFAULT]
        enabled_backends=ceph
  cephBackend:
    cephFsid: <CephClusterFSID>
    cephMons: <CephMons>
    cephClientKey: <cephClientKey>
    cephUser: openstack
    cephPools:
      cinder:
        name: volumes
      nova:
        name: vms
      glance:
        name: images
      cinder_backup:
        name: backup
      extra_pool1:
        name: ceph_ssd_tier
      extra_pool2:
        name: ceph_nvme_tier
      extra_pool3:
        name: ceph_hdd_tier
```

When the service is up and running, it's possible to interact with the cinder
API and create the Ceph `cinder type` backend which is associated with the Ceph
tier specified in the config file.


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

