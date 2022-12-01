# dfanout

`dfanout` allows you to manage request paths with a lot of flexibility. 
Introduce new fanouts, or chain different fanouts to dynamically manage
your HTTP or RPC endpoints with minimal or no code changes. 
`dfanout` provides observability
into the health and performance of request paths out of the box.
Perform data migrations with confidence, or repurpose your request paths
with minimal efforts.

## Use cases

* Data migrations where you want to implement dual reads and/or dual writes with minimal or no code changes.
* Populating new datastores or message queues without code changes or new hooks.
* Improve the auditing capabilities for read and write paths with minimal or no code changes.
* Managing retry or security policies of your read and write path with minimal or no code changes.

## Limits

* Limited to HTTP endpoints (e.g. REST endpoints) for now. Native gRPC and Twirp support is coming in the future.
* No transactional capabilities, e.g. no rollbacks on partial failures.
* Endpoints should share the request and response contract.
