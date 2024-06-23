# CINDER-OPERATOR

The cinder-operator is an OpenShift Operator built using the Operator Framework
for Go. The Operator provides a way to easily install and manage an OpenStack
Cinder service on OpenShift. This Operator was developed using RDO containers
for openStack but is not limited to them.

Conceptually there are 2 types of nodes: Control Plane Nodes, running OpenStack
control plane services on OpenShift, and External Data Plane (EDPM) Nodes,
running RHEL for computes (and maybe swift).

Some links of interest:

- [OpenStack Kubernetes Operators](https://github.com/openstack-k8s-operators/)
- [Developer Docs](https://github.com/openstack-k8s-operators/dev-docs)
- [User Docs](https://openstack-k8s-operators.github.io/openstack-operator/)

## Getting Started

In this section we'll deploy an operator deployed OpenStack system with a
single compute node and using the LVM backend for block storage. The control
plane will run in a single node OpenShift cluster inside a VM and the compute
node will be a single EDPM node in another VM.

Cloning this repository is not necessary to follow this getting started
section, though it will be for development.

**You need to get your [pull-secrets from Red Hat](
https://cloud.redhat.com/openshift/create/local) and store it in the machine,
for example on your home directory as `pull-secret`.**

Ensure prerequisites are installed:

```sh
sudo dnf -y install git make wget ansible-core
git clone https://github.com/openstack-k8s-operators/install_yamls.git
```

Get the OpenShift cluster up and running:

```sh
cd install_yamls/devsetup

PULL_SECRET=~/pull-secret CPUS=6 MEMORY=24576 DISK=50 make download_tools crc

make crc_attach_default_interface
```

Install the OpenStack operators:

```sh
eval $(crc oc-env)
cd ..
make crc_storage openstack_wait
```

Create the deployment config for the LVM backend and deploy the OpenStack
control plane:

```sh
oc kustomize \
  https://github.com/openstack-k8s-operators/cinder-operator.git/config/samples/backends/lvm/iscsi?ref=main \
  > ~/openstack-deployment.yaml

oc label node --all openstack.org/cinder-lvm=

OPENSTACK_CR=~/openstack-deployment.yaml make openstack_deploy
```

*NOTE*: We are using `--all` to label all the nodes because we only have 1 node
in our OpenShift cluster. For 3 node cluster a node name must be provided
instead.

This will reboot the OpenShift cluster, so it will take a while to be up and
running.

```sh
oc wait --for condition=Ready --timeout=300s
```

Create the EDPM (compute) VM and provisioning:

```sh
cd devsetup
make edpm_compute

cd ..
DATAPLANE_TIMEOUT=40m make edpm_wait_deploy
```

We should now have a working OpenStack deployment and we will be able to run
commands using the `openstackclient` pod:

```sh
oc exec -t openstackclient -- openstack volume service list
oc exec -t openstackclient -- openstack compute service list
```

## Other information

- [Configure Cinder with additional networks](docs/additional_network.md)
- [Expose Cinder API to an isolated network](docs/api_isolated_network.md)
- [Configure Cinder with Ceph backend](docs/ceph_backend.md)
- [Contributing](CONTRIBUTING.md)

## Troubleshooting

### Failure creating PVs

Running `make crc_storage openstack_wait` may fail creating the PVs. It is safe
to just retry the command.

Or we can clean existing PVs with `make crc_storage_cleanup` and retry.

### Timeout deploying EDPM

Running `DATAPLANE_TIMEOUT=40m make edpm_wait_deploy` usually fails due to a
timeout.

You can `watch oc get OpenStackDataPlaneDeployment` until it is ready and then
run `make edpm_nova_discover_hosts`.

### Problems deploying EDPM

We can see the pods that are running ansible playbooks to configure the
compute node with `oc get pod -l app=openstackansibleee --watch` and the do an
`oc logs` on a specific pod.

We can also check the logs filtering by label:
```sh
oc logs -f -l app=openstackansibleee
```

Or we can also run a complex command to log the different pods automatically:

```sh
while true; do oc logs -n openstack -f $(oc get pod -l osdpd=openstack-edpm --field-selector='status.phase=Running' --no-headers -o name) 2>/dev/null || echo -n .; sleep 1; done
```

### Go into the EDPM node:

```sh
ssh -i ~/install_yamls/out/edpm/ansibleee-ssh-key-id_rsa root@192.168.122.100
```

Once provisioned cloud-admin user will exist.

### Go into the OpenShift node

```sh
oc debug $(oc get node -o name)
chroot /host
```

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
