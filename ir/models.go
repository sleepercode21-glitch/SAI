package ir

type ExposureKind string

const (
	ExposurePrivate    ExposureKind = "private"
	ExposurePublicHTTP ExposureKind = "public_http"
)

type ApplicationIR struct {
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Cloud     string `json:"cloud"`
	Region    string `json:"region"`
	Users     int    `json:"users"`
	BudgetUSD int    `json:"budget_usd"`
	Env       string `json:"env"`
}

type ServiceIR struct {
	Name            string       `json:"name"`
	Runtime         string       `json:"runtime"`
	Path            string       `json:"path"`
	Port            int          `json:"port"`
	Exposure        ExposureKind `json:"exposure"`
	HealthCheckPath string       `json:"health_check_path"`
	Connects        []string     `json:"connects"`
	ScaleHint       string       `json:"scale_hint"`
}

type ResourceIR struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Type string `json:"type"`
	Size string `json:"size"`
}

type ProgramIR struct {
	Application ApplicationIR `json:"application"`
	Service     ServiceIR     `json:"service"`
	Resources   []ResourceIR  `json:"resources"`
}
