#!/usr/bin/env bash

LOCATION=$(realpath `dirname -- $BASH_SOURCE[0]`)

mkdir -p ~/.config/openstack

if [[ ! -e ~/.config/openstack/clouds.yaml ]]; then
    cp $LOCATION/clouds.yaml ~/.config/openstack/clouds.yaml
fi

# Install OSC because some services no longer have individual clients
# This installs the cinderclient as well, which has feature parity with cinder
pip3 install openstackclient

export OS_CLOUD=default
export OS_PASSWORD=12345678

source $LOCATION/admin-rc
