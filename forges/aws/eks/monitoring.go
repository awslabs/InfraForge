// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	
	"github.com/awslabs/InfraForge/core/utils/security"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/jsii-runtime-go"
)

// deployPrometheusStack 部署 Prometheus + Grafana 监控栈
// 使用gp2 StorageClass（EKS默认提供）
// Grafana密码存储在Secrets Manager中，可通过以下命令获取：
// aws secretsmanager get-secret-value --secret-id <stack-name>-grafana-password --query SecretString --output text
func deployPrometheusStack(stack awscdk.Stack, cluster awseks.Cluster, eksInstance *EksInstanceConfig) awseks.HelmChart {
	retention := "30d"
	if eksInstance.PrometheusRetention != "" {
		retention = eksInstance.PrometheusRetention
	}

	prometheusStorage := "50Gi"
	if eksInstance.PrometheusStorageSize != "" {
		prometheusStorage = eksInstance.PrometheusStorageSize
	}

	grafanaStorage := "10Gi"
	if eksInstance.GrafanaStorageSize != "" {
		grafanaStorage = eksInstance.GrafanaStorageSize
	}

	// 生成或获取Grafana密码（幂等）
	secretName := fmt.Sprintf("%s-grafana-password", *stack.StackName())
	grafanaPassword, _ := security.GetOrCreateSecretPassword(
		stack, 
		"GrafanaPassword", 
		secretName, 
		"Grafana admin password for monitoring stack",
		20,
	)

	helmOptions := &awseks.HelmChartOptions{
		Chart:           jsii.String("kube-prometheus-stack"),
		Repository:      jsii.String("https://prometheus-community.github.io/helm-charts"),
		Namespace:       jsii.String("monitoring"),
		CreateNamespace: jsii.Bool(true),
		Values: &map[string]interface{}{
			"prometheus": map[string]interface{}{
				"prometheusSpec": map[string]interface{}{
					"retention": retention,
					"storageSpec": map[string]interface{}{
						"volumeClaimTemplate": map[string]interface{}{
							"spec": map[string]interface{}{
								"storageClassName": "gp2",
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{
										"storage": prometheusStorage,
									},
								},
							},
						},
					},
					"maximumStartupDurationSeconds": 300,
				},
			},
			"grafana": map[string]interface{}{
				"adminPassword": grafanaPassword,
				"persistence": map[string]interface{}{
					"enabled":          true,
					"storageClassName": "gp2",
					"size":             grafanaStorage,
				},
			},
		},
	}

	if eksInstance.PrometheusStackVersion != "" && eksInstance.PrometheusStackVersion != "latest" {
		helmOptions.Version = jsii.String(eksInstance.PrometheusStackVersion)
	}

	return cluster.AddHelmChart(jsii.String("prometheus-stack"), helmOptions)
}
