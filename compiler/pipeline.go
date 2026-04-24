package compiler

import (
	"os"

	"github.com/sleepercode/sai/ast"
	buildartifact "github.com/sleepercode/sai/compiler/build"
	deployartifact "github.com/sleepercode/sai/compiler/deploy"
	infraplan "github.com/sleepercode/sai/compiler/infra"
	"github.com/sleepercode/sai/ir"
	"github.com/sleepercode/sai/parser"
	"github.com/sleepercode/sai/planner"
)

type Result struct {
	AST           *ast.Program            `json:"ast"`
	IR            *ir.ProgramIR           `json:"ir"`
	Plan          *planner.Plan           `json:"plan,omitempty"`
	Lowered       *LoweredPlans           `json:"lowered,omitempty"`
	InfraArtifact *infraplan.Artifact     `json:"infra_artifact,omitempty"`
	BuildArtifact *buildartifact.Artifact `json:"build_artifact,omitempty"`
	DeployBundle  *deployartifact.Bundle  `json:"deploy_bundle,omitempty"`
	TerraformJSON string                  `json:"terraform_json,omitempty"`
}

func CompileFile(path string) (*Result, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	programAST, err := parser.Parse(string(source))
	if err != nil {
		return nil, err
	}

	programIR, err := ir.Build(programAST)
	if err != nil {
		return nil, err
	}

	return &Result{
		AST: programAST,
		IR:  programIR,
	}, nil
}

func PlanFile(path string) (*Result, error) {
	result, err := CompileFile(path)
	if err != nil {
		return nil, err
	}

	plan, err := planner.Build(result.IR)
	if err != nil {
		return nil, err
	}
	result.Plan = plan

	lowered, err := Lower(result.IR, result.Plan)
	if err != nil {
		return nil, err
	}
	result.Lowered = lowered
	if err := validateLowered(result); err != nil {
		return nil, err
	}

	infraArtifact, err := infraplan.GenerateArtifact(result.Lowered.Infra)
	if err != nil {
		return nil, err
	}
	result.InfraArtifact = infraArtifact
	if infraArtifact.Format == infraplan.ArtifactFormatTerraformJSON {
		result.TerraformJSON = infraArtifact.Content
	}

	dockerfileArtifact, err := buildartifact.GenerateDockerfile(result.Lowered.Build)
	if err != nil {
		return nil, err
	}
	result.BuildArtifact = dockerfileArtifact

	deployBundle, err := deployartifact.GenerateBundle(result.Lowered.Deploy, result.Lowered.Infra, result.InfraArtifact, result.BuildArtifact)
	if err != nil {
		return nil, err
	}
	result.DeployBundle = deployBundle
	return result, nil
}
