package httpapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/func/func/auth"
)

func TestAuthRoundTripper(t *testing.T) {
	token := &auth.Token{
		AccessToken: "abc",
		Type:        "Bearer",
	}
	rt := &AuthRoundTripper{Token: token}
	cli := &http.Client{Transport: rt}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "received header: %q", r.Header.Get("Authorization"))
	}))
	defer ts.Close()

	req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
	resp, err := cli.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	want := "received header: \"Bearer abc\""
	if string(body) != want {
		t.Errorf("Response does not match\nGot  %q\nWant %q", string(body), want)
	}
}
