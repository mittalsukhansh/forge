package engine

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/mittalsukhansh/forge/internal/pipeline"
)

type Engine struct {
	StepTimeout time.Duration
}

func New() *Engine {
	return &Engine{
		StepTimeout: 5 * time.Minute,
	}
}

func containerName() (string, error) {
	b := make([]byte, 8)

	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return "forge-" + hex.EncodeToString(b), nil
}

func (e *Engine) RunStep(
	ctx context.Context,
	step pipeline.Step,
	output io.Writer,
) error {
	ctx, cancel := context.WithTimeout(ctx, e.StepTimeout)
	defer cancel()

	name, err := containerName()

	if err != nil {
		return err
	}

	defer func() {
		cleanup, stop := context.WithTimeout(
			context.Background(), 15*time.Second,
		)
		defer stop()

		exec.CommandContext(
			cleanup, "docker", "rm", "-f", name,
		).Run()
	}()

	fmt.Fprintf(output, "\n[STEP] %s\n", step.Name)
	fmt.Fprintf(output, "[IMAGE] %s\n", step.Image)

	script := "set -e\n" + strings.Join(step.Commands, "\n")

	cmd := exec.CommandContext(
		ctx,
		"docker", "run", "--rm",
		"--name", name,
		"--network", "none",
		"--cap-drop", "ALL",
		"--security-opt", "no-new-privileges",
		"--pids-limit", "128",
		"--entrypoint", "/bin/sh",
		step.Image,
		"-c", script,
	)

	cmd.Stdout = output
	cmd.Stderr = output

	started := time.Now()

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("step timed out or cancelled: %w", ctx.Err())
		}

		fmt.Fprintf(output, "[FAILED] %s\n", step.Name)

		return fmt.Errorf("step %q failed: %w", step.Name, err)
	}

	fmt.Fprintf(
		output,
		"[PASSED] %s (%s)\n",
		step.Name,
		time.Since(started).Round(time.Millisecond),
	)

	return nil
}

func (e *Engine) Run(
	ctx context.Context,
	p *pipeline.Pipeline,
	output io.Writer,
) error {
	started := time.Now()

	fmt.Fprintf(output, "[PIPELINE] %s\n", p.Name)
	fmt.Fprintf(output, "[STEPS] %d\n", len(p.Steps))

	for i, step := range p.Steps {
		fmt.Fprintf(output, "\n[%d/%d]\n", i+1, len(p.Steps))

		if err := e.RunStep(ctx, step, output); err != nil {
			fmt.Fprintln(output, "[PIPELINE FAILED]")
			return err
		}
	}

	fmt.Fprintf(
		output,
		"\n[PIPELINE SUCCESS] Duration: %s\n",
		time.Since(started).Round(time.Millisecond),
	)

	return nil
}
