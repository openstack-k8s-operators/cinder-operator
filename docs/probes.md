# Probes Defaults Configuration

The cinder-operator provides flexible health check configuration through
Kubernetes probes. This document explains the probe configuration, timing
calculations, and best practices.

## Overview

The operator supports three types of probes:
- **Liveness Probes** - Determines if a container needs to be restarted
- **Readiness Probes** - Determines if a container can receive traffic
- **Startup Probes** - Handles initial container startup before other probes begin

Cinder has two distinct service categories, each with its own probe strategy:

| Service       | Liveness | Readiness | Startup | Health Check Method         |
|---------------|----------|-----------|---------|-----------------------------|
| CinderAPI     | Yes      | Yes       | No      | HTTP `/healthcheck` endpoint|
| CinderScheduler | Yes    | No        | Yes     | Sidecar `healthcheck.py`    |
| CinderBackup  | Yes      | No        | Yes     | Sidecar `healthcheck.py`    |
| CinderVolume  | Yes      | No        | Yes     | Sidecar `healthcheck.py`    |

**CinderAPI** exposes an HTTP endpoint and serves external traffic, so it uses
readiness probes to control traffic routing. Startup probes are not applied
because the API process initializes quickly behind HAProxy/Apache.

**RPC worker processes** (scheduler, backup, volume) do not expose HTTP
endpoints. Instead, a dedicated sidecar container runs `healthcheck.py`, which
verifies the service state by checking the cinder service report timestamps
against the configuration files. These workers use startup probes to handle
potentially slow initialization (e.g., connecting to storage backends), but
do not need readiness probes since they do not receive direct traffic.

## Default Probe Configuration

### CinderAPI: Dynamic Timing Based on APITimeout

The CinderAPI probe timings are dynamically calculated based on the `apiTimeout`
parameter to ensure alignment with HAProxy and Apache timeout settings:

```go
period = floor(apiTimeout / failureCount)   // failureCount = 3
timeout = 5                                  // fixed value
startupPeriod = max(5, period / 2)           // faster startup detection
```

### RPC Workers: Timing Based on ServiceDownTime

The RPC worker probe timings (cinder-scheduler, cinder-backup, cinder-volume)
are derived from the upstream cinder `service_down_time` configuration value,
which defaults to 60 seconds. This value represents the maximum time since
the last service check-in before a service is considered down.

A dedicated `serviceDownTime` field is not currently exposed in the operator's
API, and we rely on the
[default provided by cinder](https://opendev.org/openstack/cinder/src/branch/master/cinder/common/config.py#L121)
to compute the default probe timings. Further tuning can be performed via the
dedicated override interface.

```go
period = floor(serviceDownTime / failureCount) // failureCount = 3
timeout = 5                                     // fixed value
startupPeriod = max(5, period / 2)              // faster startup detection
```

### Default Values (apiTimeout=60s, serviceDownTime=60s)

#### CinderAPI

| Probe Type | Timeout | Period | Initial Delay | Failure Threshold | Total Unhealthy Time |
|------------|---------|--------|---------------|-------------------|----------------------|
| Liveness   | 5s      | 20s    | 5s            | 3 (default)       | 60s (3 x 20s)       |
| Readiness  | 5s      | 20s    | 5s            | 3 (default)       | 60s (3 x 20s)       |

#### RPC Workers (Scheduler, Backup, Volume)

| Probe Type | Timeout | Period | Initial Delay | Failure Threshold | Total Unhealthy Time |
|------------|---------|--------|---------------|-------------------|----------------------|
| Liveness   | 5s      | 20s    | 15s           | 3 (default)       | 60s (3 x 20s)       |
| Startup    | 5s      | 10s    | 20s           | 12                | 120s (12 x 10s)     |

> **Note**: RPC workers use a longer `initialDelaySeconds` (15s for liveness,
> `period` for startup) compared to CinderAPI (5s), because RPC processes need
> additional time to establish AMQP connections and register with the message bus.

### CinderAPI Timing Examples for Different APITimeout Values

| apiTimeout | Period | Startup Period | Max Startup Time | Total Unhealthy Time |
|-----------|--------|----------------|------------------|----------------------|
| 30s       | 10s    | 5s (min)       | 60s (1 min)      | 30s (3 x 10s)       |
| 60s       | 20s    | 10s            | 120s (2 min)     | 60s (3 x 20s)       |
| 120s      | 40s    | 20s            | 240s (4 min)     | 120s (3 x 40s)      |
| 300s      | 100s   | 50s            | 600s (10 min)    | 300s (3 x 100s)     |

> **Note**: Unlike the glance-operator, the cinder-operator does not cap the
> `startupPeriod` at 10 seconds. For large `apiTimeout` values, the startup
> detection window grows proportionally. Since CinderAPI does not currently
> apply startup probes in its StatefulSet, this only affects RPC workers when
> their defaults are manually overridden. The RPC worker timing remains
> constant since `serviceDownTime` is not exposed as a configurable parameter.

## Design Decisions

### 1. Fixed Timeout (5 seconds)

Unlike some operators that compute timeout as a percentage of period, the
cinder-operator uses a fixed 5-second timeout for all probes. This provides a
consistent and predictable health check window regardless of the `apiTimeout`
value, and avoids unnecessarily long probe timeouts that could delay failure
detection.

### 2. Period = APITimeout / FailureCount

The probe period is derived by dividing `apiTimeout` (or `serviceDownTime`) by
the failure count (3). This ensures the total unhealthy time aligns exactly with
the configured timeout:

`Total Unhealthy Time = periodSeconds x failureThreshold = (apiTimeout / 3) x 3 = apiTimeout`

This means a pod is marked unhealthy at exactly the `apiTimeout` boundary,
providing a tight health checking window that detects failures in sync with
the HAProxy/Apache timeout configuration.

### 3. Sidecar-Based Probes for RPC Workers

RPC worker processes (cinder-volume, cinder-scheduler, cinder-backup) do not
expose HTTP endpoints. Instead, the operator deploys a sidecar container
running `healthcheck.py` that:

1. Reads the cinder configuration from `/etc/cinder/cinder.conf.d`
2. Queries the cinder database for the service's last check-in timestamp
3. Compares against `service_down_time` to determine health
4. Exposes the result on port 8080 for Kubernetes probes to query

This approach checks actual service health rather than merely verifying process
liveness, catching scenarios where the process is running but unable to
communicate with the message bus.

### 4. No Readiness Probes for RPC Workers

RPC workers communicate exclusively via the message bus (RabbitMQ) and never
receive direct HTTP traffic. Readiness probes control whether a pod receives
traffic from Kubernetes Services, which is not relevant for these workers.
Liveness probes ensure the process is restarted if it becomes unresponsive,
while startup probes handle the initialization window.

### 5. No Startup Probes for CinderAPI

CinderAPI starts behind HAProxy and Apache, which handle connection queuing
during initialization. The readiness probe already prevents traffic from
reaching the pod before it is ready to serve, making a separate startup probe
unnecessary. The liveness probe uses a short `initialDelaySeconds` (5s) since
the WSGI process initializes quickly.

## Customizing Probes

### CinderAPI CR Override

You can override default probe settings for the CinderAPI service:

```yaml
apiVersion: cinder.openstack.org/v1beta1
kind: CinderAPI
metadata:
  name: cinder-api
spec:
  apiTimeout: 120
  override:
    probes:
      livenessProbes:
        path: "/healthcheck"
        initialDelaySeconds: 10
        timeoutSeconds: 30
        periodSeconds: 40
        failureThreshold: 5
      readinessProbes:
        path: "/healthcheck"
        initialDelaySeconds: 10
        timeoutSeconds: 30
        periodSeconds: 40
        failureThreshold: 3
```

All probe overrides are configured through the top-level Cinder CR, which
propagates settings to the individual sub-services:

```yaml
apiVersion: cinder.openstack.org/v1beta1
kind: Cinder
metadata:
  name: cinder
spec:
  apiTimeout: 120
  cinderAPI:
    override:
      probes:
        livenessProbes:
          path: "/healthcheck"
          timeoutSeconds: 30
          periodSeconds: 40
  cinderScheduler:
    override:
      probes:
        livenessProbes:
          timeoutSeconds: 10
          periodSeconds: 25
        startupProbes:
          timeoutSeconds: 10
          periodSeconds: 5
          failureThreshold: 20
  cinderVolumes:
    volume1:
      override:
        probes:
          livenessProbes:
            timeoutSeconds: 10
            periodSeconds: 25
          startupProbes:
            timeoutSeconds: 10
            periodSeconds: 5
            failureThreshold: 20
  cinderBackups:
    backup1:
      override:
        probes:
          livenessProbes:
            timeoutSeconds: 10
            periodSeconds: 25
```

### Field-Level Overrides

Users can customize individual probe parameters while other settings retain
their computed defaults. For example, to only increase the failure threshold
on a CinderVolume instance:

```yaml
cinderVolumes:
  ceph-backend:
    override:
      probes:
        startupProbes:
          failureThreshold: 24
```

This doubles the startup detection window (from 120s to 240s) without changing
any other probe timing, which can be useful for backends that require additional
initialization time.

## TLS Considerations

When TLS is enabled for the CinderAPI endpoints:
- Probe scheme automatically switches to `HTTPS`
- No configuration changes needed

The operator automatically configures the correct scheme based on the TLS
settings:

```go
if instance.Spec.TLS.API.Enabled(service.EndpointPublic) {
    scheme = corev1.URISchemeHTTPS
}
```

RPC workers are not affected by TLS settings since their probe sidecar
communicates over localhost on port 8080 using plain HTTP.

## Best Practices

### 1. Align apiTimeout with Expected Response Times

Set `apiTimeout` based on your slowest expected Cinder API operation:
- Standard deployments: 60s (default)
- Large volume operations: 120-300s
- High-latency storage backends: 180-300s

Changing `apiTimeout` automatically adjusts CinderAPI probe timings. RPC worker
probe timings are not affected by `apiTimeout`.

### 2. Tune RPC Worker Probes for Slow Backends

Storage backends with slow initialization (e.g., establishing iSCSI sessions,
connecting to remote Ceph clusters) may require adjusted startup probes:

```yaml
cinderVolumes:
  slow-backend:
    override:
      probes:
        startupProbes:
          failureThreshold: 24  # 240s max startup time
          periodSeconds: 15
```

### 3. Monitor Probe Failures

Watch for probe failure patterns:

```bash
# Check probe failures in pod events
oc describe pod cinder-volume-vol1-0 -n openstack

# View probe timing in container spec
oc get pod cinder-api-0 -n openstack -o jsonpath='{.spec.containers[1].livenessProbe}'
```

### 4. Understand the Sidecar Probe Architecture

For RPC workers, remember that the probes target the sidecar `probe` container,
not the main service container. If probe failures occur, check both containers:

```bash
# Check main container logs
oc logs cinder-volume-vol1-0 -c cinder-volume -n openstack

# Check probe sidecar logs
oc logs cinder-volume-vol1-0 -c probe -n openstack
```

## References

- [Kubernetes Probe Documentation](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [lib-common Probes Module](https://github.com/openstack-k8s-operators/lib-common/tree/main/modules/common/probes)
- [Cinder ServiceDownTime Configuration](https://opendev.org/openstack/cinder/src/branch/master/cinder/common/config.py#L121)
- [CinderAPI CRD Reference](../api/v1beta1/cinderapi_types.go)
- [Probes Defaults Implementation](../internal/cinder/funcs.go)
