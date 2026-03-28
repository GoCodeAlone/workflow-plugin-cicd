// Package internal implements the workflow-plugin-cicd external plugin,
// providing CI/CD pipeline step types and the aws.codebuild module type.
package internal

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// cicdPlugin implements sdk.PluginProvider.
type cicdPlugin struct{}

// NewCICDPlugin returns a new cicdPlugin instance.
func NewCICDPlugin() sdk.PluginProvider {
	return &cicdPlugin{}
}

// Manifest returns plugin metadata.
func (p *cicdPlugin) Manifest() sdk.PluginManifest {
	return sdk.PluginManifest{
		Name:        "workflow-plugin-cicd",
		Version:     "0.1.0",
		Author:      "GoCodeAlone",
		Description: "CI/CD pipeline steps: shell exec, Docker, artifact management, security scanning, git operations, AWS CodeBuild",
	}
}

// ModuleTypes returns the module type names this plugin provides.
func (p *cicdPlugin) ModuleTypes() []string {
	return []string{"aws.codebuild"}
}

// CreateModule creates a module instance of the given type.
func (p *cicdPlugin) CreateModule(typeName, name string, config map[string]any) (sdk.ModuleInstance, error) {
	switch typeName {
	case "aws.codebuild":
		return &codebuildModule{name: name, config: config}, nil
	default:
		return nil, fmt.Errorf("cicd plugin: unknown module type %q", typeName)
	}
}

// StepTypes returns the step type names this plugin provides.
func (p *cicdPlugin) StepTypes() []string {
	return []string{
		"step.shell_exec",
		"step.artifact_pull",
		"step.artifact_push",
		"step.docker_build",
		"step.docker_push",
		"step.docker_run",
		"step.scan_sast",
		"step.scan_container",
		"step.scan_deps",
		"step.gate",
		"step.build_ui",
		"step.build_from_config",
		"step.build_binary",
		"step.git_clone",
		"step.git_commit",
		"step.git_push",
		"step.git_tag",
		"step.git_checkout",
		"step.codebuild_create_project",
		"step.codebuild_start",
		"step.codebuild_status",
		"step.codebuild_logs",
		"step.codebuild_delete_project",
		"step.codebuild_list_builds",
	}
}

// CreateStep creates a step instance of the given type.
func (p *cicdPlugin) CreateStep(typeName, name string, config map[string]any) (sdk.StepInstance, error) {
	switch typeName {
	case "step.shell_exec":
		return &shellExecStep{name: name, config: config}, nil
	case "step.git_clone":
		return &gitStep{name: name, stepType: typeName, config: config}, nil
	case "step.git_commit":
		return &gitStep{name: name, stepType: typeName, config: config}, nil
	case "step.git_push":
		return &gitStep{name: name, stepType: typeName, config: config}, nil
	case "step.git_tag":
		return &gitStep{name: name, stepType: typeName, config: config}, nil
	case "step.git_checkout":
		return &gitStep{name: name, stepType: typeName, config: config}, nil
	case "step.artifact_pull", "step.artifact_push",
		"step.docker_build", "step.docker_push", "step.docker_run",
		"step.scan_sast", "step.scan_container", "step.scan_deps",
		"step.gate", "step.build_ui", "step.build_from_config", "step.build_binary",
		"step.codebuild_create_project", "step.codebuild_start", "step.codebuild_status",
		"step.codebuild_logs", "step.codebuild_delete_project", "step.codebuild_list_builds":
		return &stubStep{name: name, stepType: typeName, config: config}, nil
	default:
		return nil, fmt.Errorf("cicd plugin: unknown step type %q", typeName)
	}
}

// ─── Modules ─────────────────────────────────────────────────────────────────

// codebuildModule is a stub for aws.codebuild.
// TODO: Integrate with AWS CodeBuild SDK.
type codebuildModule struct {
	name   string
	config map[string]any
}

func (m *codebuildModule) Init() error { return nil }
func (m *codebuildModule) Start(_ context.Context) error { return nil }
func (m *codebuildModule) Stop(_ context.Context) error  { return nil }

// ─── Steps ───────────────────────────────────────────────────────────────────

// shellExecStep executes a shell command.
type shellExecStep struct {
	name   string
	config map[string]any
}

func (s *shellExecStep) Execute(
	ctx context.Context,
	_ map[string]any,
	_ map[string]map[string]any,
	_ map[string]any,
	_ map[string]any,
	_ map[string]any,
) (*sdk.StepResult, error) {
	command, _ := s.config["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("step.shell_exec %q: 'command' is required", s.name)
	}

	shell := "/bin/sh"
	if sh, _ := s.config["shell"].(string); sh != "" {
		shell = sh
	}

	cmd := exec.CommandContext(ctx, shell, "-c", command)
	if workdir, _ := s.config["workdir"].(string); workdir != "" {
		cmd.Dir = workdir
	}

	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			// Only fail if fail_on_error is not explicitly false
			failOnError := true
			if v, ok := s.config["fail_on_error"].(bool); ok {
				failOnError = v
			}
			if failOnError {
				return nil, fmt.Errorf("step.shell_exec %q: command failed (exit %d): %s", s.name, exitCode, string(out))
			}
		} else {
			return nil, fmt.Errorf("step.shell_exec %q: %w", s.name, err)
		}
	}

	return &sdk.StepResult{
		Output: map[string]any{
			"output":    strings.TrimSpace(string(out)),
			"exit_code": exitCode,
		},
	}, nil
}

// gitStep executes git operations.
// TODO: Use go-git library for proper git operations.
type gitStep struct {
	name     string
	stepType string
	config   map[string]any
}

func (s *gitStep) Execute(
	ctx context.Context,
	_ map[string]any,
	_ map[string]map[string]any,
	_ map[string]any,
	_ map[string]any,
	_ map[string]any,
) (*sdk.StepResult, error) {
	switch s.stepType {
	case "step.git_clone":
		repo, _ := s.config["repo"].(string)
		dest, _ := s.config["dest"].(string)
		if repo == "" {
			return nil, fmt.Errorf("step.git_clone %q: 'repo' is required", s.name)
		}
		args := []string{"clone", repo}
		if branch, _ := s.config["branch"].(string); branch != "" {
			args = append(args, "--branch", branch)
		}
		if dest != "" {
			args = append(args, dest)
		}
		out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("step.git_clone %q: %w: %s", s.name, err, string(out))
		}
		return &sdk.StepResult{Output: map[string]any{"cloned": true, "repo": repo, "dest": dest}}, nil

	case "step.git_commit":
		workdir, _ := s.config["workdir"].(string)
		message, _ := s.config["message"].(string)
		if message == "" {
			message = "automated commit"
		}
		cmd := exec.CommandContext(ctx, "git", "commit", "-m", message)
		if workdir != "" {
			cmd.Dir = workdir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("step.git_commit %q: %w: %s", s.name, err, string(out))
		}
		return &sdk.StepResult{Output: map[string]any{"committed": true, "message": message}}, nil

	case "step.git_push":
		workdir, _ := s.config["workdir"].(string)
		remote, _ := s.config["remote"].(string)
		if remote == "" {
			remote = "origin"
		}
		branch, _ := s.config["branch"].(string)
		args := []string{"push", remote}
		if branch != "" {
			args = append(args, branch)
		}
		cmd := exec.CommandContext(ctx, "git", args...)
		if workdir != "" {
			cmd.Dir = workdir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("step.git_push %q: %w: %s", s.name, err, string(out))
		}
		return &sdk.StepResult{Output: map[string]any{"pushed": true}}, nil

	case "step.git_tag":
		workdir, _ := s.config["workdir"].(string)
		tag, _ := s.config["tag"].(string)
		if tag == "" {
			return nil, fmt.Errorf("step.git_tag %q: 'tag' is required", s.name)
		}
		cmd := exec.CommandContext(ctx, "git", "tag", tag)
		if workdir != "" {
			cmd.Dir = workdir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("step.git_tag %q: %w: %s", s.name, err, string(out))
		}
		return &sdk.StepResult{Output: map[string]any{"tagged": true, "tag": tag}}, nil

	case "step.git_checkout":
		workdir, _ := s.config["workdir"].(string)
		ref, _ := s.config["ref"].(string)
		if ref == "" {
			ref, _ = s.config["branch"].(string)
		}
		if ref == "" {
			return nil, fmt.Errorf("step.git_checkout %q: 'ref' or 'branch' is required", s.name)
		}
		cmd := exec.CommandContext(ctx, "git", "checkout", ref)
		if workdir != "" {
			cmd.Dir = workdir
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("step.git_checkout %q: %w: %s", s.name, err, string(out))
		}
		return &sdk.StepResult{Output: map[string]any{"checked_out": ref}}, nil
	}

	return &sdk.StepResult{Output: map[string]any{"ok": true}}, nil
}

// stubStep is a stub for CI/CD steps not yet fully implemented.
// TODO: Implement Docker, artifact, scanning, build, and CodeBuild steps.
type stubStep struct {
	name     string
	stepType string
	config   map[string]any
}

func (s *stubStep) Execute(
	_ context.Context,
	_ map[string]any,
	_ map[string]map[string]any,
	_ map[string]any,
	_ map[string]any,
	_ map[string]any,
) (*sdk.StepResult, error) {
	return &sdk.StepResult{
		Output: map[string]any{
			"status":  "ok",
			"message": fmt.Sprintf("TODO: %s not yet implemented in external plugin", s.stepType),
		},
	}, nil
}
