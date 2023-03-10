
### Unhandled Command error loop
...

Temporal Server Version: 1.18

#### start the worker
```
cd worker
go run main.go
```
#### send a request
the requests are "proxy workflow requests" where a workflow is started and sends a singal to the task-queue.
```
cd starter
go run main.go
```

#### last step
Kill the worker `ctrl-c` and start it again `go run main.go` and send a request again to the workflow by running the starter. this will cause the workflow to "panic"
