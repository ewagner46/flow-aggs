# Approach

The REST API sits in front of a `sqlite` store where results are
aggregated with existing entries prior to storage. Since the data can only be
queried in aggregate and the aggregation is well known at data creation time, we
aggregate the data prior to storage without losing any required fidelity.

When documents are created by `POST` requests, the new records are sent over the
a channel to a goroutine that then updates the sqlite store with the new
documents. Sqlite is selected for this demo application, but could be replaced
with another SQL database that supports locking by row. As it is, since `sqlite`
can only support a single writer at one time, the benefit of multiple writers is
limited. The writer thread does pre-aggregate all waiting writes (up to a
maximum configurable `batch_size`).

With a more parallizable SQL database, we could spawn a pool of writers to read
from a pool of channels. The channel and worker for a given update can be assigned 
as `hour % num_write_workers`.  This ensures all updates that would be batched
are processed by the same goroutine to reduce the number of pending transactions
on a given row.

Reads and writes are not serialized, so it is possible to read slightly stale
data if the latest writes have not yet been applied.  While we could enforce
all known pending writes be applied before reads are processed, this still
doesn't preclude the possibility of stale data since there may be writes in
flight over the network, so it is probably not worth the performance tradeoff.

The REST API is implemented with the Gin framework, which supplies REST
operations.
