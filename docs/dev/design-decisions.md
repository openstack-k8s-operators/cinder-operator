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
