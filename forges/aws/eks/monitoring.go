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

	prometheusSpec := map[string]interface{}{
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
	}

	// 如果启用GPU监控，添加额外的scrape配置
	if eksInstance.DcgmExporterVersion != "" {
		prometheusSpec["additionalScrapeConfigs"] = []map[string]interface{}{
			{
				"job_name":        "gpu-metrics",
				"scrape_interval": "15s",
				"metrics_path":    "/metrics",
				"scheme":          "http",
				"kubernetes_sd_configs": []map[string]interface{}{
					{
						"role": "endpoints",
						"namespaces": map[string]interface{}{
							"names": []string{"monitoring"},
						},
					},
				},
				"relabel_configs": []map[string]interface{}{
					{
						"source_labels": []string{"__meta_kubernetes_service_name"},
						"action":        "keep",
						"regex":         ".*dcgm-exporter.*",
					},
					{
						"source_labels": []string{"__meta_kubernetes_pod_node_name"},
						"action":        "replace",
						"target_label":  "kubernetes_node",
					},
					{
						"source_labels": []string{"__address__"},
						"action":        "replace",
						"target_label":  "pod_ip",
					},
					{
						"source_labels": []string{"__meta_kubernetes_pod_node_name"},
						"action":        "replace",
						"target_label":  "instance",
					},
				},
			},
		}
	}

	grafanaConfig := map[string]interface{}{
		"adminPassword": grafanaPassword,
		"persistence": map[string]interface{}{
			"enabled":          true,
			"storageClassName": "gp2",
			"size":             grafanaStorage,
		},
	}

	// 如果启用GPU监控，自动导入NVIDIA DCGM Dashboard
	if eksInstance.DcgmExporterVersion != "" {
		grafanaConfig["dashboardProviders"] = map[string]interface{}{
			"dashboardproviders.yaml": map[string]interface{}{
				"apiVersion": 1,
				"providers": []map[string]interface{}{
					{
						"name":            "default",
						"orgId":           1,
						"folder":          "",
						"type":            "file",
						"disableDeletion": false,
						"editable":        true,
						"options": map[string]interface{}{
							"path": "/var/lib/grafana/dashboards/default",
						},
					},
				},
			},
		}
		grafanaConfig["dashboards"] = map[string]interface{}{
			"default": map[string]interface{}{
				"nvidia-dcgm": map[string]interface{}{
					"gnetId":     12239,
					"revision":   2,
					"datasource": "prometheus",
				},
			},
		}
	}

	helmOptions := &awseks.HelmChartOptions{
		Chart:           jsii.String("kube-prometheus-stack"),
		Repository:      jsii.String("https://prometheus-community.github.io/helm-charts"),
		Namespace:       jsii.String("monitoring"),
		CreateNamespace: jsii.Bool(true),
		Values: &map[string]interface{}{
			"prometheus": map[string]interface{}{
				"prometheusSpec": prometheusSpec,
			},
			"grafana": grafanaConfig,
		},
	}

	if eksInstance.PrometheusStackVersion != "" && eksInstance.PrometheusStackVersion != "latest" {
		helmOptions.Version = jsii.String(eksInstance.PrometheusStackVersion)
	}

	return cluster.AddHelmChart(jsii.String("prometheus-stack"), helmOptions)
}

// deployDcgmExporter 部署 DCGM Exporter 用于GPU监控
func deployDcgmExporter(cluster awseks.Cluster, eksInstance *EksInstanceConfig) awseks.HelmChart {
	helmOptions := &awseks.HelmChartOptions{
		Chart:           jsii.String("dcgm-exporter"),
		Repository:      jsii.String("https://nvidia.github.io/dcgm-exporter/helm-charts"),
		Namespace:       jsii.String("monitoring"),
		CreateNamespace: jsii.Bool(false),
		Values: &map[string]interface{}{
			"nodeSelector": map[string]interface{}{
				"accelerator": "nvidia",
			},
			"tolerations": []map[string]interface{}{
				{
					"key":      "nvidia.com/gpu",
					"operator": "Exists",
					"effect":   "NoSchedule",
				},
			},
			"serviceMonitor": map[string]interface{}{
				"enabled": true,
			},
		},
	}

	if eksInstance.DcgmExporterVersion != "" && eksInstance.DcgmExporterVersion != "latest" {
		helmOptions.Version = jsii.String(eksInstance.DcgmExporterVersion)
	}

	return cluster.AddHelmChart(jsii.String("dcgm-exporter"), helmOptions)
}
