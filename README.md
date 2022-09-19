# CINDER-OPERATOR

The cinder-operator is an OpenShift Operator built using the Operator Framework
for Go. The Operator provides a way to easily install and manage an OpenStack
Cinder installation on OpenShift. This Operator was developed using RDO
containers for openStack.

## Getting started

**NOTES:**

- *The project is in a rapid development phase and not yet intended for
  production consumption, so instructions are meant for developers.*

- *If possible don't run things in your own machine to avoid the risk of
  affecting the development of your other projects.*

Here we'll explain how to get a functiona OpenShift deployment running inside a
VM that is running MariaDB, RabbitMQ, KeyStone, Glance, and Cinder services
against a Ceph backend.

There are 4 steps:

- [Install prerequisites](#prerequisites)
- [Deploy an OpenShift cluster](#openshift-cluster)
- [Prepare Storage](#storage)
- [Deploy OpenStack](#deploy)

### Prerequisites

There are some tools that will be required through this process, so the first
thing we do is install them:

```sh
sudo dnf install -y git wget make ansible-core python-pip podman gcc
```

We'll also need this repository as well as `install_yamls`:

```sh
cd ~
git clone https://github.com/openstack-k8s-operators/install_yamls.git
git clone https://github.com/openstack-k8s-operators/cinder-operator.git
```

### OpenShift cluster

There are many ways get an OpenShift cluster, and our recommendation for the
time being is to use [OpenShift Local](https://access.redhat.com/documentation/en-us/red_hat_openshift_local/2.5/html/getting_started_guide/index)
(formerly known as CRC / Code Ready Containers).

To help with the deployment we have [companion development tools](https://github.com/openstack-k8s-operators/install_yamls/blob/master/devsetup)
available that will install OpenShift Local for you and will also help with
later steps.

Running OpenShift requires a considerable amount of resources, even more when
running all the operators and services required for an OpenStack deployment,
so make sure that you have enough resources in the machine to run everything.

You will need at least 5 CPUS and 16GB of RAM, preferably more, just for the
local OpenShift VM.

**You will also need to get your [pull-secrets from Red Hat](
https://cloud.redhat.com/openshift/create/local) and store it in the machine,
for example on your home directory as `pull-secret`.**

```sh
cd ~/install_yamls/devsetup
PULL_SECRET=~/pull-secret CPUS=6 MEMORY=20480 make download_tools crc
```

This will take a while, but once it has completed you'll have an OpenShift
cluster ready.

Now you need to set the right environmental variables for the OCP cluster, and
you may want to logging to the cluster manually (although the previous step
already logs in at the end):

```sh
eval $(crc oc-env)
```

**NOTE**: When CRC finishes the deployment the `oc` client is logged in, but
the token will eventually expire, in that case we can login again with
`oc login -u kubeadmin -p 12345678 https://api.crc.testing:6443`, or use the
[helper functions](CONTRIBUTING.md#helpful-scripts).

Let's now get the cluster version confirming we have access to it:

```sh
oc get clusterversion
```

If you are running OCP on a different machine you'll need additional steps to
[access its dashboard from an external system](https://github.com/openstack-k8s-operators/install_yamls/tree/master/devsetup#access-ocp-from-external-systems).

### Storage

There are 2 kinds of storage we'll need: One for the pods to run, for example
for the MariaDB database files, and another for the OpenStack services to use
for the VMs.

To create the pod storage we run:

```sh
cd ~/install_yamls
make crc_storage
```

As for the storage for the OpenStack services, at the time of this writing only
NFS and Ceph are supported.

For simplicity's sake we'll use a *toy* Ceph cluster that runs in a single
local container using a simple script provided by this project. Beware that
this script overrides things under `/etc/ceph`:

**NOTE**: This step must be run after the OpenShift VM is running because it
binds to an IP address created by it.

```sh
~/cinder-operator/hack/dev/create-ceph.sh
```

Using an external Ceph cluster is also possible, but out of the scope of this
document, and the manifest we'll use have been tailor made for this specific
*toy* Ceph cluster.

### Deploy

Deploying the podified OpenStack control plane is a 2 step process. First
deploying the operators, and then telling the openstack-operator how we want
our OpenStack deployment to look like.

Deploying the openstack operator:

```sh
cd ~/install_yamls
make openstack
```

Once all the operator ready we'll see the pod with:

```sh
oc get pod -l control-plane=controller-manager
```

And now we can tell this operator to deploy RabbitMQ, MariaDB, Keystone, Glance
and Cinder using the Ceph *toy* cluster:

```sh
export OPENSTACK_CR=`realpath ~/cinder-operator/hack/dev/openstack-ceph.yaml`
cd ~/install_yamls
make openstack_deploy
```

After a bit we can see the 5 operators are running:

```sh
oc get pods -l control-plane=controller-manager
```

And a while later the services will also appear:

```sh
oc get pods -l app=mariadb
oc get pods -l app.kubernetes.io/component=rabbitmq
oc get pods -l service=keystone
oc get pods -l service=glance
oc get pods -l service=cinder
```

### Configure Clients

Now that we have the OpenStack services running we'll want to setup the
different OpenStack clients.

For convenience this project has a simple script that does it for us:

```sh
source ~/cinder-operator/hack/dev/osp-clients-cfg.sh
```

We can now see available endpoints and services to confirm that the clients and
the Keystone service work as expected:

```sh
openstack service list
openstack endpoint list
```

Upload a glance image:

```sh
cd
wget http://download.cirros-cloud.net/0.5.2/cirros-0.5.2-x86_64-disk.img -O cirros.img
openstack image create cirros --container-format=bare --disk-format=qcow2 < cirros.img
openstack image list
```

And create a cinder volume:

```sh
openstack volume create --size 1 myvolume
```

## Cleanup

To delete the deployed OpenStack we can do:

```sh
cd ~/install_yamls
make openstack_deploy_cleanup
```

Once we've done this we need to recreate the PVs that we created at the start,
since some of them will be in failed state:

```sh
make crc_storage_cleanup crc_storage
```

We can now remove the openstack-operator as well:

```sh
make openstack_cleanup
```

# ADDITIONAL INFORMATION

**NOTE:** Run `make --help` for more information on all potential `make`
targets.

More information about the Makefile can be found via the [Kubebuilder
Documentation]( https://book.kubebuilder.io/introduction.html).

# LICENSE

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
