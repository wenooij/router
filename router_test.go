package router

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestRouter(t *testing.T) {
	var r Router
	r.InsertFunc("test", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test"))
	}).InsertFunc(":id", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "id=", r.URL.Path)
	})

	for _, route := range r.Routes() {
		t.Log(route)
	}

	id := "00000000-0000-0000-0000-000000000000"

	s := httptest.NewServer(&r)
	defer s.Close()

	url := fmt.Sprint(s.URL, "/test/", id)
	fmt.Println(url)

	cli := s.Client()
	resp, err := cli.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	want := []byte(fmt.Sprint("id=", id))
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Get(): got diff (-want, +got):\n%s", diff)
	}
}
