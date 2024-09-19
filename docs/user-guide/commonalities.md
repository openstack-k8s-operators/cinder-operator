# General concepts

This section covers some aspects common to the different storage services, and
they are the base for most of the topics covered in the `cinder` specific guide,
that assumes these are well understood.

Table of Contents:

- [1. Storage Back-end](#1-storage-back-end)
- [2. Enabling a service](#2-enabling-a-service)
- [3. Scaling services](#3-scaling-services)
- [4. Setting service configuration options](#4-setting-service-configuration-options)
- [5 Restricting where services run](#5-restricting-where-services-run)
- [6 Using external data in the services](#6-using-external-data-in-the-services)
- [7 Restricting resources used by a service](#7-restricting-resources-used-by-a-service)
- [8 Setting API timeouts](#8-setting-api-timeouts)
- [9 Storage networking](#9-storage-networking)
- [10 Using other container images](#10-using-other-container-images)

## 1. Storage Back-end

All *persistent* storage services need to store data somewhere, and that is
usually in a remote storage solution or back-end. OpenStack services support the
different types of back-ends using their own specific drivers.

These storage back-end solutions are usually vendor specific and are only
supported by one or two of the *persistent* storage services, with the exception
of Ceph Storage, that can serve as a back-end for all four types of
*persistent* storage available in OpenStack.

It is beyond the scope of this guide to go into all the benefits that a
clustered storage solution like Ceph provides (please refer to the [Red Hat Ceph
Storage 7 Architecture
Guide](https://access.redhat.com/documentation/en-us/red_hat_ceph_storage/7/html/architecture_guide/index)
for additional information), but it is worth mentioning that when using Ceph in
OpenStack on OpenShift we get some additional benefits beyond its inherent
scaling and redundancy, because OpenStack services such as Block Storage, Image,
and Compute have specific optimizations and behaviors to make the most out of
having Ceph as the common back-end.

When using Ceph for Object Storage the swift service will not be deployed and
Ceph will serve as a direct stand-in, and OpenStack services will access it
using a common interface.

Please refer to the [Configure Cinder with Ceph backend](../ceph_backend.md)
guide for additional information on how to use Ceph in OpenStack on OpenShift.

### Certification

To promote the use of best practices and provide a path for improved
supportability and interoperability, Red Hat has a certification process for
OpenStack back-ends, but this project in itself doesn't have them, so
additional work may be necessary to deploy specific back-ends.

### Transport protocols

Different back-ends available in the storage services may use different storage
transport protocols, and these protocols may or may not be enabled by default
at the Operating System level on deployments. Additional steps are necessary
for transport protocols that are not enabled by default.

The recommendation is to install all the transport protocol requirements as a
day-0 operation during the installation of OpenShift or once it is
installed and we haven’t started deploying additional operators, as these
requirements usually require rebooting all the OpenShift nodes, which can be
a slow process if there are already additional pods running.

## 2. Enabling a service

There are two parts to enabling a service, first is indicating to the installer
that we want the service to be deployed and the second is to have a `replicas`
value greater than `0`.

To tell the installer that the service must be deployed, which will create a
database, keystone entries and other resources necessary for the deployment, we
use the field called `enabled: true`.

Most storage services (cinder, glance, swift) are enabled by default, so we
don’t need to specify it, but the Shared File Systems service (manila) defaults
to `false`, so it needs to be explicitly set for it to be enabled.

---

> **⚠ Attention:** Changing `enabled` to `false` will delete the service
database.  More on this topic on [Scrubbing a Service of the Performing Cinder
operations guide](operations.md).

---

Most components within a service have a `replicas` value set to `1`, so they
will be automatically deployed once `enabled` is set to `true`. For some other
services the default value for `replicas`  is `0`, so we need to change them
explicitly; for example for `cinderBackup` and elements in the `cinderVolumes`
field the default is `0`.


## 3. Scaling services

Most OpenStack components support being deployed in Active-Active High
Availability mode, and the way to do this is to set `replicas` to a number
greater than `1`.

There is no predefined value for the `replicas` field, and we find different
defaults based on the component specifics:

* Components that have the number of replicas set to a positive number by
  default. These are services that don’t require additional configuration and
  can run with their defaults. E.g. cinder api and cinder scheduler have
  `replicas: 1` by default.

* Components that have the number of replicas set to `0`. These are optional
  components or components that need a specific configuration to work (e.g.
  cinder backup).

* Components that have a default value greater than `1`.

When we configure a component to work as Active-Active, by setting its
`replicas` to a number greater than `1`, the pods will be distributed according
to the [OpenShift affinity
rules](https://docs.openshift.com/container-platform/4.16/nodes/scheduling/nodes-scheduler-pod-affinity.html)
defined by the installer.

The standard affinity rule is to distribute the pods of the same component to
different OpenShift (OCP) hosts whenever possible. If at a given time there are
more pods that need to be running than nodes then some of the nodes will be
running multiple pods for the same component.

Each OpenStack has its own restrictions and recommendations for their individual
components, so please refer to their respective configuration sections to find
out specific recommendations and restrictions.

## 4. Setting service configuration options

The installation configuration for OpenStack on OpenShift is defined in an
`OpenStackControlPlane` manifest that has general sections as well as
per-service specific sections. Each storage service has its own section in the
`OpenStackControlPlane` spec for its configuration. The naming will match the
service codename `cinder`, `glance`, `manila`, and `swift`:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:

  glance:

  manila:

  swfit:
```

This guide assumes that the contents of this manifest exists in a file called
`openstack_control_plane.yaml`, and that we have either deployed it already in
our OpenShift cluster or we are editing it and creating its content, so when the
guide asks to make sure something is defined, it is saying that it should be
present in that file. It is possible to edit the contents directly on the
OpenShift cluster using `oc edit`, but it is not recommended, and using a file
and then applying it is preferred.

---

> **⚠ Attention:** Any change applied to a service configuration will
immediately trigger a restart of the service to use the new configuration.

---

The most important thing to understand when configuring storage services is the
concept of a *snippet*. A snippet is a fragment of a service configuration file
that has meaning in itself and could be a valid configuration file on its own
and is used “as is” to construct the final service configuration provided to the
service.

The following example is not a valid *snippet* because it doesn’t have a
section, so it wouldn’t be a valid service configuration file on its own.

```
db_max_retries = 45
```

This other *snippet* would be valid, as it has a section:

```
[database]
db_max_retries = 45
```

Except in very few occasions, there is no interpretation of the contents of a
*snippet*, so a specific configuration option will not trigger a series of
configuration changes elsewhere, in the same service or in other OpenStack
service, to make things consistent or reasonable. It is the system
administrator’s responsibility to configure things correctly. If a feature
requires changes in multiple OpenStack services to work, then the system
administrator will have to make the appropriate changes in the configuration
sections of the different services.

This has the downside of being a bit manual, but it has the benefits of reducing
the installer specific knowledge needed and leverage the OpenShift and OpenStack
service specific knowledge, while giving full control and transparency over all
existing configuration options.

To this effect OpenStack on OpenShift services come with sensible defaults and
only need *snippets* for the specific deployment configurations, such as the
Block Storage service backend configuration.

There are two types of snippets based on their privacy: Public and sensitive,
and services can have one or both of these, depending on our needs and
preferences.

### Public *snippets*:

These are *Snippets* that contain basic service configuration, for example the
value of the `debug` configuration option on a service. This information can be
present directly as plain text in the manifest file as they have no sensitive
information. They are set in the `customServiceConfig` of a service section
within the `OpenStackControlPlane` spec.

Here is an extract of an `OpenStackControlPlane` file to show a *snippet* that
enables debug level logs in the cinder scheduler:

```
----
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderScheduler:
        customServiceConfig: |
          [DEFAULT]
          debug = True
```

Some of the services have a top level `customServiceConfig` to facilitate
setting configuration options, like the above mentioned `debug` mode, in all its
services. So for example enabling debug log levels in all the Block Storage
services can be done like this:

```
----
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      customServiceConfig: |
        [DEFAULT]
        debug = True

```

### Sensitive *snippets*:

These are *Snippets* that contain sensitive configuration options that, if
exposed, pose a security risk. For example the credentials, username and
password, to access the management interface of a Block Storage service
back-end.

In OpenShift it is not recommended to store sensitive information in the CRs
themselves, so most OpenStack operators have a mechanism to use OpenShift
`Secrets` for sensitive configuration parameters of the services and then use
them by reference in the `customServiceConfigSecrets` section of the service.
This `customServiceConfigSecrets` is similar to the `customServiceConfig` except
it doesn’t have the *snippet* itself but a reference to secrets with the
*snippets*.

Another difference is that there is no global `customServiceConfigSecrets`
section for the whole service like there was for the `customServiceConfig`.

Let’s see an example of how we could use a secret to set an NFS configuration
options in cinder, first creating a `Secret` with the sensitive *snippet*:

```
apiVersion: v1
kind: Secret
metadata:
  labels:
	service: cinder
	component: cinder-volume
  name: cinder-volume-nfs-secrets
type: Opaque
stringData:
  cinder-volume-nfs-secrets: |
	[nfs]
	nas_host=192.168.130.1
	nas_share_path=/var/nfs/cinder
```

Then using this secret named `cinder-volume-nfs-secrets` in the `cinder`
section:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderVolumes:
        nfs:
          customServiceConfigSecrets:
          - cinder-volume-nfs-secrets
          customServiceConfig: |
            [nfs]
            volume_backend_name=nfs
            volume_driver=cinder.volume.drivers.nfs.NfsDriver
```

---

> **⚠ Attention:** Remember that all snippets must be valid, so the
configuration section (`[nfs]` in the above example) must be present in all of
them.

---

It is possible, and common, to combine multiple `customServiceConfig` and
`customServiceConfigSecrets` to build the configuration for a specific service.
For example having a global *snippet* in the top level `customServiceConfig`
with the `debug` level, a *snippet* with generic back-end configuration options
in the service’s `customServiceConfig` and a `customServiceConfigSecrets` with
the sensitive information. See this example that combines above examples:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      customServiceConfig: |
        [DEFAULT]
        debug = True
      cinderVolumes:
        nfs:
          customServiceConfigSecrets:
          - cinder-volume-nfs-secrets
          customServiceConfig: |
            [nfs]
            volume_backend_name=nfs
            volume_driver=cinder.volume.drivers.nfs.NfsDriver
```

When there is sensitive information in the service configuration then it becomes
a matter of personal preference whether to store all the configuration in the
`Secret` or only the sensitive parts.

---

> **⚠ Attention:** There is no validation of the snippets used during the
installation, they are passed to the service as-is, so any configuration option
in a snippet added outside a section could have unexpected consequences.

---

## 5 Restricting where services run

In OpenStack on OpenShift deployments, whether it’s a 3 nodes master/worker
deployment or a 3 masters and 3 workers deployment, the installer will run the
services in any worker node available based on some affinity rules to prevent
having multiple identical pods running on the same node.

Storage services are very diverse, and so are their individual components, to
show this we can look at the Block Storage service (cinder) where one can
clearly see different requirements for its individual components: the cinder
scheduler is a very light service with low memory, disk, network, and CPU usage;
cinder api has a higher network usage due to resource listing requests; the
cinder volume will have a high disk and network usage since many of its
operations are in the data path (offline volume migration, create volume from
image, etc.), and then we have the cinder backup which has high memory, network,
and CPU (to compress data) requirements.

Given these requirements it may be preferable not to let these services wander
all over your OCP worker nodes with the possibility of impacting other services,
or maybe you don’t mind the light services wandering around but you want to pin
down the heavy ones to a set of infrastructure nodes.

There are also hardware restrictions to take into consideration, because when
using a Fibre Channel (FC) Block Storage service back-end then the cinder
volume, cinder backup, and maybe even the Image Service (glance) (if it’s using
the Block Storage service as a backend) services need to run on an OpenShift
host that has an HBA card. If all the OpenShift worker nodes meet the hardware
requirements of the services (e.g. have HBA cards), then there’s no problem to
let the installer freely place the pods, but if only a subset of the nodes meet
the requirements, we need to indicate this restriction to the installer.

The OpenStack on OpenShift installer allows a great deal of flexibility on where
to run the OSP services, as it leverages existing OpenShift functionality.
Using node labels to identify OpenShift nodes that are eligible to run the
different services and then using those labels in the `nodeSelector` field.

The `nodeSelector` field follows the standard OpenShift `nodeSelector` field
behaving in the exact same way. For more information, see [About node
selectors](https://docs.openshift.com/container-platform/4.16/nodes/scheduling/nodes-scheduler-node-selectors.html)
in OpenShift Container Platform Documentation.

This field is present at all the different levels of the deployment manifest:

* Deployment: At the `OpenStackControlPlane` level.

* Service: For example the `cinder` component section within the
  `OpenStackControlPlane`.

* Component: For example the cinder backup (`cinderBackup`) in the `cinder`
  section.

Values of the `nodeSelector` are propagated to the next levels unless they are
overwritten. This means that a `nodeSelector` value at the deployment level will
affect all the OpenStack services.

This allows a fine grained control of the placement of the OpenStack services
with minimal repetition.

---

> **🛈 Note:** The Block Storage service does not currently have the possibility
of defining the `nodeSelector` globally for all the cinder volumes (inside the
`cinderVolumes` section), so it is necessary to specify it on each of the
individual back-ends.

---

Labels are used to filter eligible nodes with the `nodeSelector`. Existing node
labels can be used or new labels can be added to selected nodes.

It is possible to leverage labels added by the Node Feature Discovery (NFD)
Operator to place OSP services. For more information, see [Node Feature
Discovery
Operator](https://docs.openshift.com/container-platform/4.16/hardware_enablement/psap-node-feature-discovery-operator.html)
in OpenShift Container Platform Documentation.

### Example \#1: Deployment level

In this scenario we have an OpenShift deployment that has more than three worker
nodes, but we want all the OpenStack services to run on three of these nodes:
`worker-0`, `worker-1`, and `worker-2`.

We’ll use a new label named `type` and we’ll set its value to `openstack` to
mark the nodes that should have the OpenStack services.

First we add this new label to the three selected nodes:

```
$ oc label nodes worker0 type=openstack
$ oc label nodes worker1 type=openstack
$ oc label nodes worker2 type=openstack
```

And then we’ll use this label in our `OpenStackControlPlane` in the top level
`nodeSelector` field:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  nodeSelector:
    type: openstack
```

### Example \#2: Component level

In this scenario we have an OpenShift deployment that has 3 worker nodes, but
only two of those (`worker0` and `worker1`) have an HBA card. We want to
configure the Block Storage service with our FC back-end, so cinder volume must
run in one of the two nodes that have an HBA.

We’ll use a new label named `rhosotype` and we’ll set it to `storage` to mark
the nodes that have access to the Block Storage data network, which requires the
presence of an HBA card.

First we add this new label to the two selected nodes:

```
$ oc label nodes worker0 rhosotype=storage
$ oc label nodes worker1 rhosotype=storage
```

Now we would ensure that the `OpenStackControlPlane` is configured to select
those nodes for the cinder volume back-end level:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderVolumes:
       fibre_channel:
         nodeSelector:
           fc_card: true
```

## 6 Using external data in the services

There are many scenarios in real life deployments where access to external data
is necessary, for example:

* A component needs deployment specific configuration and credential files for
  the storage back-end to exist in a specific location in the filesystem. The
  Ceph cluster configuration and keyring files is one such case, and those files
  are needed in the Block Storage service (cinder volume and cinder backup),
  Image service, and Compute nodes. So it’s necessary for a system administrator
  to provide arbitrary files to the pods.

* It’s very difficult to estimate the disk space needs of the nodes since we
  can’t anticipate the number of parallel operations or size needed, so a node
  can run out of disk space because of the local image conversion that happens
  on the “create volume from image” operation. For those scenarios being able to
  access an external NFS share and use it for the temporary image location is an
  appealing alternative.

* Some back-end drivers expect to run on a persistent filesystem, as they need
  to preserve data they store during runtime between reboots.

To resolve these and other scenarios OpenStack on OpenShift has the concept of
`extraMounts` that has the ability of using all the [types of volumes available
in OpenShift](https://kubernetes.io/docs/concepts/storage/volumes/):
`ConfigMaps`, `Secrets`, `NFS` shares, etc.

These `extraMounts` can be defined at different levels: Deployment, Service, and
Component. For the higher levels we can define where these should be applied by
using the propagation functionality, otherwise they will propagate to all
available resources below; so at the deployment level they would go to all the
services (cinder, glance, manila, neutron, horizon…), at the service level they
would go to all the individual components (e.g.: cinder api, cinder scheduler,
cinder volume, cinder backup), and at the component level it can only go to the
component itself.

This the the general structure of the `extraMounts` which contains a list of
“mounts”:

```
extraMounts:
  - name: <extramount-name> [1]
    region: <openstack-region> [2]
    extraVol:
      - propagation: [3]
        - <location>
        extraVolType: <Ceph | Undefined> [4]
        volumes: [5]
        - <pod-volume-structure>
        mounts: [6]
        - <pod-mount-structure>
```

- \[1\]: Optional name of the extra mount, not of much significance since it’s not
referenced from other parts of the manifest.

- \[2\]: Optional name of the OpenStack region where this `extraMount` should be
applied.

- \[3\]: Location to propagate this specific `extraMount`.

  There are multiple names that can be used each with different granularity:

  * Service names: `Glance`, `Cinder`, `Manila`
  * Component names: `CinderAPI`, `CinderScheduler`, `CinderVolume`,
    `CinderBackup`, `GlanceAPI`, `ManilaAPI`, `ManilaScheduler`,  `ManilaShare`
  * Back-end names: For cinder it would be any name in the `cinderVolumes` map.
    In the examples we’ve shown before it could have been `fibre_channel` or
    `iscsi` or `nfs`.
  * Step name: `DBSync`

  **⚠ Attention:** Defining a broad propagation at a high level can result in
  the mount happening in many different components. For example using `DBSync`
  propagation at the deployment level will propagate to the cinder, glance, and
  manila DB sync pods.

- \[4\] Optional field declaring the type of extra volume. It’s possible values
are `Ceph` and `Undefined`, so in reality it only makes sense to set it when
used to propagate Ceph configuration files.

- \[5\]: List of volume sources, where volumes refers to the broader term of a
“volume” as understood by OpenShift.

  This has the exact same structure as the `volumes` section in a `Pod` and is
  dependent on the type of volume that is being defined, as it is not the same
  to reference an NFS location as it is to reference a secret.

  The name used here is very important, as it will be used as a reference in the
  `mounts` section.

  An example of the excerpt of a `volumes` element that defines that the volume
  is a secret called `ceph-conf-files`:

  ```
       - name: ceph
         secret:
           name: ceph-conf-files
  ```

- \[6\]: List of instructions on where the different volumes should be mounted
within the Pods. Here the name of a volume from the `volumes` section is used as
a reference as well as the path where it should be mounted.

  This has the exact same structure as the `volumeMounts` in a `Pod`.

  An example of how we would use the `cephf-conf-files` secret that was named as
  `ceph` in the earlier `volumes` example:

  ```
       - name: ceph
         mountPath: "/etc/ceph"
         readOnly: true
  ```

Now that we know the `extraMounts` structure and possible values, let’s have a
look at a couple of examples to tie this all together:

### Example \#1: At the deployment level

A very good example is the propagation of the Ceph configuration that is stored
in a secret called `ceph-conf-files` to the different storage services (glance,
manila, and cinder components in the data path):

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
spec:
  extraMounts:
    - name: v1
      region: r1
      extraVol:
        - propagation:
          - CinderVolume
          - CinderBackup
          - GlanceAPI
          - ManilaShare
          extraVolType: Ceph
          volumes:
          - name: ceph
            secret:
              name: ceph-conf-files
          mounts:
          - name: ceph
            mountPath: "/etc/ceph"
            readOnly: true
```

### Example \#2: At the component level

Let’s add a custom policy to the cinder API component. We have previously
created this policy as a `configMap` with the name `my-cinder-policy` and the
contents are stored in the `policy.yaml` key:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
spec:
  cinder:
    extraMounts:
    - extraVol:
      - volumes:
        - name: policy
          configMap:
            name: my-cinder-policy
        mounts:
        - mountPath: /etc/cinder/policy.yaml
          subPath: policy.yaml
          name: policy
          readOnly: true
        propagation:
        - CinderAPI
```

## 7 Restricting resources used by a service

Many storage services allow you to specify for each component how much of each
resource its container needs. The most common resources to specify are CPU and
memory (RAM).

Requested resources information is used by the OpenShift scheduler to decide
which node to place the Pod on, while the *limit* is enforced so that the
container is not allowed to go beyond the set limit.

A system administrator can do this using the `resources` field which has both
the `requests` and `limits` as [defined in the official
documentation](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
for the same `requests`.

As an example (not necessarily a good one) where we are setting limits for the
memory used by the cinder scheduler so that if it goes beyond that the container
gets killed:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderScheduler:
        resources:
          limits:
            memory: "500Mi"
```

## 8 Setting API timeouts

All storage services have a REST API interface to interact with users and other
OpenStack components, and when a connection is opened but no data is transmitted
after a period of time the connection is considered stale, a connection timeout
occurs and the connection is closed.

There are cases where the default API timeout is insufficient and the timeout
needs to be increased for proper service operation.

The `cinder`, `glance`, and `manila` components have a field called `apiTimeout`
that accepts an integer with the implicit unit being seconds and a default value
of `60` seconds.

This value is used to set all the appropriate configuration options in the
different areas to make it effective. In other words, by setting this single
value the installer will set the HAProxy timeout, the Apache timeout, and even
internal RPC timeouts accordingly.

These three timeouts can still be individually configured using the `apiTimeout`
and `apiOverride` sections of the manifest and the `rpc_response_timeout`
configuration option in the snippet.

Here’s an example of setting the HAProxy to 121 seconds, the Apache timeout to
120 seconds, and the RPC timeout to 119 seconds for Cinder.

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    apiOverride:
      route:
        public:
          "haproxy.router.openshift.io/timeout": 121
      apiTimeout: 120
      customServiceConfig: |
        [DEFAULT]
        rpc_response_timeout = 119
```

---

> **🛈 Note:** The reason why it is possible to individually configure each
timeout (HAProxy, Apache, and RPC) is that they may return different errors to
REST API clients, so admins may prefer one to the other.

---

## 9 Storage networking

Storage best practices recommend using two different networks: one for the data
I/O and another for storage management.

There is no technical impediment, as in it will work, to have a single storage
network or even have no specific storage network and have it all mixed with the
rest of your network traffic, but we recommend deploying OpenStack in OpenShift
with a network architecture that follows as closely as possible the best
practices.

In this and other guides and their examples these networks will be referred to
as `storage` and `storageMgmt`. If your deployment diverges from the two
networks reference architecture, please adapt the examples as necessary. For
example if the storage system’s management interface is available on the
`storage` network, replace any mention of `storageMgmt` with `storage` when
there’s only one network and remove it altogether when the `storage` network is
already present.

Most storage services, with the possible exception of the Object Storage
service, require access to the `storage` and `storageMgmt` networks, and they
are configured in the `networkAttachments` field that accepts a list of strings
with all the networks the component should have access to for proper operation.
Different components can have different network requirements, for example Cinder
API component doesn’t need access to any of the storage networks.

This is an example for the cinder volume:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderVolumes:
        iscsi:
          networkAttachments:
          - storage
          - storageMgmt
```

## 10 Using other container images

OpenStack services are deployed using their respective container images for the
specific OpenStack release and version. There are times when a deployment may
require using different container images. The most common cases of using a non
default container image are: deploying a hotfix and using a certified vendor
provided container image.

The container images used by the installer for the OpenStack services are
controlled via the `OpenStackVersion` CR. An `OpenStackVersion` CR is
automatically created by the openstack operator during the deployment of the
OpenStack services, or we can create it manually before we apply the
`OpenStackControlPlane` but after after the openstack operator has been
installed.

Using the `OpenStackVersion` we can change the container image for any service
and component individually. The granularity of what can have different images
depends on the service, for example for the Block Storage service (cinder) all
the cinder API pods will have the same image, the same is true for the scheduler
and backup components, but for the volume service the container image is defined
for each of the `cinderVolumes`.

For example, let’s assume we have the following cinder volume configuration with
two volume back-ends, Ceph and one called custom-fc that requires a certified
vendor provided container image, and we also want to change the other component
images. And excerpt of the `OpenStackControlPlan` could look like:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderVolumes:
        ceph:
          networkAttachments:
          - storage
< . . . >
        custom-fc:
          networkAttachments:
          - storage
```

---

> **⚠ Attention:** The name of the `OpenStackVersion` must match the name of
your `OpenStackControlPlane`, so in your case it may be other than openstack.

---

Then the `OpenStackVersion` that would change the container images would look
something like this::

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackVersion
metadata:
  name: openstack
spec:
  customContainerImages:
    cinderAPIImages: <hotfix-api-image>
    cinderBackupImages: <hotfix-backup-image>
    cinderSchedulerImages: <hotfix-scheduler-image>
    cinderVolumeImages:
      custom-fc: <vendor-volume-volume-image>
```

In this scenario only the Ceph volume back-end pod would use the default cinder
volume image.
