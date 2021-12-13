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

The `concwg.WaitGroup` works similarly to the standard version, but has one, crucial change to the interface: **`Add` returns a bool value**.

Since `Add` and `Wait` methods could be called asynchronously, there is no way to guarantee that `Add` won't be called accidentally after the `Wait`.
So in some cases you must have a way to know if it is still safe to schedule a job after the call to `Add`.

That's why after `Wait` is called, `Add` always returns false. In this case you can't schedule a job and be sure that `Wait` will wait for it to finish.

## Usage

Example use case:


```go

type myWorker struct {
	wg *concwg.WaitGroup
}

func newWorker() *myWorker {
	return &myWorker{
		wg: concwg.New(),
	}
}

func (s *myWorker) HandleTask(name string) error {
	if ok := s.wg.Add(1); !ok {
		return errors.New("server is closing")
	}
	defer s.wg.Done()

	// Simulate doing some work.
	time.Sleep(time.Second)
	fmt.Printf("task '%s' done", name)

	return nil
}

func (s *myWorker) Stop() {
	s.wg.Wait()
}

// This example shows the simple use case of the concwg.WaitGroup
func ExampleWaitGroup() {
	worker := newWorker()
	handler := func(w http.ResponseWriter, r *http.Request) {
		err := worker.HandleTask(r.URL.Path)
		if err != nil {
			log.Printf("calling worer: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusAccepted)
	}

	// Start a server.
	srv := httptest.NewServer(http.HandlerFunc(handler))

	// Handle a request.
	resp, err := http.DefaultClient.Get(srv.URL + "/foo")
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != http.StatusAccepted {
		panic("unexpected status code")
	}

	// Close the server
	srv.Close()

	// Stop the worker, wait for all tasks to be finished.
	worker.Stop()

	// You can safely exit the program now.

	// Output:
	// task '/foo' done
}
```
