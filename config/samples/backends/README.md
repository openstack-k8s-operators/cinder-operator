# Cinder Backend Samples

In each directory there is a `backend.yaml` file containing an overlay for the
`OpenStackControlPlane` with just the storage related information.

Each backend has some prerequirements that will be listed in the `backend.yaml`
file.  These can range from having to replace the storage system's address and
credentials in a different yaml file, to having to create secrets, to having to
deploy the backend like in the `LVM` case.

Once the OpenStack operators are running in your OpenShift cluster and
the secret `osp-secret` is present, one can deploy OpenStack with a
specific storage backend with single command.  For example for Ceph we can do:
`oc kustomize ceph | oc apply -f -`.

The result of the `oc kustomize ceph` command is a complete
`OpenStackControlPlane` manifest, and we can see its contents by redirecting it
to a file or just piping it to `less`: `oc kustomize ceph | less`.

Creating the basic secret that our samples require can be done using the
`install_yamls` target called `input`.

A complete example when we already have CRC running would be:

```
$ cd install_yamls
$ make ceph TIMEOUT=90
$ make crc_storage openstack input
$ cd ../cinder-operator
$ oc kustomize config/samples/backends/ceph | oc apply -f -
```
