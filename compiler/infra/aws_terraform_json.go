package infra

import (
	"encoding/json"
	"fmt"
)

func generateAWSTerraformJSON(plan *Plan) ([]byte, error) {
	stack := slugify(fmt.Sprintf("%s-%s", plan.ApplicationName, plan.Environment))
	resources := map[string]map[string]any{
		"aws_ecr_repository": {
			"service": map[string]any{
				"name":                 fmt.Sprintf("%s/%s", slugify(plan.ApplicationName), plan.Compute.ServiceName),
				"image_tag_mutability": "MUTABLE",
				"image_scanning_configuration": map[string]any{
					"scan_on_push": true,
				},
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		},
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
				"ingress":     awsServiceIngress(plan),
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
		"aws_iam_role": {
			"task_execution": map[string]any{
				"name":               stack + "-task-execution",
				"assume_role_policy": ecsTaskAssumeRolePolicy(),
				"tags": map[string]any{
					"Stack": stack,
				},
			},
			"task": map[string]any{
				"name":               stack + "-task",
				"assume_role_policy": ecsTaskAssumeRolePolicy(),
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		},
		"aws_iam_role_policy_attachment": {
			"task_execution": map[string]any{
				"role":       "${aws_iam_role.task_execution.name}",
				"policy_arn": "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy",
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
				"execution_role_arn":       "${aws_iam_role.task_execution.arn}",
				"task_role_arn":            "${aws_iam_role.task.arn}",
				"container_definitions":    awsTaskDefinitionJSON(plan, stack),
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
				"depends_on":                         awsServiceDependsOn(plan),
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		},
	}

	if plan.Network.InternetIngress {
		resources["aws_lb"] = map[string]any{
			"public": map[string]any{
				"name":               stack + "-alb",
				"internal":           false,
				"load_balancer_type": "application",
				"security_groups":    []string{"${aws_security_group.service.id}"},
				"subnets":            []string{"${aws_subnet.app.id}"},
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		}
		resources["aws_lb_target_group"] = map[string]any{
			"service": map[string]any{
				"name":        stack + "-tg",
				"port":        plan.Network.ServicePort,
				"protocol":    "HTTP",
				"target_type": "ip",
				"vpc_id":      "${aws_vpc.main.id}",
				"health_check": map[string]any{
					"path":                plan.Compute.HealthCheckPath,
					"matcher":             "200-399",
					"healthy_threshold":   2,
					"unhealthy_threshold": 2,
				},
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		}
		resources["aws_lb_listener"] = map[string]any{
			"http": map[string]any{
				"load_balancer_arn": "${aws_lb.public.arn}",
				"port":              80,
				"protocol":          "HTTP",
				"default_action": []map[string]any{
					{
						"type":             "forward",
						"target_group_arn": "${aws_lb_target_group.service.arn}",
					},
				},
			},
		}
		resources["aws_ecs_service"]["service"].(map[string]any)["load_balancer"] = []map[string]any{
			{
				"target_group_arn": "${aws_lb_target_group.service.arn}",
				"container_name":   plan.Compute.ServiceName,
				"container_port":   plan.Network.ServicePort,
			},
		}
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
				"password":               "${var.database_admin_password}",
				"skip_final_snapshot":    true,
				"publicly_accessible":    false,
				"tags": map[string]any{
					"Stack": stack,
				},
			},
		}
	}
	for _, resource := range plan.Resources {
		switch resource.Kind {
		case "cache":
			if !resource.Managed {
				continue
			}
			resources["aws_elasticache_cluster"] = map[string]any{
				resource.Name: map[string]any{
					"cluster_id":           stack + "-" + resource.Name,
					"engine":               "redis",
					"node_type":            awsCacheNodeType(resource.Class),
					"num_cache_nodes":      1,
					"port":                 6379,
					"parameter_group_name": "default.redis7",
				},
			}
		case "queue":
			if !resource.Managed {
				continue
			}
			if resources["aws_sqs_queue"] == nil {
				resources["aws_sqs_queue"] = map[string]any{}
			}
			resources["aws_sqs_queue"][resource.Name] = map[string]any{
				"name":                       stack + "-" + resource.Name,
				"visibility_timeout_seconds": 30,
				"message_retention_seconds":  345600,
				"tags": map[string]any{
					"Stack": stack,
				},
			}
		}
	}
	if len(plan.AppConfig.Secrets) > 0 {
		resources["aws_secretsmanager_secret"] = map[string]any{}
		for _, secret := range plan.AppConfig.Secrets {
			secretID := terraformName(secret.Name)
			resources["aws_secretsmanager_secret"][secretID] = map[string]any{
				"name":                    stack + "-" + secret.Name,
				"recovery_window_in_days": 0,
				"tags": map[string]any{
					"Stack": stack,
				},
			}
		}
	}
	if len(plan.AppConfig.Environment) > 0 {
		resources["aws_ssm_parameter"] = map[string]any{}
		for _, env := range plan.AppConfig.Environment {
			if env.SecretRef != "" {
				continue
			}
			envID := terraformName(env.Name)
			resources["aws_ssm_parameter"][envID] = map[string]any{
				"name":  "/sai/" + stack + "/" + env.Name,
				"type":  "String",
				"value": env.Literal,
				"tags": map[string]any{
					"Stack": stack,
				},
			}
		}
	}

	document := terraformDocument{
		Terraform: terraformSettings{
			RequiredVersion: ">= 1.6.0",
			RequiredProviders: map[string]providerRequirement{
				"aws": {Source: "hashicorp/aws", Version: "~> 5.0"},
			},
		},
		Variable: map[string]variableValue{
			"database_admin_password": {Type: "string", Sensitive: true},
		},
		Locals:   map[string]any{"stack_name": stack},
		Resource: resources,
		Output: map[string]outputValue{
			"cluster_name":         {Value: "${aws_ecs_cluster.main.name}"},
			"service_name":         {Value: "${aws_ecs_service.service.name}"},
			"container_repository": {Value: "${aws_ecr_repository.service.repository_url}"},
		},
	}
	if plan.Network.InternetIngress {
		document.Output["service_url"] = outputValue{Value: "http://${aws_lb.public.dns_name}"}
	}
	if plan.Database != nil && plan.Database.Managed {
		document.Output["database_endpoint"] = outputValue{Value: "${aws_db_instance.main.address}"}
	}
	for _, resource := range plan.Resources {
		switch resource.Kind {
		case "cache":
			if resource.Managed {
				document.Output[resource.Name+"_endpoint"] = outputValue{Value: "${aws_elasticache_cluster." + resource.Name + ".cache_nodes[0].address}"}
			}
		case "queue":
			if resource.Managed {
				document.Output[resource.Name+"_url"] = outputValue{Value: "${aws_sqs_queue." + resource.Name + ".url}"}
			}
		}
	}
	return json.MarshalIndent(document, "", "  ")
}

func awsServiceIngress(plan *Plan) []map[string]any {
	if !plan.Network.InternetIngress {
		return []map[string]any{}
	}
	return []map[string]any{{
		"from_port":   80,
		"to_port":     80,
		"protocol":    "tcp",
		"cidr_blocks": []string{"0.0.0.0/0"},
	}}
}

func awsTaskDefinitionJSON(plan *Plan, stack string) string {
	image := "${aws_ecr_repository.service.repository_url}:latest"
	container := map[string]any{
		"name":      plan.Compute.ServiceName,
		"image":     image,
		"essential": true,
		"portMappings": []map[string]any{{
			"containerPort": plan.Network.ServicePort,
			"hostPort":      plan.Network.ServicePort,
			"protocol":      "tcp",
		}},
		"healthCheck": map[string]any{
			"command":     []string{"CMD-SHELL", fmt.Sprintf("curl -f http://localhost:%d%s || exit 1", plan.Network.ServicePort, plan.Compute.HealthCheckPath)},
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
	}
	if env := awsContainerEnvironment(plan); len(env) > 0 {
		container["environment"] = env
	}
	if secrets := awsContainerSecrets(plan); len(secrets) > 0 {
		container["secrets"] = secrets
	}
	definition := []map[string]any{container}
	data, _ := json.Marshal(definition)
	return string(data)
}

func ecsTaskAssumeRolePolicy() string {
	return `{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"ecs-tasks.amazonaws.com"},"Action":"sts:AssumeRole"}]}`
}

func awsServiceDependsOn(plan *Plan) []string {
	dependsOn := []string{"aws_iam_role_policy_attachment.task_execution"}
	if plan.Network.InternetIngress {
		dependsOn = append(dependsOn, "aws_lb_listener.http")
	}
	return dependsOn
}

func awsCacheNodeType(class string) string {
	switch class {
	case "medium":
		return "cache.t4g.small"
	default:
		return "cache.t4g.micro"
	}
}

func awsContainerEnvironment(plan *Plan) []map[string]any {
	env := make([]map[string]any, 0, len(plan.AppConfig.Environment))
	for _, item := range plan.AppConfig.Environment {
		if item.SecretRef != "" {
			continue
		}
		env = append(env, map[string]any{
			"name":  item.Name,
			"value": item.Literal,
		})
	}
	return env
}

func awsContainerSecrets(plan *Plan) []map[string]any {
	secretIndex := map[string]string{}
	for _, secret := range plan.AppConfig.Secrets {
		secretIndex[secret.Name] = terraformName(secret.Name)
	}
	secrets := make([]map[string]any, 0, len(plan.AppConfig.Environment))
	for _, item := range plan.AppConfig.Environment {
		if item.SecretRef == "" {
			continue
		}
		secretID, ok := secretIndex[item.SecretRef]
		if !ok {
			continue
		}
		secrets = append(secrets, map[string]any{
			"name":      item.Name,
			"valueFrom": "${aws_secretsmanager_secret." + secretID + ".arn}",
		})
	}
	return secrets
}

func terraformName(value string) string {
	return resourceName(value)
}
