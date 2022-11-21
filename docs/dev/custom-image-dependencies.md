# Running the operator in OpenShift with external dependencies

**NOTE**: This article [makes some assumptions](assumptions.md), make sure they
are correct or adapt the steps accordingly.

If you've already [run the operator in OpenShift](custom-image.md) before,
you'll be familiar with the process of running custom cinder-operator code
by building and deploying a custom image.

Let's see what we have to do to run our operator in OpenShift when we need to
compile and build containers with external dependencies.

This short guide builds on the knowledge provided in [guide to running the
operator in OpenShift](custom-image.md) to explain external dependencies usage.

The steps will be the same, the only difference is that some additional steps
are necessary between the [Preparation](custom-image.md#preparation) and the
[Image build](custom-image.md#image-build) steps to indicate the new
references.

To avoid repeating things again, this text assumes familiarity with the
 concepts presented in the [running locally with external
dependencies](local-dependencies.md) article.

All the cases described in running the operator locally with external
dependencies where the dependency was expressed as github repositories will
work fine when building the containers, so we don't need to do anything
special.

The difference comes when using local paths to reference the external
dependencies, because those are outside of the root directory of the
`Dockerfile` and therefore won't be present in the building container.

The solution is to use bind mounts.

### Unmerged local simple reference

Assuming the same example as running things locally we would do:

```
cd ~/cinder-operator
mkdir -p tmp/lib-common
sudo mount --bind `realpath ../lib-common` `realpath tmp/lib-common`

go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/common=tmp/lib-common/modules/common
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/database=tmp/lib-common/modules/database
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/storage=tmp/lib-common/modules/storage
```

Alternatively we can replace the last 3 commands with a single command:

```
hack/setdeps.py lib-common=tmp/lib-common
```

Then we can just continue with [building the
image](custom-image.md#image-build) like we would normally do with the exception
of having to add `GOWORK=` as well:

```
GOWORK= make docker-build
```

### Circular references

In the circular references case when using local paths it's a bit more
complicated than the local run case, because the bind mount created from the
cinder-operator to the openstack-operator is not preserved when we bind mount
the cinder-operator from the openstack-operator, so we need to link it again.

We'll assume we have the dependency between cinder-operator and
openstack-operator and that our local repositories have the code we want to
use.

```
cd ~/cinder-operator
mkdir -p tmp/openstack-operator
sudo mount --bind `realpath ../openstack-operator` `realpath tmp/openstack-operator`

go work edit -replace  github.com/openstack-k8s-operators/openstack-operator/apis=tmp/openstack-operator


cd ~/openstack-operator
mkdir -p tmp/cinder-operator
sudo mount --bind `realpath ../cinder-operator` `realpath tmp/cinder-operator`
sudo mount --bind `realpath ./` `realpath tmp/cinder-operator/tmp/openstack-operator`

go work edit -replace github.com/openstack-k8s-operators/cinder-operator/api=tmp/cinder-operator/api
```

Now we can continue with the [Image build steps](custom-image.md#image-build),
again adding `GOWORK=` to the `docker-build`.
