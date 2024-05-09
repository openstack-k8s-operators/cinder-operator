# Expose Cinder API to an isolated network

The Cinder spec can be used to configure Cinder API to register e.g.
the internal endpoint to an isolated network. MetalLB is used for this
scenario.

As a pre requisite, MetalLB needs to be installed and worker nodes
prepared to work as MetalLB nodes to serve the LoadBalancer service.

In this example the following MetalLB IPAddressPool is used:

```
---
apiVersion: metallb.io/v1beta1
kind: IPAddressPool
metadata:
  name: osp-internalapi
  namespace: metallb-system
spec:
  addresses:
  - 172.17.0.200-172.17.0.210
  autoAssign: false
```

The following represents an example of Cinder resource that can be used
to trigger the service deployment, and have the cinderAPI endpoint
registerd as a MetalLB service using the IPAddressPool `osp-internal`,
request to use the IP `172.17.0.202` as the VIP and the IP is shared with
other services.

```
apiVersion: cinder.openstack.org/v1beta1
kind: Cinder
metadata:
  name: cinder
spec:
  ...
  cinderAPI:
    ...
    override:
      service:
        internal:
          metadata:
            annotations:
              metallb.universe.tf/address-pool: osp-internalapi
              metallb.universe.tf/allow-shared-ip: internalapi
              metallb.universe.tf/loadBalancerIPs: 172.17.0.202
          spec:
            type: LoadBalancer
    ...
...
```

The internal cinder endpoint gets registered with its service name. This
service name needs to resolve to the `LoadBalancerIP` on the isolated network
either by DNS or via /etc/hosts:

```
# openstack endpoint list -c 'Service Name' -c Interface -c URL --service cinderv3
+--------------+-----------+------------------------------------------------------------------+
| Service Name | Interface | URL                                                              |
+--------------+-----------+------------------------------------------------------------------+
| cinderv3     | internal  | http://cinder-internal.openstack.svc:8776/v3                     |
| cinderv3     | public    | http://cinder-public-openstack.apps.ostest.test.metalkube.org/v3 |
+--------------+-----------+------------------------------------------------------------------+
```
