# Approach

The REST API sits in front of an embedded `postgres` store. Since the data can
only be queried in aggregate and the aggregation is well known at data creation
time, I aggregate the data prior to storage without losing any required
fidelity.

All API endpoints are parallelized by the `gin` go library.  The `sqlx` handler
offers a pool of database connections for parallelized reads and writes. This
handler also retries database operations up to 10 times if there is a transient
type of failure.

`GET` requests query directly from the `sqlx` handler.

`POST` requests write to a pool of channels and corresponding write workers.

For additional write throughput, I added batching logic to handle cases where we
receive a large burst of messages for a given `src_app + dst_app + vpc_id + hour`.
Rather than issue a large number of updates to the same database row, these are
first aggregated into a single update before they are then combined with the
stored row in the database.

This bursty situation can arise, for example, if the machine in the field that
was serving that flow was temporarily partitioned from, or otherwise unable to
send updates to, the `flowagg` server. When it reconnects, it then submits a
large burst for each of the flows it continued to serve in that time. Due to the
nature of connectivity and availability errors, it's likely that more than one
of these machines in the field was disconnected from the `flowagg` server, so it's
possible for substantial bursts to occur.  I would like to gracefully absorb these.

To handle multiples for a given key and to allow additional write
parallelization, I added a pool of channels and aggregators/writers.
The channel and worker for a given update can be assigned as 
`hash(src_app + dst_app + vpc_id + hour) % num_write_workers`. This ensures
all updates that would be batched are processed by the same goroutine to greatly
reduce the number of pending transactions on a given row, while still providing
a reasonably even distribution among the workers with the degrees of freedom we
have available to us.

Reads and writes are not serialized, so it is possible to read slightly stale
data if the latest writes have not yet been applied.  While we could enforce
all known pending writes be applied before reads are processed,
we can still end up with stale reads since there may be writes in
flight over the network. Because of this, I decided that enforcing this ordering
is probably not worth the performance trade-off.

Unfortunately, I ran out of time to write unit tests with Go and was only able
to test using shell scripts invoking curl commands, but I would have preferred
to submit this with a unit test suite.

# Scale and Availability

While this implementation would be able to utilize a single machine well,
for larger scale and better availability, it should be deployed
across multiple machines in multiple regions. As soon we are deployed in
multiple machines, we must swap the embedded database implementation for
one or more remote databases. To continue to benefit from batching,
and improve responsiveness, we would ideally prefer to direct flow updates
from machines in the field to the closest region and then prefer to direct
updates from specific machines in that region to the same backend server,
as long as that server is not too loaded. Off the shelf proxy/load balancer
software can monitor request response time and act accordingly after preferring 
the initial results of backend server selection based on a hash of the sender IP
and port.  Despite this ordered preference, we should be able to fall back to
other machines and other regions.


