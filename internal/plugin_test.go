package internal_test

import (
	"context"
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/GoCodeAlone/workflow-plugin-cicd/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

// TestNewCICDPlugin verifies that NewCICDPlugin returns a non-nil provider that
// satisfies the required SDK interfaces.
func TestNewCICDPlugin(t *testing.T) {
	p := internal.NewCICDPlugin()
	if p == nil {
		t.Fatal("NewCICDPlugin() returned nil")
	}

	m := p.Manifest()
	if m.Name != "workflow-plugin-cicd" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "workflow-plugin-cicd")
	}
	if m.Author != "GoCodeAlone" {
		t.Errorf("Manifest().Author = %q, want %q", m.Author, "GoCodeAlone")
	}
	if m.Description == "" {
		t.Error("Manifest().Description is empty")
	}
}

// TestModuleProvider verifies that the plugin implements ModuleProvider and
// returns the expected module types.
func TestModuleProvider(t *testing.T) {
	p := internal.NewCICDPlugin()
	mp, ok := p.(sdk.ModuleProvider)
	if !ok {
		t.Fatal("plugin does not implement sdk.ModuleProvider")
	}

	types := mp.ModuleTypes()
	if len(types) == 0 {
		t.Fatal("ModuleTypes() returned empty slice")
	}

	wantTypes := []string{"aws.codebuild"}
	for _, want := range wantTypes {
		found := false
		for _, got := range types {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("module type %q not found in ModuleTypes()", want)
		}
	}
}

// TestCreateModule verifies that CreateModule returns a valid ModuleInstance for
// known types and an error for unknown types.
func TestCreateModule(t *testing.T) {
	p := internal.NewCICDPlugin()
	mp, ok := p.(sdk.ModuleProvider)
	if !ok {
		t.Fatal("plugin does not implement sdk.ModuleProvider")
	}

	inst, err := mp.CreateModule("aws.codebuild", "my-codebuild", map[string]any{
		"region": "us-east-1",
	})
	if err != nil {
		t.Fatalf("CreateModule(aws.codebuild) error: %v", err)
	}
	if inst == nil {
		t.Fatal("CreateModule(aws.codebuild) returned nil instance")
	}

	// Init / Start / Stop must not error for the stub module.
	if err := inst.Init(); err != nil {
		t.Errorf("Init() error: %v", err)
	}
	if err := inst.Start(context.Background()); err != nil {
		t.Errorf("Start() error: %v", err)
	}
	if err := inst.Stop(context.Background()); err != nil {
		t.Errorf("Stop() error: %v", err)
	}

	// Unknown module type must return an error.
	_, err = mp.CreateModule("unknown.type", "x", nil)
	if err == nil {
		t.Error("CreateModule(unknown.type) expected error, got nil")
	}
}

// TestSchemaProvider verifies that the plugin implements SchemaProvider and
// declares a schema for every module type it advertises.
func TestSchemaProvider(t *testing.T) {
	p := internal.NewCICDPlugin()
	sp, ok := p.(sdk.SchemaProvider)
	if !ok {
		t.Fatal("plugin does not implement sdk.SchemaProvider")
	}

	schemas := sp.ModuleSchemas()
	if len(schemas) == 0 {
		t.Fatal("ModuleSchemas() returned empty slice")
	}

	// Every module type returned by ModuleTypes must have a matching schema.
	mp := p.(sdk.ModuleProvider)
	for _, modType := range mp.ModuleTypes() {
		found := false
		for _, s := range schemas {
			if s.Type == modType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("no schema declared for module type %q", modType)
		}
	}

	// aws.codebuild schema must have at least one config field.
	for _, s := range schemas {
		if s.Type == "aws.codebuild" {
			if len(s.ConfigFields) == 0 {
				t.Error("aws.codebuild schema has no config fields")
			}
			if s.Description == "" {
				t.Error("aws.codebuild schema has no description")
			}
		}
	}
}

// TestStepProvider verifies that the plugin implements StepProvider and
// returns all expected step types.
func TestStepProvider(t *testing.T) {
	p := internal.NewCICDPlugin()
	sp, ok := p.(sdk.StepProvider)
	if !ok {
		t.Fatal("plugin does not implement sdk.StepProvider")
	}

	types := sp.StepTypes()
	if len(types) == 0 {
		t.Fatal("StepTypes() returned empty slice")
	}

	expectedSteps := []string{
		"step.shell_exec",
		"step.artifact_pull", "step.artifact_push",
		"step.docker_build", "step.docker_push", "step.docker_run",
		"step.scan_sast", "step.scan_container", "step.scan_deps",
		"step.gate",
		"step.build_ui", "step.build_from_config", "step.build_binary",
		"step.git_clone", "step.git_commit", "step.git_push", "step.git_tag", "step.git_checkout",
		"step.codebuild_create_project", "step.codebuild_start", "step.codebuild_status",
		"step.codebuild_logs", "step.codebuild_delete_project", "step.codebuild_list_builds",
		"step.deploy", "step.deploy_rolling", "step.deploy_blue_green",
		"step.deploy_canary", "step.deploy_verify", "step.deploy_rollback",
		"step.container_build",
	}

	typeSet := make(map[string]bool, len(types))
	for _, t := range types {
		typeSet[t] = true
	}
	for _, want := range expectedSteps {
		if !typeSet[want] {
			t.Errorf("step type %q not found in StepTypes()", want)
		}
	}
}

// TestCreateStep verifies that CreateStep returns a valid StepInstance for all
// declared step types and returns an error for unknown types.
func TestCreateStep(t *testing.T) {
	p := internal.NewCICDPlugin()
	sp, ok := p.(sdk.StepProvider)
	if !ok {
		t.Fatal("plugin does not implement sdk.StepProvider")
	}

	for _, stepType := range sp.StepTypes() {
		inst, err := sp.CreateStep(stepType, "test-step", map[string]any{})
		if err != nil {
			t.Errorf("CreateStep(%q) unexpected error: %v", stepType, err)
			continue
		}
		if inst == nil {
			t.Errorf("CreateStep(%q) returned nil instance", stepType)
		}
	}

	_, err := sp.CreateStep("step.unknown", "x", nil)
	if err == nil {
		t.Error("CreateStep(step.unknown) expected error, got nil")
	}
}

// TestShellExecStep verifies that the shell_exec step executes a command and
// returns combined output and exit code.
func TestShellExecStep(t *testing.T) {
	p := internal.NewCICDPlugin()
	sp := p.(sdk.StepProvider)

	inst, err := sp.CreateStep("step.shell_exec", "test-shell", map[string]any{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatalf("CreateStep error: %v", err)
	}

	result, err := inst.Execute(context.Background(), nil, nil, nil, nil, map[string]any{
		"command": "echo hello",
	})
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if result == nil {
		t.Fatal("Execute returned nil result")
	}
	output, ok := result.Output["output"].(string)
	if !ok {
		t.Fatalf("output type = %T, want string", result.Output["output"])
	}
	if output != "hello" {
		t.Errorf("output = %q, want %q", output, "hello")
	}
	exitCode, ok := result.Output["exit_code"].(int)
	if !ok {
		t.Fatalf("exit_code type = %T, want int", result.Output["exit_code"])
	}
	if exitCode != 0 {
		t.Errorf("exit_code = %d, want 0", exitCode)
	}
}

// TestShellExecStepMissingCommand verifies that shell_exec returns an error
// when the required 'command' config field is absent.
func TestShellExecStepMissingCommand(t *testing.T) {
	p := internal.NewCICDPlugin()
	sp := p.(sdk.StepProvider)

	inst, err := sp.CreateStep("step.shell_exec", "test-shell", map[string]any{})
	if err != nil {
		t.Fatalf("CreateStep error: %v", err)
	}

	_, err = inst.Execute(context.Background(), nil, nil, nil, nil, map[string]any{})
	if err == nil {
		t.Error("Execute with missing command expected error, got nil")
	}
}

// TestShellExecStepFailOnError verifies that shell_exec respects fail_on_error=false.
func TestShellExecStepFailOnError(t *testing.T) {
	p := internal.NewCICDPlugin()
	sp := p.(sdk.StepProvider)

	inst, err := sp.CreateStep("step.shell_exec", "test-shell", map[string]any{
		"command":       "exit 1",
		"fail_on_error": false,
	})
	if err != nil {
		t.Fatalf("CreateStep error: %v", err)
	}

	result, err := inst.Execute(context.Background(), nil, nil, nil, nil, map[string]any{
		"command":       "exit 1",
		"fail_on_error": false,
	})
	if err != nil {
		t.Fatalf("Execute with fail_on_error=false should not error, got: %v", err)
	}
	exitCode, ok := result.Output["exit_code"].(int)
	if !ok {
		t.Fatalf("exit_code type = %T, want int", result.Output["exit_code"])
	}
	if exitCode != 1 {
		t.Errorf("exit_code = %d, want 1", exitCode)
	}
}

// TestDeploySteps verifies that all deploy step types execute without error and
// return the expected output keys.
func TestDeploySteps(t *testing.T) {
	deployTypes := []string{
		"step.deploy",
		"step.deploy_rolling",
		"step.deploy_blue_green",
		"step.deploy_canary",
		"step.deploy_verify",
		"step.deploy_rollback",
		"step.container_build",
	}

	p := internal.NewCICDPlugin()
	sp := p.(sdk.StepProvider)

	for _, stepType := range deployTypes {
		t.Run(stepType, func(t *testing.T) {
			inst, err := sp.CreateStep(stepType, "test-deploy", map[string]any{
				"service": "my-service",
				"image":   "myimage:latest",
			})
			if err != nil {
				t.Fatalf("CreateStep error: %v", err)
			}

			result, err := inst.Execute(context.Background(), nil, nil, nil, nil, map[string]any{
				"service": "my-service",
				"image":   "myimage:latest",
			})
			if err != nil {
				t.Fatalf("Execute error: %v", err)
			}
			if result == nil {
				t.Fatal("Execute returned nil result")
			}
			if _, ok := result.Output["status"]; !ok {
				t.Error("output missing 'status' key")
			}
		})
	}
}

// TestPluginJSONStepSchemas verifies that plugin.json declares a stepSchema for
// every step type the plugin advertises, ensuring strict contract coverage.
func TestPluginJSONStepSchemas(t *testing.T) {
	data, err := os.ReadFile("../plugin.json")
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}

	var manifest struct {
		Capabilities struct {
			StepTypes []string `json:"stepTypes"`
		} `json:"capabilities"`
		StepSchemas []struct {
			Type string `json:"type"`
		} `json:"stepSchemas"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse plugin.json: %v", err)
	}

	schemaTypes := make(map[string]bool, len(manifest.StepSchemas))
	for _, s := range manifest.StepSchemas {
		schemaTypes[s.Type] = true
	}

	for _, stepType := range manifest.Capabilities.StepTypes {
		if !schemaTypes[stepType] {
			t.Errorf("plugin.json is missing stepSchema for step type %q", stepType)
		}
	}

	if len(manifest.StepSchemas) == 0 {
		t.Error("plugin.json has no stepSchemas")
	}
}

// TestPluginContractsJSON verifies that plugin.contracts.json is valid JSON
// and declares a contract for every module and step type.
func TestPluginContractsJSON(t *testing.T) {
	data, err := os.ReadFile("../plugin.contracts.json")
	if err != nil {
		t.Fatalf("read plugin.contracts.json: %v", err)
	}

	var contracts struct {
		Version string `json:"version"`
		Plugin  string `json:"plugin"`
		Strict  bool   `json:"strict"`
		Modules []struct {
			Type string `json:"type"`
		} `json:"modules"`
		Steps []struct {
			Type string `json:"type"`
		} `json:"steps"`
	}
	if err := json.Unmarshal(data, &contracts); err != nil {
		t.Fatalf("parse plugin.contracts.json: %v", err)
	}

	if contracts.Version == "" {
		t.Error("contracts.version is empty")
	}
	if !contracts.Strict {
		t.Error("contracts.strict must be true")
	}

	// Verify modules coverage.
	p := internal.NewCICDPlugin()
	mp := p.(sdk.ModuleProvider)
	moduleSet := make(map[string]bool, len(contracts.Modules))
	for _, m := range contracts.Modules {
		moduleSet[m.Type] = true
	}
	for _, modType := range mp.ModuleTypes() {
		if !moduleSet[modType] {
			t.Errorf("plugin.contracts.json missing module descriptor for %q", modType)
		}
	}

	// Verify steps coverage.
	sp := p.(sdk.StepProvider)
	stepSet := make(map[string]bool, len(contracts.Steps))
	for _, s := range contracts.Steps {
		stepSet[s.Type] = true
	}
	for _, stepType := range sp.StepTypes() {
		if !stepSet[stepType] {
			t.Errorf("plugin.contracts.json missing step descriptor for %q", stepType)
		}
	}
}

// TestStepTypesConsistency verifies that the step types declared in plugin.json
// match exactly what the plugin binary advertises via StepProvider.
func TestStepTypesConsistency(t *testing.T) {
	data, err := os.ReadFile("../plugin.json")
	if err != nil {
		t.Fatalf("read plugin.json: %v", err)
	}

	var manifest struct {
		Capabilities struct {
			StepTypes []string `json:"stepTypes"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("parse plugin.json: %v", err)
	}

	p := internal.NewCICDPlugin()
	sp := p.(sdk.StepProvider)
	codeTypes := sp.StepTypes()

	sort.Strings(manifest.Capabilities.StepTypes)
	sort.Strings(codeTypes)

	jsonSet := make(map[string]bool, len(manifest.Capabilities.StepTypes))
	for _, t := range manifest.Capabilities.StepTypes {
		jsonSet[t] = true
	}
	codeSet := make(map[string]bool, len(codeTypes))
	for _, t := range codeTypes {
		codeSet[t] = true
	}

	for _, stepType := range codeTypes {
		if !jsonSet[stepType] {
			t.Errorf("step type %q is in plugin code but missing from plugin.json capabilities", stepType)
		}
	}
	for _, stepType := range manifest.Capabilities.StepTypes {
		if !codeSet[stepType] {
			t.Errorf("step type %q is in plugin.json capabilities but missing from plugin code", stepType)
		}
	}
}
