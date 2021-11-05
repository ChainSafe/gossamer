# Pprof

There is a built pprof server to faciliate profiling the program.

Note it does not affect performance unless the server is queried.

We assume Gossamer runs on `localhost` and the Pprof server is listening
on the default `6060` port.

You need to have [Go](https://golang.org/dl/) installed to profile the program.

The easiest way to visualize profiling data is through your browser.

The following commands are available and will show the result at [http://localhost:8000](http://localhost:8000):

```sh
# Check the heap
go tool pprof -http=localhost:8000 http://localhost:6060/debug/pprof/heap
```
