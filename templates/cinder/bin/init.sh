#!/bin/bash
#
# Copyright 2020 Red Hat Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License. You may obtain
# a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
# License for the specific language governing permissions and limitations
# under the License.
set -ex

# This script generates cinder.conf.d files and copies the result to the
# ephemeral /var/lib/config-data/merged volume.
#
# Secrets are obtained from ENV variables.
export DB=${DatabaseName:-"cinder"}
export DBHOST=${DatabaseHost:?"Please specify a DatabaseHost variable."}
export DBUSER=${DatabaseUser:-"cinder"}
export DBPASSWORD=${DatabasePassword:?"Please specify a DatabasePassword variable."}
export PASSWORD=${CinderPassword:?"Please specify a CinderPassword variable."}
export TRANSPORTURL=${TransportURL:-""}

DEFAULT_DIR=/var/lib/config-data/default
CUSTOM_DIR=/var/lib/config-data/custom
MERGED_DIR=/var/lib/config-data/merged
SVC_CFG=/etc/cinder/cinder.conf
SVC_CFG_MERGED=${MERGED_DIR}/cinder.conf
SVC_CFG_MERGED_DIR=${MERGED_DIR}/cinder.conf.d

mkdir -p ${SVC_CFG_MERGED_DIR}

cp ${DEFAULT_DIR}/* ${MERGED_DIR}

# Save the default service config from container image as cinder.conf.sample,
# and create a small cinder.conf file that directs people to files in
# cinder.conf.d.
cp -a ${SVC_CFG} ${SVC_CFG_MERGED}.sample
cat <<EOF > ${SVC_CFG_MERGED}
# Service configuration snippets are stored in the cinder.conf.d subdirectory.
EOF

cp ${DEFAULT_DIR}/cinder.conf ${SVC_CFG_MERGED_DIR}/00-default.conf

# Generate 01-secrets.conf
SVC_CFG_SECRETS=${SVC_CFG_MERGED_DIR}/01-secrets.conf
if [ -n "$TRANSPORTURL" ]; then
    cat <<EOF > ${SVC_CFG_SECRETS}
[DEFAULT]
transport_url = ${TRANSPORTURL}

EOF
fi

# TODO: service token
cat <<EOF >> ${SVC_CFG_SECRETS}
[database]
connection = mysql+pymysql://${DBUSER}:${DBPASSWORD}@${DBHOST}/${DB}

[keystone_authtoken]
password = ${PASSWORD}

[nova]
password = ${PASSWORD}
EOF

if [ -f ${DEFAULT_DIR}/custom.conf ]; then
    cp ${DEFAULT_DIR}/custom.conf ${SVC_CFG_MERGED_DIR}/02-global.conf
fi

if [ -f ${CUSTOM_DIR}/custom.conf ]; then
    cp ${CUSTOM_DIR}/custom.conf ${SVC_CFG_MERGED_DIR}/03-service.conf
fi

# Probes cannot run kolla_set_configs because it uses the 'cinder' uid
# and gid and doesn't have permission to make files be owned by root.
# This means the probe must use files in the "merged" location, and the
# files must be readable by 'cinder'.
chown -R :cinder ${SVC_CFG_MERGED_DIR}
