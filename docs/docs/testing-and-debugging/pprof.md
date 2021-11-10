---
layout: default
title: Pprof
permalink: /testing-and-debugging/pprof
---

## Pprof

There is a built-in pprof server to faciliate profiling the program.
You can enable it with the flag `--pprofserver` or by modifying the TOML configuration file.

Note it does not affect performance unless the server is queried.

We assume Gossamer runs on `localhost` and the Pprof server is listening
on the default `6060` port. You can configure the Pprof server listening address with the pprof TOML key `listening-address` or the flag `--pprofaddress`.

You need to have [Go](https://golang.org/dl/) installed to profile the program.

### Browser

The easiest way to visualize profiling data is through your browser.

For example, the following commands will show interactive results at [http://localhost:8000](http://localhost:8000):

```sh
# Check the heap
go tool pprof -http=localhost:8000 http://localhost:6060/debug/pprof/heap
# Check the CPU time spent for 10 seconds
go tool pprof -http=localhost:8000 http://localhost:6060/debug/pprof/profile?seconds=10
```

### Compare heaps

You can compare heaps with Go's pprof, this is especially useful to find memory leaks.

1. Download the first heap profile `wget -qO heap.1.out http://localhost:6060/debug/pprof/heap`
1. Download the second heap profile `wget -qO heap.2.out http://localhost:6060/debug/pprof/heap`
1. Compare the second heap profile with the first one using `go tool pprof -base ./heap.1.out heap.2.out`

### More routes

More routes are available in the HTTP pprof server. You can also list them at [http://localhost:6060/debug/pprof/](http://localhost:6060/debug/pprof/).
Notable ones are written below:

#### Goroutine blocking profile

The route `/debug/pprof/block` is available but requires to set the block profile rate, using either the toml value `block-rate` or the flag `--pprofblockrate`.

#### Mutex contention profile

The route `/debug/pprof/mutex` is available but requires to set the mutex profile rate, using either the toml value `mutex-rate` or the flag `--pprofmutexrate`.

#### Other routes

The other routes are listed below, if you need them:

- `/debug/pprof/cmdline`
- `/debug/pprof/symbol`
- `/debug/pprof/trace`
- `/debug/pprof/goroutine`
- `/debug/pprof/threadcreate`
