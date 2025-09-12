// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package eks

import (
	"fmt"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awseks"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/awslabs/InfraForge/core/partition"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// KarpenterIamResources 包含所有 Karpenter 相关的 IAM 资源
type KarpenterIamResources struct {
	NodeRole         awsiam.Role
	ControllerRole   awsiam.Role
	ControllerPolicy awsiam.ManagedPolicy
	InstanceProfile  awsiam.CfnInstanceProfile
}

// createKarpenterIamResources 创建 Karpenter 所需的所有 IAM 资源
func createKarpenterIamResources(scope constructs.Construct, id string, cluster awseks.Cluster) *KarpenterIamResources {
	// 创建 Karpenter 控制器角色
	clusterName := *cluster.ClusterName()
	nodeRoleName := fmt.Sprintf("KarpenterNodeRole-%s-%s", clusterName, partition.DefaultRegion)

	// 创建 Karpenter 节点角色
	nodeRole := awsiam.NewRole(scope, jsii.String("KarpenterNodeRole"), &awsiam.RoleProps{
		RoleName: jsii.String(nodeRoleName),
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), &awsiam.ServicePrincipalOpts{}),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonEKS_CNI_Policy")),
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonEKSWorkerNodePolicy")),
			// 修正: 使用正确的策略名称 AmazonEC2ContainerRegistryPullOnly 而非 AmazonEC2ContainerRegistryReadOnly
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonEC2ContainerRegistryPullOnly")),
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSSMManagedInstanceCore")),
		},
	})

	// 创建实例配置文件
	instanceProfile := awsiam.NewCfnInstanceProfile(scope, jsii.String(fmt.Sprintf("%s-instance-profile", id)), &awsiam.CfnInstanceProfileProps{
		InstanceProfileName: jsii.String(nodeRoleName),  // 使用与角色相同的名称
		Roles: jsii.Strings(*nodeRole.RoleName()),
	})

	// 创建 Karpenter 控制器策略
	controllerPolicy := awsiam.NewManagedPolicy(scope, jsii.String("KarpenterControllerPolicy"), &awsiam.ManagedPolicyProps{
		ManagedPolicyName: jsii.String(fmt.Sprintf("KarpenterControllerPolicy-%s-%s", clusterName, partition.DefaultRegion)),
		Document: awsiam.NewPolicyDocument(&awsiam.PolicyDocumentProps{
			Statements: &[]awsiam.PolicyStatement{
				// 允许 EC2 实例访问操作
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowScopedEC2InstanceAccessActions"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"ec2:RunInstances",
						"ec2:CreateFleet",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:ec2:%s::image/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s::snapshot/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:security-group/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:subnet/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:capacity-reservation/*", partition.DefaultPartition, partition.DefaultRegion),
					),
				}),
				// 允许 EC2 启动模板访问操作
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowScopedEC2LaunchTemplateAccessActions"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"ec2:RunInstances",
						"ec2:CreateFleet",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:ec2:%s:*:launch-template/*", partition.DefaultPartition, partition.DefaultRegion),
					),
					Conditions: &map[string]interface{}{
						"StringEquals": awscdk.NewCfnJson(scope, jsii.String("LaunchTemplateTagCondition"), &awscdk.CfnJsonProps{
							Value: map[string]string{
								fmt.Sprintf("aws:ResourceTag/kubernetes.io/cluster/%s", clusterName): "owned",
							},
						}).Value(),
						"StringLike": map[string]string{
							"aws:ResourceTag/karpenter.sh/nodepool": "*",
						},
					},
				}),
				// 允许带标签的 EC2 实例操作
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowScopedEC2InstanceActionsWithTags"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"ec2:RunInstances",
						"ec2:CreateFleet",
						"ec2:CreateLaunchTemplate",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:ec2:%s:*:fleet/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:instance/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:volume/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:network-interface/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:launch-template/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:spot-instances-request/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:capacity-reservation/*", partition.DefaultPartition, partition.DefaultRegion),
					),
					Conditions: &map[string]interface{}{
						"StringEquals": awscdk.NewCfnJson(scope, jsii.String("InstanceTagCondition"), &awscdk.CfnJsonProps{
							Value: map[string]string{
								fmt.Sprintf("aws:RequestTag/kubernetes.io/cluster/%s", clusterName): "owned",
								fmt.Sprintf("aws:RequestTag/eks:eks-cluster-name"): clusterName,
							},
						}).Value(),
						"StringLike": map[string]string{
							"aws:RequestTag/karpenter.sh/nodepool": "*",
						},
					},
				}),
				// 允许资源创建标记
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowScopedResourceCreationTagging"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"ec2:CreateTags",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:ec2:%s:*:fleet/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:instance/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:volume/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:network-interface/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:launch-template/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:spot-instances-request/*", partition.DefaultPartition, partition.DefaultRegion),
					),
					Conditions: &map[string]interface{}{
						"StringEquals": awscdk.NewCfnJson(scope, jsii.String("TaggingCondition"), &awscdk.CfnJsonProps{
							Value: map[string]interface{}{
								fmt.Sprintf("aws:RequestTag/kubernetes.io/cluster/%s", clusterName): "owned",
								fmt.Sprintf("aws:RequestTag/eks:eks-cluster-name"): clusterName,
								"ec2:CreateAction": []string{
									"RunInstances",
									"CreateFleet",
									"CreateLaunchTemplate",
								},
							},
						}).Value(),
						"StringLike": map[string]string{
							"aws:RequestTag/karpenter.sh/nodepool": "*",
						},
					},
				}),
				
				// 添加: 允许有范围的资源标记
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowScopedResourceTagging"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"ec2:CreateTags",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:ec2:%s:*:instance/*", partition.DefaultPartition, partition.DefaultRegion),
					),
					Conditions: &map[string]interface{}{
						"StringEquals": awscdk.NewCfnJson(scope, jsii.String("ResourceTaggingCondition"), &awscdk.CfnJsonProps{
							Value: map[string]string{
								fmt.Sprintf("aws:ResourceTag/kubernetes.io/cluster/%s", clusterName): "owned",
							},
						}).Value(),
						"StringLike": map[string]string{
							"aws:ResourceTag/karpenter.sh/nodepool": "*",
						},
						"StringEqualsIfExists": map[string]string{
							fmt.Sprintf("aws:RequestTag/eks:eks-cluster-name"): clusterName,
						},
						"ForAllValues:StringEquals": map[string][]string{
							"aws:TagKeys": {
								"eks:eks-cluster-name",
								"karpenter.sh/nodeclaim",
								"Name",
							},
						},
					},
				}),
				
				// 添加: 允许有范围的删除操作
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowScopedDeletion"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"ec2:TerminateInstances",
						"ec2:DeleteLaunchTemplate",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:ec2:%s:*:instance/*", partition.DefaultPartition, partition.DefaultRegion),
						fmt.Sprintf("arn:%s:ec2:%s:*:launch-template/*", partition.DefaultPartition, partition.DefaultRegion),
					),
					Conditions: &map[string]interface{}{
						"StringEquals": awscdk.NewCfnJson(scope, jsii.String("DeletionTagCondition"), &awscdk.CfnJsonProps{
							Value: map[string]string{
								fmt.Sprintf("aws:ResourceTag/kubernetes.io/cluster/%s", clusterName): "owned",
							},
						}).Value(),
						"StringLike": map[string]string{
							"aws:ResourceTag/karpenter.sh/nodepool": "*",
						},
					},
				}),
				
				// 允许 EC2 实例操作 - 修改为更符合官方模板的权限列表
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowEc2InstanceActions"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"ec2:DescribeAvailabilityZones",
						"ec2:DescribeImages",
						"ec2:DescribeInstances",
						"ec2:DescribeInstanceTypeOfferings",
						"ec2:DescribeInstanceTypes",
						"ec2:DescribeLaunchTemplates",
						"ec2:DescribeSecurityGroups",
						"ec2:DescribeSpotPriceHistory",
						"ec2:DescribeSubnets",
						"pricing:GetProducts",
					),
					Resources: jsii.Strings("*"),
				}),
				
				// 允许 IAM 实例配置文件传递
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowPassingInstanceRole"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"iam:PassRole",
					),
					Resources: &[]*string{
						nodeRole.RoleArn(),
					},
					Conditions: &map[string]interface{}{
						"StringEquals": map[string]string{
							"iam:PassedToService": "ec2.amazonaws.com",
						},
					},
				}),
				
				// 允许 EKS 操作
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowScopedEKSNodeActions"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"eks:DescribeCluster",
					),
					Resources: &[]*string{
						cluster.ClusterArn(),
					},
				}),
				
				// 允许 SSM 参数访问
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowSSMAccess"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"ssm:GetParameter",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:ssm:%s:*:parameter/aws/service/eks/optimized-ami/*", partition.DefaultPartition, *awscdk.Aws_REGION()),
						fmt.Sprintf("arn:%s:ssm:%s:*:parameter/aws/service/ami-windows-latest/*", partition.DefaultPartition, *awscdk.Aws_REGION()),
						fmt.Sprintf("arn:%s:ssm:%s:*:parameter/aws/service/bottlerocket/*", partition.DefaultPartition, *awscdk.Aws_REGION()),
					),
				}),
				
				// 添加: 允许 SQS 操作
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowSQSActions"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"sqs:DeleteMessage",
						"sqs:GetQueueAttributes",
						"sqs:GetQueueUrl",
						"sqs:ReceiveMessage",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:sqs:%s:%s:karpenter-%s", 
							partition.DefaultPartition, partition.DefaultRegion, *awscdk.Aws_ACCOUNT_ID(), clusterName),
					),
				}),
				
				// 添加: 允许 EventBridge 操作
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Sid:    jsii.String("AllowEventBridgeActions"),
					Effect: awsiam.Effect_ALLOW,
					Actions: jsii.Strings(
						"events:PutRule",
						"events:PutTargets",
						"events:DeleteRule",
						"events:RemoveTargets",
					),
					Resources: jsii.Strings(
						fmt.Sprintf("arn:%s:events:%s:%s:rule/karpenter-%s-*", 
							partition.DefaultPartition, partition.DefaultRegion, *awscdk.Aws_ACCOUNT_ID(), clusterName),
					),
				}),
			},
		}),
	})

	// 创建 Karpenter 控制器角色
	issuerUrl := cluster.OpenIdConnectProvider().OpenIdConnectProviderIssuer()

	// 使用 CfnJson 创建条件对象，延迟解析到部署时间
	stringEquals := awscdk.NewCfnJson(scope, jsii.String("KarpenterOIDCCondition"), &awscdk.CfnJsonProps{
		Value: &map[string]interface{}{
			fmt.Sprintf("%s:sub", *issuerUrl): "system:serviceaccount:kube-system:karpenter-controller", // 修改命名空间为 kube-system
			fmt.Sprintf("%s:aud", *issuerUrl): "sts.amazonaws.com",
		},
	})

	// 创建 Karpenter 控制器角色
	controllerRole := awsiam.NewRole(scope, jsii.String("KarpenterControllerRole"), &awsiam.RoleProps{
		RoleName: jsii.String(fmt.Sprintf("KarpenterControllerRole-%s-%s", clusterName, partition.DefaultRegion)),
		AssumedBy: awsiam.NewWebIdentityPrincipal(
			cluster.OpenIdConnectProvider().OpenIdConnectProviderArn(),
			&map[string]interface{}{
				"StringEquals": stringEquals.Value(),
			},
		),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			controllerPolicy,
		},
	})

	// 添加 Karpenter 节点角色到 aws-auth ConfigMap
	cluster.AwsAuth().AddRoleMapping(nodeRole, &awseks.AwsAuthMapping{
		Groups: &[]*string{
			jsii.String("system:bootstrappers"),
			jsii.String("system:nodes"),
		},
		Username: jsii.String("system:node:{{EC2PrivateDNSName}}"),
	})

	return &KarpenterIamResources{
		NodeRole:        nodeRole,
		ControllerRole:  controllerRole,
		ControllerPolicy: controllerPolicy,
		InstanceProfile: instanceProfile,
	}
}
