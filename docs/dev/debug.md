# Debugging

When we deploy OpenStack using operators there are many moving pieces that must
work to get a running OpenStack deployment: OLM, OpenStack Operator, MariaDB
operator, RabbbitMQ operator, Keystone Operator, Cinder Operator, etc. For that
reason it's good to know a bit about the different pieces and how they connect
to each other.

Besides reading this guide it is recommended to read the [Debug Running Pods
documentation](
https://kubernetes.io/docs/tasks/debug/debug-application/debug-running-pod).

Usually the first step to resolve issues is to figure out **where** the issue
is happening and we can do it starting from the OLM and go forward through the
steps, do it in reverse starting from the cinder-operator and move backwards,
or anything in between.

### General flow

To be able to locate where things are failing we first need to know what the
expected steps are:

- [Deploying operators](#deploying-operators)
- [Propagating CRs](#propagating-crs)
- [Waiting for services](#waiting-for-services)
- [Deploying Cinder](#deploying-cinder)

##### Deploying operators

The expected result of running `make openstack` is to have the OpenStack
operators running in our OpenShift cluster.

The [Operator Lifecycle Manager (OLM)](https://olm.operatorframework.io/docs/)
is used to deploy the operators, and it is recommended to read its
documentation to understand it, but let's have a quick overview here.

When we are packaging our operator to be delivered through the OLM we create 3
container images, the operator itself, a bundle, and an index.

The bundle image contains the [`ClusterServiceVersion`
(CSV)](https://olm.operatorframework.io/docs/concepts/crds/clusterserviceversion/),
which we can think to be something like an RPM. It contains metadata about a
specific operator version as well as a template to be used by the OpenShift
Deployment operator to create our operator pods.

The index index image holds a sqlite database with bundle definitions and it
runs a grpc service when executed that lets consumers query the operators.

The operator contains the service with the controllers for a number of CRDs.

So how is the `make openstack` command actually deploying our operators using
those images?

The first thing it does is create an [`OperatorGroup`](https://docs.openshift.com/container-platform/4.11/operators/understanding/olm/olm-understanding-operatorgroups.html) to
provide multitenant configuration selecting the target namespaces in which to
generate required RBAC access for its member Operators.

Then it creates a [`CatalogSource`](
https://olm.operatorframework.io/docs/concepts/crds/catalogsource/) to
represent a specific location that can be queried to discover and install
operators and their dependencies.  In our case this points to the index image,
that runs a grpc service as mentioned before.

This step is necessary because our operator we are installing is not present in
the default catalog included in OpenShift.

At this point OpenShift knows everything about our operators, but we still need
to tell it that we want to install that specific operator.  This is where the
[`Subscription`](
https://olm.operatorframework.io/docs/concepts/crds/subscription/) comes into
play.

A Subscription represents an intention to install an operator and the `make
openstack` command creates one for our operator and specifies our
`CatalogSource` as the source to find our operator.  This means that it will
install our custom operator even if we have an official operator already
released in the official operator catalog.

This newly created `Subscription` triggers the creation of the index pod so the
OLM can query the information, then an [`InstallPlan`](
https://docs.openshift.com/container-platform/4.11/rest_api/operatorhub_apis/installplan-operators-coreos-com-v1alpha1.html)
gets created to take care of installing the resources for the operator.

If the operators are not correctly deployed after running `make openstack`,
then we should look into the `InstallPlan` and check for errors.

```
oc describe InstallPlan | less
```

If there is no `InstallPlan` we have to check if the index pod is running:

```
oc get pod -l olm.catalogSource=openstack-operator-index
```

If it isn't, check for the `CatalogSource`:

```
oc describe catalogsource openstack-operator-index
```

##### Propagating CRs

When we run `make openstack_deploy` we are basically applying our
`OpenStackControlPlane` manifest, as defined in the `OPENSTACK_CR`
environmental variable, which is a CRD defined by the openstack-operator.

The openstack-operator has a controller watching for `OpenStackControlPlane`
resources, so when it sees a new one it starts working to reconcile it. In this
case that means propagating the `template` in the `cinder` section into a new
`Cinder` resource.

So after the `OpenStackControlPlane` resource we should be able to see the
`Cinder` resource created by the openstack-operator, and this should contain
the same information present in the `template` section in the `cinder` section
of our manifest.

```
oc get cinder
```

If we don't see this resource, then we need to check first that the `enabled`
key inside the `cinder` section is not set to `false` and then look at the
openstack-operator logs and search for reconciliation errors.

```
OPENSTACK_OPERATOR=`oc get pod -l control-plane=controller-manager -o custom-columns=POD:.metadata.name|grep openstack`
oc logs $OPENSTACK_OPERATOR
```

Something similar should happen for the keystone and glance operators.

```
oc get keystoneapis
oc get glances
```

##### Waiting for services

Now that we have a `Cinder` resource it's the cinder-operator's turn, that has
a specific controller waiting for these `Cinder` resource, but before it starts
deploying the cinder services it has to make sure that everything is in place
for the services to run correctly.

The `Cinder` controller keeps the status of each of the steps it needs to
perform as conditions, which can be checked with `oc describe cinder`.

The steps are:

- Request the RabbitMQ transport url information: A `TransportURL` resource is
  created by the `Cinder` controller and will be handled the openstack-operator
  that waits until the RabbitMQ has been deployed and is ready.

  While RabbitMQ is not ready we can see that the `TransportURL` condition
  called `TransportURLReady` has a `False` status.

  Once RabbitMQ can accept requests the openstack-operator sets the
  `TransportURLReady` condition to `True` and creates a secret and references
  it in the `TransportURL` status field called `Secret Name`.

  The condition in the `Cinder` resource for this step is called
  `CinderRabbitMqTransportURLReady`.

  If we never see the condition changing we should check the RabbitMQ pod (`oc
  describe rabbitmq-server-0`) or its operator (`oc describe
  controller-manager-8674c4db5c-lq56w`)

- Check for required OpenStack secret: Condition `InputReady`.

- Create `ConfigMap`s and `Secret`s: Condition `ServiceConfigReady`.

- Request a database: Condition `DBReady`. The new database for cinder is
  requested using the `mariadbdatabase` resource that is handled by the
  mariadb-operator.

- Run the database initialization code: Condition `DBSyncReady`.  The db sync
  is executed using an OpenShift [`Job`](
  https://docs.openshift.com/container-platform/4.11/nodes/jobs/nodes-nodes-jobs.html).

##### Deploying Cinder

Now that everything is ready the `Cinder` controller will request the creation
of the different cinder services: API, backup, scheduler, and volume.

Each of the services has its own CRD (`cinderapi`, `cinderbackup`,
`cinderscheduler`, and `cindervolume`) and there is a specific controller in
the cinder-operator to reconcile each of these CRDs, just like there was one
for the top level `Cinder` CRD.

If the cinder-operator is running successfully then it should generate those
resources based on the top level `Cinder` resource and the information gathered
by the `TransportURL`, the `mariadbdatabase`, and the generated `ConfigMap`s
and `Secret`s.

If we don't see a specific resource kind it may be because its section in
`Cinder` didn't have the `replication` field set to 1 or greater, or because
there is a failure during it's creation.  In this last case we should check the
cinder-operator's log and look for the error:

```
CINDER_OPERATOR=`oc get pod -l control-plane=controller-manager -o custom-columns=POD:.metadata.name|grep cinder`
oc logs $CINDER_OPERATOR
```

Each of these 4 controllers is responsible for a specific cinder service, and
the API will use a [`Deployment`](
https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) but the
other 3 services will use a [`StatefulSet`](
https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
instead, because the first doesn't care about the hostname but the other 3 do,
since it is used by the Cinder code.

The operator deploys an independent pod for each cinder volume backend instead
of deploying them all in a single pod.

At this point we should see the pods for each service running, or at least
trying to run.  If they cannot run successfully we should `describe` the pod to
see what is failing.

It's important to know that these 4 services use the [Kolla project](
https://wiki.openstack.org/wiki/Kolla) to prepare and start the service.

### OpenStack CRDs

The specific CRDs for the whole OpenStack effort can be listed after
successfully running `make openstack` with:

```
oc get crd -l operators.coreos.com/openstack-operator.openstack=
```

### Configuration generation

In the [waiting for services section](#waiting-for-services) we described the
creation of `ConfigMap`s and `Secret`s.  Those are basically the scripts and
default configuration from the `templates` directory, but then we need to add
to that default configuration the customizations from the user, such as the
backend configuration.

The final configuration is created by an [init container](
https://kubernetes.io/docs/concepts/workloads/pods/init-containers/) in each
service pod running `templates/cinder/bin/init.sh` and using templates and
environmental variables.

The script generates files in `/var/lib/config-data/merged` before any other
container in the pod is started and then the directory is available in the
probe and service containers.

There may be times where we want to debug what is happening in this
configuration generation, for the purpose we have the `initContainer` key in
the `debug` section of our manifests.

For example, if we wanted to debug this creation in the cinder-scheduler
service we would edit the top level `OpenStackControlPlane`:

```
oc edit OpenStackControlPlane openstack
```

And then in the `template` section of the `cinder` section look for
`cinderScheduler` and in the `debug` key change `initContainer: false` to
`initContainer: true`, then save and exit.

After saving the openstack-operator will notice the change and propagate this
change to the `Cinder` resource, which in turn will get propagated by the
cinder-operator into the `CinderScheduler` resource, which will terminate the
existing pod and create a new one changing the command that is run.

When debugging the init container the command that is run is no longer the
`init.sh` script mentioned before, instead it executes a loop that sleeps for
5 seconds while file `/tmp/stop-init-container` exist.

So once the debug mode has been applied we can see that the status of the
container is `Init`:

```
$ oc get pod cinder-scheduler-0
NAME                 READY   STATUS     RESTARTS   AGE
cinder-scheduler-0   0/2     Init:0/1   0          3m2s
```

And we can go into the init container and manually run the script:

```
oc exec -it cinder-scheduler-0 -c init /bin/bash

# See source files
ls /etc/cinder/cinder.conf
ls /var/lib/config-data/

# Run script
/usr/local/bin/container-scripts/init.sh

# See merged result
ls /var/lib/config-data/merged
```

Once we are satisfied with the resulting configuration files we can `rm
/tmp/stop-init-container` and the container will be destroyed (we'll see
*command terminated with exit code 137* message and the service and probe
containers will be started.

### Cinder services

Even though we are not working on the cinder service there will be times when
we need to debug it.

Similarly to the [init container debugging](#configuration-generation) we also
have a way to enable debugging in the service with the `service` key under
`debug` in the specific service section we want to debug.

Here the command for the service container is changed to an infinite sleep and
the service container probes are disabled to prevent the container from getting
constantly restarted.

**NOTE**: Setting the service debug mode to `true` the easiest way to debug a
container that is in `CrashLoopBackOff` due to probe failures.

Once the pod has been restarted and the service container is just sleeping we
can go into the container.

```
oc exec -it cinder-scheduler-0 /bin/bash
```

As mentioned before we are using [Kolla](https://wiki.openstack.org/wiki/Kolla)
to prepare and start the service, so it's a good idea to be familiar with [its
API](https://docs.openstack.org/kolla/ussuri/admin/kolla_api.html).

The first thing we need to do is to ask Kolla to move some files around for the
service to be able to run.  The files are defined in the kolla configuration
json file that lives in `$KOLLA_CONFIG_FILE`.  This is important because in
this step the merged config file is moved to `/etc/cinder/cinder.conf`.

To trigger the file moves we run:

```
/usr/local/bin/kolla_set_configs
```

Now we can check that everything looks right, and then start the service:

```
/usr/local/bin/kolla_start
```

If we see an unexpected behavior we can hit Ctrl+C to stop the service and
explore the system, make changes to files or code (for example set a `pdb`
trace), etc.

Alternatively to setting the `service` to `debug` we can also run:

```
oc debug cinder-scheduler-0 -c cinder-scheduler
```

And we'll get a pod in debug mode which is an exact copy of our running pod
configuration but going into a shell.

### Probes

Difference API and the others

There are 3 kind of probes in OpenShift: Liveness, Readiness and Startup.
Please [refer to the documentation for more information on them](
https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes).

The cinder API service has the liveness and readiness probes.  The readiness
probe checks that the API can respond to HTTP requests so it can be taken off
the OpenShift service load balancer if it can't.  The liveness checks the
`/healthcheck` endpoint.

The other cinder services -scheduler, volume, and backup- don't have the
readiness probe because they don't leverage the OpenShift Service and its load
balancer, as they communicate through REST API but RabbitMQ RPCs instead.  So
these 3 services only have a Startup and Liveness probes.

The probes of these 3 internal services do HTTP requests to a very simple HTTP
server that runs in a container within the service pod.  The probe server code
is at `templates/cinder/bin/healthcheck.py` and its port is not exposed outside
the pod because OpenShift probe checking is done from the same network
namespace as the pod.

The probe server is quite simple right now and all it does is go to the Cinder
database and check the status of the service.  If the service doesn't appears
as up, then it will return 500 status error code.

In the future the probe may be extended to send RabbitMQ requests (for example
a get logs call) to check for connectivity.

If we want to debug the probe we should set the service on debug mode and start
the service as described in the [Cinder services section](#cinder-services),
and then go into a different terminal and go into the probe container and
run the script, probably with `pdb`, at this point we can make the query from
another terminal.

For example, after the cinder-scheduler is manually started we run in terminal
number 2:

```
oc exec -it cinder-scheduler-0 -c probe /bin/bash
python3 -m pdb /usr/local/bin/container-scripts/healthcheck.py scheduler /var/lib/config-data/merged/cinder.conf

# now we run the script, set break points, etc
```

Once the probe server is listening to requests we can go to our terminal 3 and:

```
oc exec -it cinder-scheduler-0 -c probe /bin/bash
curl localhost:8080
```

### Operator

The easiest way to debug the operator is running it locally and modify the
command used to run the operator to include the debugger.  For example if we
are using [Delve](https://github.com/go-delve/delve) we can do:

```sh
make build
OPERATOR_TEMPLATES=$PWD/templates dlv exec ./bin/manager
```

Debugging using your own IDE [locally](
https://golangforall.com/en/post/goland-debugger.html) or [remotely when using
a VM](https://golangforall.com/en/post/go-docker-delve-remote-debug.html)
requires some extra steps but is also possible.

If we want to debug the container inside OpenShift we could build a custome
container image that has the dlv command, set the service to `debug`, and then
run the debugger:

```
oc exec -it /dlv exec /manager
```
