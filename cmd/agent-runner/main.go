// Command agent-runner serves jobs for the profiles of one deployment: each
// job runs Claude Code headless in a fresh working folder with only the
// permissions its profile gives. See the README.
//
// Settings come from flags, falling back to environment variables so a
// container needs no arguments:
//
//	--profiles / AGENT_PROFILES   the profiles file (required)
//	--addr / PORT                 where to listen (default :8080)
//	--work-root / AGENT_WORK_ROOT where working folders are made (default: temp)
//	--claude / AGENT_CLAUDE       the Claude Code program (default: claude on PATH)
//
// The credential (CLAUDE_CODE_OAUTH_TOKEN, or ANTHROPIC_API_KEY) is read from
// the environment by Claude Code itself; the runner never looks at it or logs it.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/javadebadi/claude-headless-agent-runner/internal/api"
	"github.com/javadebadi/claude-headless-agent-runner/internal/job"
	"github.com/javadebadi/claude-headless-agent-runner/internal/profile"
)

// version is stamped at release time (-ldflags "-X main.version=v0.1.0").
var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "agent-runner:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) > 0 && (args[0] == "version" || args[0] == "--version") {
		fmt.Println(version)
		return nil
	}
	if len(args) > 0 && args[0] == "serve" { // optional, so `serve` works as a container command
		args = args[1:]
	}
	fs := flag.NewFlagSet("agent-runner", flag.ContinueOnError)
	profiles := fs.String("profiles", os.Getenv("AGENT_PROFILES"), "the profiles file")
	addr := fs.String("addr", "", "address to listen on (default :$PORT, or :8080)")
	workRoot := fs.String("work-root", os.Getenv("AGENT_WORK_ROOT"), "where working folders are made")
	claude := fs.String("claude", os.Getenv("AGENT_CLAUDE"), "the Claude Code program")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *profiles == "" {
		return errors.New("no profiles file: pass --profiles or set AGENT_PROFILES")
	}
	set, err := profile.Load(*profiles)
	if err != nil {
		return err
	}
	if *addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		*addr = ":" + port
	}
	runner := job.New(set)
	runner.Claude = *claude
	runner.WorkRoot = *workRoot

	srv := &http.Server{
		Addr:              *addr,
		Handler:           api.New(runner),
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		srv.Shutdown(shutdown)
	}()
	fmt.Printf("agent-runner %s listening on %s, profiles: %v\n", version, *addr, set.Names())
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
