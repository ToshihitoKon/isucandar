package agent

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/isucon/isucandar/failure"
	"github.com/stretchr/testify/assert"
)

func TestNoCookie(t *testing.T) {
	agent, err := NewAgent(WithNoCookie(), WithDefaultTransport())
	if err != nil {
		t.Fatal(err)
	}

	if agent.HttpClient.Jar != nil {
		t.Fatal("Not removed cookie jar")
	}
}

func TestNoCache(t *testing.T) {
	agent, err := NewAgent(WithNoCache(), WithDefaultTransport())
	if err != nil {
		t.Fatal(err)
	}

	if agent.CacheStore != nil {
		t.Fatal("Not removed cache store")
	}
}

func TestUserAgent(t *testing.T) {
	agent, err := NewAgent(WithUserAgent("Hello"), WithDefaultTransport())
	if err != nil {
		t.Fatal(err)
	}

	if agent.Name != "Hello" {
		t.Fatalf("missmatch ua: %s", agent.Name)
	}
}

func TestBaseURL(t *testing.T) {
	agent, err := NewAgent(WithBaseURL("http://base.example.com"), WithDefaultTransport())
	if err != nil {
		t.Fatal(err)
	}

	if agent.BaseURL.String() != "http://base.example.com" {
		t.Fatalf("missmatch base URL: %s", agent.BaseURL.String())
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-time.After(2 * time.Second)
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)

		io.WriteString(w, "Hello, World")
	}))
	defer func() {
		go srv.Close()
	}()

	agent, err := NewAgent(WithTimeout(1*time.Microsecond), WithBaseURL(srv.URL), WithDefaultTransport())
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = get(agent, "/")
	var nerr net.Error
	if ok := failure.As(err, &nerr); !ok || !nerr.Timeout() {
		t.Fatalf("expected timeout error: %+v", err)
	}
}

func TestWithoutTransport(t *testing.T) {
	_, err := NewAgent()
	assert.NotNil(t, err)
	if err != nil {
		assert.Same(t, ErrTransportInvalid, err)
	}
}

func TestDefaultTransport(t *testing.T) {
	agent1, err := NewAgent(WithDefaultTransport())
	if err != nil {
		t.Fatal(err)
	}

	agent2, err := NewAgent(WithDefaultTransport())
	if err != nil {
		t.Fatal(err)
	}

	assert.Same(t, agent1.HttpClient.Transport, agent2.HttpClient.Transport)
}

func TestTransport(t *testing.T) {
	trs := DefaultTransport.Clone()

	agent1, err := NewAgent(WithTransport(trs))
	if err != nil {
		t.Fatal(err)
	}

	assert.Same(t, agent1.HttpClient.Transport, trs)
}

func TestCloneTransport(t *testing.T) {
	agent1, err := NewAgent(WithCloneTransport(DefaultTransport))
	if err != nil {
		t.Fatal(err)
	}

	agent2, err := NewAgent(WithCloneTransport(DefaultTransport))
	if err != nil {
		t.Fatal(err)
	}

	assert.NotSame(t, agent1.HttpClient.Transport, agent2.HttpClient.Transport)
}
