// Package api serves the runner over HTTP: POST /run does one job and answers
// with its result, GET /health says the service is up.
//
// It has no authentication of its own. Run it where something else decides who
// may call it (Cloud Run's IAM): a job spends the deployment's credential.
package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/javadebadi/claude-headless-agent-runner/internal/job"
)

const maxBody = 2 << 20

// New returns the handler for a runner.
func New(runner *job.Runner) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /profiles", func(w http.ResponseWriter, _ *http.Request) {
		type entry struct {
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Inputs      []string `json:"inputs"`
			Params      []string `json:"params"`
		}
		var out []entry
		for _, name := range runner.Profiles.Names() {
			p := runner.Profiles[name]
			out = append(out, entry{name, p.Description, p.Inputs, p.Params})
		}
		writeJSON(w, http.StatusOK, out)
	})
	mux.HandleFunc("POST /run", func(w http.ResponseWriter, r *http.Request) {
		var req job.Request
		// A field the runner does not know is refused: a caller that sends
		// "tools" believes it has set the agent's permissions, and it has not.
		dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBody))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad request body: " + err.Error()})
			return
		}
		res, err := runner.Run(r.Context(), req)
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, res)
		case job.IsBadRequest(err):
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		case errors.Is(err, r.Context().Err()) && r.Context().Err() != nil:
			// The caller went away or gave up; nobody is listening.
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "cancelled"})
		default:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		}
	})
	return mux
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
