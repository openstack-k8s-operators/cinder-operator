#!/bin//bash
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

export DatabasePassword=${DatabasePassword:?"Please specify a DatabasePassword variable."}
export DatabaseHost=${DatabaseHost:?"Please specify a DatabaseHost variable."}
export Database=${Database:-"cinder"}
export DatabaseConnection="mysql+pymysql://$Database:$DatabasePassword@$DatabaseHost/$Database"

# Bootstrap and exit if KOLLA_BOOTSTRAP variable is set. This catches all cases
# of the KOLLA_BOOTSTRAP variable being set, including empty.
if [[ "${!KOLLA_BOOTSTRAP[@]}" ]]; then
    cinder-manage db sync
    exit 0
fi

if [[ "${!KOLLA_OSM[@]}" ]]; then
    cinder-manage db online_data_migrations
    exit 0
fi
