package concwg_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/m-zajac/concwg"
)

// This example shows the simple use case of the concwg.WaitGroup
func ExampleWaitGroup() {
	wg := concwg.New()

	handler := func(w http.ResponseWriter, _ *http.Request) {
		wg.Add(1) // There's some job to be done for this request.

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

	// Wait for all the jobs to complete.
	// It is safe to call it here, even if adds were called in different goroutines.
	wg.Wait()
}
