# HPE 3PAR Cinder Volume container for Antelope deployments
FROM quay.io/podified-antelope-centos9/openstack-cinder-volume:current-podified

ARG min_3parclient_version=4.2.10
MAINTAINER HPE

LABEL name="geguileo/podified-antelope-openstack-cinder-volume-3par:current-podified" \
      maintainer="HPE" \
      vendor="HPE" \
      version="2.0" \
      release="Antelope" \
      summary="OpenStack cinder-volume HPE" \
      description="Cinder plugin for HPE 3PAR"

# switch to root and install a custom RPM, etc.
USER "root"

RUN curl "https://bootstrap.pypa.io/get-pip.py" -o "get-pip.py" && \
    python3 get-pip.py && \
    pip3 install "python-3parclient>=${min_3parclient_version}" && \
    rm -f get-pip.py


# switch the container back to the default user
USER "cinder"
