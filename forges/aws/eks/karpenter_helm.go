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

// deployKarpenterHelm 部署 Karpenter Helm Chart
func deployKarpenterHelm(scope constructs.Construct, id string, props *KarpenterHelmProps) awseks.HelmChart {
	// 部署 Karpenter Helm Chart
	karpenterChart := awseks.NewHelmChart(scope, jsii.String("KarpenterChart"), &awseks.HelmChartProps{
		Cluster: props.Cluster,
		Chart: jsii.String("karpenter"),
		Repository: jsii.String("oci://public.ecr.aws/karpenter/karpenter"),
		Namespace: jsii.String("kube-system"),
		Release: jsii.String("karpenter"),
		Version: jsii.String(props.KarpenterVersion),
		Values: &map[string]interface{}{
			"serviceAccount": map[string]interface{}{
				"name": "karpenter-controller", // 指定服务账户名称
				"annotations": map[string]string{
					"eks.amazonaws.com/role-arn": props.ControllerRoleArn,
				},
			},
			"settings": map[string]interface{}{
				"clusterName": props.ClusterName,
				"clusterEndpoint": *props.Cluster.ClusterEndpoint(),
				"interruptionQueueName": fmt.Sprintf("Karpenter-%s", props.ClusterName),
			},
			"controller": map[string]interface{}{
				"resources": map[string]interface{}{
					"requests": map[string]string{
						"cpu": "1",
						"memory": "1Gi",
					},
					"limits": map[string]string{
						"cpu": "1",
						"memory": "1Gi",
					},
				},
			},
		},
	})

	return karpenterChart
}
