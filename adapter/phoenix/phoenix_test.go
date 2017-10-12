package phoenix

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"
)

func TestElixirPhoenix(t *testing.T) {

	adapter, err := New("phoenix.test", "./test/phoenix1.3")
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

	re := "Hello PhoenixTest!"
	match := true
	if got := regexp.MustCompile(re).Match(rr.Body.Bytes()); got != match {
		t.Errorf("%s: ~ /%s/ = %v, want %v", rr.Body, re, got, match)
	}

	fmt.Println(rr)
}
