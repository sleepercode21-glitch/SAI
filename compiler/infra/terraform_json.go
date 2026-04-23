package infra

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

// GenerateTerraformJSON emits deterministic Terraform JSON from the infra plan.
func GenerateTerraformJSON(plan *Plan) ([]byte, error) {
	if plan.Cloud != "aws" {
		return nil, fmt.Errorf("infra codegen currently supports only aws, got %q", plan.Cloud)
	}

	stack := slugify(fmt.Sprintf("%s-%s", plan.ApplicationName, plan.Environment))
	resources := map[string]map[string]any{
		"aws_vpc": {
			"main": map[string]any{
				"cidr_block":           "10.0.0.0/16",
				"enable_dns_hostnames": true,
				"tags": map[string]any{
					"Name":  stack + "-vpc",
					"Stack": stack,
				},
			},
		},
		"aws_subnet": {
			"app": map[string]any{
				"vpc_id":                  "${aws_vpc.main.id}",
				"cidr_block":              "10.0.1.0/24",
				"map_public_ip_on_launch": plan.Network.InternetIngress,
				"availability_zone":       plan.Region + "a",
				"tags": map[string]any{
					"Name":  stack + "-subnet-app",
					"Stack": stack,
				},
			},
		},
		"aws_security_group": {
			"service": map[string]any{
				"name":        stack + "-service",
				"description": "Managed service security group",
				"vpc_id":      "${aws_vpc.main.id}",
				"ingress":     serviceIngress(plan),
				"egress": []map[string]any{
					{
						"from_port":   0,
						"to_port":     0,
						"protocol":    "-1",
						"cidr_blocks": []string{"0.0.0.0/0"},
					},
				},
				"tags": map[string]any{
					"Name":  stack + "-service-sg",
					"Stack": stack,
				},
			},
		},
		"aws_ecs_cluster": {
			"main": map[string]any{
				"name": stack,
				"setting": []map[string]any{
					{
						"name":  "containerInsights",
						"value": "enabled",
					},
				},
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		},
		"aws_cloudwatch_log_group": {
			"service": map[string]any{
				"name":              "/sai/" + stack + "/" + plan.Compute.ServiceName,
				"retention_in_days": 14,
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		},
		"aws_ecs_task_definition": {
			"service": map[string]any{
				"family":                   stack + "-" + plan.Compute.ServiceName,
				"network_mode":             "awsvpc",
				"requires_compatibilities": []string{"FARGATE"},
				"cpu":                      cpuForClass(plan.Compute.InfraClass),
				"memory":                   memoryForClass(plan.Compute.InfraClass),
				"container_definitions":    taskDefinitionJSON(plan, stack),
			},
		},
		"aws_ecs_service": {
			"service": map[string]any{
				"name":            stack + "-" + plan.Compute.ServiceName,
				"cluster":         "${aws_ecs_cluster.main.id}",
				"task_definition": "${aws_ecs_task_definition.service.arn}",
				"desired_count":   plan.Compute.MinInstances,
				"launch_type":     "FARGATE",
				"network_configuration": map[string]any{
					"subnets":          []string{"${aws_subnet.app.id}"},
					"security_groups":  []string{"${aws_security_group.service.id}"},
					"assign_public_ip": plan.Network.InternetIngress,
				},
				"deployment_minimum_healthy_percent": 100,
				"deployment_maximum_percent":         maxPercent(plan.Compute.MaxInstances),
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		},
	}

	if plan.Database != nil && plan.Database.Managed {
		resources["aws_db_subnet_group"] = map[string]any{
			"main": map[string]any{
				"name":       stack + "-db-subnets",
				"subnet_ids": []string{"${aws_subnet.app.id}"},
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		}
		resources["aws_db_instance"] = map[string]any{
			"main": map[string]any{
				"identifier":             stack + "-" + plan.Database.Name,
				"engine":                 plan.Database.Engine,
				"instance_class":         dbInstanceClass(plan.Database.Class),
				"allocated_storage":      dbStorageForClass(plan.Database.Class),
				"db_subnet_group_name":   "${aws_db_subnet_group.main.name}",
				"vpc_security_group_ids": []string{"${aws_security_group.service.id}"},
				"username":               "app",
				"password":               "change-me-in-secrets",
				"skip_final_snapshot":    true,
				"publicly_accessible":    false,
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		}
	}

	document := terraformDocument{
		Terraform: terraformSettings{
			RequiredVersion: ">= 1.6.0",
			RequiredProviders: map[string]providerRequirement{
				"aws": {
					Source:  "hashicorp/aws",
					Version: "~> 5.0",
				},
			},
		},
		Locals: map[string]any{
			"stack_name": stack,
		},
		Resource: resources,
		Output: map[string]outputValue{
			"cluster_name": {
				Value: "${aws_ecs_cluster.main.name}",
			},
			"service_name": {
				Value: "${aws_ecs_service.service.name}",
			},
		},
	}

	if plan.Database != nil && plan.Database.Managed {
		document.Output["database_endpoint"] = outputValue{
			Value: "${aws_db_instance.main.address}",
		}
	}

	return json.MarshalIndent(document, "", "  ")
}

func serviceIngress(plan *Plan) []map[string]any {
	if !plan.Network.InternetIngress {
		return []map[string]any{}
	}
	return []map[string]any{
		{
			"from_port":   plan.Network.ServicePort,
			"to_port":     plan.Network.ServicePort,
			"protocol":    "tcp",
			"cidr_blocks": []string{"0.0.0.0/0"},
		},
	}
}

func taskDefinitionJSON(plan *Plan, stack string) string {
	image := fmt.Sprintf("%s/%s:latest", slugify(plan.ApplicationName), plan.Compute.ServiceName)
	definition := []map[string]any{
		{
			"name":      plan.Compute.ServiceName,
			"image":     image,
			"essential": true,
			"portMappings": []map[string]any{
				{
					"containerPort": plan.Network.ServicePort,
					"hostPort":      plan.Network.ServicePort,
					"protocol":      "tcp",
				},
			},
			"healthCheck": map[string]any{
				"command": []string{
					"CMD-SHELL",
					fmt.Sprintf("curl -f http://localhost:%d%s || exit 1", plan.Network.ServicePort, plan.Compute.HealthCheckPath),
				},
				"interval":    30,
				"timeout":     5,
				"retries":     3,
				"startPeriod": 10,
			},
			"logConfiguration": map[string]any{
				"logDriver": "awslogs",
				"options": map[string]any{
					"awslogs-group":         "${aws_cloudwatch_log_group.service.name}",
					"awslogs-region":        plan.Region,
					"awslogs-stream-prefix": stack,
				},
			},
		},
	}

	data, _ := json.Marshal(definition)
	return string(data)
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
