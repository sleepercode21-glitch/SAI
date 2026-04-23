package infra

type ArtifactFormat string

const (
	ArtifactFormatTerraformJSON ArtifactFormat = "terraform-json"
	ArtifactFormatBicep         ArtifactFormat = "bicep"
)

type Artifact struct {
	Format  ArtifactFormat `json:"format"`
	Content string         `json:"content"`
}
