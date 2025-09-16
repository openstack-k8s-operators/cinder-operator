# Cinder configuration guide

The OpenStack Block Storage service (cinder) allows users to access block
storage devices through *volumes* to provide persistent storage in the Compute
instances and can also be used by the Image Service (glance) as a back-end to
store its images.

This guide focuses on general concepts covered before but from the cinder
perspective as well as the configuration of the different cinder components as
day-0 and day-1 operations; that is, creating the manifest content with the
cinder configuration and configuring it once the API component is running.

The Block Storage service (cinder) has three mandatory services (api, scheduler,
and volume), and one optional service (backup).

All the cinder services are configured using the `cinder` section of the
`OpenStackControlPlane` manifest:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
```

Within this `cinder` section there are a couple of options for the
cinder-operator that have effects on all the services -such as `enabled` and
`uniquePodNames`-, but most of the global configuration options are in the
`template` section within this `cinder` section, which includes sections -such
as `nodeSelector`, `preserveJobs`, `dbPurge`, etc.

Service specific sections all hang right below the `template` section. We have
the `cinderAPI` section for the API, `cinderScheduler` for the scheduler,
`cinderBackup` for backups, and `cinderVolumes` for the volume backends.

A rough view of this structure would be:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    <global-options>
    template:
      <global-options>
      cinderAPI:
        <cinder-api-options>
      cinderScheduler:
        <cinder-scheduler-options>
      cinderVolumes:
        <name1>: <cinder-volume-options>
        <name2>: <cinder-volume-options>
      cinderBackup:
        <cinder-backup-options>
```

Table of contents:
- [1. Terminology](#1-terminology)
- [2. Changes from older releases](#2-changes-from-older-releases)
- [3. Prepare transport protocol requirements](#3-prepare-transport-protocol-requirements)
- [4. Setting initial defaults](#4-setting-initial-defaults)
- [5. Configuring the API service](#5-configuring-the-api-service)
- [6. Configuring the scheduler service](#6-configuring-the-scheduler-service)
- [7. Configuring the volume service](#7-configuring-the-volume-service)
- [8. Configuring the backup service](#8-configuring-the-backup-service)
- [9. Automatic database cleanup](#9-automatic-database-cleanup)
- [10. Preserving jobs](#10-preserving-jobs)
- [11. Resolving hostname conflicts](#11-resolving-hostname-conflicts)


## 1. Terminology

There are some terminology that is important to clarify when discussing cinder:

* Storage back-end: Refers to a physical Storage System where the data for the
  *volumes* is actually stored.
* Cinder driver: Is the code in cinder that knows how to talk to a Storage
  back-end. Configured using the `volume_driver` option.
* Cinder back-end: Refers to a logical representation of the grouping of a
  cinder driver with its configuration. This grouping is used to manage and
  address the volumes present in a specific Storage back-end. The name of this
  logical construct is defined using the `volume_backend_name` option.
* Storage pool: Logical grouping of volumes in a given Storage back-end.
* Cinder pool: The representation in Cinder of a Storage pool.
* Volume host: This usually refers to the way cinder uses to address volumes.
  There are two different representations, short `<hostname>@<backend-name>` and
  full `<hostname>@<backend-name>#<pool-name>`.

## 2. Changes from older releases

If you are familiar with previous Red Hat OpenStack Platform releases that were
deployed using TripleO, you will notice enhancements and improvements as well as
some differences such as:

* Multiple volume back-ends:
  * Very easy to deploy.
  * Adding back-ends is fast and doesn’t affect other running back-ends.
  * Removing back-ends is fast and doesn’t affect other running back-ends.
  * Making configuration changes to one back-end doesn’t affect other back-ends.
  * Each back-end can use its own vendor specific container image. No more
    building custom images to hold dependencies from two drivers.
* Recommended deployment avoids inconsistent scheduling issues.
* No more pacemaker. OpenShift functionality has been leveraged to not need it.
* Easier way to debug service code when faced with difficult to resolve issues.


## 3. Prepare transport protocol requirements

The OpenStack control plane services that use volumes, such as cinder volume and
cinder backup, require the OpenShift cluster’s support for them, as it needs to
run some daemons and kernel modules, such as `iscsid` and `multipathd`.

These daemon and kernel modules must be available in all the OpenShift nodes
where the cinder volume and backup services could run, not only on the ones
where they are currently running. More information on where services run is
available in section 1.5 Restricting where services run.

To ensure those daemons and kernel modules are running on the OCP hosts some
changes may be required on the OpenShift side using a `MachineConfig`. For
additional information on `MachineConfig` please refer to the [OpenShift
documentation](https://docs.openshift.com/container-platform/4.12/post_installation_configuration/machine-configuration-tasks.html).

It is recommended to make these changes during the installation of the OpenShift
cluster or before deploying any of the OpenStack control plane services with the
`OpenStackControlPlane` manifest.

There are complete examples of backend and transport protocols available, as
described in the [back-end examples section](#76-back-end-examples).

---

> **🛈 Note:** This section’s only purpose is to serve as a guide to the topic
of transport protocol requirements for illustration purposes, but each vendor
may have specific recommendations on how to configure the storage transport
protocol to use against their storage system, so it is recommended to always
check with the vendor.

---

> **⚠ Attention:** The node will reboot any time a `MachineConfig` is used to
make changes to OpenShift nodes\!\! OpenShift and OpenStack administrators may
be different, so please consult your OpenShift administrators before applying
any `MachineConfig` to ensure safety of the OpenShift workloads.

---

> **⚠ Attention:** When using `nodeSelector` described later, remember to also use
*a `MachineConfigPool` and adapt the `MachineConfig` to use it as described in
*the [OpenShift
*documentation](https://docs.openshift.com/container-platform/4.12/post_installation_configuration/machine-configuration-tasks.html).

> **⚠ Attention:** `MachineConfig` examples in the following sections are using
label `machineconfiguration.openshift.io/role: worker` instructing OpenShift to
apply those changes on `worker` nodes (`workers` is an automatically created
`MachineConfigPool`)  This assumes we have an OpenShift cluster with 3 master
nodes and 3 `worker` nodes.  If we are deploying in a 3 node OpenShift cluster
where all nodes are `master` and `worker` we need to change it to use `master`
instead (`master` is another automatically created `MachineConfigPool`).

---

### 3.1. iSCSI

Connecting to iSCSI volumes from the OpenShift nodes requires the iSCSI
initiator to be running, because the Linux Open iSCSI initiator doesn't
currently support network namespaces, so we must only run 1 instance of the
`iscsid` service for the normal OpenShift usage, the OpenShift CSI plugins, and
the OpenStack services.

If we are not already running `iscsid` on the OpenShift nodes then we'll need to
apply a `MachineConfig` to the nodes that could be running cinder volume and
backup services. Here is an example that starts the `iscsid` service with the
default configuration options in **all** the OpenShift worker nodes:

```
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
    service: cinder
  name: 99-worker-cinder-enable-iscsid
spec:
  config:
    ignition:
      version: 3.2.0
    systemd:
      units:
      - enabled: true
        name: iscsid.service
```

For production deployments using iSCSI volumes we encourage setting up
multipathing, please look at the [multipathing section below](#multipathing) to
see how to configure it.

The iSCSI initiator is automatically loaded on provisioned Data Plane nodes
where the Compute service is running, so no additional steps are required for
instances to access the volumes.

### 3.2. FC

Connecting FC volumes from the OpenShift nodes do not require any
`MachineConfig` to work, but production deployments using FC volumes should
always be set up to use multipathing. Please look at the [multipathing section
below](#multipathing) to see how to configure it.

It is of utmost importance that all OpenShift nodes that are going to run cinder
volume and cinder backups have a HBA card.

Node selectors can be used to determine what nodes can run cinder volume and
backup pods if our deployment doesn’t have HBA cards in all the nodes. Please
refer to the [Service placement
section](commonalities#5-restricting-where-services-run) for detailed
information on how to use node selectors.

No additional steps are necessary in Data Plane nodes where the Compute service
is running for instances to have access to FC volumes.

### 3.3. NVMe-oF

Connecting to NVMe-oF volumes from the OpenShift nodes requires that the nvme
kernel modules are loaded on the OpenShift hosts.

If we are not already loading the nvme modules on the OpenShift nodes where
volume and backup services are going to run, then we'll need to apply a
`MachineConfig` similar to this one that applies the change to **all** OpenShift
nodes:

```
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
    service: cinder
  name: 99-master-cinder-load-nvme-fabrics
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
        - path: /etc/modules-load.d/nvme_fabrics.conf
          overwrite: false
          mode: 420
          user:
            name: root
          group:
            name: root
          contents:
            source: data:,nvme-fabrics%0Anvme-tcp
```

We are only loading the `nvme-fabrics` module because it takes care of loading
the transport specific modules (tcp, rdma, fc) as needed.

For production deployments using NVMe-oF volumes we encourage using
multipathing. For NVMe-oF volumes OpenStack uses native multipathing, called
[ANA](https://nvmexpress.org/faq-items/what-is-ana-nvme-multipathing/).

Once the OpenShift nodes have rebooted and are loading the `nvme-fabrics` module
we can confirm that the Operating System is configured and supports ANA by
checking on the host:

```
cat /sys/module/nvme_core/parameters/multipath
```

The nvme kernel modules are automatically loaded on provisioned Data Plane nodes
where the Compute service is running, so no additional steps are required for
instances to access the volumes.

---

> **⚠ Attention:** Even though ANA doesn't use the Linux Multipathing Device
Mapper the current OpenStack code requires `multipathd` on compute nodes to be
running for the Compute nodes to be able to use multipathing when connecting
volumes to instances, so please remember to follow the [multipathing
section](#multipathing) to be able to use it in the control plane nodes.

#### OCP BUG #32629

In some deployments the OpenShift image has the NVMe `hostid` and `hostnqn`
hardcoded, so it ends up being the same in all the OpenShift nodes used for the
OpenStack control plane, which is problematic for Cinder when connecting
volumes.

Bugs:
- [Red Hat bug #34629](https://issues.redhat.com/browse/OCPBUGS-34629)
- [Upstream issue](https://github.com/openshift/os/issues/1519)
- [PR](https://github.com/openshift/os/pull/1520)

To resolve this issue we can use the following `MachineConfig` that fixes the
issue by recreating both files when the `hostid` doesn't match the system-uuid
of the machine it is running on.

```
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    component: fix-nvme-ids
    machineconfiguration.openshift.io/role: worker
    service: cinder
  name: 99-worker-cinder-fix-nvme-ids
spec:
  config:
    Systemd:
      Units:
      - name: cinder-nvme-fix.service
        enabled: true
        Contents: |
          [Unit]
          Description=Cinder fix nvme ids

          [Service]
          Type=oneshot
          RemainAfterExit=yes
          Restart=on-failure
          RestartSec=5
          ExecStart=bash -c "if ! grep $(/usr/sbin/dmidecode -s system-uuid) /etc/nvme/hostid; then /usr/sbin/dmidecode -s system-uuid > /etc/nvme/hostid; /usr/sbin/nvme gen-hostnqn > /etc/nvme/hostnqn; fi"

          [Install]
          WantedBy=multi-user.target
    ignition:
      version: 3.2.0
```

---

### 3.4. Multipathing

Cinder back-ends using iSCSI and FC transport protocols (and NVMe-oF due to an
OpenStack limitation) in production should always be configured using
multipathing to provide additional resilience and optionally additional
throughput.

Setting up multipathing on OpenShift nodes requires a `MachineConfig` that
creates the `multipath.conf` file and starts the service.

A basic `multipath.conf` file would be:

```
defaults {
  user_friendly_names no
  recheck_wwid yes
  skip_kpartx yes
  find_multipaths yes
}

blacklist {
}
```

Here is an example of how that same configuration file could be written in
**all** OpenShift worker nodes and then make the `multipathd` service start on
*boot:

```
apiVersion: machineconfiguration.openshift.io/v1
kind: MachineConfig
metadata:
  labels:
    machineconfiguration.openshift.io/role: worker
    service: cinder
  name: 99-master-cinder-enable-multipathd
spec:
  config:
    ignition:
      version: 3.2.0
    storage:
      files:
        - path: /etc/multipath.conf
          overwrite: false
          mode: 384
          user:
            name: root
          group:
            name: root
          contents:
            source: data:,defaults%20%7B%0A%20%20user_friendly_names%20no%0A%20%20recheck_wwid%20yes%0A%20%20skip_kpartx%20yes%0A%20%20find_multipaths%20yes%0A%7D%0A%0Ablacklist%20%7B%0A%7D
    systemd:
      units:
      - enabled: true
        name: multipathd.service
```

Configuring the cinder services to use multipathing is usually done using the
`use_multipath_for_image_xfer` configuration option in all the backend sections
(and in the `[DEFAULT]` section for the backup service), but in OpenStack on
OpenShift deployments there’s no need to worry about it, because that's the
default. So as long as we don't override it by setting
`use_multipath_for_image_xfer = false`.

The multipathing daemon is automatically loaded on provisioned Data Plane nodes.

## 4. Setting initial defaults

There are some configuration options in the service that are only used once
during the database creation process, so they should be configured the first
time we enable the cinder service, and they have to be defined at the top
`customServiceConfig` or they won’t have any effect and defaults will be used.

The configuration options that are used during the database creations are the following:

* `quota_volumes`: Number of volumes allowed per project.
  Integer value.
  Defaults to `10`
* `quota_snapshots`: Number of volume snapshots allowed per project.
  Integer value.
  Defaults to `10`
* `quota_consistencygroups`: Number of consistency groups allowed per project.
  Integer value.
  Defaults to `10`
* `quota_groups`: Number of groups allowed per project.
  Integer value.
  Defaults to `10`
* `quota_gigabytes` : Total amount of storage, in gigabytes, allowed for volumes and snapshots per project.
  Integer value.
  Defaults to `1000`
* `no_snapshot_gb_quota`:  Whether snapshots count against gigabyte quota.
  Bboolean value.
  Defaults to `false`
* `quota_backups`: Number of volume backups allowed per project.
  Integer value.
  Defaults to `10`
* `quota_backup_gigabytes`: Total amount of storage, in gigabytes, allowed for backups per project.
  Integer value.
  Defaults to `1000`
* `per_volume_size_limit`: Max size allowed per volume, in gigabytes.
  Integer value.
  Defaults to `-1` (no limit).
* `default_volume_type`: Default volume type to use. More details on the CUSTOMIZING PERSISTENT STORAGE guide.
  String value.
  Defaults to `__DEFAULT__` (automatically created on installation).

Changes are still possible once the database has been created, but the
`openstack` client needs to be used to modify any of these values, and changing
these configuration options in a snippet will have no impact on the deployment.
Please refer to setting the quota on the customizing persistent storage.

Here’s an example with some of these options set in the manifest:

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
        quota_volumes = 20
        quota_snapshots = 15
```

## 5. Configuring the API service

The cinder API offers a REST API interface for all external interaction with the
service for both users and other OpenStack services.

The API component is relatively simple to configure, because it doesn’t need
additional networks, doesn’t need sensitive information to function, can run on
any OCP node, and doesn’t need any configuration snippet to operate.

From a configuration point of view the only thing that needs to be configured is
the internal OpenShift service’s load balancer as previously described in
Creating the control plane from the Deploying OpenStack Services on OpenShift.

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderAPI:
        override:
          service:
            internal:
              metadata:
                annotations:
                  metallb.universe.tf/address-pool: internalapi
                  metallb.universe.tf/allow-shared-ip: internalapi
                  metallb.universe.tf/loadBalancerIPs: 172.17.0.80
              spec:
                type: LoadBalancer
```

The default REST API inactivity timeout is 60 seconds, changing this value is
documented in [Setting API timeouts](commonalities.md#8-setting-api-timeouts).

### 5.1. Setting the number of replicas

For the cinder API the default number of replicas is `1`, but the recommendation
for production is to run multiple instances simultaneously in Active-Active
mode.

This can be easily achieved by setting `replicas: 3` in the `cinderAPI` section
of the configuration:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderAPI:
        replicas: 3
```

### 5.2. Setting cinder API options

Like all the cinder components, cinder API inherits the configuration snippet
from the top `customServiceConfig` section defined under `cinder`, but it also
has its own `customServiceConfig` section under the `cinderAPI` section.

There are multiple configuration options that can be set in the snippets under
the `[DEFAULT]` group, but the most relevant ones are:

* `debug`: When set to `true` the logging level will be set to `DEBUG` instead
  of the default `INFO` level. Debug log levels for the API can also be
  dynamically set without restart using the dynamic log level API functionality.
  Boolean value.
  Defaults to `false`
* `api_rate_limit`: Enables or disables rate limit of the API.
  Boolean value.
  Defaults to `true`
* `osapi_volume_workers`: Number of workers for the cinder API Component.
  Integer value.
  Defaults to the number of CPUs available.
* `osapi_max_limit`: Maximum number of items that a collection resource returns
  in a single response.
  Integer value.
  Defaults to 1,000.

As an example, here’s how we would enable debug logs globally for all the cinder
components, API included, and set the number of API workers to `3` specifically
for the API component.

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
        debug = true
      cinderAPI:
        customServiceConfig: |
          [DEFAULT]
          osapi_volume_workers = 3
```

### 5.3. Creating custom access policies

Cinder, like other OpenStack services, has its own sensible default access
policies for its API; this is necessary to restrict the operations users can
perform.

To override the default values of these policies a YAML formatted file can be
provided to the API component. This file only needs to have the specific
policies that want to be changed from their default values, there’s no need to
provide all the policies in the file for it to be valid.

The default location where cinder API will look for the policy file is
`/etc/cinder/policy.yaml` if a different location is used then the `policy_file`
option of the `oslo_policy` section of the cinder configuration must be set to
match.

A complete list of available policies in cinder as well as their default values
can be found in [the project
documentation](https://docs.openstack.org/cinder/2023.1/configuration/block-storage/policy.html).

Here’s an example of how to change the policy to allow any user to force delete
snapshots. For illustration purposes we’ll use a non-default location for the
policy file.

First let’s see what the `ConfigMap` with the custom cinder policy would look
like:

```
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-cinder-conf
  namespace: openstack
data:
  policy: |
    "volume_extension:snapshot_admin_actions:force_delete": "rule:xena_system_admin_or_project_member"
```

And now leverage the `extraMounts` feature using a non standard location for the
file.

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
spec:
  cinder:
    template:
      cinderAPI:
        customServiceConfig: |
          [oslo_policy]
          policy_file=/etc/cinder/api/policy.yaml
    extraMounts:
    - extraVol:
      - volumes:
        - name: policy
          projected:
            sources:
            - configMap:
                name: my-cinder-policy
                items:
                - key: policy
                  path: policy.yaml
        mounts:
        - mountPath: /etc/cinder/api
          name: policy
          readOnly: true
      propagation:
      - CinderAPI
```

For additional information on `extraMounts`, including an example of how to
mount the policy in the default location, please refer to section [Using
external data in the
services](commonalities.md#6-using-external-data-in-the-services).

## 6. Configuring the scheduler service

The cinder Scheduler is responsible for making decisions such as  selecting
where (as in what cinder back-end) new volumes should be created, whether
there’s enough free space to perform an operation (e.g. creating a snapshot) or
not, or deciding where an existing volume should be moved to on some specific
operations.

The Scheduler component is the simplest component to configure, because it
doesn’t need any changes to its defaults to run.

### 6.1. Setting the number of replicas

The cinder Scheduler component can run multiple instances in Active-Active mode,
but since it follows an eventual consistency model, running multiple instances
makes it considerably harder to understand its behavior when doing operations.

The recommendation is to use a single instance for the Scheduler unless your
specific deployment needs find that this becomes a bottleneck. Increasing the
number of instances can be done at any time without disrupting the existing
instances.

### 6.2. Setting service down detection timeout

On scheduling operations only services that are up and running will be taken
into account, and the rest will be ignored.

To detect services that are up a database heartbeat is used, and then the time
of this heartbeat is used to see if it has been too long since the service did
the heartbeat.

The options used to configure the service down detection timeout are:

* `report_interval`: Interval, in seconds, between components (scheduler,
  volume, and backup) reporting `up` state in the form of a heartbeat through
  the database.
  Integer value.
  Defaults to `10`
* `service_down_time`: Maximum time since last check-in (heartbeat) for a
  component to be considered up.
  Integer value.
  Defaults to `60`

It is recommended to define these options at the top cinder
`customServiceConfig` so all services have a consistent heartbeat interval and
because the service down time is used by multiple services, not only the
scheduler.

Here’s an example of doubling the default reporting intervals:

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
        report_interval = 20
        service_down_time = 120
```

### 6.3. Setting the stats reporting interval

Many scheduling operations require that the cinder schedulers know details about
the storage back-end, so cinder volumes and backups periodically report stats to
the schedulers that include, but are not limited to, used space, available
space, and capabilities.

Having frequent stats updates is most important when a storage back-end is close
to its full capacity, as we may get failures on operations due to lack of space
that theoretically would fit.

The configuration option for volumes is called `backend_stats_polling_interval`
and dictates the time in seconds between the cinder volume requests for usage
statistics from the Storage back-end. The default value is `60`. This option
must be set in the `[DEFAULT]` section.

The equivalent configuration option for backups is called
`backup_driver_stats_polling_interval` and has the same default of `60`.

---

> **⚠ Attention:** Generating these usage statistics is expensive for some
backends, so setting this value too low may adversely affect performance.

---

Because these stats are generated by the cinder volume and cinder backup, the
configuration option must be present in their respective snippet and not the
scheduler. This can be achieved by setting it globally in the top
`customServiceConfig` or  individually on each of the `cinderVolumes` or in the
`cinderBackup` depending on the granularity we want.

Example of setting it globally to double the default value for backups and
volumes:

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
        backend_stats_polling_interval = 120
        backup_driver_stats_polling_interval = 120
```

Example of setting it on a per back-end basis:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderBackup:
        customServiceConfig: |
          [DEFAULT]
          backup_driver_stats_polling_interval = 120
          < rest of the config >
      cinderVolumes:
        nfs:
          customServiceConfig: |
            [DEFAULT]
            backend_stats_polling_interval = 120
            < rest of the config >
```

### 6.4. Setting other cinder Scheduler options

Like all the cinder components, cinder Scheduler inherits the configuration
snippet from the top `customServiceConfig` section defined under `cinder`, but
it also has its own `customServiceConfig` section under the `cinderScheduler`
section.

There are multiple configuration options that can be set in the snippets under
the `[DEFAULT]` group, but the most relevant ones are:

* `debug`: When set to `true` the logging level will be set to `DEBUG` instead
  of the default `INFO` level.  Debug log levels for the Scheduler can also be
  dynamically set without restart using the dynamic log level API functionality.
  Boolean value.
  Defaults to `false`
* `scheduler_max_attempts`: Maximum number of attempts to schedule a volume.
  Integer value.
  Defaults to `3`
* `scheduler_default_filters`: Filter class names to use for filtering hosts
  when not specified in the request. List of available filters can be found in
  the [upstream project
  documentation](https://docs.openstack.org/cinder/2023.1/configuration/block-storage/scheduler-filters.html).
  Comma separated list value.
  Defaults to `AvailabilityZoneFilter,CapacityFilter,CapabilitiesFilter`
* `scheduler_default_weighers`: Weigher class names to use for weighing hosts.
  List of available weighers can be found in the [upstream project
  documentation](https://docs.openstack.org/cinder/2023.1/configuration/block-storage/scheduler-weights.html).
  Comma separated list value.
  Defaults to `CapacityWeigher`.
* `scheduler_weight_handler`: Handler to use for selecting the host/pool after
  weighing. Available values are:
  `cinder.scheduler.weights.OrderedHostWeightHandler` that selects the first
  host from the list of hosts that passed the filters, and
  `cinder.scheduler.weights.stochastic.stochasticHostWeightHandler`  which gives
  every pool a chance to be chosen where the probability is proportional to each
  pools’ weight.
  String value.
  Defaults to `cinder.scheduler.weights.OrderedHostWeightHandler`.

As an example, here’s how we would enable debug logs globally for all the cinder
components, Scheduler included, and set the number of scheduling attempts to 2
specifically in the Scheduler component.

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
        debug = true
      cinderScheduler:
        customServiceConfig: |
          [DEFAULT]
          scheduler_max_attempts = 3
```

## 7. Configuring the volume service

Cinder Volume is responsible for managing operations related to Volumes,
Snapshots, and Groups (previously known as Consistency Groups\`); such as
creating, deleting, cloning, snapshotting, etc.

The component requires access to the Storage back-end I/O network (`storage`) as
well as its management network (`storageMgmt`) in the `networkAttachments`. Some
operations, such as creating an empty volume or a snapshot, do not require any
data I/O between the cinder Volume pod and the Storage back-end, but there are
other operations, such as migrating data from one storage back-end to a
different one, that requires the data to pass through the Volume pod.

Configuring volume services is done under the `cinderVolumes` section, and most
of the time requires using `customServiceConfig`, `customServiceConfigSecrets`,
`networkAttachments`, `replicas`, although in some cases even the
`nodeSelector`.

Make sure you’ve reviewed section 1.4. Setting service configuration options to
understand what configuration options should go in the `customServiceConfig` and
which ones in the `customServiceConfigSecrets`.

### 7.1. Back-ends

In OpenStack on OpenShift each cinder back-end should have its own entry in the
`cinderVolumes` section, that way each cinder back-end will run in its own
pod. This not only removes a good number of limitations but also brings in a lot
of benefits:

* Increased isolation
* Adding back-ends is fast and doesn’t affect other running back-ends.
* Removing back-ends is fast and doesn’t affect other running back-ends.
* Making configuration changes to a back-end doesn’t affect other back-ends.
* Automatically spreads the Volume pods into different nodes.

Each cinder back-end uses a storage transport protocol to access the data in the
volumes, and each of these protocols have their own requirements, as described
in section [prepare transport protocol
requirements](#3-prepare-transport-protocol-requirements), but this is usually
also documented by the vendor.

---

> **🛈 Note:** In older OpenStack releases the only deployment option was to run
all the back-ends together in the same container, but this is no longer
recommended for OpenStack on OpenShift, and using an independent pod for each
configured back-end.

---

> **⚠ Attention:** In OpenStack on OpenShift no back-end is deployed by
default, so there will be no cinder Volume services running unless a back-end is
manually configured in the deployment.

---

### 7.2. Setting the number of replicas

The cinder volume component cannot currently run multiple instances in
Active-Active mode; all deployments must use `replicas: 1`, which is the default
value, so there’s no need to explicitly specify it.

Cinder volume leverages the OpenShift functionality to always maintain one pod
running to serve volumes.

### 7.3. Setting cinder Volume options

A volume back-end configuration snippet must have its own configuration group
and cannot be configured in the `[DEFAULT]` group.

As mentioned before, each back-end has its own configuration options that are
documented by the storage vendor, but there are some common options for all
cinder back-ends.

* `backend_availability_zone`: Availability zone of this cinder back-end. Can be
  set in the `[DEFAULT]` section using the `storage_availability_zone` option.
  String value.
  Defaults to the value of `storage_availability_zone` which in turn defaults to `nova`.
* `volume_backend_name`: Cinder back-end name for a given driver implementation.
  String value.
  No default value.
* `volume_driver`: Driver to use for volume creation in the form of python
  namespace for the specific class.
  String value.
* `enabled_backends`: A list of backend names to use. These backend names should
  be backed by a unique `[CONFIG]` group with its options.
  Comma separated list of string values.
  Defaults to the name of the section with a `volume_backend_name` option.
* `image_conversion_dir`: Directory used for temporary storage during image
  conversion. This option is useful when replacing image conversion location
  with a remote NFS location.
  String value.
  Defaults to `/var/lib/cinder/conversion`
* `backend_stats_polling_interval:` Time in seconds between the cinder volume
  requests for usage statistics from the Storage back-end. Be aware that
  generating usage statistics is expensive for some backends, so setting this
  value too low may adversely affect performance.
  Integer value.
  Defaults to `60`

In OpenStack on OpenShift there is no need to configure `enabled_backends` when
running a single pod per back-end as it will be automatically configured for us.
So this snippet in the `customServiceConfig` section:

```
[iscsi]
volume_backend_name = myiscsi
volume_driver = cinder.volume.drivers...
```

Is equivalent to its more verbose counterpart:

```
[DEFAULT]
enabled_backends = iscsi
[iscsi]
volume_backend_name = myiscsi
volume_driver = cinder.volume.drivers...
```

---

> **⚠ Attention:** You may be aware of the `host` and `backend_host`
configuration options. We recommend not using them unless strictly necessary,
such as when adopting an existing deployment.

---

As an example, here’s how we would enable debug logs globally for all the cinder
components, Volume included, and set the backend name and volume driver for a
Ceph back-end:

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
        debug = true
      cinderVolumes:
        ceph:
          customServiceConfig: |
            [DEFAULT]
            volume_backend_name = ceph
            volume_driver = cinder.volume.drivers.rbd.RBDDriver
```

### 7.4. Configuring multiple back-ends

You can deploy multiple back-ends for the Block Storage service (cinder), each
will use its own pod, so they are independent regarding updating, changing
configuration, node placement, container image to be used, etc.

Configuring multiple back-ends is as easy as adding another entry to the
`cinderVolumes` section.

For example we could have two independent back-ends, one for iSCSI and another
for NFS::

```
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      cinderVolumes:
        nfs:
          networkAttachments:
          - storage
          - storageMgmt
          customServiceConfigSecrets:
          - cinder-volume-nfs-secrets
          customServiceConfig: |
            [nfs]
            volume_backend_name=nfs
        iSCSI:
          networkAttachments:
          - storage
          - storageMgmt
          customServiceConfig: |
            [iscsi]
            volume_backend_name=iscsi
```

### 7.5. Configuring an NFS back-end

The Block Storage service (cinder) can be configured with a generic NFS back-end
to provide an alternative storage solution for volumes as well as backups.

As with most back-ends the snippet with sensitive information would be stored in
a secret (that needs to be created in OpenShift on its own with `oc create` or
`oc apply`) and referenced in the `customServiceConfigSecrets` and the rest of
the configuration in the `customServiceConfig`.

The single pod per back-end should still be the norm for NFS shares, so to
configure multiple shares a separate back-end should be added with its own
`Secret` referenced in `customServiceConfigSecrets`.

---

> **⚠ Attention:** Use a certified third-party NFS driver when using OpenStack
Services on OpenShift in a production environment. The generic NFS driver is not
recommended for a production environment.

---

> **🛈 Note:** *This section discussed using NFS as a cinder volume back-end,
not using an external NFS share for image conversion. For that purpose the
`extraMounts` feature should be used, as described in section [using external
data](commonalities.md#6-using-external-data-in-the-services).

---

#### Supported NFS storage

* For production deployments we recommends that you use a vendor specific storage back-end and driver. We
don't recommend using NFS storage that comes from the generic NFS back end for
production, because its capabilities are limited compared to a vendor NFS
system.  For example, the generic NFS back-end does not support features such as
volume encryption and volume multi-attach.

* For Block Storage (cinder) and Compute (nova) services, you must use NFS
version 4.0 or later. OpenStack on OpenShift does not support earlier versions
of NFS.

#### Unsupported NFS configuration

* OpenStack on OpenShift does not support the NetApp NAS secure feature, because
it interferes with normal volume operations, so these must be disabled in the
`customServiceConfig` in the specific backend configuration, as seen in the
later example, by setting:

  * `nas_secure_file_operation=false`
  * `nas_secure_file_permissions=false`

* Do not configure the `nfs_mount_options` option as the default value is the
most suitable NFS options for OpenStack on OpenShift environments. If you
experience issues when you configure multiple services to share the same NFS
server, contact Red Hat Support.

#### Limitations when using NFS shares

* Instances that have a swap disk cannot be resized or rebuilt when the back-end
is an NFS share.

---

> **Attention:** Use a vendor specific NFS driver when using OpenStack on
OpenShift in a production environment. The generic NFS driver is not recommended
for a production environment.

---

#### NFS Sample configuration

In this example we are just going to configure cinder volume to use the same NFS
system.

Volume secret:

```
apiVersion: v1
kind: Secret
metadata:
  name: cinder-volume-nfs-secrets [1]
type: Opaque
stringData:
  cinder-volume-nfs-secrets: |
	[nfs]
	nas_host=192.168.130.1
	nas_share_path=/var/nfs/cinder
```

The `OpenStackControlPlane` configuration:

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
          networkAttachments: [2]
          - storage
          customServiceConfig: |
            [nfs]
            volume_backend_name=nfs
            volume_driver=cinder.volume.drivers.nfs.NfsDriver
            nfs_snapshot_support=true
            nas_secure_file_operations=false
            nas_secure_file_permissions=false
          customServiceConfigSecrets:
          - cinder-volume-nfs-secrets [1]
```

\[1\]: The name in the `Secret` and the `OpenStackControlPlane` must match.

\[2\]: Note how the `storageMgmt` network is not present, that’s because in
vanilla NFS there is no management interface, as all operations are done
directly through the data path.

### 7.6. Back-end examples

To make things easier the `cinder-operator` code repository includes a good
number of samples to illustrate different configurations.

Provided back-end samples use the `kustomize` tool which can be executed
directly using `oc kustomize` to get a complete `OpenStackControlPlane` sample
file.

For example to get the iSCSI backend sample configuration into a
`~/openstack-deployment.yaml` file we can run:

```bash
oc kustomize \
  https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/lvm/iscsi \
  > ~/openstack-deployment.yaml
```

That will generate not only the `OpenStackControlPlane` object but also the
required `MachineConfig` objects for iSCSI, Multipathing, and even the creation
of an LVM VG.

Some of the samples may require additional steps, like adding a label to a node
for the LVM samples or deploying a Ceph cluster for the Ceph example, or Pure
that needs a custom container. Please look at the storage link as well as the
specific samples for more information.

All examples using Swift as the backup back-end, with the exception of the
Ceph sample that uses Ceph.

Currently [available samples are](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends):

- [Ceph](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/ceph)
- [NFS](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/nfs)
- [LVM](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/lvm)
  - [Using iSCSI](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/lvm/iscsi)
  - [Using NVMe-TCP](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/lvm/nvme-tcp)
- [HPE](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/hpe)
    - [3PAR using iSCSI](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/hpe/3par/iscsi)
    - [3PAR using FC](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/hpe/3par/fc)
- [NetApp](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/netapp/ontap)
  - [ONTAP using iSCSI](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/netapp/ontap/iscsi)
  - [ONTAP using FC](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/netapp/ontap/fc)
  - [ONTAP using NFS](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/netapp/ontap/nfs)
- [Pure Storage](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/pure)
  - [Using iSCSI](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/pure/iscsi)
  - [Using FC](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/pure/fc)
  - [Using NVMe-RoCE](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/pure/nvme-roce)
- [Dell PowerMax iSCSI](https://github.com/openstack-k8s-operators/cinder-operator/tree/main/config/samples/backends/dell/powermax/iscsi)

## 8. Configuring the backup service

The Block Storage service (cinder) provides an optional backup service that you
can deploy in your OpenStack on OpenShift environment.

You can use the Block Storage backup service to create and restore full or
incremental backups of your Block Storage volumes.

A volume backup is a persistent copy of the contents of a Block Storage volume
that is saved to a backup repository.

Configuring cinder backup is done under the `cinderBackup` section, and most of
the time requires using `customServiceConfig`, `customServiceConfigSecrets`,
`networkAttachments`, `replicas`, although in some cases even the
`nodeSelector`.

### 8.1. Back-ends

You can use Ceph Storage RBD, the Object Storage service (swift), NFS, or S3 as
your backup back end, and there are no vendor containers necessary for them.

The Block Storage backup service can back up volumes on any back end that the
Block Storage service (cinder) supports, regardless of which back end you choose
to use for your backup repository.

Only one back-end can be configured for backups, unlike in cinder volume where
multiple back-ends can be configured and used.

Even though the backup back-ends don’t have transport protocol requirements that
need to be run on the OpenShift node, like the volume back-ends do, the pods are
still affected by those because the backup pods need to connect to the volumes.
Refer to section [preparing transport protocol
requirements](#3-prepare-transport-protocol-requirements) section to ensure the
nodes are properly configured.

### 8.2. Setting the number of replicas

The cinder backup component can run multiple instances in Active-Active mode,
and that’s the  recommendation.

As explained earlier this is achieved by setting `replicas` to a value greater
than `1`.

The default value is `0`, so it always needs to be explicitly defined for it to
be defined.

Example:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
spec:
  cinder:
    template:
      cinderBackup:
        replicas: 3
```

### 8.3. Setting cinder Backup options

Like all the cinder components, cinder Backup inherits the configuration snippet
from the top `customServiceConfig` section defined under `cinder`, but it also
has its own `customServiceConfig` section under the `cinderBackup` section.

The backup back-end configuration snippet must be configured in the `[DEFAULT]`
group.

Each back-end has its own configuration options that are documented in their
respective sections, but here are some common options to all the drivers.

* `debug`: When set to `true` the logging level will be set to `DEBUG` instead
  of the default `INFO` level.  Debug log levels for the Scheduler can also be
  dynamically set without restart using the dynamic log level API functionality.
  Boolean value.
  Defaults to `false`

* `backup_service_inithost_offload`: Offload pending backup delete during backup
  service startup. If set to `false`, the backup service will remain down until
  all pending backups are deleted.
  Boolean value.
  Defaults to `true`

* `backup_workers`: Number of processes to launch in the backup pod. Improves
  performance with concurrent backups.
  Integer value
  Defaults to `1`

* `backup_max_operations`: Maximum number of concurrent memory, and possibly
  CPU, heavy operations (backup and restore) that can be executed on each pod.
  The number limits all workers within a pod but not across pods. Value of 0
  means unlimited.
  Integer value.
  Defaults to `15`

* `backup_native_threads_pool_size`: Size of the native threads pool used for
  the backup data related operations. Most backup drivers rely heavily on this,
  it can be increased for specific drivers that don’t.
  Integer value.
  Defaults to `60`

As an example here’s how we could enable debug logs, double the number of
processes and increase the maximum number of operations per pod to `20`:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
spec:
  cinder:
    template:
      customServiceConfig: |
        [DEFAULT]
        debug = true
      cinderBackup:
        customServiceConfig: |
          [DEFAULT]
          backup_workers = 2
          backup_max_operations = 20
```

## 9. Automatic database cleanup

The Block Storage (cinder) service does what’s called a soft-deletion of
database entries, this allows some level of auditing of deleted resources.

These deleted database rows will grow endlessly if not purged. OpenStack on
OpenShift automatically purges database entries marked for deletion for a set
number of days. By default, records are marked for deletion for 30 days. You can
configure a different record age and schedule for purge jobs.

Automatic DB purging is configured using the `dbPurge` section under the
`cinder` section, which has 2 fields: `age` and `schedule`.

Here is an example to purge the database every `20` days just after midnight:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      dbPurge:
        age: 20   [1]
        schedule: 1 0 * * 0   [2]
```

\[1\]: The number of days a record has been marked for deletion before it is
purged. The default value is 30 days. The minimum value is 1 day.

\[2\]: The schedule of when to run the job in a `crontab` format. The default
value is `1 0 * * *`.


## 10. Preserving jobs

The Block Storage service requires maintenance operations that are run
automatically, some are one-off and some periodic. These operations are run
using OpenShift Jobs.

Administrators may want to check the logs of these operations, which won’t be
possible if the Jobs and their pods are automatically removed on completion; for
this reason there is a mechanism to stop the automatic removal of Jobs.

To preserve pods for cinder we use the `preserveJob` field like this:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    template:
      preserveJobs: true
```

## 11. Resolving hostname conflicts

Most storage back-ends in cinder require the hosts that connect to them to have
unique hostnames, as these hostnames are used to identify the permissions and
the addresses (iSCSI initiator name, HBA WWN and WWPN, etc).

Because we are deploying in OpenShift the hostnames that the cinder volume and
cinder backup will be reporting are not the OpenShift hostnames but the pod
names instead.

These pod names are formed using a predetermined template:

* For volumes: `cinder-volume-<backend\_key>-0`
* For backups: `cinder-backup-<replica-number>`

If we use the same storage backend in multiple deployments we may end up not
honouring this unique hostname requirement, resulting in many operational
problems.

To resolve this, we can request the installer to have unique pod names, and
hence unique hostnames using a field called `uniquePodNames`.

When `uniquePodNames` are set to `true` a short hash will be added to these pod
names, which will resolve hostname conflicts.

Here is an example requesting unique podnames/hostnames:

```
apiVersion: core.openstack.org/v1beta1
kind: OpenStackControlPlane
metadata:
  name: openstack
spec:
  cinder:
    uniquePodNames: true
```
