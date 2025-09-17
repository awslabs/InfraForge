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

// createHyperPodKarpenterResources 创建 HyperPod 的 NodeClass 和 NodePool
func createHyperPodKarpenterResources(hyperPodInstance *HyperPodInstanceConfig, cluster awssagemaker.CfnCluster) {
	// 只有启用了 Karpenter scaling 才创建资源
	if hyperPodInstance.EnableKarpenterScaling == nil || !*hyperPodInstance.EnableKarpenterScaling {
		return
	}

	if hyperPodInstance.DependsOn == "" {
		return
	}

	// 获取 EKS forge 对象
	eksForge, exists := dependency.GlobalManager.Get(hyperPodInstance.DependsOn)
	if !exists {
		return
	}

	// 转换为 EksForge 类型并获取集群和依赖组件
	if eksForgeObj, ok := eksForge.(*eks.EksForge); ok {
		eksCluster := eksForgeObj.GetCluster()
		
		// 获取 Karpenter Helm chart 用于 Karpenter 资源依赖
		var karpenterChart interface{}
		if chart := eksForgeObj.GetProperties()["karpenterChart"]; chart != nil {
			karpenterChart = chart
		}
		
		createKarpenterManifests(hyperPodInstance, cluster, eksCluster, karpenterChart)
	}
}

func createKarpenterManifests(hyperPodInstance *HyperPodInstanceConfig, cluster awssagemaker.CfnCluster, eksCluster awseks.Cluster, karpenterChart interface{}) {
	nodeClassName := "karpenter-hyperpod"

	// Create HyperPodNodeClass manifest
	nodeClassManifest := map[string]interface{}{
		"apiVersion": "karpenter.sagemaker.amazonaws.com/v1",
		"kind":       "HyperpodNodeClass",
		"metadata": map[string]interface{}{
			"name": nodeClassName,
		},
		"spec": map[string]interface{}{
			"instanceGroups": []string{hyperPodInstance.InstanceGroupName},
		},
	}

	// Create NodePool manifest
	nodePoolManifest := map[string]interface{}{
		"apiVersion": "karpenter.sh/v1",
		"kind":       "NodePool",
		"metadata": map[string]interface{}{
			"name": nodeClassName,
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"nodeClassRef": map[string]interface{}{
						"group": "karpenter.sagemaker.amazonaws.com",
						"kind":  "HyperpodNodeClass",
						"name":  nodeClassName,
					},
					"requirements": []map[string]interface{}{
						{
							"key":      "node.kubernetes.io/instance-type",
							"operator": "In",
							"values":   []string{hyperPodInstance.InstanceType},
						},
					},
				},
			},
		},
	}

	// Deploy HyperPodNodeClass
	nodeClassResource := awseks.NewKubernetesManifest(eksCluster.Stack(), jsii.String(fmt.Sprintf("%s-hyperpod-nodeclass", hyperPodInstance.GetID())), &awseks.KubernetesManifestProps{
		Cluster:  eksCluster,
		Manifest: &[]*map[string]interface{}{&nodeClassManifest},
	})

	// Deploy NodePool
	nodePoolResource := awseks.NewKubernetesManifest(eksCluster.Stack(), jsii.String(fmt.Sprintf("%s-hyperpod-nodepool", hyperPodInstance.GetID())), &awseks.KubernetesManifestProps{
		Cluster:  eksCluster,
		Manifest: &[]*map[string]interface{}{&nodePoolManifest},
	})

	// Add dependency on Karpenter Helm chart to ensure CRDs are installed
	if karpenterChart != nil {
		nodeClassResource.Node().AddDependency(karpenterChart)
		nodePoolResource.Node().AddDependency(karpenterChart)
	}
}
