// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/jsii-runtime-go"
)

// 部署 HyperPod 专用组件 - 简化的 Job 方式
func deployHyperPodComponents(stack awscdk.Stack, cluster awseks.Cluster, eksInstance *EksInstanceConfig) awseks.KubernetesManifest {
	// 创建支持 IRSA 的 IAM 角色
	issuerUrl := cluster.OpenIdConnectProvider().OpenIdConnectProviderIssuer()
	
	// 1. HyperPod Inference Execution Role
	inferenceStringEquals := awscdk.NewCfnJson(stack, jsii.String("HyperPodInferenceOIDCCondition"), &awscdk.CfnJsonProps{
		Value: &map[string]interface{}{
			fmt.Sprintf("%s:sub", *issuerUrl): "system:serviceaccount:hyperpod-inference-system:hyperpod-inference-operator-controller-manager",
			fmt.Sprintf("%s:aud", *issuerUrl): "sts.amazonaws.com",
		},
	})
	
	executionRole := awsiam.NewRole(stack, jsii.String("HyperPodInferenceOperatorRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewCompositePrincipal(
			awsiam.NewServicePrincipal(jsii.String("sagemaker.amazonaws.com"), nil),
			awsiam.NewWebIdentityPrincipal(
				cluster.OpenIdConnectProvider().OpenIdConnectProviderArn(),
				&map[string]interface{}{
					"StringEquals": inferenceStringEquals.Value(),
				},
			),
		),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSageMakerFullAccess")),
		},
		InlinePolicies: &map[string]awsiam.PolicyDocument{
			"HyperPodInferencePolicy": awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
				Statements: &[]awsiam.PolicyStatement{
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Effect: awsiam.Effect_ALLOW,
						Actions: jsii.Strings(
							"sagemaker:DescribeClusterInference",
							"sagemaker:UpdateClusterInference",
							"eks:ListAssociatedAccessPolicies",
							"eks:AssociateAccessPolicy",
							"eks:DisassociateAccessPolicy",
						),
						Resources: jsii.Strings("*"),
					}),
				},
			}),
		},
	})

	// 2. KEDA Operator Role
	kedaStringEquals := awscdk.NewCfnJson(stack, jsii.String("KedaOIDCCondition"), &awscdk.CfnJsonProps{
		Value: &map[string]interface{}{
			fmt.Sprintf("%s:sub", *issuerUrl): "system:serviceaccount:kube-system:keda-operator",
			fmt.Sprintf("%s:aud", *issuerUrl): "sts.amazonaws.com",
		},
	})
	
	kedaRole := awsiam.NewRole(stack, jsii.String("KedaOperatorRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewCompositePrincipal(
			awsiam.NewServicePrincipal(jsii.String("sagemaker.amazonaws.com"), nil),
			awsiam.NewWebIdentityPrincipal(
				cluster.OpenIdConnectProvider().OpenIdConnectProviderArn(),
				&map[string]interface{}{
					"StringEquals": kedaStringEquals.Value(),
				},
			),
		),
		InlinePolicies: &map[string]awsiam.PolicyDocument{
			"KedaPolicy": awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
				Statements: &[]awsiam.PolicyStatement{
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Effect: awsiam.Effect_ALLOW,
						Actions: jsii.Strings(
							"cloudwatch:GetMetricData",
							"cloudwatch:GetMetricStatistics",
							"cloudwatch:ListMetrics",
						),
						Resources: jsii.Strings("*"),
					}),
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Effect: awsiam.Effect_ALLOW,
						Actions: jsii.Strings(
							"aps:QueryMetrics",
							"aps:GetLabels",
							"aps:GetSeries",
							"aps:GetMetricMetadata",
						),
						Resources: jsii.Strings("*"),
					}),
				},
			}),
		},
	})

	// 3. JumpStart Gated Model Download Role
	jumpstartRole := awsiam.NewRole(stack, jsii.String("JumpStartGatedRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("sagemaker.amazonaws.com"), nil),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSageMakerFullAccess")),
		},
	})

	// 4. 复用用户指定的 S3 桶用于 TLS 证书存储
	// 如果用户没有指定 S3 桶，则创建一个专用的 TLS 桶
	var tlsBucketName string
	if eksInstance.S3BucketName != "" {
		tlsBucketName = eksInstance.S3BucketName
	} else {
		tlsBucket := awss3.NewBucket(stack, jsii.String(fmt.Sprintf("%s-hyperpod-tls", eksInstance.GetID())), &awss3.BucketProps{
			RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
			AutoDeleteObjects: jsii.Bool(true),
		})
		tlsBucketName = *tlsBucket.BucketName()
	}

	// 5. 获取已部署的 S3 CSI Driver 角色 ARN
	// S3 CSI Driver 是 EKS 组件，角色名称是固定的
	s3CsiRoleArn := fmt.Sprintf("arn:%s:iam::%s:role/s3-csi-driver-sa-role", *stack.Partition(), *stack.Account())

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
									"helm template charts/inference-operator " +
									"--set aws-fsx-csi-driver.enabled=false " +
									"--set nvidia-device-plugin.enabled=false " +
									"--set aws-mountpoint-s3-csi-driver.enabled=false " +
									"--set metrics-server.enabled=false " +
									"--set alb.enabled=false " +
									"--set s3.enabled=false " +
									"--set fsx.enabled=false " +
									"--set cert-manager.enabled=false " +
									"--set region=" + *stack.Region() + " " +
									"--set eksClusterName=" + *cluster.ClusterName() + " " +
									"--set executionRoleArn=" + *executionRole.RoleArn() + " " +
									"--set s3.serviceAccountRoleArn=" + s3CsiRoleArn + " " +
									"--set keda.podIdentity.aws.irsa.roleArn=" + *kedaRole.RoleArn() + " " +
									"--set jumpstartGatedModelDownloadRoleArn=" + *jumpstartRole.RoleArn() + " " +
									"--set tlsCertificateS3Bucket=" + tlsBucketName + " " +
									"--set alb.region=" + *stack.Region() + " " +
									"--set alb.clusterName=" + *cluster.ClusterName() + " " +
									"--set alb.vpcId=" + *cluster.Vpc().VpcId() + " " +
									"--set serviceAccount.annotations.'eks\\.amazonaws\\.com/role-arn'=" + *executionRole.RoleArn() + " " +
									"| kubectl apply --server-side --force-conflicts -f - && echo 'inference-operator done'",
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
