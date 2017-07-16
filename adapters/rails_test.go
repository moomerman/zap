package adapters

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
)

func TestRubyRails(t *testing.T) {

	adapter, err := CreateRailsAdapter("localhost", "./test/rails5.1")
	if err != nil {
		panic(err)
	}

	if err = adapter.Start(); err != nil {
		panic(err)
	}
	time.Sleep(5 * time.Second)

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	adapter.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	re := "Ruby on Rails"
	match := true
	if got := regexp.MustCompile(re).Match(rr.Body.Bytes()); got != match {
		t.Errorf("%s: ~ /%s/ = %v, want %v", rr.Body, re, got, match)
	}

	adapter.Stop()
	time.Sleep(5 * time.Second)
}
