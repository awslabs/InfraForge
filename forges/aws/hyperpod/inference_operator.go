// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package hyperpod

import (
	"fmt"

	"github.com/awslabs/InfraForge/core/dependency"
	"github.com/awslabs/InfraForge/forges/aws/eks"

	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssagemaker"
	"github.com/aws/jsii-runtime-go"
)

// updateInferenceOperator 更新 inference operator 的 HyperPod 集群 ARN
func updateInferenceOperator(hyperPodInstance *HyperPodInstanceConfig, cluster awssagemaker.CfnCluster) {
	if hyperPodInstance.DependsOn == "" {
		return
	}

	// 直接使用 DependsOn（"EKS:eks" 格式）
	eksForge, exists := dependency.GlobalManager.Get(hyperPodInstance.DependsOn)
	if !exists {
		return
	}

	// 直接转换为 EksForge 类型并获取集群
	if eksForgeObj, ok := eksForge.(*eks.EksForge); ok {
		createUpdateJob(hyperPodInstance, cluster, eksForgeObj.GetCluster())
	}
}

func createUpdateJob(hyperPodInstance *HyperPodInstanceConfig, cluster awssagemaker.CfnCluster, eksCluster awseks.Cluster) {
	// 使用 kubectl set env 命令来更新环境变量，然后强制重启 deployment 并删除所有 pods
	patchCommand := fmt.Sprintf(`kubectl set env deployment/hyperpod-inference-operator-controller-manager -n hyperpod-inference-system HYPERPOD_CLUSTER_ARN=%s && kubectl rollout restart deployment/hyperpod-inference-operator-controller-manager -n hyperpod-inference-system && kubectl delete pod --all -n hyperpod-inference-system`, 
		*cluster.AttrClusterArn())

	// 创建 Job 来执行 kubectl patch 命令
	jobManifest := map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-inference-update", hyperPodInstance.GetID()),
			"namespace": "kube-system",
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"serviceAccountName": "eks-admin",
					"restartPolicy":      "Never",
					"containers": []map[string]interface{}{
						{
							"name":    "kubectl",
							"image":   "bitnami/kubectl:latest",
							"command": []string{"/bin/sh", "-c", patchCommand},
						},
					},
				},
			},
		},
	}

	eksCluster.AddManifest(jsii.String(fmt.Sprintf("%s-inference-update", hyperPodInstance.GetID())), &jobManifest)
}
