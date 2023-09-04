# Design decisions

We have agreed on a very basic set of principles we would like to follow during
the development of the OpenStack Operators:

- OpenShift is the only intended Container Orchestration system we aim to
  support, and code can depend on OpenShift only available features.

- Should use configuration snippets for the system administrators (or the
  meta-operator) to provide the service specific configuration. This will
  reduce the domain specific knowledge required, making it easy for anyone that
  knows how to configure the specific service to use the operator.

- Should aim to follow the Kubernetes [Operator pattern](
  https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

- Must use [Controllers](
  https://kubernetes.io/docs/concepts/architecture/controller/) which provide
  a reconcile function responsible for synchronizing resources until the
  desired state is reached on the cluster.

- Intended to be deployed via OLM [Operator Lifecycle Manager](
  https://github.com/operator-framework/operator-lifecycle-manager).

- Directory structure should try to adhere to the [Standard Go Project Layout](
https://github.com/golang-standards/project-layout) whenever possible.

## Transport protocol considerations

OpenShift deployments are different from TripleO deployments.  In TripleO we
run `iscsid` and `multipathd` daemons in containers and their client side
commands are in sync (version-wise) in the service containers.  It is also
required that there are no other daemons running in the host or other
containers.

In the OpenShift world things are different. A big difference is that the
underlying operating system is CoreOS, and as an immutable OS the system cannot
be modified by users and it's not under our control, so we cannot ensure that
the `iscsid` and `multipathd` services OpenStack runs in its containers are
compatible with the operating system.

The solution is running `iscsid` and `multipathd` on the host, which also
allows these damons to be shared between the different things that may need it:
OpenShift itself, CSI plugins, and in this case also OpenStack services.

The way to use the host iscsi and multipath services is by exiting the pod
namespace and running the commands on the host namespace using `nsenter`.  Even
if this may look nasty/weird, this is the recommended way by the storage
OpenShift team and what CSI plugins do.

The fix consists in replacing the iscsiadm, multipathd, multipath, and scsi_id
commands with a simple script (`run-on-host`) that calls `nsenter`.  The
replacements of the binary is done using the kolla config file.

To be able to use `nsenter` the container needs to be run as privileged and the
pod needs to share the PID namespace with the host.
