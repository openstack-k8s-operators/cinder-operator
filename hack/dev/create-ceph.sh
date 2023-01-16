#!/bin/env bash
set -e
LOCATION=$(realpath `dirname -- $BASH_SOURCE[0]`)
SECRET_NAME=${1:-ceph-client-files}
OPENSTACK_YAMLS="$LOCATION/openstack-ceph.yaml $LOCATION/openstack-lvm-ceph.yaml"

# Change Ceph default features (if we want to attach using krbd)
# echo -e "\nrbd default features = 3" | sudo tee -a /etc/ceph/ceph.conf
if sudo podman container exists ceph; then
  echo 'Ceph container exists reusing it'
else
  echo 'Starting ceph Pacific demo cluster'
  sudo podman run -d --name ceph --net=host -e MON_IP=192.168.130.1 -e CEPH_PUBLIC_NETWORK=0.0.0.0/0 -e DEMO_DAEMONS='osd' quay.io/ceph/daemon:latest-pacific demo
fi

echo 'Waiting for Ceph config files to be created'
until sudo podman exec -t ceph test -e /etc/ceph/I_AM_A_DEMO
do
  echo -n .
  sleep 0.5
done
echo

TEMPDIR=`mktemp -d`
trap 'sudo rm -rf -- "$TEMPDIR"' EXIT
echo 'Copying Ceph config files from the container to $TEMPDIR'
sudo podman cp ceph:/etc/ceph/ceph.conf $TEMPDIR
sudo podman cp ceph:/etc/ceph/ceph.client.admin.keyring $TEMPDIR
sudo chown `whoami` $TEMPDIR/*

echo "Replacing openshift secret $SECRET_NAME"
oc delete secret $SECRET_NAME 2>/dev/null || true
oc create secret generic $SECRET_NAME --from-file=$TEMPDIR/ceph.conf --from-file=$TEMPDIR/ceph.client.admin.keyring

FSID=`grep fsid $TEMPDIR/ceph.conf | cut -d' ' -f 3`
for manifest in $OPENSTACK_YAMLS
do
  # echo "Replacing Ceph FSID in $OPENSTACK_YAML with $FSID"
  echo "Replacing Ceph FSID in $manifest with $FSID"
  sed -i "s/rbd_secret_uuid\\s*=.*/rbd_secret_uuid=$FSID/g" "$manifest"
done

sudo podman exec -it ceph bash -c 'ceph osd pool create volumes 4 && ceph osd pool application enable volumes rbd'
sudo podman exec -it ceph bash -c 'ceph osd pool create backups 4 && ceph osd pool application enable backups rbd'
sudo podman exec -it ceph bash -c 'ceph osd pool create images 4 && ceph osd pool application enable images rgw'
