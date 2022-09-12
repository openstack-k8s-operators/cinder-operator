#!/usr/bin/env python3
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

# Trivial HTTP server to check health of scheduler, backup and volume services.
# Cinder-API hast its own health check endpoint and does not need this.
#
# The only check this server currently does is using the heartbeat in the
# database service table, accessing the DB directly here using the cinder.conf
# configuration options.
#
# The benefit of accessing the DB directly is that it doesn't depend on the
# Cinder-API service being up and we can also differentiate between the
# container not having a connection to the DB and the cinder service not doing
# the heartbeats.
#
# For volume services all enabled backends must be up to return 200, so it is
# recommended to use a different pod for each backend to avoid one backend
# affecting others.
#
# Requires the name of the service as the first argument (volume, backup,
# scheduler) and optionally a second argument with the location of the
# configuration file (defaults to /etc/cinder/cinder.conf)

from http import server
import signal
import sys
import time
import threading

from oslo_config import cfg

from cinder import context
from cinder.volume import configuration as vol_conf
from cinder import objects


HOSTNAME = ''
SERVER_PORT = 8080
CONF = cfg.CONF
BINARIES = ('volume', 'backup', 'scheduler')


class HeartbeatServer(server.BaseHTTPRequestHandler):
    @classmethod
    def initialize_class(cls, binary):
        """Calculate and initialize constants"""
        cls.binary = 'cinder-' + binary
        cls.ctxt = context.get_admin_context()

        if binary != 'volume':
            services_filters = [{'host': CONF.host}]
        else:
            backend_opts = [
                cfg.StrOpt('backend_host'),
                cfg.StrOpt('backend_availability_zone', default=None),
            ]
            services_filters = []
            for backend in CONF.enabled_backends:
                # This instance gets from backend section and falls back to
                # backend_defaults if it's not present.
                conf = vol_conf.BackendGroupConfiguration(backend_opts,
                                                          backend)
                host = "%s@%s" % (conf.backend_host or CONF.host, backend)

                # We also want to set cluster to None on empty strings, and we
                # ignore leading and trailing spaces.
                cluster = CONF.cluster and CONF.cluster.strip()
                cluster = (cluster or None) and f'{cluster}@{backend}'

                az = (conf.backend_availability_zone or
                      CONF.storage_availability_zone)
                services_filters.append({'host': host,
                                         'cluster': cluster,
                                         'availability_zone': az})

        # Store the filters used to find out services. For volume it's a list
        # of dict/filters, and for others is a single element with the filters.
        cls.services_filters = services_filters

    @staticmethod
    def check_service(services, **filters):
        # Check services DB connectivity by looking at their DB heartbeat
        our_services = [service for service in services
                        if all(getattr(service, key, None) == value
                               for key, value in filters.items())]
        if not our_services:
            raise Exception('Not found', f'Service not found with: {filters}')

        num_services = len(our_services)
        if num_services != 1:
            print(f'Too many services found! {num_services}')

        ok = all(service.is_up for service in our_services)
        if not ok:
            raise Exception('Service error', 'Service is not UP')

        # TODO: Add a RabbitMQ connectivity check, for example using the
        #       logging get feature

    def do_GET(self):
        try:
            services = objects.ServiceList.get_all_by_binary(self.ctxt,
                                                             self.binary)
        except Exception as exc:
            return self.send_error(500,
                                   'DB access error',
                                   f'Failed to connect to the database: {exc}')

        for service_filters in self.services_filters:
            try:
                self.check_service(services, **service_filters)
            except Exception as exc:
                return self.send_error(500, exc.args[0], exc.args[1])

        self.send_response(200)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        self.wfile.write('<html><body>OK</body></html>'.encode('utf-8'))


def get_stopper(server):
    def stopper(signal_number=None, frame=None):
        print("Stopping server.")
        server.shutdown()
        server.server_close()
        print("Server stopped.")
        sys.exit(0)
    return stopper


if __name__ == "__main__":
    if 3 < len(sys.argv) < 2 or sys.argv[1] not in BINARIES:
        print('Healthcheck requires the binary type as argument (one of: '
              + ', '.join(BINARIES) +
              ') and optionally the location of the config file.')
        sys.exit(1)
    binary = sys.argv[1]

    cfg_file = sys.argv[2] if len(sys.argv) == 3 else '/etc/cinder/cinder.conf'

    objects.register_all()

    # Due to our imports cinder.common.config.global_opts are registered in
    # cinder/volume/__init__.py bringing in the following config options we
    # want: host, storage_availability_zone, service_down_time, cluster,
    # enabled_backends
    # Initialize Oslo Config
    CONF.register_opt(cfg.StrOpt('cluster', default=None))
    CONF(['--config-file', cfg_file], project='cinder')

    HeartbeatServer.initialize_class(binary)

    webServer = server.HTTPServer((HOSTNAME, SERVER_PORT), HeartbeatServer)
    stop = get_stopper(webServer)

    # Need to run the server on a different thread because its shutdown method
    # will block if called from the same thread, and the signal handler must be
    # on the main thread in Python.
    thread = threading.Thread(target=webServer.serve_forever)
    thread.daemon = True
    thread.start()
    print(f"Cinder Healthcheck Server started http://{HOSTNAME}:{SERVER_PORT}")
    signal.signal(signal.SIGTERM, stop)

    try:
        while True:
            time.sleep(60)
    except KeyboardInterrupt:
        pass
    finally:
        stop()
