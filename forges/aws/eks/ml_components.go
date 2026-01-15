// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/jsii-runtime-go"
)

// deployMlflow 部署 MLflow
func deployMlflow(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.HelmChart {
	// 如果未指定版本，使用默认版本
	if version == "" {
		version = "1.8.0" // 更新到当前可用版本
	}

	return cluster.AddHelmChart(jsii.String("mlflow"), &awseks.HelmChartOptions{
		Chart:           jsii.String("mlflow"),
		Repository:      jsii.String("https://community-charts.github.io/helm-charts"),
		Version:         jsii.String(version),
		Namespace:       jsii.String("mlflow"),
		CreateNamespace: jsii.Bool(true),
		Values: &map[string]interface{}{
			"service": map[string]interface{}{
				"type": "ClusterIP",
			},
		},
	})
}

// 部署 Legacy Training Operator 的函数
func deployLegacyTrainingOperator(stack awscdk.Stack, cluster awseks.Cluster, version string, eksAdminSA awseks.KubernetesManifest, eksAdminCRB awseks.KubernetesManifest) awseks.KubernetesManifest {
	// 创建 kubeflow 命名空间
	kubeflowNamespaceMap := map[string]interface{}{
		"apiVersion": "v1",
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "kubeflow",
		},
	}
	kubeflowNamespace := cluster.AddManifest(jsii.String("kubeflow-namespace"), &kubeflowNamespaceMap)
	
	// 创建 Job 来执行 kubectl apply 命令（使用传入的 eks-admin 资源）
	applyCommand := fmt.Sprintf("kubectl apply --server-side -k \"github.com/kubeflow/training-operator.git/manifests/overlays/standalone?ref=v%s\"", version)
	
	trainingOperatorJobMap := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind": "Job",
		"metadata": map[string]interface{}{
			"name": "training-operator-install",
			"namespace": "kube-system",
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"serviceAccountName": "eks-admin",
					"containers": []map[string]interface{}{
						{
							"name": "kubectl",
							"image": "public.ecr.aws/amazonlinux/amazonlinux:latest",
							"command": []string{
								"sh",
								"-c",
								"dnf install -y git && " +
								"curl -LO \"https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')/kubectl\" && " +
								"chmod +x kubectl && mv kubectl /usr/local/bin/ && " +
								applyCommand,
							},
						},
					},
					"restartPolicy": "Never",
				},
			},
			"backoffLimit": 0,
		},
	}
	trainingOperatorJob := cluster.AddManifest(jsii.String("training-operator-install-job"), &trainingOperatorJobMap)
	
	// 添加依赖关系，确保按正确顺序创建资源
	trainingOperatorJob.Node().AddDependency(eksAdminSA)
	trainingOperatorJob.Node().AddDependency(eksAdminCRB)
	trainingOperatorJob.Node().AddDependency(kubeflowNamespace)
	
	//fmt.Printf("已配置 Legacy Training Operator %s 安装\n", version)
	return trainingOperatorJob
}

// 部署新版 Training Operator 的函数
func deployModernTrainingOperator(stack awscdk.Stack, cluster awseks.Cluster, version string, eksAdminSA awseks.KubernetesManifest, eksAdminCRB awseks.KubernetesManifest) awseks.KubernetesManifest {
	// 创建 kubeflow-system 命名空间
	kubeflowSystemNamespaceMap := map[string]interface{}{
		"apiVersion": "v1",
		"kind": "Namespace",
		"metadata": map[string]interface{}{
			"name": "kubeflow-system",
		},
	}
	kubeflowSystemNamespace := cluster.AddManifest(jsii.String("kubeflow-system-namespace"), &kubeflowSystemNamespaceMap)
	
	// 使用传入的版本参数
	// 创建 Job 来执行安装 Kubeflow Trainer Controller Manager 的命令
	managerCommand := fmt.Sprintf("kubectl apply --server-side -k \"https://github.com/kubeflow/trainer.git/manifests/overlays/manager?ref=v%s\"", version)
	
	trainerManagerJobMap := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind": "Job",
		"metadata": map[string]interface{}{
			"name": "trainer-controller-manager-install",
			"namespace": "kube-system",
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"serviceAccountName": "eks-admin",
					"containers": []map[string]interface{}{
						{
							"name": "kubectl",
							"image": "public.ecr.aws/amazonlinux/amazonlinux:latest",
							"command": []string{
								"sh",
								"-c",
								"curl -LO \"https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')/kubectl\" && " +
								"chmod +x kubectl && mv kubectl /usr/local/bin/ && " +
								managerCommand,
							},
						},
					},
					"restartPolicy": "Never",
				},
			},
			"backoffLimit": 0,
		},
	}
	trainerManagerJob := cluster.AddManifest(jsii.String("trainer-controller-manager-install-job"), &trainerManagerJobMap)
	
	// 创建 Job 来执行安装 Kubeflow Training Runtimes 的命令
	runtimesCommand := fmt.Sprintf("kubectl apply --server-side -k \"https://github.com/kubeflow/trainer.git/manifests/overlays/runtimes?ref=v%s\"", version)
	
	trainingRuntimesJobMap := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind": "Job",
		"metadata": map[string]interface{}{
			"name": "training-runtimes-install",
			"namespace": "kube-system",
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"serviceAccountName": "eks-admin",
					"containers": []map[string]interface{}{
						{
							"name": "kubectl",
							"image": "public.ecr.aws/amazonlinux/amazonlinux:latest",
							"command": []string{
								"sh",
								"-c",
								"curl -LO \"https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/$(uname -m | sed 's/x86_64/amd64/' | sed 's/aarch64/arm64/')/kubectl\" && " +
								"chmod +x kubectl && mv kubectl /usr/local/bin/ && " +
								runtimesCommand,
							},
						},
					},
					"restartPolicy": "Never",
				},
			},
			"backoffLimit": 0,
		},
	}
	trainingRuntimesJob := cluster.AddManifest(jsii.String("training-runtimes-install-job"), &trainingRuntimesJobMap)
	
	// 添加依赖关系，确保按正确顺序创建资源
	trainerManagerJob.Node().AddDependency(eksAdminSA)
	trainerManagerJob.Node().AddDependency(eksAdminCRB)
	trainerManagerJob.Node().AddDependency(kubeflowSystemNamespace)
	
	// 确保 Training Runtimes 在 Controller Manager 之后安装
	trainingRuntimesJob.Node().AddDependency(trainerManagerJob)
	
	//fmt.Printf("已配置新版 Training Operator %s 安装\n", version)
	return trainingRuntimesJob
}

// deployRayOperator 部署 Ray Operator
func deployRayOperator(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.HelmChart {
	helmOptions := &awseks.HelmChartOptions{
		Chart:           jsii.String("kuberay-operator"),
		Repository:      jsii.String("https://ray-project.github.io/kuberay-helm/"),
		Namespace:       jsii.String("ray-system"),
		CreateNamespace: jsii.Bool(true),
	}

	if version != "" && version != "latest" {
		helmOptions.Version = jsii.String(version)
	}

	return cluster.AddHelmChart(jsii.String("ray-operator"), helmOptions)
}
