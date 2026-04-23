package ir

import (
	"fmt"
	"strings"

	"github.com/sleepercode/sai/ast"
	"github.com/sleepercode/sai/utils"
)

// Build converts the AST into the typed IR used by every downstream phase.
func Build(program *ast.Program) (*ProgramIR, error) {
	diagnostics := &utils.Diagnostics{}

	if program.App == nil {
		diagnostics.Add("ir", "an app declaration is required", program.Span)
		return nil, diagnostics
	}
	if len(program.Services) == 0 {
		diagnostics.Add("ir", "a service declaration is required for the MVP", program.Span)
		return nil, diagnostics
	}
	if len(program.Services) > 1 {
		diagnostics.Add("ir", "the MVP supports exactly one service", program.Services[1].Span)
		return nil, diagnostics
	}

	appIR := normalizeApp(program.App, diagnostics)
	serviceIR := normalizeService(program.Services[0], diagnostics)
	resourceIRs := normalizeResources(program.Resources, diagnostics)

	validateServiceReferences(serviceIR, resourceIRs, diagnostics)

	if diagnostics.HasErrors() {
		return nil, diagnostics
	}

	return &ProgramIR{
		Application: appIR,
		Service:     serviceIR,
		Resources:   resourceIRs,
	}, nil
}

func normalizeApp(node *ast.AppDecl, diagnostics *utils.Diagnostics) ApplicationIR {
	result := ApplicationIR{
		Name:      node.Name,
		Slug:      slugify(node.Name),
		Cloud:     "aws",
		Region:    "us-east-1",
		Users:     0,
		BudgetUSD: 0,
		Env:       "dev",
	}

	seen := map[string]bool{}
	for _, field := range node.Fields {
		if seen[field.Kind] {
			diagnostics.Add("ir", fmt.Sprintf("duplicate app field %q", field.Kind), field.Span)
			continue
		}
		seen[field.Kind] = true

		switch field.Kind {
		case "cloud":
			result.Cloud = field.Ident
		case "region":
			result.Region = field.String
		case "users":
			result.Users = field.Int
		case "budget":
			result.BudgetUSD = field.Money
		case "env":
			result.Env = field.Ident
		}
	}

	if result.Users <= 0 {
		diagnostics.Add("ir", "app.users must be greater than zero", node.Span)
	}
	if result.BudgetUSD <= 0 {
		diagnostics.Add("ir", "app.budget must be greater than zero", node.Span)
	}

	return result
}

func normalizeService(node *ast.ServiceDecl, diagnostics *utils.Diagnostics) ServiceIR {
	result := ServiceIR{
		Name:            node.Name,
		Runtime:         "node",
		Path:            "server",
		Port:            3000,
		Exposure:        ExposurePrivate,
		HealthCheckPath: "/health",
		Connects:        []string{},
		ScaleHint:       "balanced",
	}

	seen := map[string]bool{}
	for _, field := range node.Fields {
		if field.Kind != "connects" && seen[field.Kind] {
			diagnostics.Add("ir", fmt.Sprintf("duplicate service field %q", field.Kind), field.Span)
			continue
		}
		seen[field.Kind] = true

		switch field.Kind {
		case "runtime":
			result.Runtime = field.Ident
		case "path":
			result.Path = field.String
		case "port":
			result.Port = field.Int
		case "public":
			if field.Protocol != "http" {
				diagnostics.Add("ir", "only public http exposure is supported in the MVP", field.Span)
				continue
			}
			result.Exposure = ExposurePublicHTTP
		case "private":
			result.Exposure = ExposurePrivate
		case "connects":
			result.Connects = append(result.Connects, field.Idents...)
		case "scale":
			result.ScaleHint = field.Ident
		case "health":
			result.HealthCheckPath = field.String
		}
	}

	if result.Port <= 0 || result.Port > 65535 {
		diagnostics.Add("ir", "service.port must be between 1 and 65535", node.Span)
	}
	if result.Runtime == "" {
		diagnostics.Add("ir", "service.runtime cannot be empty", node.Span)
	}

	return result
}

func normalizeResources(nodes []*ast.ResourceDecl, diagnostics *utils.Diagnostics) []ResourceIR {
	results := make([]ResourceIR, 0, len(nodes))
	names := map[string]bool{}

	for _, node := range nodes {
		if names[node.Name] {
			diagnostics.Add("ir", fmt.Sprintf("duplicate resource %q", node.Name), node.Span)
			continue
		}
		names[node.Name] = true

		resource := ResourceIR{
			Name: node.Name,
			Kind: node.Kind,
			Type: "managed",
			Size: "small",
		}

		seen := map[string]bool{}
		for _, field := range node.Fields {
			if seen[field.Kind] {
				diagnostics.Add("ir", fmt.Sprintf("duplicate resource field %q", field.Kind), field.Span)
				continue
			}
			seen[field.Kind] = true

			switch field.Kind {
			case "type":
				resource.Type = field.Ident
			case "size":
				resource.Size = field.Ident
			}
		}

		results = append(results, resource)
	}

	return results
}

func validateServiceReferences(service ServiceIR, resources []ResourceIR, diagnostics *utils.Diagnostics) {
	index := map[string]bool{}
	for _, resource := range resources {
		index[resource.Name] = true
	}
	for _, dep := range service.Connects {
		if !index[dep] {
			diagnostics.Add("ir", fmt.Sprintf("service references unknown resource %q", dep), ast.Span{})
		}
	}
}

func slugify(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, " ", "-")
	normalized = strings.ReplaceAll(normalized, "_", "-")
	return normalized
}
