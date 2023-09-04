# Pure Storage Cinder Volume container for Antelope deployments

FROM quay.io/podified-antelope-centos9/openstack-cinder-volume:current-podified

ARG min_pureclient_version=1.17.0

LABEL maintainer="Pure Storage" \
      description="OpenStack cinder-volume Pure Storage" \
      summary="OpenStack cinder-volume Pure Storage" \
      name="geguileo/podified-antelope-openstack-cinder-volume-pure:current-podified" \
      vendor="Pure Storage" \
      min_pureclient="${pureclient_version}"

# Switch to root to install packages
USER root

RUN curl "https://bootstrap.pypa.io/get-pip.py" -o "get-pip.py" && \
    python3 get-pip.py && \
    pip3 install "purestorage>=${min_pureclient_version}" && \
    rm -f get-pip.py

# Switch to cinder user
USER cinder
