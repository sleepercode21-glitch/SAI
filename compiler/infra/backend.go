package infra

import "fmt"

// GenerateArtifact emits the provider-native infrastructure artifact for the plan.
func GenerateArtifact(plan *Plan) (*Artifact, error) {
	switch plan.Cloud {
	case "aws":
		content, err := generateAWSTerraformJSON(plan)
		if err != nil {
			return nil, err
		}
		return &Artifact{Format: ArtifactFormatTerraformJSON, Content: string(content)}, nil
	case "azure":
		content, err := generateAzureBicep(plan)
		if err != nil {
			return nil, err
		}
		return &Artifact{Format: ArtifactFormatBicep, Content: content}, nil
	case "gcp":
		content, err := generateGCPTerraformJSON(plan)
		if err != nil {
			return nil, err
		}
		return &Artifact{Format: ArtifactFormatTerraformJSON, Content: string(content)}, nil
	default:
		return nil, fmt.Errorf("infra codegen does not support cloud %q", plan.Cloud)
	}
}

// GenerateTerraformJSON emits Terraform JSON only for Terraform-backed clouds.
func GenerateTerraformJSON(plan *Plan) ([]byte, error) {
	switch plan.Cloud {
	case "aws":
		return generateAWSTerraformJSON(plan)
	case "gcp":
		return generateGCPTerraformJSON(plan)
	default:
		return nil, fmt.Errorf("terraform json output is not available for cloud %q", plan.Cloud)
	}
}
