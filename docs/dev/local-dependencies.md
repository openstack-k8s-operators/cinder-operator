# Running the operator locally with external dependencies

**NOTE**: This article [makes some assumptions](assumptions.md), make sure they
are correct or adapt the steps accordingly.

If you've already [run the operator locally](local.md) before, you'll be
familiar with the process of removing the cinder podified services and the
cinder-operator pod from the OpenShift cluster, building the operator and
running it locally.

It's very likely that during the development of the cinder-operator you'll find
yourself needing to use a newer version of a dependency or even needing to use
an unmerged code of a dependency.

This short guide builds on the knowledge provided in [guide to locally run the
operator](local.md) to explain those external dependencies usages.

The steps will be the same, the only difference is that some additional steps
are necessary between the [Preparation](local.md#preparation) and the [Build
and Run](local.md#build-and-run) steps to indicate the new references.

There are slight differences between the possible cases, lib-common dependency
update, circular reference with the openstack-operator, the external dependency
code being merged or not, having the external dependency's code locally or not,
etc., so to facilitate its understanding we'll provide separate explanations
for most of them.

### Merged simple reference

If we want to compile the cinder-operator with newer lib-common code that has
already merged in its repository but don't want to wait for dependabot to
 propose an update to the project's dependencies and for it to get merged we
can just run the following commands from the `cinder-operator` repository
before [Building and Running the operator](local.md#build-and-run):

```
go get github.com/openstack-k8s-operators/lib-common/modules/common
go get github.com/openstack-k8s-operators/lib-common/modules/database
go get github.com/openstack-k8s-operators/lib-common/modules/storage
```

**NOTE**: Using the ``go get`` command ensures that other `go.mod` adjustments
are made as needed to satisfy constraints imposed by other modules.

After running these commands we can see that the `go.mod` and `go.sum` have
been updated:

```
$ git diff go.*

diff --git a/go.mod b/go.mod
index 599f056..8e7d423 100644
--- a/go.mod
+++ b/go.mod
@@ -9,9 +9,9 @@ require (
 	github.com/openshift/api v3.9.0+incompatible
 	github.com/openstack-k8s-operators/cinder-operator/api v0.0.0-20221010180347-a9a8efadf3c3
 	github.com/openstack-k8s-operators/keystone-operator/api v0.0.0-20220927090553-6b3218c776f7
-	github.com/openstack-k8s-operators/lib-common/modules/common v0.0.0-20221103175706-2c39582ce513
-	github.com/openstack-k8s-operators/lib-common/modules/database v0.0.0-20220923094431-9fca0c85a9dc
-	github.com/openstack-k8s-operators/lib-common/modules/storage v0.0.0-20220923094431-9fca0c85a9dc
+	github.com/openstack-k8s-operators/lib-common/modules/common v0.0.0-20221117092428-c1190ea3bf3d
+	github.com/openstack-k8s-operators/lib-common/modules/database v0.0.0-20221117092428-c1190ea3bf3d
+	github.com/openstack-k8s-operators/lib-common/modules/storage v0.0.0-20221117092428-c1190ea3bf3d
 	github.com/openstack-k8s-operators/mariadb-operator/api v0.0.0-20221014164348-0a612ae8b391
 	github.com/openstack-k8s-operators/openstack-operator/apis v0.0.0-20221107090218-8d63dba1ec13
 	k8s.io/api v0.25.4
diff --git a/go.sum b/go.sum
index c19f296..2130ed6 100644
--- a/go.sum
+++ b/go.sum
@@ -323,12 +323,18 @@ github.com/openstack-k8s-operators/keystone-operator/api v0.0.0-20220927090553-6
 github.com/openstack-k8s-operators/keystone-operator/api v0.0.0-20220927090553-6b3218c776f7/go.mod h1:q/owiyXlI2W4uQR4TeHPeeN75AGDfyZgQdNHeKUYN68=
 github.com/openstack-k8s-operators/lib-common/modules/common v0.0.0-20221103175706-2c39582ce513 h1:PSXOLFTskoG9R/YR4Pg5AOJYS3CEnFbZ2yVdrk9xOE4=
 github.com/openstack-k8s-operators/lib-common/modules/common v0.0.0-20221103175706-2c39582ce513/go.mod h1:KWqK7l2ej+rIYngoNUrxE2YjKGlRAAgJXXM0uU2R6XY=
+github.com/openstack-k8s-operators/lib-common/modules/common v0.0.0-20221117092428-c1190ea3bf3d h1:/1FTHxBQJo4xM0GmJCX5wPCYmyLWTw1uHQKCydGH3mY=
+github.com/openstack-k8s-operators/lib-common/modules/common v0.0.0-20221117092428-c1190ea3bf3d/go.mod h1:KWqK7l2ej+rIYngoNUrxE2YjKGlRAAgJXXM0uU2R6XY=
 github.com/openstack-k8s-operators/lib-common/modules/database v0.0.0-20220923094431-9fca0c85a9dc h1:87lUVT3MLRI4Vg0nHpupwPKXtykGX3hZzPl5k6Kcyng=
 github.com/openstack-k8s-operators/lib-common/modules/database v0.0.0-20220923094431-9fca0c85a9dc/go.mod h1:umGUqQO4JtgefAaIwZjP+TxfxsLMEEeK/6VNzk8ooaI=
+github.com/openstack-k8s-operators/lib-common/modules/database v0.0.0-20221117092428-c1190ea3bf3d h1:lO5WmV9RjVAIxbr1HPjvqVy6niatdEPG+FyEbL4FMpc=
+github.com/openstack-k8s-operators/lib-common/modules/database v0.0.0-20221117092428-c1190ea3bf3d/go.mod h1:umGUqQO4JtgefAaIwZjP+TxfxsLMEEeK/6VNzk8ooaI=
 github.com/openstack-k8s-operators/lib-common/modules/openstack v0.0.0-20220915080953-f73a201a1da6 h1:MVNEHyqD0ZdO9jiyUSKw5M2T9Lc4l4Wx1pdC2/BSJ5Y=
 github.com/openstack-k8s-operators/lib-common/modules/openstack v0.0.0-20220915080953-f73a201a1da6/go.mod h1:YsqouRH8DoZAjFaxcIErspk59BcwXtVjPxK/yV17Wrc=
 github.com/openstack-k8s-operators/lib-common/modules/storage v0.0.0-20220923094431-9fca0c85a9dc h1:Dud2dr25VhaZF9Av28nqmCeBfNkGWDckZ5TaajEcGFc=
 github.com/openstack-k8s-operators/lib-common/modules/storage v0.0.0-20220923094431-9fca0c85a9dc/go.mod h1:fhM62I45VF/5WVpOP1h9OpTfFn+lF2XGrT5jUBKEHVc=
+github.com/openstack-k8s-operators/lib-common/modules/storage v0.0.0-20221117092428-c1190ea3bf3d h1:b2dqfShyX4NO/NMe3rTK1xWRF91ITobjQEfO6ftO6yM=
+github.com/openstack-k8s-operators/lib-common/modules/storage v0.0.0-20221117092428-c1190ea3bf3d/go.mod h1:fhM62I45VF/5WVpOP1h9OpTfFn+lF2XGrT5jUBKEHVc=
 github.com/openstack-k8s-operators/mariadb-operator/api v0.0.0-20221014164348-0a612ae8b391 h1:Bd1e4CG/0gQbRoSH1EJLS1tin9XUjPR2s1e+dpBHiUs=
 github.com/openstack-k8s-operators/mariadb-operator/api v0.0.0-20221014164348-0a612ae8b391/go.mod h1:HiEKXmDSJ6Gl+pN7kK5CX1sgOjrxybux4Ob5pdUim1M=
 github.com/openstack-k8s-operators/openstack-operator/apis v0.0.0-20221107090218-8d63dba1ec13 h1:GkYSRpfdDav5HipJ2oIYqpY8crLaDVRBmhqlUztDTV4=
```

Now we can proceed as usual with the [Building and
Running](local.md#build-and-run) steps.

Once we have confirmed that our code changes work as expected with the new
dependency version we can submit our PR as we normally do but including the
`go.mod` and `go.sum` changes.

If we want to use a specific tag instead of `HEAD` we can provide it in our
calls:

```
go get github.com/openstack-k8s-operators/lib-common/modules/common@v0.1.0
go get github.com/openstack-k8s-operators/lib-common/modules/database@v0.1.0
go get github.com/openstack-k8s-operators/lib-common/modules/storage@v0.1.0
```

### Unmerged local simple reference

There will be times when we'll be working on lib-common code in our local
repository while using it in the cinder-operator and we want to test this.

In this case we cannot use the `go get` because our lib-common module is not
 within the cinder-operator command so we'll use the `go work edit` command
instead:

```
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/common=../lib-common/modules/common
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/database=../lib-common/modules/database
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/storage=../lib-common/modules/storage
```

Alternatively we can use the `hack/setdeps.py` script to do the same thing:

```
hack/setdeps.py lib-common=../lib-common
```

Now we can proceed as usual with the [Building and
Running](local.md#build-and-run) steps.  If we use `make build` then we'll need
to let it know that it should use our workspace changes:

```
GOWORK= make build
```

Once we have confirmed that our code changes work as expected with the new
dependency version we will need to submit the lib-common PR first and then the
one in cinder-operator with the new dependency, as explained in the previous
section.

### Simple reference in PR

Somebody may have submitted a PR to lib-common and we want to start writing
code in the cinder-operator before that PR gets merged.

Here we can either download the PR to our own local repository or just
reference the one from GitHub.

As an example, let's look at the `extraVolumes` effort and the [lib-common
PR#88](https://github.com/openstack-k8s-operators/lib-common/pull/88).

To reference it directly we'll go to GitHub's website and look for the source
 of the PR, which in this case it's [fmount's extra_volumes branch](
https://github.com/fmount/lib-common/tree/extra_volumes).

So now we replace our lib-common with that one:

```
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/common=github.com/fmount/lib-common/modules/common@extra_volumes
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/database=github.com/fmount/lib-common/modules/database@extra_volumes
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/storage=github.com/fmount/lib-common/modules/storage@extra_volumes

go mod tidy
```

Alternatively we can use the `hack/setdeps.py` to do all that for us:

```
hack/setdeps.py lib-common=88
```

Or just the last part:

```
go work edit -replace lib-common=fmount/lib-common
```

If we have a local repository and we may want to make changes to the lib-common
code ourselves (in case we find some issue) we can pull the PR and then
reference it:

```
cd ../lib-common

git fetch upstream pull/88/head:extra_volumes
git checkout extra_volumes

cd ../cinder-operator

go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/common=../lib-common/modules/common
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/database=../lib-common/modules/database
go work edit -replace github.com/openstack-k8s-operators/lib-common/modules/storage=../lib-common/modules/storage
```

Alternatively we can replace the last 3 commands with a single command:

```
hack/setdeps.py lib-common=../lib-common
```

Now we can proceed as usual with the [Building and
Running](local.md#build-and-run) steps.  If we use `make build` then we'll need
to let it know that it should use our workspace changes:

```
GOWORK= make build
```

### Circular references

There are times where we have circular dependencies where cinder-operator needs
 a new structure defined in the openstack-operator but at the same time the
openstack-operator uses the cinder-operator structures to define the
`OpenStackControlPlane`.

An example of this happening is the work on the `TrasportURL` that was
 introduced in [openstack-operator
 PR#37](https://github.com/openstack-k8s-operators/openstack-operator/pull/27)
and used in the [cinder-operator
PR#62](https://github.com/openstack-k8s-operators/cinder-operator/pull/62).

The solution of how to replace the dependencies is pretty much the same as the
one we've seen in the previous section, the only difference is that we need to
do it in both repositories.

We see in github that the cinder-operator PR source branch is
`use_transport_url_crd` from `abays` and the openstack-operator PR source
branch is `rabbitmq_transporturl` from `dprince`, so we replace their
dependencies:

```
cd ~/cinder-operator
go work edit -replace  github.com/openstack-k8s-operators/openstack-operator/apis=github.com/dprince/openstack-operator@rabbitmq_transporturl

cd ~/openstack-operator
go work edit -replace github.com/openstack-k8s-operators/cinder-operator/api=github.com/abays/cinder-operator@use_transport_url_crd
```

**NOTE**: If we are using local directory references we should use absolute
paths instead of relative ones like we did before.

In this case since we have also modified the openstack-operator we'll have to
stop it in our OpenShift cluster and run it locally before we can proceed with
[building and running the cinder-operator](local.md#build-and-run).  Remember
to add the `GOWORK=` to the `make build` call.

```sh
CSV=`oc get -l operators.coreos.com/openstack-operator.openstack= -o custom-columns=CSV:.metadata.name --no-headers csv`

oc edit csv $CSV
```

This will drop us in our editor with the contents of CSV YAML manifest where
we'll search for the first instance of `name:
openstack-operator-controller-manager`, and we should see something like:

```
      - label:
          control-plane: controller-manager
        name: openstack-operator-controller-manager
        spec:
          replicas: 1
          selector:
            matchLabels:
              control-plane: controller-manager
```

Where we see `replicas: 1` change it to `replicas: 0`, save and exit. This
triggers the termination of the openstack-operator pod.

Now we just build and run our local openstack-operator:

```
cd ~/openstack-operator
GOWORK= make run
```

Now we can continue, in another terminal, [running the cinder-operator
locally](local.md#build-and-run) but this time we cannot use `make run`,
because it will fail due to ports :8080 and :8081 already being in use:

```
bin/manager -metrics-bind-address=:8082 -health-probe-bind-address=:8083
```
