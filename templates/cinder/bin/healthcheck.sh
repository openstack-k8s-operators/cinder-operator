#!/bin//bash
#
# Copyright 2022 Red Hat Inc.
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

#
# Used to check health of scheduler, backup and volume services.  API
# comes with its own health check endpoint and does not need this.  Of
# course, if the API isn't healthy, this check won't be either because
# it ultimately relies on being able to contact the API.
#

set -ex

CINDER_SERVICE=$1

if [ -z "$CINDER_SERVICE" ]; then
    printf "No service provided"
    exit 1
fi

CONF_FILE=/etc/cinder/cinder.conf
export OS_AUTH_URL=$(crudini --get ${CONF_FILE} keystone_authtoken auth_url)
export OS_AUTH_TYPE=$(crudini --get ${CONF_FILE} keystone_authtoken auth_type)
export OS_USERNAME=$(crudini --get ${CONF_FILE} keystone_authtoken username)
export OS_PASSWORD=$(crudini --get ${CONF_FILE} keystone_authtoken password)
export OS_PROJECT_NAME=$(crudini --get ${CONF_FILE} keystone_authtoken project_name)
export OS_PROJECT_DOMAIN_NAME=$(crudini --get ${CONF_FILE} keystone_authtoken project_domain_name)
export OS_USER_DOMAIN_NAME=$(crudini --get ${CONF_FILE} keystone_authtoken user_domain_name)

CINDER_STATE=$(cinder service-list --binary ${CINDER_SERVICE} | grep ${CINDER_SERVICE} | awk {'print $10'})

if [ "$CINDER_STATE" != "up" ]; then
    printf "${CINDER_SERVICE} is not healthy"
    exit 1
fi

exit 0
