package infra

import (
	"encoding/json"
	"fmt"
)

func generateGCPTerraformJSON(plan *Plan) ([]byte, error) {
	stack := slugify(fmt.Sprintf("%s-%s", plan.ApplicationName, plan.Environment))
	resources := map[string]map[string]any{
		"google_compute_network": {
			"main": map[string]any{
				"name":                    stack + "-vpc",
				"auto_create_subnetworks": false,
			},
		},
		"google_compute_subnetwork": {
			"app": map[string]any{
				"name":          stack + "-subnet",
				"ip_cidr_range": "10.0.1.0/24",
				"region":        plan.Region,
				"network":       "${google_compute_network.main.id}",
			},
		},
		"google_compute_firewall": {
			"service": map[string]any{
				"name":    stack + "-service",
				"network": "${google_compute_network.main.name}",
				"allow": []map[string]any{
					{
						"protocol": "tcp",
						"ports":    []int{plan.Network.ServicePort},
					},
				},
				"source_ranges": sourceRanges(plan.Network.InternetIngress),
			},
		},
		"google_cloud_run_v2_service": {
			"service": map[string]any{
				"name":     stack + "-" + plan.Compute.ServiceName,
				"location": plan.Region,
				"template": []map[string]any{
					{
						"service_account": "${google_service_account.runtime.email}",
						"scaling": []map[string]any{
							{
								"min_instance_count": plan.Compute.MinInstances,
								"max_instance_count": plan.Compute.MaxInstances,
							},
						},
						"containers": []map[string]any{
							{
								"image": fmt.Sprintf("gcr.io/project/%s/%s:latest", slugify(plan.ApplicationName), plan.Compute.ServiceName),
								"ports": []map[string]any{{"container_port": plan.Network.ServicePort}},
								"env":   gcpContainerEnvironment(plan),
							},
						},
					},
				},
			},
		},
		"google_service_account": {
			"runtime": map[string]any{
				"account_id":   truncateLabel(stack+"-runtime", 28),
				"display_name": stack + " runtime",
			},
		},
	}
	if len(plan.AppConfig.Secrets) > 0 {
		resources["google_secret_manager_secret"] = map[string]any{}
		resources["google_secret_manager_secret_iam_member"] = map[string]any{}
		for _, secret := range plan.AppConfig.Secrets {
			secretID := terraformName(secret.Name)
			resources["google_secret_manager_secret"][secretID] = map[string]any{
				"secret_id":   stack + "-" + secret.Name,
				"replication": []map[string]any{{"auto": []map[string]any{{}}}},
			}
			resources["google_secret_manager_secret_iam_member"][secretID] = map[string]any{
				"secret_id": "${google_secret_manager_secret." + secretID + ".secret_id}",
				"role":      "roles/secretmanager.secretAccessor",
				"member":    "serviceAccount:${google_service_account.runtime.email}",
			}
		}
	}

	if plan.Database != nil && plan.Database.Managed {
		resources["google_sql_database_instance"] = map[string]any{
			"main": map[string]any{
				"name":             stack + "-" + plan.Database.Name,
				"region":           plan.Region,
				"database_version": "POSTGRES_15",
				"settings": []map[string]any{
					{
						"tier": cloudSQLTier(plan.Database.Class),
					},
				},
			},
		}
	}

	document := terraformDocument{
		Terraform: terraformSettings{
			RequiredVersion: ">= 1.6.0",
			RequiredProviders: map[string]providerRequirement{
				"google": {Source: "hashicorp/google", Version: "~> 5.0"},
			},
		},
		Locals:   map[string]any{"stack_name": stack},
		Resource: resources,
		Output: map[string]outputValue{
			"service_name": {Value: "${google_cloud_run_v2_service.service.name}"},
			"network_name": {Value: "${google_compute_network.main.name}"},
		},
	}
	if plan.Database != nil && plan.Database.Managed {
		document.Output["database_connection_name"] = outputValue{Value: "${google_sql_database_instance.main.connection_name}"}
	}
	return json.MarshalIndent(document, "", "  ")
}

func gcpContainerEnvironment(plan *Plan) []map[string]any {
	env := make([]map[string]any, 0, len(plan.AppConfig.Environment))
	for _, item := range plan.AppConfig.Environment {
		entry := map[string]any{
			"name": item.Name,
		}
		if item.SecretRef != "" {
			entry["value_source"] = []map[string]any{
				{
					"secret_key_ref": []map[string]any{
						{
							"secret":  "${google_secret_manager_secret." + terraformName(item.SecretRef) + ".secret_id}",
							"version": "latest",
						},
					},
				},
			}
		} else {
			entry["value"] = item.Literal
		}
		env = append(env, entry)
	}
	return env
}

func truncateLabel(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
