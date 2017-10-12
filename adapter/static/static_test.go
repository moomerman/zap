package static

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStatic(t *testing.T) {

	adapter, err := New("./test/static")
	if err != nil {
		panic(err)
	}

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

	fmt.Println(rr)
}
