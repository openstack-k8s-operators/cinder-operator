#!/bin/env bash
LOCATION=$(realpath `dirname -- $BASH_SOURCE[0]`)
sudo cp -R "${LOCATION}/ceph" /etc

# Change Ceph default features (if we want to attach using krbd)
# echo -e "\nrbd default features = 3" | sudo tee -a /etc/ceph/ceph.conf

echo 'Running ceph Pacific demo cluster'
sudo podman run -d --name ceph --net=host -v /etc/ceph:/etc/ceph:z -v /lib/modules:/lib/modules -e MON_IP=192.168.130.1 -e CEPH_PUBLIC_NETWORK=0.0.0.0/0 -e DEMO_DAEMONS='osd' quay.io/ceph/daemon:latest-pacific demo

sleep 3

sudo podman exec -it ceph bash -c 'ceph osd pool create volumes 4 && ceph osd pool application enable volumes rbd'
sudo podman exec -it ceph bash -c 'ceph osd pool create backups 4 && ceph osd pool application enable backups rbd'
sudo podman exec -it ceph bash -c 'ceph osd pool create images 4 && ceph osd pool application enable images rgw'
