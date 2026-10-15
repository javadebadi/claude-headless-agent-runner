// Package job runs one job of a profile: it prepares a fresh working folder
// (the profile's skills, the request's inputs and the folders it asked to
// copy), runs Claude Code headless there with only the permissions the
// profile gives, and returns the one file the agent was asked to write (or the
// folder of files, for a profile with output_dir).
//
// It runs on Linux and macOS (a timed-out session's whole process group is
// stopped). It knows nothing about what the jobs are for. Every job is: some inputs, a
// prompt, a few folders to read, one file (or folder) back.
package job

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/javadebadi/claude-headless-agent-runner/internal/profile"
)

// Request is one job: which profile, and what it is about.
type Request struct {
	Profile string `json:"profile"`
	// Inputs are named texts the job works from (a story outline). They are
	// written to files in the working folder; the prompt refers to their paths.
	Inputs map[string]string `json:"inputs"`
	// Refs are folders to copy into the working folder (a theme). Each source
	// must be inside a place the profile may read.
	Refs []Ref `json:"refs"`
	// Params fill the profile's templates (a language, a theme name).
	Params map[string]string `json:"params"`
}

// Ref copies the folder From (an absolute path on this machine) to To (relative
// to the working folder).
type Ref struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// Usage is the tokens a session used, as Claude Code counted them. Cache reads
// are the cheap part: a session that starts cold has few of them.
type Usage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheReadTokens     int `json:"cache_read_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
}

// Spend is what a session cost. CostUSD is Claude Code's own estimate (tokens
// times its price table), not a bill. It is reported for failed jobs too: they
// spend tokens like any other.
type Spend struct {
	CostUSD    float64 `json:"cost_usd"`
	Turns      int     `json:"turns"`
	DurationMs int64   `json:"duration_ms"`
	Usage      Usage   `json:"usage"`
}

// JobError is a job that failed after its session ran, and what the session
// spent before failing. A job that never started a session (a refused request,
// a timeout) is a plain error.
type JobError struct {
	Err   error
	Spend Spend
}

func (e *JobError) Error() string { return e.Err.Error() }
func (e *JobError) Unwrap() error { return e.Err }

// Result is what a job returns.
type Result struct {
	Output string `json:"output,omitempty"` // the contents of the profile's output file
	// Files is the profile's output folder: each file's contents by its path
	// inside the folder, with / separators. Set instead of Output when the
	// profile has output_dir.
	Files map[string]string `json:"files,omitempty"`
	Notes string            `json:"notes"` // the agent's own closing remarks
	Model string            `json:"model"`
	Spend
	// Denied lists tools the agent tried to use and was refused, which usually
	// means the profile needs a rule added (or the agent was misled).
	Denied []string `json:"denied,omitempty"`
}

// Runner runs jobs of a deployment's profiles, one at a time: a session is
// heavy, and one agent's browser or files must not be another's.
type Runner struct {
	Profiles profile.Set
	// Claude is the Claude Code program (default: claude on PATH).
	Claude string
	// WorkRoot is where working folders are made (default: the temp folder).
	WorkRoot string
	// Env is added to the session's environment (the credential comes from the
	// runner's own environment, set by whoever deploys it).
	Env []string

	slot chan struct{}
}

// New returns a Runner.
func New(profiles profile.Set) *Runner {
	return &Runner{Profiles: profiles, slot: make(chan struct{}, 1)}
}

var (
	paramValue = regexp.MustCompile(`^[A-Za-z0-9 ._:/-]{0,100}$`)
	inputName  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,40}$`)
)

// Limits on what a job returns: one file, or a folder of files.
const (
	maxOutputBytes    = 1 << 20
	maxOutputDirBytes = 8 << 20
	maxOutputFiles    = 500
)

type templateData struct {
	Output string
	Inputs map[string]string
	Refs   []string
	Params map[string]string
}

// Run does the job. A refusal of the request is a *BadRequest; anything else
// that goes wrong is an ordinary error.
func (r *Runner) Run(ctx context.Context, req Request) (*Result, error) {
	p, ok := r.Profiles[req.Profile]
	if !ok {
		return nil, badRequestf("no profile %q (have: %s)", req.Profile, strings.Join(r.Profiles.Names(), ", "))
	}
	if err := checkRequest(p, req); err != nil {
		return nil, err
	}

	select {
	case r.slot <- struct{}{}:
		defer func() { <-r.slot }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	work, err := os.MkdirTemp(r.WorkRoot, "job-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(work)
	// Claude Code names files by their real path, and permission rules are
	// matched against that: a working folder reached through a symlink (macOS's
	// /var, for one) would make its own file look like it is somewhere else.
	if work, err = filepath.EvalSymlinks(work); err != nil {
		return nil, err
	}

	data, err := r.prepare(work, p, req)
	if err != nil {
		return nil, err
	}
	system, err := renderSystem(p, data, work)
	if err != nil {
		return nil, err
	}
	prompt, err := render("prompt", p.Prompt, data)
	if err != nil {
		return nil, err
	}
	allowed, err := renderAll("allowed_tools", p.AllowedTools, data)
	if err != nil {
		return nil, err
	}
	denied, err := renderAll("denied_tools", p.DeniedTools, data)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(p.Timeout))
	defer cancel()
	report, err := r.claude(ctx, work, p, prompt, system, allowed, denied)
	if err != nil {
		if report != nil {
			err = &JobError{Err: err, Spend: report.spend()}
		}
		return nil, err
	}

	res := &Result{
		Notes: tail(report.Result),
		Model: p.Model,
		Spend: report.spend(),
	}
	if err := collect(work, p, report, res); err != nil {
		return nil, err
	}
	res.Denied = report.denied()
	return res, nil
}

// checkRequest refuses anything the profile did not declare: the request can
// only fill in what the deployment decided it may.
func checkRequest(p *profile.Profile, req Request) error {
	declared := map[string]bool{}
	for _, n := range p.Inputs {
		declared[n] = true
		if _, ok := req.Inputs[n]; !ok {
			return badRequestf("input %q is missing", n)
		}
	}
	total := 0
	for n, v := range req.Inputs {
		if !declared[n] || !inputName.MatchString(n) {
			return badRequestf("input %q is not one this profile takes", n)
		}
		total += len(v)
	}
	if total > p.MaxInputBytes {
		return badRequestf("the inputs are %d bytes; this profile takes at most %d", total, p.MaxInputBytes)
	}
	allowed := map[string]bool{}
	for _, n := range p.Params {
		allowed[n] = true
	}
	for n, v := range req.Params {
		if !allowed[n] {
			return badRequestf("parameter %q is not one this profile takes", n)
		}
		// Parameters go into the prompt: keep them to plain words.
		if !paramValue.MatchString(v) {
			return badRequestf("parameter %q has characters a parameter may not have", n)
		}
	}
	if len(req.Refs) > 20 {
		return badRequestf("at most 20 refs")
	}
	return nil
}

// prepare builds the working folder and returns the template data.
func (r *Runner) prepare(work string, p *profile.Profile, req Request) (*templateData, error) {
	output := p.Output
	if p.OutputDir != "" {
		output = p.OutputDir
	}
	data := &templateData{Output: output, Inputs: map[string]string{}, Params: req.Params}
	if data.Params == nil {
		data.Params = map[string]string{}
	}
	for _, n := range p.Params { // a declared parameter left out reads as empty, not as an error
		if _, ok := data.Params[n]; !ok {
			data.Params[n] = ""
		}
	}
	c := &copier{}

	for _, s := range p.Skills {
		real, err := filepath.EvalSymlinks(s.From)
		if err != nil {
			return nil, fmt.Errorf("skill %q: %w", s.Name, err)
		}
		if err := c.copyTree(real, filepath.Join(work, ".claude", "skills", s.Name)); err != nil {
			return nil, fmt.Errorf("skill %q: %w", s.Name, err)
		}
	}
	if len(req.Inputs) > 0 {
		if err := os.MkdirAll(filepath.Join(work, "inputs"), 0o755); err != nil {
			return nil, err
		}
	}
	for n, v := range req.Inputs {
		rel := filepath.Join("inputs", n)
		if err := os.WriteFile(filepath.Join(work, rel), []byte(v), 0o644); err != nil {
			return nil, err
		}
		data.Inputs[n] = rel
	}
	for _, ref := range req.Refs {
		src, err := resolveRead(ref.From, p.Reads)
		if err != nil {
			return nil, err
		}
		dst, err := destination(work, ref.To)
		if err != nil {
			return nil, err
		}
		if p.OutputDir != "" && within(filepath.Join(work, p.OutputDir), dst) {
			return nil, badRequestf("ref target %q may not be in the output folder", ref.To)
		}
		if err := c.copyTree(src, dst); err != nil {
			return nil, err
		}
		data.Refs = append(data.Refs, filepath.Clean(ref.To))
	}
	if p.OutputDir != "" {
		if err := os.MkdirAll(filepath.Join(work, p.OutputDir), 0o755); err != nil {
			return nil, err
		}
	}
	return data, nil
}

// collect reads what the agent wrote into res: the profile's one file, or all
// the files of its output folder.
func collect(work string, p *profile.Profile, report *report, res *Result) error {
	fail := func(err error) error { return &JobError{Err: err, Spend: report.spend()} }
	missing := func(what string) error {
		msg := fmt.Sprintf("the agent finished without writing %s: %s", what, tail(report.Result))
		if d := report.denied(); len(d) > 0 {
			msg += fmt.Sprintf(" (refused by the profile's permissions: %s)", strings.Join(d, "; "))
		} else if report.Turns <= 1 {
			// Reading the inputs and writing the output take at least a few turns, so
			// one turn means the agent answered in text and used no tool. Its reply
			// then reads like a problem with the request; say where it really lies.
			msg += " (the session ended after one turn without using any tool, so it never read the inputs or wrote the output: " +
				"the prompt may be too vague for this model; name the input files and the tools to use, or try a larger model)"
		}
		return fail(errors.New(msg))
	}

	if p.OutputDir == "" {
		out, err := os.ReadFile(filepath.Join(work, p.Output))
		if err != nil || len(bytes.TrimSpace(out)) == 0 {
			return missing(p.Output)
		}
		if len(out) > maxOutputBytes {
			return fail(fmt.Errorf("%s is larger than %d bytes", p.Output, maxOutputBytes))
		}
		res.Output = string(out)
		return nil
	}

	root := filepath.Join(work, p.OutputDir)
	files := map[string]string{}
	total := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			return nil
		case !d.Type().IsRegular():
			// A link could lead out of the working folder: refuse, don't follow.
			return fmt.Errorf("%s is not a plain file", path[len(work)+1:])
		}
		rel, _ := filepath.Rel(root, path)
		if len(files) >= maxOutputFiles {
			return fmt.Errorf("more than %d files in %s", maxOutputFiles, p.OutputDir)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if total += len(data); total > maxOutputDirBytes {
			return fmt.Errorf("%s holds more than %d bytes", p.OutputDir, maxOutputDirBytes)
		}
		if !utf8.Valid(data) {
			return fmt.Errorf("%s is not text; an output folder may hold only text files", filepath.ToSlash(rel))
		}
		files[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) || (err == nil && len(files) == 0) {
		return missing(p.OutputDir + "/")
	}
	if err != nil {
		return fail(err)
	}
	res.Files = files
	return nil
}

func render(name, text string, data *templateData) (string, error) {
	t, err := template.New(name).Option("missingkey=error").Parse(text)
	if err != nil {
		return "", fmt.Errorf("profile %s: %w", name, err)
	}
	var b strings.Builder
	if err := t.Execute(&b, data); err != nil {
		return "", fmt.Errorf("profile %s: %w", name, err)
	}
	return b.String(), nil
}

func renderAll(name string, items []string, data *templateData) ([]string, error) {
	var out []string
	for _, it := range items {
		s, err := render(name, it, data)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// renderSystem is the profile's system text followed by the body of each
// inline skill.
func renderSystem(p *profile.Profile, data *templateData, work string) (string, error) {
	var parts []string
	if strings.TrimSpace(p.System) != "" {
		s, err := render("system", p.System, data)
		if err != nil {
			return "", err
		}
		parts = append(parts, strings.TrimSpace(s))
	}
	for _, s := range p.Skills {
		if !s.Inline {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(work, ".claude", "skills", s.Name, "SKILL.md"))
		if err != nil {
			return "", fmt.Errorf("skill %q has no SKILL.md to include", s.Name)
		}
		parts = append(parts, fmt.Sprintf("## Skill: %s\n\n%s", s.Name, skillBody(string(raw))))
	}
	return strings.Join(parts, "\n\n"), nil
}

// skillBody is a SKILL.md without its YAML front matter.
func skillBody(s string) string {
	if strings.HasPrefix(s, "---") {
		if end := strings.Index(s[3:], "\n---"); end >= 0 {
			s = s[3+end+4:]
		}
	}
	return strings.TrimSpace(s)
}

type report struct {
	Result  string  `json:"result"`
	IsError bool    `json:"is_error"`
	Subtype string  `json:"subtype"`
	CostUSD float64 `json:"total_cost_usd"`
	Turns   int     `json:"num_turns"`
	Millis  int64   `json:"duration_ms"`
	Usage   struct {
		Input         int `json:"input_tokens"`
		Output        int `json:"output_tokens"`
		CacheRead     int `json:"cache_read_input_tokens"`
		CacheCreation int `json:"cache_creation_input_tokens"`
	} `json:"usage"`
	Denials []struct {
		Tool  string          `json:"tool_name"`
		Input json.RawMessage `json:"tool_input"`
	} `json:"permission_denials"`
}

func (r *report) spend() Spend {
	return Spend{
		CostUSD:    r.CostUSD,
		Turns:      r.Turns,
		DurationMs: r.Millis,
		Usage: Usage{
			InputTokens:         r.Usage.Input,
			OutputTokens:        r.Usage.Output,
			CacheReadTokens:     r.Usage.CacheRead,
			CacheCreationTokens: r.Usage.CacheCreation,
		},
	}
}

// denied describes the refused tool calls: which tool and what it was after.
func (r *report) denied() []string {
	var out []string
	for _, d := range r.Denials {
		out = append(out, strings.TrimSpace(d.Tool+" "+string(d.Input)))
	}
	return out
}

func (r *Runner) claude(ctx context.Context, work string, p *profile.Profile, prompt, system string, allowed, denied []string) (*report, error) {
	bin := r.Claude
	if bin == "" {
		var err error
		if bin, err = exec.LookPath("claude"); err != nil {
			return nil, errors.New("Claude Code (`claude`) is not installed on this machine")
		}
	}
	args := []string{"-p", prompt, "--output-format", "json", "--permission-mode", p.PermissionMode}
	if system != "" {
		args = append(args, "--append-system-prompt", system)
	}
	args = append(args, "--allowedTools")
	args = append(args, allowed...)
	if len(denied) > 0 {
		args = append(args, "--disallowedTools")
		args = append(args, denied...)
	}
	if p.Model != "" {
		args = append(args, "--model", p.Model)
	}
	if p.Budget > 0 {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", p.Budget))
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = work
	cmd.Env = append(os.Environ(), r.Env...)
	// A session starts helpers of its own (a browser, shell commands). When the
	// job times out, stop the whole group, or they keep running unseen.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL) }
	cmd.WaitDelay = 2 * time.Second
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("the job did not finish within %s", time.Duration(p.Timeout))
	}
	var rep report
	if err := json.Unmarshal(stdout.Bytes(), &rep); err != nil {
		if runErr != nil {
			return nil, fmt.Errorf("claude failed: %w\n%s", runErr, tail(stderr.String()))
		}
		return nil, fmt.Errorf("claude's output was not the expected JSON: %w", err)
	}
	if rep.IsError {
		return &rep, fmt.Errorf("the session ended with an error (%s): %s", rep.Subtype, tail(rep.Result))
	}
	return &rep, nil
}

// tail keeps the end of a message: a session's closing lines are what matter.
func tail(s string) string {
	s = strings.TrimSpace(s)
	const max = 4000
	if len(s) > max {
		return "…" + s[len(s)-max:]
	}
	return s
}
