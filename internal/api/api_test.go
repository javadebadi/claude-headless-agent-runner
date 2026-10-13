package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/javadebadi/claude-headless-agent-runner/internal/job"
	"github.com/javadebadi/claude-headless-agent-runner/internal/profile"
)

const fake = `#!/bin/sh
case "$FAKE_MODE" in
  nofile) ;;
  *) echo "hello" > out.txt ;;
esac
echo '{"result":"ok","is_error":false,"total_cost_usd":0.1,"num_turns":2,"duration_ms":5,"usage":{"input_tokens":3,"output_tokens":4}}'
`

func server(t *testing.T, mode string) http.Handler {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "claude")
	if err := os.WriteFile(bin, []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	set, err := profile.Parse([]byte(`
profiles:
  hello:
    description: says hello
    output: out.txt
    params: [name]
    allowed_tools: ["Write({{.Output}})"]
    prompt: "Write hello to {{.Output}}"
`), "test")
	if err != nil {
		t.Fatal(err)
	}
	r := job.New(set)
	r.Claude = bin
	r.WorkRoot = t.TempDir()
	r.Env = []string{"FAKE_MODE=" + mode}
	return New(r)
}

func post(h http.Handler, path, body string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", path, strings.NewReader(body)))
	return rec
}

func TestHealthAndProfiles(t *testing.T) {
	h := server(t, "ok")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/health", nil))
	if rec.Code != 200 {
		t.Fatalf("health: %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/profiles", nil))
	var got []map[string]any
	json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 1 || got[0]["name"] != "hello" {
		t.Errorf("profiles: %s", rec.Body)
	}
	// What a profile may do is not advertised, only how to ask for it.
	if strings.Contains(rec.Body.String(), "Write(") {
		t.Errorf("the permissions leaked into the listing: %s", rec.Body)
	}
}

func TestRunAnswersWithTheResult(t *testing.T) {
	rec := post(server(t, "ok"), "/run", `{"profile":"hello"}`)
	if rec.Code != 200 {
		t.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
	var res job.Result
	json.Unmarshal(rec.Body.Bytes(), &res)
	if res.Output != "hello\n" || res.CostUSD != 0.1 || res.Turns != 2 || res.Usage.InputTokens != 3 || res.Usage.OutputTokens != 4 {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestStatusCodesSayWhoseFaultItIs(t *testing.T) {
	h := server(t, "ok")
	for name, tc := range map[string]struct {
		body string
		want int
	}{
		"not json":        {`nope`, http.StatusBadRequest},
		"unknown profile": {`{"profile":"deploy"}`, http.StatusBadRequest},
		"unknown param":   {`{"profile":"hello","params":{"x":"y"}}`, http.StatusBadRequest},
		"unknown field":   {`{"profile":"hello","tools":["Bash"]}`, http.StatusBadRequest},
	} {
		if rec := post(h, "/run", tc.body); rec.Code != tc.want {
			t.Errorf("%s: status %d, want %d (%s)", name, rec.Code, tc.want, rec.Body)
		}
	}
	// A job that fails is a gateway problem, not a bad request.
	if rec := post(server(t, "nofile"), "/run", `{"profile":"hello"}`); rec.Code != http.StatusBadGateway {
		t.Errorf("a job that wrote nothing: status %d, want 502 (%s)", rec.Code, rec.Body)
	} else if body := rec.Body.String(); !strings.Contains(body, `"error"`) || !strings.Contains(body, `"cost_usd":0.1`) || !strings.Contains(body, `"output_tokens":4`) {
		// It ran, so it spent tokens: the failure says how many.
		t.Errorf("a failed job should report what it spent: %s", body)
	}
}
