# Running the operator in OpenShift

**NOTE**: This article [makes some assumptions](assumptions.md), make sure they
are correct or adapt the steps accordingly.

This development model is the closest to the *real thing* because the
cinder-operator will be running in OpenShift in a pod and using the RBACs we
have defined, but since we have to build the container and upload it to a
registry, and then OpenShift needs to download it, it will be considerably
slower than just running [the operator locally](local.md).

Before we go and build the container image we should decide what container
image registry we want to use, because we can use a public registry,such as
quay.io, a private registry, or even run a *toy* registry locally.

Running a *toy* registry may require running a couple more commands the first
time, but it will prove to be much faster, since image pushing and pulling will
not go through your internet connection.

To make things easier we [have a simplified explanation of how to run a local
toy registry](local-registry.md) in the IP address (`192.168.130.1`) that CRC
assigns the host when deployint the VM.

The next of the document assumes the *toy* registry is being used.

### Preparation

This article assumes we have followed the [Getting
Started](../../README.md#getting-started) section successfully so we'll not
only have a cinder-operator pod running, but also the different cinder
services.

Unlike when we are [running the operator locally](local.md) here we don't need
to manually stop the existing operator, we can leave it running.  And we won't
stop the Cinder services either to illustrate another way of doing things,
though we could do it here if we want, just like we did when [running the
operator locally](local.md#preparation).

### Image build

Before continuing make sure you are in the `cinder-operator` directory where
the changes to test are.

Now we will build the container image for the cinder-operator:

```sh
export IMG=192.168.130.1:5000/cinder-operator:latest

make docker-build
```

This command takes a while to execute, but once it completes we'll have the new
cinder-operator container image in our local registry (don't mistake this for
the *toy* registry) and we can confirm the presence of the image with `podman
images`.

Now it's time to push it to our *toy* registry:

```sh
export VERIFY_TLS=false

make docker-push
```

At this point OpenShift will be able to pull the new image container from this
*toy* registry instead of having to access a registry from Internet.

### Run

Before we try to run the new operator we may have to generate and install the
new CRDs and RBACs if they have changed. This can be easily done by running
`make install`.

As mentioned before we haven't stopped the cinder-operator that is running in
OpenShift, so what we are going to do is replace the running operator to make
it use the one we just built.

We start by searching for the name of the `ClusterServiceVersion` of the
OpenStack operator and editing its current CR:

```sh
CSV=`oc get -l operators.coreos.com/openstack-operator.openstack= -o custom-columns=CSV:.metadata.name --no-headers csv`

oc edit csv $CSV
```

Now we search for the first instance of `name:
cinder-operator-controller-manager`, and within its `spec.containers` sections
look for the `image:` definition where the cinder-container image location is
defined.

Now we change the location to `192.168.130.1:5000/cinder-operator:latest`, save
and exit the editor.

At this point OpenShift should detect the change and try to reconcile the CSV,
this will terminate the existing cinder-operator pod that is running and start
a new one with the new image. We can confirm it using `oc describe pod` with
the name of the new cinder-operator pod.

**NOTE**: If the new image is not used we may need to force a faster respose by
changing the cinder-operator replicas to 0 in the CSV, save and exit, wait
until the pod is terminated and then change it back to 1.

Since we didn't remove the Cinder services that were running, once the new
cinder-operator is running it should reconcile the existing `Cinder` CR,
modifying existing `StatefulSet` and `Deployment` manifests used for the Cinder
services according to the new code.

We may also want to uninstall all cinder services and see how our newly
deployed operator deploys them from scratch.  To do that we'll need to edit the
`OpenStackControlPlane` CR that was used to deploy OpenStack and currently
exists in the OpenShift cluster.

```sh
oc edit OpenStackControlPlane openstack
```

Now we search for the `cinder` section and in its `template` section we change
the `replicas` value to `0` for the 4 services, then save and exit the editor.

This will make the openstack-operator notice the change and modify the `Cinder`
CR, which in turn will be detected by the cinder-operator triggering the
termination of the cinder services in order during the reconciliation.

Once the cinder service pods are gone we can set the `replicas` back to `1` in
the `cinder` section of the `OpenStackControlPlane` CR to trigger the
deployment of the Cinder services.  The easiest way to do so is to apply the
original manifest we used to deploy OpenStack in the first place:

```sh
oc apply -f hack/dev/openstack.yaml
```

**NOTE**: The Cinder DB is not deleted when uninstalling Cinder services, so
Cinder DB migrations will run faster this time (they won't do anything).

To see what the cinder-operator is doing we'll need to do `oc logs -f` for the
cinder-operator.

### Final notes

If we need to make changes to the operator we'll need to go through the `make
install` and `make docker-build docker-push` cycle again to create a new image.

Making OpenShift use the new image we just built should be easy enough, we just
need to delete the cinder-operator pod, since the `imagePullPolicy` policy of
the pod is `Always`, so it checks that the source image is up to date every
time it runs the pod.
