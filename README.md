# KVStor

KVStor is a simple in-memory key-value storage.

To start the KVStor instance run: 

`go run main/kvstor.go`

After it KVStor start a server listening on port :8666

To connect to server type: `telnet localhost 8666`

To set some key in the command line prompt type:

`set somekey somevalue [optional TTL in seconds]`

To get this key:

`get somekey`

To delete:

`delete somekey`

