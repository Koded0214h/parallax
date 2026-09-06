package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/holiday-heartbreaks/parallax/backend/internal/session"
	"github.com/holiday-heartbreaks/parallax/backend/pkg/contracts"
)

func testServer(t *testing.T) (*httptest.Server, *session.Store) {
	t.Helper()
	sessions := session.New()
	h := NewRouterWithDeps(Deps{
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Sessions: sessions,
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv, sessions
}

func postEvent(t *testing.T, srv *httptest.Server, body string) *http.Response {
	t.Helper()
	res, err := http.Post(srv.URL+"/v1/events", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	defer res.Body.Close()
	var v T
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

func TestHealthz(t *testing.T) {
	srv, _ := testServer(t)
	res, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	body := decode[map[string]any](t, res)
	if body["status"] != "ok" {
		t.Errorf("status field = %v", body["status"])
	}
	if _, ok := body["sessions"]; !ok {
		t.Errorf("healthz missing sessions count: %v", body)
	}
}

func TestPostEventNormalizesAndSequences(t *testing.T) {
	srv, sessions := testServer(t)

	res := postEvent(t, srv, `{"session_id":"s1","user_id":"u1","type":"login"}`)
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", res.StatusCode)
	}
	first := decode[map[string]any](t, res)
	if first["seq"].(float64) != 1 {
		t.Errorf("first seq = %v, want 1", first["seq"])
	}
	if !strings.HasPrefix(first["event_id"].(string), "evt_") {
		t.Errorf("event_id not generated: %v", first["event_id"])
	}

	res = postEvent(t, srv, `{"session_id":"s1","user_id":"u1","type":"AMOUNT_ENTERED","metadata":{"amount":450000}}`)
	second := decode[map[string]any](t, res)
	if second["seq"].(float64) != 2 || second["events"].(float64) != 2 {
		t.Errorf("second post = %v", second)
	}

	tr, ok := sessions.Trajectory("s1")
	if !ok || len(tr) != 2 {
		t.Fatalf("store has %d events, want 2", len(tr))
	}
	if tr[0].Type != contracts.EventLogin {
		t.Errorf("type not normalized in store: %q", tr[0].Type)
	}
	if tr[0].EventID == "" || tr[0].Timestamp == 0 || tr[0].IngestedAtNanos == 0 {
		t.Errorf("server fields not stamped: %+v", tr[0])
	}
	if tr[1].Metadata["amount"] != float64(450000) {
		t.Errorf("metadata lost: %+v", tr[1].Metadata)
	}
}

func TestPostEventRejects(t *testing.T) {
	srv, _ := testServer(t)
	cases := map[string]string{
		"malformed json": `{"session_id":`,
		"empty body":     ``,
		"not an object":  `["nope"]`,
		"missing user":   `{"session_id":"s1","type":"LOGIN"}`,
		"missing type":   `{"session_id":"s1","user_id":"u1"}`,
		"unknown type":   `{"session_id":"s1","user_id":"u1","type":"WAT"}`,
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			res := postEvent(t, srv, body)
			defer res.Body.Close()
			if res.StatusCode != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400", res.StatusCode)
			}
			if got := decode[map[string]string](t, res); got["error"] == "" {
				t.Errorf("missing error message in %v", got)
			}
		})
	}
}

func TestIntentUnknownSession(t *testing.T) {
	srv, _ := testServer(t)
	res, err := http.Get(srv.URL + "/v1/sessions/nope/intent")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
}

func TestIntentKnownSessionShape(t *testing.T) {
	srv, _ := testServer(t)
	postEvent(t, srv, `{"session_id":"s1","user_id":"u1","type":"LOGIN"}`).Body.Close()

	res, err := http.Get(srv.URL + "/v1/sessions/s1/intent")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
	got := decode[contracts.IntentResponse](t, res)
	if got.SessionID != "s1" || got.EventsSeen != 1 {
		t.Errorf("unexpected intent envelope: %+v", got)
	}
	total := 0.0
	for _, k := range []string{
		contracts.IntentLegitimate, contracts.IntentAccidental,
		contracts.IntentSocialEngineering, contracts.IntentAccountTakeover,
	} {
		v, ok := got.Hypotheses[k]
		if !ok {
			t.Fatalf("hypotheses missing %q: %+v", k, got.Hypotheses)
		}
		total += v
	}
	if total < 0.99 || total > 1.01 {
		t.Errorf("hypotheses sum to %v, want ~1", total)
	}
}

func TestCORSPreflight(t *testing.T) {
	srv, _ := testServer(t)
	req, _ := http.NewRequest(http.MethodOptions, srv.URL+"/v1/events", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", res.StatusCode)
	}
	if res.Header.Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("missing CORS origin header")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	srv, _ := testServer(t)
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/v1/events", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", res.StatusCode)
	}
}

func TestNewRouterDefaultsAreUsable(t *testing.T) {
	// NewRouter with a nil logger must still build a working handler.
	srv := httptest.NewServer(NewRouter(nil))
	t.Cleanup(srv.Close)
	res, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
}
