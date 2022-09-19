#!/bin/env bash
set -ev
set -x

LOCATION=$(realpath `dirname -- $BASH_SOURCE[0]`)
source "$LOCATION/helpers.sh"

# Enable iSCSI because it always fails to start for some reason, and just to be
# extra sure create the initiator name if it doesn't exist.
crc_ssh 'if [[ ! -e /etc/iscsi/initiatorname.iscsi ]]; then echo InitiatorName=`iscsi-iname` | sudo tee /etc/iscsi/initiatorname.iscsi; fi; if ! systemctl --no-pager status iscsid; then sudo systemctl restart iscsid; fi'

# Multipath failed to start because it doesn't have a configuration, create it
# and restart the service
crc_ssh 'if [[ ! -e /etc/multipath.conf ]]; then sudo mpathconf --enable --with_multipathd y --user_friendly_names n --find_multipaths y && sudo systemctl start multipathd; fi'

loopback_file="/var/home/core/cinder-volumes"
echo Creating $loopback_file in CRC VM
crc_ssh "if [[ ! -e $loopback_file ]]; then truncate -s 10G $loopback_file; fi"
crc_ssh "if ! sudo vgdisplay cinder-volumes; then sudo vgcreate cinder-volumes \`sudo losetup --show -f $loopback_file\` && sudo vgscan; fi"
