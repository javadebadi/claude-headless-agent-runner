package profile

import (
	"strings"
	"testing"
	"time"
)

const good = `
profiles:
  spec:
    description: Draft a carousel spec
    model: claude-sonnet-5-5
    budget: 2
    timeout: 10m
    output: spec.yaml
    inputs: [outline]
    params: [language]
    reads: [/themes]
    allowed_tools: ["Read", "Write({{.Output}})"]
    denied_tools: ["Read(**/.env)"]
    skills:
      - {name: guide, from: /skills/guide, inline: true}
    prompt: Write {{.Output}}
`

func TestParse(t *testing.T) {
	set, err := Parse([]byte(good), "test")
	if err != nil {
		t.Fatal(err)
	}
	p := set["spec"]
	if p == nil || p.Name != "spec" || p.Budget != 2 || time.Duration(p.Timeout) != 10*time.Minute {
		t.Fatalf("unexpected profile: %+v", p)
	}
	if p.PermissionMode != "acceptEdits" {
		t.Errorf("the default permission mode is %q", p.PermissionMode)
	}
	if !p.Skills[0].Inline || p.MaxInputBytes != DefaultMaxInputBytes {
		t.Errorf("defaults/skills wrong: %+v", p)
	}
	if got := set.Names(); len(got) != 1 || got[0] != "spec" {
		t.Errorf("names: %v", got)
	}
}

func TestADefaultTimeoutIsFilledIn(t *testing.T) {
	set, err := Parse([]byte(strings.Replace(good, "    timeout: 10m\n", "", 1)), "test")
	if err != nil {
		t.Fatal(err)
	}
	if time.Duration(set["spec"].Timeout) != DefaultTimeout {
		t.Errorf("timeout %v", set["spec"].Timeout)
	}
}

func TestRefusesAProfileThatCouldBeMoreOpenThanMeant(t *testing.T) {
	for name, mutate := range map[string]func(string) string{
		"unknown field (a typo in a permission)": func(s string) string { return strings.Replace(s, "allowed_tools", "alowed_tools", 1) },
		"no allowed tools": func(s string) string {
			return strings.Replace(s, `allowed_tools: ["Read", "Write({{.Output}})"]`, "allowed_tools: []", 1)
		},
		"output with a folder":  func(s string) string { return strings.Replace(s, "output: spec.yaml", "output: ../spec.yaml", 1) },
		"relative read root":    func(s string) string { return strings.Replace(s, "reads: [/themes]", "reads: [themes]", 1) },
		"relative skill folder": func(s string) string { return strings.Replace(s, "from: /skills/guide", "from: skills/guide", 1) },
		"empty prompt":          func(s string) string { return strings.Replace(s, "prompt: Write {{.Output}}", "prompt: ''", 1) },
		"bad profile name":      func(s string) string { return strings.Replace(s, "  spec:", "  Spec Job:", 1) },
		"bad param name":        func(s string) string { return strings.Replace(s, "params: [language]", "params: [Lang]", 1) },
		"bypassing permissions": func(s string) string {
			return strings.Replace(s, "    output: spec.yaml", "    permission_mode: bypassPermissions\n    output: spec.yaml", 1)
		},
		"negative budget": func(s string) string { return strings.Replace(s, "budget: 2", "budget: -1", 1) },
		"bad timeout":     func(s string) string { return strings.Replace(s, "timeout: 10m", "timeout: soon", 1) },
	} {
		if _, err := Parse([]byte(mutate(good)), "test"); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	if _, err := Parse([]byte("profiles: {}"), "test"); err == nil {
		t.Error("an empty file should be an error")
	}
}
