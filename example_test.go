package concwg_test

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/m-zajac/concwg"
)

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
