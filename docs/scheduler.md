# Cinder Scheduler

## Replicas

The cinder scheduler follows an eventual consistency model, which has proven
problematic for deployments with storage backends close to reaching their full
capacity.  Having multiple scheduler services exacerbates this issue and also
makes it more difficult to do RCAs.

There are large production deployments running with a single cinder scheduler
service that have not reported any bottleneck in the operations that go through
the scheduler.

So given these 2 facts -issues with scheduler and single scheduler performance-
we decided to do some characterization of the scheduler.

Not all cinder operations go through the scheduler, and there are differences in
complexity between the ones that go through it.  For example it's not the same
thing doing a create volume operation than a create snapshot.

For the create volume operation a single scheduler seems to be able to handle
around 15.7 requests per second (63.7ms per request). This includes the
reception of the RabbitMQ RPC message from the scheduler's queue, the DB
operations, and sending the RPC call to the volume's queue in RabbitMQ, so our
DB and RabbitMQ speeds are a big factor here.  If we exclude the RabbitMQ
message handling and the DB access this goes down to less than 13ms per request.

At the time of this writing for the create snapshot operation, which doesn't
have DB requests, we only have the timing of the code without external services
calls, which is around 1ms.

Given this information it is our recommendation to run a single scheduler
(`replicas: 1`) as it seems to be performant enough for most deployments. For
deployments that require better performance we recommend careful monitoring of
available space in the backends to always enough space to avoid issues.

There is room for improvement in the scheduler code, and during these tests we
have already located a [DB read during the create volume operation that can be
removed](https://review.opendev.org/c/openstack/cinder/+/888535) (tests
presented in here were run with the read).

For a detailed explanation of the performed tests please refer to the [scheduler
performance test details](scheduler-perf/scheduler-perf.md)
