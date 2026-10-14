// Package profile reads the job descriptions a deployment of the runner
// offers. A profile is written by whoever deploys the runner, never by the
// caller of a job: it fixes what the agent may read, which commands it may
// run, which file it must hand back, and how much it may spend. A request can
// only pick a profile and fill in its declared inputs and parameters.
package profile

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Skill is a folder of instructions for the agent (a SKILL.md and whatever it
// refers to). It is copied into the job's working folder under
// .claude/skills/<name>/ so Claude Code finds it; with Inline its SKILL.md
// body is also added to the system prompt, for guidance that must always be
// followed.
type Skill struct {
	Name   string `yaml:"name"`
	From   string `yaml:"from"` // an absolute folder on this machine
	Inline bool   `yaml:"inline"`
}

// Profile is one kind of job.
type Profile struct {
	Name        string
	Description string   `yaml:"description"`
	Model       string   `yaml:"model"`
	Budget      float64  `yaml:"budget"` // USD; the session stops if it would spend more
	Timeout     Duration `yaml:"timeout"`

	// System and Prompt are templates. Data available: .Output (the file or
	// folder the agent writes, relative to its working folder), .Inputs (input name ->
	// its path in the working folder), .Refs (the copied folders, as paths in
	// the working folder) and .Params (the request's parameters).
	System string `yaml:"system"`
	Prompt string `yaml:"prompt"`

	// PermissionMode is Claude Code's permission mode. The default,
	// acceptEdits, lets the agent write inside its working folder (and refuses
	// writes anywhere else); dontAsk refuses everything not allowed below,
	// writing the output file included; plan reads only. Path-scoped Write rules
	// such as Write(spec.yaml) are not honored by headless Claude Code (checked
	// against the real CLI), which is why writing is confined by the mode and the
	// scratch working folder, not by a rule.
	PermissionMode string `yaml:"permission_mode"`

	// AllowedTools and DeniedTools are Claude Code permission rules, also
	// templates. Commands and tools not allowed here are refused (a headless
	// session cannot ask), with one exception worth knowing: Claude Code always
	// allows reading files and running read-only commands such as ls. The rules
	// are a guard rail; the container is the boundary (see the README).
	AllowedTools []string `yaml:"allowed_tools"`
	DeniedTools  []string `yaml:"denied_tools"`

	Skills []Skill `yaml:"skills"`

	// Reads are the only places a request may copy folders from (Refs): absolute
	// folders on this machine, which in a container are mounted buckets.
	Reads []string `yaml:"reads"`

	// Inputs and Params are the names a request must (Inputs) or may (Params)
	// supply. Anything else is refused.
	Inputs []string `yaml:"inputs"`
	Params []string `yaml:"params"`

	// Output is the one file the agent may write, and what the job returns.
	// OutputDir is the alternative for work made of several related files: a
	// folder the agent fills, all of whose files the job returns. A profile
	// sets exactly one of them.
	Output    string `yaml:"output"`
	OutputDir string `yaml:"output_dir"`

	MaxInputBytes int `yaml:"max_input_bytes"`
}

// Duration is a time.Duration that reads "15m" in YAML.
type Duration time.Duration

func (d *Duration) UnmarshalYAML(n *yaml.Node) error {
	var s string
	if err := n.Decode(&s); err != nil {
		return err
	}
	v, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("%q is not a duration like 15m", s)
	}
	*d = Duration(v)
	return nil
}

// Set is a deployment's profiles by name.
type Set map[string]*Profile

var (
	namePattern   = regexp.MustCompile(`^[a-z][a-z0-9-]{0,40}$`)
	paramPattern  = regexp.MustCompile(`^[a-z][a-z0-9_]{0,40}$`)
	outputPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,60}$`)
)

// Defaults for what a profile leaves out.
const (
	DefaultTimeout       = 15 * time.Minute
	DefaultMaxInputBytes = 256 << 10
)

// Load reads a profiles file:
//
//	profiles:
//	  spec: {description: ..., output: spec.yaml, ...}
//
// A profile returns one file (output) or a folder of files (output_dir).
func Load(path string) (Set, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(data, path)
}

// Parse is Load for text already read. Unknown fields are an error, so a typo
// in a permission field cannot silently leave a profile more open than meant.
func Parse(data []byte, source string) (Set, error) {
	var file struct {
		Profiles map[string]*Profile `yaml:"profiles"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("%s: %w", source, err)
	}
	if len(file.Profiles) == 0 {
		return nil, fmt.Errorf("%s: no profiles", source)
	}
	set := Set{}
	for name, p := range file.Profiles {
		if p == nil {
			return nil, fmt.Errorf("%s: profile %q is empty", source, name)
		}
		p.Name = name
		if err := p.validate(); err != nil {
			return nil, fmt.Errorf("%s: profile %q: %w", source, name, err)
		}
		set[name] = p
	}
	return set, nil
}

func (p *Profile) validate() error {
	if !namePattern.MatchString(p.Name) {
		return errors.New("the name must be lowercase letters, digits and dashes")
	}
	if strings.TrimSpace(p.Prompt) == "" {
		return errors.New("prompt is empty")
	}
	switch {
	case p.Output != "" && p.OutputDir != "":
		return errors.New("set output (one file) or output_dir (a folder of files), not both")
	case p.Output == "" && p.OutputDir == "":
		return errors.New("set output (the one file the job returns) or output_dir (a folder of files)")
	case p.Output != "" && !outputPattern.MatchString(p.Output):
		return errors.New("output must be one plain file name (the file the agent writes and the job returns)")
	case p.OutputDir != "" && (!outputPattern.MatchString(p.OutputDir) || p.OutputDir == "inputs"):
		return errors.New("output_dir must be one plain folder name, other than inputs")
	}
	if p.Budget < 0 {
		return errors.New("budget is negative")
	}
	switch p.PermissionMode {
	case "":
		p.PermissionMode = "acceptEdits"
	case "acceptEdits", "dontAsk", "plan":
	default:
		return fmt.Errorf("permission_mode %q must be acceptEdits, dontAsk or plan", p.PermissionMode)
	}
	if p.Timeout == 0 {
		p.Timeout = Duration(DefaultTimeout)
	}
	if p.MaxInputBytes == 0 {
		p.MaxInputBytes = DefaultMaxInputBytes
	}
	if len(p.AllowedTools) == 0 {
		return errors.New("allowed_tools is empty: the agent could do nothing")
	}
	for _, r := range p.Reads {
		if !filepath.IsAbs(r) {
			return fmt.Errorf("reads: %q must be an absolute folder", r)
		}
	}
	seen := map[string]bool{}
	for _, s := range p.Skills {
		if !namePattern.MatchString(s.Name) {
			return fmt.Errorf("skill name %q must be lowercase letters, digits and dashes", s.Name)
		}
		if !filepath.IsAbs(s.From) {
			return fmt.Errorf("skill %q: from must be an absolute folder", s.Name)
		}
		if seen[s.Name] {
			return fmt.Errorf("skill %q is listed twice", s.Name)
		}
		seen[s.Name] = true
	}
	for _, n := range append(append([]string{}, p.Inputs...), p.Params...) {
		if !paramPattern.MatchString(n) {
			return fmt.Errorf("input/param name %q must be lowercase letters, digits and underscores", n)
		}
	}
	return nil
}

// Names lists the profiles, for messages.
func (s Set) Names() []string {
	var out []string
	for n := range s {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
