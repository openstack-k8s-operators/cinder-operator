

# PERSISTENT STORAGE GUIDE

The OpenStack on OpenShift Operators support multiple persistent storage types:
Volumes, Images, Shares, and Objects. Each of these types is handled by a
different OpenStack service, and each OpenStack services has its own operator.
These operators have some commonalities in their behavior and their data
structures in their CRDs, but due to their specific needs they also have some
differences.

This guide is intended as a source of contextual information on configuring and
deploying the OpenStack Block Storage service (`cinder`), though it may also
include some procedures.  The reader is expected to be familiar with basic
OpenShift or Kubernetes concepts and they won't be covered in this guide. The
reader is also expected to refer to the [user documentation](
https://openstack-k8s-operators.github.io/openstack-operator/) to be able to
configure and install a full OpenStack deployment.

---

> **🛈 NOTE:** For users familiar with previous OpenStack installer (TripleO),
> it is important to know that the new installation no longer uses TripleO or
> Tripleo-Heat-Templates (THT), it uses a completely new mechanism, so the
> abstraction layer between the installer configuration options and the
> individual services configuration options has been removed.

---

As mentioned before there are some commonalities among persistent storage
services and some differences, and we cover these in different documents. The
recommendation is to go over the commonalities documentation first and then the
specific `cinder` guide.

- [General storage concepts](commonalities.md)
- [Cinder configuration guide](cinder.md)
