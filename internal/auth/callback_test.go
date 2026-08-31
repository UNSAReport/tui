package auth

import (
	"net/http"
	"testing"
	"time"
)

func TestCallbackStateValidation(t *testing.T) {
	state, err := GenerateState()
	if err != nil {
		t.Fatal(err)
	}
	if len(state) < 5 || state[:3] != "st_" {
		t.Fatalf("state %q", state)
	}
	cb := NewCallbackServer(state)
	url, err := cb.Start()
	if err != nil {
		t.Fatal(err)
	}
	defer cb.Close()
	if !IsLoopbackURL(url) {
		t.Fatalf("url not loopback %q", url)
	}
	go func() {
		_, _ = http.Get(url + "?state=bad&pat=unsareport_pat_test")
	}()
	select {
	case res := <-cb.Done:
		if res.Err == nil {
			t.Fatal("should error on bad state")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout")
	}

	cb2 := NewCallbackServer(state)
	url2, _ := cb2.Start()
	defer cb2.Close()
	go func() {
		_, _ = http.Get(url2 + "?state=" + state + "&pat=unsareport_pat_good")
	}()
	select {
	case res := <-cb2.Done:
		if res.Err != nil {
			t.Fatal(res.Err)
		}
		if res.PAT != "unsareport_pat_good" {
			t.Fatalf("pat %q", res.PAT)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout valid")
	}
}

func TestIsLoopbackURL(t *testing.T) {
	if !IsLoopbackURL("http://127.0.0.1:1234/callback") {
		t.Fatal("should be loopback")
	}
	if !IsLoopbackURL("http://localhost:8080/callback") {
		t.Fatal("should be loopback localhost")
	}
	if IsLoopbackURL("https://127.0.0.1:1234/callback") {
		t.Fatal("https should not pass")
	}
	if IsLoopbackURL("http://example.com/callback") {
		t.Fatal("external should not pass")
	}
	if IsLoopbackURL("http://127.0.0.1.evil.com/callback") {
		t.Fatal("evil should not pass")
	}
}

func TestHeadlessDetection(t *testing.T) {
	t.Setenv("SSH_CONNECTION", "1")
	if !IsHeadless() {
		t.Fatal("should be headless with SSH_CONNECTION")
	}
	t.Setenv("SSH_CONNECTION", "")
}
