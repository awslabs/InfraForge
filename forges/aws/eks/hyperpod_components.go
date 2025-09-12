// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/jsii-runtime-go"
)

// 部署 HyperPod 专用组件 - 简化的 Job 方式
func deployHyperPodComponents(stack awscdk.Stack, cluster awseks.Cluster, eksInstance *EksInstanceConfig) awseks.KubernetesManifest {
	// 创建支持 IRSA 的 IAM 角色
	issuerUrl := cluster.OpenIdConnectProvider().OpenIdConnectProviderIssuer()
	
	// 使用 CfnJson 创建条件对象，延迟解析到部署时间
	stringEquals := awscdk.NewCfnJson(stack, jsii.String("HyperPodOIDCCondition"), &awscdk.CfnJsonProps{
		Value: &map[string]interface{}{
			fmt.Sprintf("%s:sub", *issuerUrl): "system:serviceaccount:hyperpod-inference-system:hyperpod-inference-operator-controller-manager",
			fmt.Sprintf("%s:aud", *issuerUrl): "sts.amazonaws.com",
		},
	})
	
	executionRole := awsiam.NewRole(stack, jsii.String("HyperPodExecutionRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewCompositePrincipal(
			awsiam.NewServicePrincipal(jsii.String("sagemaker.amazonaws.com"), nil),
			awsiam.NewWebIdentityPrincipal(
				cluster.OpenIdConnectProvider().OpenIdConnectProviderArn(),
				&map[string]interface{}{
					"StringEquals": stringEquals.Value(),
				},
			),
		),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSageMakerFullAccess")),
		},
	})

	// 1. 创建简单的 Job 来安装 HyperPod 组件
	hyperPodJob := cluster.AddManifest(jsii.String("hyperpod-installer-job"), &map[string]interface{}{
		"apiVersion": "batch/v1",
		"kind":       "Job",
		"metadata": map[string]interface{}{
			"name":      "hyperpod-installer",
			"namespace": "kube-system",
		},
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"serviceAccountName": "eks-admin",
					"containers": []interface{}{
						map[string]interface{}{
							"name":  "installer",
							"image": "alpine/helm:latest",
							"command": []interface{}{
								"sh", "-c",
								"apk add curl git && " +
									"ARCH=$(uname -m) && " +
									"if [ \"$ARCH\" = \"aarch64\" ]; then ARCH=\"arm64\"; else ARCH=\"amd64\"; fi && " +
									"VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt) && " +
									"curl -LO https://dl.k8s.io/release/$VERSION/bin/linux/$ARCH/kubectl && " +
									"chmod +x kubectl && mv kubectl /usr/local/bin/ && " +
									"kubectl create namespace aws-hyperpod || true && " +
									"git clone --depth 1 --filter=blob:none --sparse https://github.com/aws/sagemaker-hyperpod-cli.git /tmp/h && " +
									"cd /tmp/h && git sparse-checkout set helm_chart/HyperPodHelmChart/charts && " +
									"cd helm_chart/HyperPodHelmChart && " +
									"helm template charts/health-monitoring-agent | kubectl apply --server-side -f - && echo 'health-monitoring-agent done' && " +
									"helm template charts/deep-health-check | kubectl apply --server-side -f - && echo 'deep-health-check done' && " +
									"helm template charts/job-auto-restart | kubectl apply --server-side -f - && echo 'job-auto-restart done' && " +
									"helm template charts/hyperpod-patching | kubectl apply --server-side -f - && echo 'hyperpod-patching done' && " +
									"helm repo add nvidia https://nvidia.github.io/k8s-device-plugin && " +
									"helm repo add eks https://aws.github.io/eks-charts && " +
									"helm repo add aws-fsx-csi-driver https://kubernetes-sigs.github.io/aws-fsx-csi-driver && " +
									"helm repo add jetstack https://charts.jetstack.io && " +
									"helm repo add kedacore https://kedacore.github.io/charts && " +
									"helm repo add aws-mountpoint-s3-csi-driver https://awslabs.github.io/mountpoint-s3-csi-driver && " +
									"helm repo add metrics-server https://kubernetes-sigs.github.io/metrics-server && " +
									"helm repo update && echo 'helm repos added' && " +
									"cd charts/inference-operator && helm dependency build && cd ../.. && echo 'inference-operator deps built' && " +
									"helm template charts/inference-operator --set aws-fsx-csi-driver.enabled=false --set nvidia-device-plugin.enabled=false --set aws-mountpoint-s3-csi-driver.enabled=false --set metrics-server.enabled=false --set alb.enabled=false --set s3.enabled=false --set fsx.enabled=false --set cert-manager.enabled=false --set executionRoleArn=" + *executionRole.RoleArn() + " --set tlsCertificateS3Bucket=" + eksInstance.S3BucketName + " --set region=" + *stack.Region() + " --set eksClusterName=" + *cluster.ClusterName() + " --set keda.podIdentity.aws.irsa.roleArn=" + *executionRole.RoleArn() + " --set serviceAccount.annotations.'eks\\.amazonaws\\.com/role-arn'=" + *executionRole.RoleArn() + " | kubectl apply --server-side --force-conflicts -f - && echo 'inference-operator done'",
							},
						},
					},
					"restartPolicy": "Never",
				},
			},
		},
	})
	
	return hyperPodJob
}
