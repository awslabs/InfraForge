// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/jsii-runtime-go"
	"github.com/awslabs/InfraForge/core/partition"
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

// deployKServe 部署 KServe (使用 Helm)
func deployKServe(stack awscdk.Stack, cluster awseks.Cluster, version string, ingressClass string, s3BucketName string) awseks.HelmChart {
	// 默认使用 istio
	if ingressClass == "" {
		ingressClass = "istio"
	}

	// 创建 ServiceAccount 用于 KServe 访问 S3
	kserveServiceAccount := cluster.AddServiceAccount(jsii.String("kserve-sa"), &awseks.ServiceAccountOptions{
		Name:      jsii.String("kserve-sa"),
		Namespace: jsii.String("default"),
	})

	// 如果指定了 S3 bucket，创建 S3 访问策略
	if s3BucketName != "" {
		kserveS3Policy := awsiam.NewPolicy(stack, jsii.String("KServeS3AccessPolicy"), &awsiam.PolicyProps{
			Statements: &[]awsiam.PolicyStatement{
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"s3:GetObject",
						"s3:ListBucket",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:s3:::%s", partition.DefaultPartition, s3BucketName),
						fmt.Sprintf("arn:%s:s3:::%s/*", partition.DefaultPartition, s3BucketName),
					),
				}),
			},
		})
		// 将策略附加到 ServiceAccount 的角色
		kserveServiceAccount.Role().AttachInlinePolicy(kserveS3Policy)
	}

	// 先安装 CRDs
	crdChart := cluster.AddHelmChart(jsii.String("kserve-crd"), &awseks.HelmChartOptions{
		Chart:           jsii.String("oci://ghcr.io/kserve/charts/kserve-crd"),
		Version:         jsii.String(fmt.Sprintf("v%s", version)),
		Release:         jsii.String("kserve-crd"),
		Namespace:       jsii.String("kserve"),
		CreateNamespace: jsii.Bool(true),
	})

	// 再安装 KServe Controller，配置为 RawDeployment 模式
	kserveChart := cluster.AddHelmChart(jsii.String("kserve"), &awseks.HelmChartOptions{
		Chart:     jsii.String("oci://ghcr.io/kserve/charts/kserve"),
		Version:   jsii.String(fmt.Sprintf("v%s", version)),
		Release:   jsii.String("kserve"),
		Namespace: jsii.String("kserve"),
		Values: &map[string]interface{}{
			"kserve": map[string]interface{}{
				"controller": map[string]interface{}{
					"deploymentMode": "RawDeployment",
					"gateway": map[string]interface{}{
						"ingressGateway": map[string]interface{}{
							"className": ingressClass,
						},
					},
				},
			},
		},
	})

	// 确保 Controller 在 CRDs 之后安装
	kserveChart.Node().AddDependency(crdChart)
	// 确保 ServiceAccount 在 Helm chart 之前创建
	kserveChart.Node().AddDependency(kserveServiceAccount)

	// 修复 KServe webhook 配置，避免删除 InferenceService 时卡住
	// 问题：默认 webhook 会验证 DELETE 操作，如果 webhook server 不可用会导致删除失败
	// 解决：1) failurePolicy=Ignore - webhook 失败时允许操作继续
	//      2) operations 只包含 CREATE/UPDATE - 跳过 DELETE 验证
	// 使用 NewKubernetesManifest + Overwrite 避免升级时资源冲突
	webhookPatchManifest := awseks.NewKubernetesManifest(stack, jsii.String("kserve-webhook-patch"), &awseks.KubernetesManifestProps{
		Cluster:   cluster,
		Overwrite: jsii.Bool(true),
		Manifest: &[]*map[string]interface{}{
			{
				"apiVersion": "admissionregistration.k8s.io/v1",
				"kind":       "ValidatingWebhookConfiguration",
				"metadata": map[string]interface{}{
					"name": "inferenceservice.serving.kserve.io",
				},
				"webhooks": []map[string]interface{}{
					{
						"name":                    "inferenceservice.kserve-webhook-server.validator",
						"admissionReviewVersions": []string{"v1", "v1beta1"},
						"clientConfig": map[string]interface{}{
							"service": map[string]interface{}{
								"name":      "kserve-webhook-server-service",
								"namespace": "kserve",
								"path":      "/validate-serving-kserve-io-v1beta1-inferenceservice",
							},
						},
						"failurePolicy":     "Ignore",
						"matchPolicy":       "Equivalent",
						"namespaceSelector": map[string]interface{}{},
						"objectSelector":    map[string]interface{}{},
						"rules": []map[string]interface{}{
							{
								"apiGroups":   []string{"serving.kserve.io"},
								"apiVersions": []string{"v1beta1"},
								"operations":  []string{"CREATE", "UPDATE"}, // 不包含 DELETE，避免删除时被 webhook 阻塞
								"resources":   []string{"inferenceservices"},
								"scope":       "*",
							},
						},
						"sideEffects":    "None",
						"timeoutSeconds": 10,
					},
				},
			},
		},
	})
	// 确保在 KServe 安装后应用 webhook patch
	webhookPatchManifest.Node().AddDependency(kserveChart)

	return kserveChart
}
