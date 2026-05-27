// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// KarpenterHelmProps 定义部署 Karpenter Helm Chart 所需的属性
type KarpenterHelmProps struct {
	ClusterName      string
	Cluster          awseks.Cluster
	ControllerRoleArn string
	KarpenterVersion string
}

// deployKarpenterHelm 部署 Karpenter（官方推荐两步方式）
// 1. 先安装 karpenter-crd chart 管理 CRDs 生命周期
// 2. 再安装 karpenter controller chart (SkipCrds)
func deployKarpenterHelm(scope constructs.Construct, id string, props *KarpenterHelmProps) awseks.HelmChart {
	// Step 1: 安装 CRDs（独立 chart，支持升级时更新 CRDs）
	crdChart := awseks.NewHelmChart(scope, jsii.String("KarpenterCrdChart"), &awseks.HelmChartProps{
		Cluster:    props.Cluster,
		Chart:      jsii.String("karpenter-crd"),
		Repository: jsii.String("oci://public.ecr.aws/karpenter/karpenter-crd"),
		Namespace:  jsii.String("kube-system"),
		Release:    jsii.String("karpenter-crd"),
		Version:    jsii.String(props.KarpenterVersion),
	})

	// Step 2: 安装 Karpenter controller（跳过 CRDs，由上面的 chart 管理）
	karpenterChart := awseks.NewHelmChart(scope, jsii.String("KarpenterChart"), &awseks.HelmChartProps{
		Cluster:  props.Cluster,
		Chart:    jsii.String("karpenter"),
		Repository: jsii.String("oci://public.ecr.aws/karpenter/karpenter"),
		Namespace: jsii.String("kube-system"),
		Release:  jsii.String("karpenter"),
		Version:  jsii.String(props.KarpenterVersion),
		SkipCrds: jsii.Bool(true),
		Wait:     jsii.Bool(true),
		Values: &map[string]interface{}{
			"serviceAccount": map[string]interface{}{
				"name": "karpenter-controller",
				"annotations": map[string]string{
					"eks.amazonaws.com/role-arn": props.ControllerRoleArn,
				},
			},
			"settings": map[string]interface{}{
				"clusterName":          props.ClusterName,
				"clusterEndpoint":      *props.Cluster.ClusterEndpoint(),
				"interruptionQueueName": fmt.Sprintf("Karpenter-%s", props.ClusterName),
			},
			"controller": map[string]interface{}{
				"resources": map[string]interface{}{
					"requests": map[string]string{
						"cpu":    "1",
						"memory": "1Gi",
					},
					"limits": map[string]string{
						"cpu":    "1",
						"memory": "1Gi",
					},
				},
			},
		},
	})

	// 确保 controller 在 CRDs 之后安装
	karpenterChart.Node().AddDependency(crdChart)

	return karpenterChart
}
