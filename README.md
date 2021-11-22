# concwg ![Build](https://github.com/m-zajac/concwg/workflows/Build/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/m-zajac/concwg)](https://goreportcard.com/report/github.com/m-zajac/concwg) [![Go Reference](https://pkg.go.dev/badge/github.com/m-zajac/concwg.svg)](https://pkg.go.dev/github.com/m-zajac/concwg) [![Coverage](https://img.shields.io/badge/coverage-gocover.io-blue)](https://gocover.io/github.com/m-zajac/concwg)

## Description

This package provides a version of `sync.WaitGroup` that allows calling `Add` and `Wait` in different goroutines.

## Motivation

`sync.WaitGroup` is designed to be used only in this kind of scenario:

```go
var wg sync.WaitGroup
for i := 0; i < 10; i++ {
    wg.Add(1)
    go func() {
        // do something
        wg.Done()
    }()
}
wg.Wait()
```

It is critical that `Add` and `Wait` are in the same goroutine. This is not well-enough documented behavior, but if you're interested, you can check:
 - [The golang issue](https://github.com/golang/go/issues/23842)
 - [The source code](https://cs.opensource.google/go/go/+/refs/tags/go1.16.7:src/sync/waitgroup.go;l=88)

The `concwg.WaitGroup` works very similar to the standard version, but it is safe to use in different scenarios.
## Usage

This version required one, crucial change to the interface vs the standard `WaitGroup`.

Since `Add` and `Wait` methods could be all called asynchronously, there is no way to guarantee that `Add` won't be called accidentally after the `Wait`.
So in some cases you must have a way to know if it is still safe to schedule a job after the call to `Add`.

That's why:
- You can call `Finish` if you want to be sure that no new jobs will be accepted before calling `Wait`.
- After `Finish` is called, `Add` always returns false. In this case you can't schedule a job and be sure that synchronous `Wait` will wait for it to finish.

Example:


```go
wg := concwg.New()

handler := func(w http.ResponseWriter, _ *http.Request) {
    // There's some job to be done for this request.
    // Note that each request is handled in a separate goroutine.
    ok := wg.Add(1)
    if !ok {
        // This means the group was "finished" and it is not safe to accept more jobs.
        w.WriteHeader(StatusServiceUnavailable)
        return
    }

    w.WriteHeader(http.StatusAccepted)
    go func() {
        // Do a background job...
        defer wg.Done()
    }()
}

// Start a server.
srv := httptest.NewServer(http.HandlerFunc(handler))

// Handler some requests.
// ...

// Close the server
srv.Close()

// Finish the group to prevent accepting new jobs before we start to wait.
wg.Finish()

// Wait for all the jobs to complete.
// It is safe to call it here.
wg.Wait()
```
