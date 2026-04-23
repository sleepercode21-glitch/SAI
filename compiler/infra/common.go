package infra

import "strings"

type terraformDocument struct {
	Terraform terraformSettings         `json:"terraform"`
	Locals    map[string]any            `json:"locals,omitempty"`
	Resource  map[string]map[string]any `json:"resource"`
	Output    map[string]outputValue    `json:"output"`
}

type terraformSettings struct {
	RequiredVersion   string                         `json:"required_version"`
	RequiredProviders map[string]providerRequirement `json:"required_providers"`
}

type providerRequirement struct {
	Source  string `json:"source"`
	Version string `json:"version"`
}

type outputValue struct {
	Value any `json:"value"`
}

func sourceRanges(external bool) []string {
	if external {
		return []string{"0.0.0.0/0"}
	}
	return []string{"10.0.0.0/16"}
}

func cpuForClass(class string) string {
	switch class {
	case "managed-medium":
		return "1024"
	default:
		return "512"
	}
}

func memoryForClass(class string) string {
	switch class {
	case "managed-medium":
		return "2048"
	default:
		return "1024"
	}
}

func dbInstanceClass(class string) string {
	switch class {
	case "medium":
		return "db.t4g.medium"
	default:
		return "db.t4g.micro"
	}
}

func dbStorageForClass(class string) int {
	switch class {
	case "medium":
		return 50
	default:
		return 20
	}
}

func cloudSQLTier(class string) string {
	switch class {
	case "medium":
		return "db-g1-small"
	default:
		return "db-f1-micro"
	}
}

func azurePostgresSKU(class string) string {
	switch class {
	case "medium":
		return "Standard_B2s"
	default:
		return "Standard_B1ms"
	}
}

func maxPercent(maxInstances int) int {
	if maxInstances > 1 {
		return 200
	}
	return 100
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "_", "-")
	return value
}
