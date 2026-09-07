package httpapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/internal/stream"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func busServer(t *testing.T) (*httptest.Server, *stream.Bus, *session.Store) {
	t.Helper()
	bus := stream.New(stream.Options{})
	sessions := session.New()
	h := NewRouterWithDeps(Deps{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Sessions: sessions,
		Bus:      bus,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(func() { srv.Close(); bus.Close() })
	return srv, bus, sessions
}

func TestEventsArePublishedToBus(t *testing.T) {
	srv, bus, _ := busServer(t)
	sub := bus.Subscribe("test")

	postEvent(t, srv, `{"session_id":"s1","user_id":"u1","type":"login"}`).Body.Close()

	select {
	case e := <-sub.C():
		if e.Type != contracts.EventLogin || e.SessionID != "s1" || e.Seq != 1 {
			t.Errorf("unexpected event on bus: %+v", e)
		}
		if e.EventID == "" || e.IngestedAtNanos == 0 {
			t.Errorf("event published before normalization stamped it: %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("event never reached the bus")
	}
}

func TestNoStreamRouteWithoutBus(t *testing.T) {
	srv, _ := testServer(t) // Deps without a Bus
	res, err := http.Get(srv.URL + "/v1/stream")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404 when no Bus configured", res.StatusCode)
	}
}

// sseLines starts a GET /v1/stream and returns a channel of its text lines plus
// a cancel func.
func sseLines(t *testing.T, url string) (<-chan string, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/event-stream") {
		res.Body.Close()
		cancel()
		t.Fatalf("bad stream response: %d %s", res.StatusCode, res.Header.Get("Content-Type"))
	}
	lines := make(chan string, 128)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(res.Body)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	t.Cleanup(func() { cancel(); res.Body.Close() })
	return lines, cancel
}

func waitLine(t *testing.T, lines <-chan string, match func(string) bool, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatal("stream closed before match")
			}
			if match(l) {
				return
			}
		case <-deadline:
			t.Fatal("timed out waiting for line")
		}
	}
}

func collectData(t *testing.T, lines <-chan string, n int, timeout time.Duration) []contracts.Event {
	t.Helper()
	out := make([]contracts.Event, 0, n)
	deadline := time.After(timeout)
	for len(out) < n {
		select {
		case l, ok := <-lines:
			if !ok {
				t.Fatalf("stream closed after %d/%d events", len(out), n)
			}
			data, found := strings.CutPrefix(l, "data: ")
			if !found {
				continue
			}
			var e contracts.Event
			if err := json.Unmarshal([]byte(data), &e); err != nil {
				t.Fatalf("bad SSE data line %q: %v", l, err)
			}
			out = append(out, e)
		case <-deadline:
			t.Fatalf("timed out with %d/%d events", len(out), n)
		}
	}
	return out
}

func TestSSEStreamDeliversEvents(t *testing.T) {
	srv, _, _ := busServer(t)
	lines, _ := sseLines(t, srv.URL+"/v1/stream")
	waitLine(t, lines, func(s string) bool { return strings.Contains(s, "connected") }, 2*time.Second)

	types := []string{"LOGIN", "AMOUNT_ENTERED", "OTP_VERIFIED"}
	for _, typ := range types {
		postEvent(t, srv, `{"session_id":"s1","user_id":"u1","type":"`+typ+`"}`).Body.Close()
	}

	got := collectData(t, lines, len(types), 3*time.Second)
	for i, e := range got {
		if string(e.Type) != types[i] || e.SessionID != "s1" {
			t.Errorf("event %d = %+v, want type %s", i, e, types[i])
		}
		if e.Seq != uint64(i+1) {
			t.Errorf("event %d seq = %d, want %d", i, e.Seq, i+1)
		}
	}
}

func TestSSEStreamSessionFilter(t *testing.T) {
	srv, _, _ := busServer(t)
	lines, _ := sseLines(t, srv.URL+"/v1/stream?session_id=s1")
	waitLine(t, lines, func(s string) bool { return strings.Contains(s, "connected") }, 2*time.Second)

	postEvent(t, srv, `{"session_id":"s2","user_id":"u2","type":"LOGIN"}`).Body.Close()
	postEvent(t, srv, `{"session_id":"s1","user_id":"u1","type":"LOGIN"}`).Body.Close()
	postEvent(t, srv, `{"session_id":"s1","user_id":"u1","type":"OTP_VERIFIED"}`).Body.Close()

	got := collectData(t, lines, 2, 3*time.Second)
	for _, e := range got {
		if e.SessionID != "s1" {
			t.Errorf("filter leaked session %s", e.SessionID)
		}
	}
}

func TestMetricsEndpoint(t *testing.T) {
	srv, _, _ := busServer(t)
	for range 5 {
		postEvent(t, srv, `{"session_id":"s1","user_id":"u1","type":"LOGIN"}`).Body.Close()
	}

	res, err := http.Get(srv.URL + "/v1/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}

	var m struct {
		Sessions int `json:"sessions"`
		Bus      struct {
			Published uint64 `json:"Published"`
		} `json:"bus"`
	}
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		t.Fatal(err)
	}
	if m.Sessions != 1 {
		t.Errorf("sessions = %d, want 1", m.Sessions)
	}
	// Give the hub a moment to count the publishes.
	deadline := time.Now().Add(time.Second)
	for m.Bus.Published < 5 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
		res2, _ := http.Get(srv.URL + "/v1/metrics")
		json.NewDecoder(res2.Body).Decode(&m)
		res2.Body.Close()
	}
	if m.Bus.Published < 5 {
		t.Errorf("bus.Published = %d, want >= 5", m.Bus.Published)
	}
}
