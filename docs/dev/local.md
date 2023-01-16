# Running the operator locally

**NOTE**: This article [makes some assumptions](assumptions.md), make sure they
are correct or adapt the steps accordingly.

This development model is useful for quick iterations of the operator code
where one can easily use debugging tools and change template and asset files on
the fly without needed for deployments.

We will build and run the operator on the host machine that is running the
OpenShift VM and it will connect to the OpenShift cluster from the outside
using our credentials.

The downside of this approach is that we are not running things in a container
inside OpenShift, so there could be differences between what is accessible in
one case and the other.  For example, we'll have admin credentials running
things on the host, whereas the operator deployed inside OpenShift will have
more restrictive ACLs.

Another downside is that we'll have to manually login into the cluster every
time our login credentials expire.

This process can be used for quick development and once we are ready we can
move to [building custom images and running them in OpenShift](custom-image.md)
where things are run as they would normally do.

### Preparation

This article assumes we have followed the [Getting
Started](../../README.md#getting-started) section successfully so we'll not
only have a cinder-operator pod running, but also the different cinder
services.

Since we have everything running we need to uninstalling both the
cinder-operator and Cinder services.

To uninstall the Cinder services we will edit the `OpenStackControlPlane` CR
that we used to deploy OpenStack and is present in the OpenShift cluster.

```sh
oc edit OpenStackControlPlane openstack
```

Now we search for the `cinder` section and in its `template` section we change
the `replicas` value to `0` for the 4 services, then save and exit the editor.

This will make the openstack-operator notice the change and modify the `Cinder`
CR, which in turn will be detected by the cinder-operator triggering the
termination of the cinder services in order during the reconciliation.

**NOTE**: The Cinder DB is not deleted when uninstalling Cinder services, so
Cinder DB migrations will run faster on the next deploy (they won't do
anything) and volume, snapshot, and backup records will not be lost.

Once we no longer have any of the cinder service pods (`oc get pod -l
service=cinder` returns no pods) we can proceed to remove the cinder-operator
pod that is currently running on OpenShift so it doesn't conflict with the one
we'll be running locally.

We search for the name of the `ClusterServiceVersion` of the OpenStack operator
and edit its current CR:

```sh
CSV=`oc get -l operators.coreos.com/openstack-operator.openstack= -o custom-columns=CSV:.metadata.name --no-headers csv`

oc edit csv $CSV
```

This will drop us in our editor with the contents of CSV YAML manifest where
we'll search for the first instance of `name:
cinder-operator-controller-manager`, and we should see something like:

```
      - label:
          control-plane: controller-manager
        name: cinder-operator-controller-manager
        spec:
          replicas: 1
          selector:
            matchLabels:
              control-plane: controller-manager
```

Where we see `replicas: 1` change it to `replicas: 0`, save and exit. This
triggers the termination of the cinder-operator pod.

### Build and Run

Before continuing make sure you are in the `cinder-operator` directory where
the changes to test are.

If our local code is changing the cinder-operator CRDs or adding new ones we
need to regenerate the manifests and change them in OpenShift.  This can be
easily done by running `make install`, which first builds the CRDs (using the
`manifests` target) and then installs them in the OpenShift cluster.

Now it's time to build the cinder operator (we'll need go version 1.18) and run
it (remember you need to be logged in with `oc login` or `crc_login` if you are
using the helper functions):

```sh
make build
OPERATOR_TEMPLATES=$PWD/templates ./bin/manager
```

We can also do it with a single make call: `OPERATOR_TEMPLATES=$PWD/templates
make install run`

Any changes in the `templates` directory will be automatically available to the
operator and there will be no need to recompile, rebuild, or restart the
cinder-operator.

Now that the cinder operator is running locally we can go back and set the
`replicas` back to `1` in the `cinder` section of the `OpenStackControlPlane`
CR to trigger the deployment of the Cinder services.  The easiest way to do so
is to apply the original manifest we used to deploy OpenStack in the first
place:

```sh
oc apply -f hack/dev/openstack.yaml
```

We should now see the local cinder-operator detecting the change and we'll be
able to see validate our code changes.

### Final notes

If there's something wrong we can stop the operator with Ctrl+C and repeat the
process: Run `make install` if there are changes to the CRDs and rebuild and
rerun the cinder-operator.

Remember that this workflow doesn't take into account the cinder-operator's
RBACs so even if things work here they could fail during a real deployment.  It
would be prudent to [run the new code in OpenShift](custom-image.md) to be sure
that there will be no ACL issues.
