package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	deployartifact "github.com/sleepercode/sai/compiler/deploy"
)

type PreflightCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

func Preflight(_ context.Context, bundle *deployartifact.Bundle) ([]PreflightCheck, error) {
	switch bundle.Provider {
	case "azure":
		return azureChecks(bundle), nil
	case "aws":
		return awsChecks(bundle), nil
	case "gcp":
		return gcpChecks(bundle), nil
	default:
		return nil, fmt.Errorf("preflight does not support provider %q", bundle.Provider)
	}
}

func azureChecks(bundle *deployartifact.Bundle) []PreflightCheck {
	checks := lookupCommands(bundle.Provider, []string{"bash", "az"})
	checks = append(checks,
		requireBundleFile(bundle, "deploy/azure/deploy.sh"),
		requireBundleFile(bundle, "deploy/azure/main.bicep"),
	)

	authConfigured := hasAnyEnv("AZURE_CLIENT_ID", "AZURE_TENANT_ID", "AZURE_SUBSCRIPTION_ID") || hasAnyEnv("AZURE_USE_AZ_LOGIN")
	if authConfigured {
		checks = append(checks, PreflightCheck{
			Name:   "azure-auth",
			Status: "ok",
			Detail: "found Azure credential environment configuration",
		})
	} else {
		checks = append(checks, PreflightCheck{
			Name:   "azure-auth",
			Status: "missing",
			Detail: "set AZURE_CLIENT_ID, AZURE_TENANT_ID, and AZURE_SUBSCRIPTION_ID, or export AZURE_USE_AZ_LOGIN=1",
		})
	}
	return checks
}

func awsChecks(bundle *deployartifact.Bundle) []PreflightCheck {
	checks := lookupCommands(bundle.Provider, []string{"bash", "terraform", "aws", "docker"})
	checks = append(checks,
		requireBundleFile(bundle, "deploy/terraform/deploy.sh"),
		requireBundleFile(bundle, "deploy/terraform/main.tf.json"),
	)
	if hasAnyEnv("AWS_PROFILE", "AWS_ACCESS_KEY_ID", "AWS_WEB_IDENTITY_TOKEN_FILE") {
		checks = append(checks, PreflightCheck{
			Name:   "aws-auth",
			Status: "ok",
			Detail: "found AWS credential environment configuration",
		})
	} else {
		checks = append(checks, PreflightCheck{
			Name:   "aws-auth",
			Status: "missing",
			Detail: "set AWS_PROFILE, AWS_ACCESS_KEY_ID, or AWS_WEB_IDENTITY_TOKEN_FILE before execution",
		})
	}
	return checks
}

func gcpChecks(bundle *deployartifact.Bundle) []PreflightCheck {
	checks := lookupCommands(bundle.Provider, []string{"bash", "terraform", "gcloud", "docker"})
	checks = append(checks,
		requireBundleFile(bundle, "deploy/terraform/deploy.sh"),
		requireBundleFile(bundle, "deploy/terraform/main.tf.json"),
	)
	if hasAnyEnv("GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_CLOUD_PROJECT", "CLOUDSDK_CONFIG") {
		checks = append(checks, PreflightCheck{
			Name:   "gcp-auth",
			Status: "ok",
			Detail: "found GCP credential environment configuration",
		})
	} else {
		checks = append(checks, PreflightCheck{
			Name:   "gcp-auth",
			Status: "missing",
			Detail: "set GOOGLE_APPLICATION_CREDENTIALS, GOOGLE_CLOUD_PROJECT, or CLOUDSDK_CONFIG before execution",
		})
	}
	return checks
}

func lookupCommands(provider string, commands []string) []PreflightCheck {
	results := make([]PreflightCheck, 0, len(commands))
	for _, name := range commands {
		path, err := exec.LookPath(name)
		if err != nil {
			results = append(results, PreflightCheck{
				Name:   name,
				Status: "missing",
				Detail: fmt.Sprintf("%s deployment requires %s to be installed", provider, name),
			})
			continue
		}
		results = append(results, PreflightCheck{
			Name:   name,
			Status: "ok",
			Detail: path,
		})
	}
	return results
}

func requireBundleFile(bundle *deployartifact.Bundle, path string) PreflightCheck {
	content, ok := bundle.Files[path]
	if !ok {
		return PreflightCheck{
			Name:   path,
			Status: "missing",
			Detail: fmt.Sprintf("%s bundle is missing %s", bundle.Provider, path),
		}
	}
	if strings.TrimSpace(content) == "" {
		return PreflightCheck{
			Name:   path,
			Status: "invalid",
			Detail: fmt.Sprintf("%s bundle contains an empty %s", bundle.Provider, path),
		}
	}
	return PreflightCheck{
		Name:   path,
		Status: "ok",
		Detail: "bundle file present",
	}
}

func hasAnyEnv(names ...string) bool {
	for _, name := range names {
		if value := strings.TrimSpace(getenv(name)); value != "" {
			return true
		}
	}
	return false
}

var getenv = func(key string) string {
	return os.Getenv(key)
}
