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

// deployMetricsServer 部署 Kubernetes Metrics Server
func deployMetricsServer(stack awscdk.Stack, cluster awseks.Cluster, version string, hasCertManager bool) awseks.HelmChart {
	// 如果未指定版本，使用默认版本
	if version == "" {
		version = "3.13.0"
	}

	// 基础配置 - 使用最小化配置
	values := map[string]interface{}{}

	// 根据是否有 cert-manager 来配置 TLS 设置
	if hasCertManager {
		// 如果有 cert-manager，使用完全默认配置（安全的 TLS 验证）
		// 不添加任何自定义配置，让 Helm Chart 使用所有默认值
	} else {
		// 如果没有 cert-manager，只添加必要的 kubelet-insecure-tls 参数
		values["args"] = []string{
			"--kubelet-insecure-tls",
		}
	}

	// 部署 Metrics Server Helm Chart
	metricsServerChart := cluster.AddHelmChart(jsii.String("ms"), &awseks.HelmChartOptions{
		Chart:      jsii.String("metrics-server"),
		Repository: jsii.String("https://kubernetes-sigs.github.io/metrics-server/"),
		Namespace:  jsii.String("kube-system"),
		Version:    jsii.String(version),
		Values:     &values,
		CreateNamespace: jsii.Bool(true),
		Wait:       jsii.Bool(true), // 确保完全部署后再继续
	})

	return metricsServerChart
}

// deployCertManager 部署 Cert Manager
func deployCertManager(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.HelmChart {
	// 如果未指定版本，使用默认版本（不带v前缀）
	if version == "" {
		version = "1.19.1"
	}

	// 部署 Cert Manager Helm Chart（配置文件中不带v，但Helm Chart需要v前缀）
	// 注意：namespace已经在调用处创建并添加了mastersRole依赖
	helmVersion := "v" + version
	certManagerChart := cluster.AddHelmChart(jsii.String("cm"), &awseks.HelmChartOptions{
		Chart:      jsii.String("cert-manager"),
		Repository: jsii.String("https://charts.jetstack.io"),
		Namespace:  jsii.String("cert-manager"),
		Version:    jsii.String(helmVersion),
		Values: &map[string]interface{}{
			"installCRDs": true, // 安装 CRDs
			"global": map[string]interface{}{
				"rbac": map[string]interface{}{
					"create": true,
				},
			},
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":    "10m",
					"memory": "32Mi",
				},
			},
			"webhook": map[string]interface{}{
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    "10m",
						"memory": "32Mi",
					},
				},
			},
			"cainjector": map[string]interface{}{
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"cpu":    "10m",
						"memory": "32Mi",
					},
				},
			},
		},
	})

	return certManagerChart
}

// deployAwsLoadBalancerController 部署 AWS Load Balancer Controller
func deployAwsLoadBalancerController(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.HelmChart {
	// 如果未指定版本，使用默认版本
	if version == "" {
		version = "1.16.0"
	}

	// 创建 AWS Load Balancer Controller 的 IAM 策略
	albPolicy := awsiam.NewPolicy(stack, jsii.String("AWSLoadBalancerControllerPolicy"), &awsiam.PolicyProps{
		Statements: &[]awsiam.PolicyStatement{
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"iam:CreateServiceLinkedRole",
					"ec2:DescribeAccountAttributes",
					"ec2:DescribeAddresses",
					"ec2:DescribeAvailabilityZones",
					"ec2:DescribeInternetGateways",
					"ec2:DescribeVpcs",
					"ec2:DescribeVpcPeeringConnections",
					"ec2:DescribeSubnets",
					"ec2:DescribeSecurityGroups",
					"ec2:DescribeInstances",
					"ec2:DescribeNetworkInterfaces",
					"ec2:DescribeTags",
					"ec2:GetCoipPoolUsage",
					"ec2:GetManagedPrefixListEntries",
					"ec2:DescribeCoipPools",
					"elasticloadbalancing:DescribeLoadBalancers",
					"elasticloadbalancing:DescribeLoadBalancerAttributes",
					"elasticloadbalancing:DescribeListeners",
					"elasticloadbalancing:DescribeListenerCertificates",
					"elasticloadbalancing:DescribeSSLPolicies",
					"elasticloadbalancing:DescribeRules",
					"elasticloadbalancing:DescribeTargetGroups",
					"elasticloadbalancing:DescribeTargetGroupAttributes",
					"elasticloadbalancing:DescribeTargetHealth",
					"elasticloadbalancing:DescribeTags",
				),
				Resources: jsii.Strings("*"),
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"cognito-idp:DescribeUserPoolClient",
					"acm:ListCertificates",
					"acm:DescribeCertificate",
					"iam:ListServerCertificates",
					"iam:GetServerCertificate",
					"waf-regional:GetWebACL",
					"waf-regional:GetWebACLForResource",
					"waf-regional:AssociateWebACL",
					"waf-regional:DisassociateWebACL",
					"wafv2:GetWebACL",
					"wafv2:GetWebACLForResource",
					"wafv2:AssociateWebACL",
					"wafv2:DisassociateWebACL",
					"shield:DescribeProtection",
					"shield:GetSubscriptionState",
					"shield:DescribeSubscription",
					"shield:CreateProtection",
					"shield:DeleteProtection",
				),
				Resources: jsii.Strings("*"),
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"ec2:AuthorizeSecurityGroupIngress",
					"ec2:RevokeSecurityGroupIngress",
				),
				Resources: jsii.Strings("*"),
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"ec2:CreateSecurityGroup",
				),
				Resources: jsii.Strings("*"),
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"ec2:CreateTags",
				),
				Resources: jsii.Strings("arn:*:ec2:*:*:security-group/*"),
				Conditions: &map[string]interface{}{
					"StringEquals": map[string]interface{}{
						"ec2:CreateAction": "CreateSecurityGroup",
					},
					"Null": map[string]interface{}{
						"aws:RequestTag/elbv2.k8s.aws/cluster": "false",
					},
				},
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"ec2:CreateTags",
					"ec2:DeleteTags",
				),
				Resources: jsii.Strings("arn:*:ec2:*:*:security-group/*"),
				Conditions: &map[string]interface{}{
					"Null": map[string]interface{}{
						"aws:RequestTag/elbv2.k8s.aws/cluster":  "true",
						"aws:ResourceTag/elbv2.k8s.aws/cluster": "false",
					},
				},
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"elasticloadbalancing:CreateLoadBalancer",
					"elasticloadbalancing:CreateTargetGroup",
				),
				Resources: jsii.Strings("*"),
				Conditions: &map[string]interface{}{
					"Null": map[string]interface{}{
						"aws:RequestTag/elbv2.k8s.aws/cluster": "false",
					},
				},
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"elasticloadbalancing:CreateListener",
					"elasticloadbalancing:DeleteListener",
					"elasticloadbalancing:CreateRule",
					"elasticloadbalancing:DeleteRule",
				),
				Resources: jsii.Strings("*"),
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"elasticloadbalancing:AddTags",
					"elasticloadbalancing:RemoveTags",
				),
				Resources: jsii.Strings(
					fmt.Sprintf("arn:%s:elasticloadbalancing:%s:%s:targetgroup/*/*", partition.DefaultPartition, *awscdk.Aws_REGION(), *awscdk.Aws_ACCOUNT_ID()),
					fmt.Sprintf("arn:%s:elasticloadbalancing:%s:%s:loadbalancer/net/*/*", partition.DefaultPartition, *awscdk.Aws_REGION(), *awscdk.Aws_ACCOUNT_ID()),
					fmt.Sprintf("arn:%s:elasticloadbalancing:%s:%s:loadbalancer/app/*/*", partition.DefaultPartition, *awscdk.Aws_REGION(), *awscdk.Aws_ACCOUNT_ID()),
				),
				Conditions: &map[string]interface{}{
					"Null": map[string]interface{}{
						"aws:RequestTag/elbv2.k8s.aws/cluster":  "true",
						"aws:ResourceTag/elbv2.k8s.aws/cluster": "false",
					},
				},
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"elasticloadbalancing:AddTags",
					"elasticloadbalancing:RemoveTags",
				),
				Resources: jsii.Strings(
					fmt.Sprintf("arn:%s:elasticloadbalancing:%s:%s:listener/net/*/*/*", partition.DefaultPartition, *awscdk.Aws_REGION(), *awscdk.Aws_ACCOUNT_ID()),
					fmt.Sprintf("arn:%s:elasticloadbalancing:%s:%s:listener/app/*/*/*", partition.DefaultPartition, *awscdk.Aws_REGION(), *awscdk.Aws_ACCOUNT_ID()),
					fmt.Sprintf("arn:%s:elasticloadbalancing:%s:%s:listener-rule/net/*/*/*", partition.DefaultPartition, *awscdk.Aws_REGION(), *awscdk.Aws_ACCOUNT_ID()),
					fmt.Sprintf("arn:%s:elasticloadbalancing:%s:%s:listener-rule/app/*/*/*", partition.DefaultPartition, *awscdk.Aws_REGION(), *awscdk.Aws_ACCOUNT_ID()),
				),
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"elasticloadbalancing:ModifyLoadBalancerAttributes",
					"elasticloadbalancing:SetIpAddressType",
					"elasticloadbalancing:SetSecurityGroups",
					"elasticloadbalancing:SetSubnets",
					"elasticloadbalancing:DeleteLoadBalancer",
					"elasticloadbalancing:ModifyTargetGroup",
					"elasticloadbalancing:ModifyTargetGroupAttributes",
					"elasticloadbalancing:DeleteTargetGroup",
				),
				Resources: jsii.Strings("*"),
				Conditions: &map[string]interface{}{
					"Null": map[string]interface{}{
						"aws:ResourceTag/elbv2.k8s.aws/cluster": "false",
					},
				},
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"elasticloadbalancing:RegisterTargets",
					"elasticloadbalancing:DeregisterTargets",
				),
				Resources: jsii.Strings("arn:*:elasticloadbalancing:*:*:targetgroup/*/*"),
			}),
			awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
				Effect: awsiam.Effect_ALLOW,
				Actions: jsii.Strings(
					"elasticloadbalancing:SetWebAcl",
					"elasticloadbalancing:ModifyListener",
					"elasticloadbalancing:AddListenerCertificates",
					"elasticloadbalancing:RemoveListenerCertificates",
					"elasticloadbalancing:ModifyRule",
				),
				Resources: jsii.Strings("*"),
			}),
		},
	})

	// 创建 Service Account
	albServiceAccount := cluster.AddServiceAccount(jsii.String("alb-sa"), &awseks.ServiceAccountOptions{
		Name:      jsii.String("aws-load-balancer-controller"),
		Namespace: jsii.String("kube-system"),
	})

	// 将策略附加到 Service Account 的角色
	albServiceAccount.Role().AttachInlinePolicy(albPolicy)

	// 按照官方文档简单部署 AWS Load Balancer Controller Helm Chart
	albChart := cluster.AddHelmChart(jsii.String("alb"), &awseks.HelmChartOptions{
		Chart:      jsii.String("aws-load-balancer-controller"),
		Repository: jsii.String("https://aws.github.io/eks-charts"),
		Namespace:  jsii.String("kube-system"),
		Version:    jsii.String(version),
		Values: &map[string]interface{}{
			"clusterName": *cluster.ClusterName(),
			"serviceAccount": map[string]interface{}{
				"create": false, // 使用我们创建的 ServiceAccount
				"name":   "aws-load-balancer-controller",
			},
			// 修改webhook配置，避免部署时的冲突
			"serviceMutatorWebhookConfig": map[string]interface{}{
				"failurePolicy": "Ignore", // 改为Ignore，避免webhook失败时阻止Service创建
			},
		},
	})

	// 添加依赖关系，确保 ServiceAccount 在 Helm Chart 之前创建
	albChart.Node().AddDependency(albServiceAccount)

	return albChart
}

// deployPodIdentityAgent 部署 EKS Pod Identity Agent
func deployPodIdentityAgent(stack awscdk.Stack, cluster awseks.Cluster, version string) awseks.KubernetesManifest {
	// 如果未指定版本，使用 latest
	if version == "" {
		version = "latest"
	}

	// 获取当前区域
	region := stack.Region()

	// Pod Identity Agent DaemonSet manifest
	podIdentityAgentManifest := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "DaemonSet",
		"metadata": map[string]interface{}{
			"name":      "eks-pod-identity-agent",
			"namespace": "kube-system",
			"labels": map[string]interface{}{
				"app.kubernetes.io/name":     "eks-pod-identity-agent",
				"app.kubernetes.io/instance": "eks-pod-identity-agent",
			},
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app.kubernetes.io/name":     "eks-pod-identity-agent",
					"app.kubernetes.io/instance": "eks-pod-identity-agent",
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app.kubernetes.io/name":     "eks-pod-identity-agent",
						"app.kubernetes.io/instance": "eks-pod-identity-agent",
					},
					"annotations": map[string]interface{}{
						"eks.amazonaws.com/skip-containers": "eks-pod-identity-agent,eks-pod-identity-agent-init",
					},
				},
				"spec": map[string]interface{}{
					"hostNetwork":                   true,
					"automountServiceAccountToken":  false,
					"priorityClassName":             "system-node-critical",
					"initContainers": []map[string]interface{}{
						{
							"name":    "eks-pod-identity-agent-init",
							"image":   "public.ecr.aws/eks/eks-pod-identity-agent:" + version,
							"command": []string{"/go-runner", "/eks-pod-identity-agent", "initialize"},
							"securityContext": map[string]interface{}{
								"privileged": true,
							},
						},
					},
					"containers": []map[string]interface{}{
						{
							"name":    "eks-pod-identity-agent",
							"image":   "public.ecr.aws/eks/eks-pod-identity-agent:" + version,
							"command": []string{"/go-runner", "/eks-pod-identity-agent", "server"},
							"args": []string{
								"--port", "80",
								"--cluster-name", *cluster.ClusterName(),
								"--probe-port", "2703",
							},
							"env": []map[string]interface{}{
								{
									"name":  "AWS_REGION",
									"value": *region,
								},
							},
							"ports": []map[string]interface{}{
								{
									"containerPort": 80,
									"name":          "proxy",
									"protocol":      "TCP",
								},
								{
									"containerPort": 2703,
									"name":          "probes-port",
									"protocol":      "TCP",
								},
							},
							"livenessProbe": map[string]interface{}{
								"httpGet": map[string]interface{}{
									"host":   "localhost",
									"path":   "/healthz",
									"port":   "probes-port",
									"scheme": "HTTP",
								},
								"initialDelaySeconds": 30,
								"periodSeconds":       10,
								"timeoutSeconds":      10,
								"failureThreshold":    3,
							},
							"readinessProbe": map[string]interface{}{
								"httpGet": map[string]interface{}{
									"host":   "localhost",
									"path":   "/readyz",
									"port":   "probes-port",
									"scheme": "HTTP",
								},
								"initialDelaySeconds": 1,
								"periodSeconds":       10,
								"timeoutSeconds":      10,
								"failureThreshold":    30,
							},
							"securityContext": map[string]interface{}{
								"capabilities": map[string]interface{}{
									"add": []string{"CAP_NET_BIND_SERVICE"},
								},
							},
						},
					},
					"affinity": map[string]interface{}{
						"nodeAffinity": map[string]interface{}{
							"requiredDuringSchedulingIgnoredDuringExecution": map[string]interface{}{
								"nodeSelectorTerms": []map[string]interface{}{
									{
										"matchExpressions": []map[string]interface{}{
											{
												"key":      "kubernetes.io/os",
												"operator": "In",
												"values":   []string{"linux"},
											},
											{
												"key":      "kubernetes.io/arch",
												"operator": "In",
												"values":   []string{"amd64", "arm64"},
											},
											{
												"key":      "eks.amazonaws.com/compute-type",
												"operator": "NotIn",
												"values":   []string{"fargate", "hybrid", "auto"},
											},
										},
									},
								},
							},
						},
					},
					"tolerations": []map[string]interface{}{
						{
							"operator": "Exists",
						},
					},
				},
			},
		},
	}

	return cluster.AddManifest(jsii.String("pod-identity-agent"), &podIdentityAgentManifest)
}
