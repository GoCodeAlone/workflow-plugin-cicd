// Command workflow-plugin-cicd is a workflow engine external plugin that
// provides CI/CD pipeline step types: shell exec, Docker, artifact management,
// security scanning, build steps, git operations, and AWS CodeBuild integration.
package main

import (
	"github.com/GoCodeAlone/workflow-plugin-cicd/internal"
	sdk "github.com/GoCodeAlone/workflow/plugin/external/sdk"
)

func main() {
	sdk.Serve(internal.NewCICDPlugin())
}
