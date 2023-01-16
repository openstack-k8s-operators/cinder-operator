# Using a local registry

During the development process there will be times when we may want to build
and test container images that we are manually building, for example with `make
docker-build`.

To use these container images we'll need to have them available in a registry
that is accessible from our OpenShift cluster.

One possibility is using our personal account in a public registry such as
quay.io, but this forces us to have access to Internet and will be slower due
to the pushing and pulling of images to the registry.

Another possibility is running a local *toy* registry, which may be a bit more
cumbersome the first time, but will make development with container images
faster.

We can easily deploy a registry in the host computer from where we'll be
running the CRC VM with:

```sh
podman run -d -p 5000:5000 --name registry registry:2
```

With this we'll be able to push container images to that local registry using
any of the host IP addresses and port `5000`.

Since this is a *toy* registry it doesn't use HTTPS but HTTP instead, so there
are some consideration to be taken into account.

## Allowing the registry

This insecure registry must be allowed in out OpenShift deployment.

We do this by changing the `/etc/containers/registries.conf` file in our
OpenShift nodes.

When using CRC we can easily do this if we leverage our helper functions.

First we import our helper functions.

```sh
source ~/cinder-operator/hack/dev/helpers.sh
```

Now we allow the insecure registry using the IP address that CRC creates on the
host during deployment (`192.168.130.1`) and then restart the `crio` engine
and the `kubelet` service:

```sh
crc_ssh "echo -e '\n[[registry]]\nlocation = \"192.168.130.1:5000\"\ninsecure = true' | sudo tee -a /etc/containers/registries.conf"

crc_ssh sudo systemctl restart crio kubelet
```

Now OpenShift will be able to pull images from our local HTTP registry without
complain.

## Pushing images

Pushing containers to the registry will fail because the client won't be able
to do the TLS verification, so we need to disable it.

To disable TLS verification we must pass the `--tls-verify=false` argument on
the `push` call.

For example:

```sh
podman push --tls-verify=false 192.168.130.1:5000/cinder-operator:latest
```

In Cinder and some other `Makefile`s this can be achieved using the environmental
variable `VERIFY_TLS`:

```sh
export VERIFY_TLS=false
make docker-push
```

## OPM

When using the `opm` CLI we must let it know that we'll be using HTTP, which
can be done with the `--use-http` argument.

There are also some project's `Makefile` files, like Cinder's, that allow us to
use the environmental variable `USE_HTTP` to automatically pass the right
parameter to `opm`.

```sh
export USE_HTTP=true

make catalog-build
```
