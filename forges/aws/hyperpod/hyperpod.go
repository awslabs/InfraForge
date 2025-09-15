// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package hyperpod

import (
	"fmt"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/dependency"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/utils/aws"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awss3"
	"github.com/aws/aws-cdk-go/awscdk/v2/awssagemaker"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type HyperPodInstanceConfig struct {
	config.BaseInstanceConfig
	
	AzIndex                  int    `json:"azIndex,omitempty"`
	ThreadsPerCore           int    `json:"threadsPerCore,omitempty"`
	// Cluster Configuration
	ClusterName              string `json:"clusterName,omitempty"`              // HyperPod cluster name
	NodeProvisioningMode     string `json:"nodeProvisioningMode,omitempty"`     // Continuous or standard
	NodeRecovery             string `json:"nodeRecovery,omitempty"`             // Automatic or None
	
	// Instance Group Configuration
	InstanceGroupName        string `json:"instanceGroupName,omitempty"`        // Instance group name
	InstanceType             string `json:"instanceType"`                       // EC2 instance type
	InstanceCount            int    `json:"instanceCount"`                      // Number of instances
	
	// Storage Configuration
	EbsVolumeSizeGb          int    `json:"ebsVolumeSizeGb,omitempty"`          // EBS volume size in GB
	
	// Lifecycle Configuration
	UseDefaultLifecycle      *bool  `json:"useDefaultLifecycle,omitempty"`      // Use default lifecycle script
	OnCreateScript           string `json:"onCreateScript,omitempty"`           // Custom lifecycle script filename
	SourceS3Uri              string `json:"sourceS3Uri,omitempty"`              // S3 URI for custom lifecycle scripts
	
	// IAM Configuration
	ExecutionRolePolicies    string `json:"executionRolePolicies,omitempty"`    // IAM policies for execution role
	
	// EKS Orchestrator (optional)
	EksVersion               string `json:"eksVersion,omitempty"`               // EKS version (e.g., "1.32")
	EksClusterArn            string `json:"eksClusterArn,omitempty"`            // EKS cluster ARN for orchestration
	DependsOn                string `json:"dependsOn,omitempty"`                // Dependency on EKS cluster ID
	
	// AutoScaling Configuration
	AutoScalingMode          string `json:"autoScalingMode,omitempty"`          // Enable/Disable
	AutoScalerType           string `json:"autoScalerType,omitempty"`           // Karpenter
	
	// Cluster Role
	ClusterRole              string `json:"clusterRole,omitempty"`              // IAM role for cluster operations
	
	// FSx Lustre Configuration (auto-create)
	CreateLustre             *bool  `json:"createLustre,omitempty"`             // Whether to create FSx Lustre
	LustreStorageCapacity    int    `json:"lustreStorageCapacity,omitempty"`    // Lustre storage capacity in GiB
	LustreThroughput         int    `json:"lustreThroughput,omitempty"`         // Per-unit storage throughput (e.g., "LUSTRE:lustre1")
}

type HyperPodForge struct {
	cluster awssagemaker.CfnCluster
}

func (h *HyperPodForge) Create(ctx *interfaces.ForgeContext) interface{} {
	hyperPodInstance, ok := (*ctx.Instance).(*HyperPodInstanceConfig)
	if !ok {
		return nil
	}

	// Create HyperPod cluster
	h.cluster = h.createHyperPodCluster(hyperPodInstance, ctx)

	return h
}

func (h *HyperPodForge) createHyperPodCluster(hyperPodInstance *HyperPodInstanceConfig, ctx *interfaces.ForgeContext) awssagemaker.CfnCluster {
	clusterName := hyperPodInstance.ClusterName
	if clusterName == "" {
		clusterName = fmt.Sprintf("%s-hyperpod", hyperPodInstance.GetID())
	}

	// Create execution role
	var executionRole awsiam.IRole
	if hyperPodInstance.ExecutionRolePolicies != "" {
		executionRoleId := fmt.Sprintf("%s-execution-role", hyperPodInstance.GetID())
		executionRole = aws.CreateRole(ctx.Stack, executionRoleId, hyperPodInstance.ExecutionRolePolicies, "sagemaker")
	}

	// Instance group name
	instanceGroupName := hyperPodInstance.InstanceGroupName
	if instanceGroupName == "" {
		instanceGroupName = fmt.Sprintf("%s-instance-group", hyperPodInstance.GetID())
	}

	// Lifecycle configuration
	var lifeCycleConfig *awssagemaker.CfnCluster_ClusterLifeCycleConfigProperty
	
	if hyperPodInstance.UseDefaultLifecycle != nil && *hyperPodInstance.UseDefaultLifecycle {
		// 使用默认 lifecycle：创建 S3 桶，从 GitHub 下载官方脚本，忽略用户配置
		
		// 创建 S3 桶（CDK 自动生成唯一名称）
		bucket := awss3.NewBucket(ctx.Stack, jsii.String(fmt.Sprintf("%s-lifecycle", hyperPodInstance.GetID())), nil)
		
		// 从 GitHub 下载官方脚本到 S3
		githubUrl := "https://raw.githubusercontent.com/aws-samples/awsome-distributed-training/refs/heads/main/1.architectures/7.sagemaker-hyperpod-eks/LifecycleScripts/base-config/on_create.sh"
		aws.CreateS3ObjectFromUrl(ctx.Stack, fmt.Sprintf("%s-download-lifecycle", hyperPodInstance.GetID()), 
			*bucket.BucketName(), "on_create.sh", githubUrl)
		
		lifeCycleConfig = &awssagemaker.CfnCluster_ClusterLifeCycleConfigProperty{
			OnCreate:    jsii.String("on_create.sh"),
			SourceS3Uri: jsii.String(fmt.Sprintf("s3://%s/", *bucket.BucketName())),
		}
	} else if hyperPodInstance.OnCreateScript != "" && hyperPodInstance.SourceS3Uri != "" {
		// 使用用户自定义配置
		lifeCycleConfig = &awssagemaker.CfnCluster_ClusterLifeCycleConfigProperty{
			OnCreate:    jsii.String(hyperPodInstance.OnCreateScript),
			SourceS3Uri: jsii.String(hyperPodInstance.SourceS3Uri),
		}
	} else {
		// 默认配置
		lifeCycleConfig = &awssagemaker.CfnCluster_ClusterLifeCycleConfigProperty{
			OnCreate:    jsii.String("on_create.sh"),
			SourceS3Uri: jsii.String("s3://aws-infra-forge/"),
		}
	}

	// EBS volume configuration
	var instanceStorageConfigs interface{}
	if hyperPodInstance.EbsVolumeSizeGb > 0 {
		ebsConfig := &awssagemaker.CfnCluster_ClusterEbsVolumeConfigProperty{
			VolumeSizeInGb: jsii.Number(float64(hyperPodInstance.EbsVolumeSizeGb)),
		}
		instanceStorageConfigs = []interface{}{
			&awssagemaker.CfnCluster_ClusterInstanceStorageConfigProperty{
				EbsVolumeConfig: ebsConfig,
			},
		}
	}

	// Instance group
	instanceGroup := &awssagemaker.CfnCluster_ClusterInstanceGroupProperty{
		ExecutionRole:     executionRole.RoleArn(),
		InstanceCount:     jsii.Number(float64(hyperPodInstance.InstanceCount)),
		InstanceGroupName: jsii.String(instanceGroupName),
		InstanceType:      jsii.String(hyperPodInstance.InstanceType),
		LifeCycleConfig:   lifeCycleConfig,
	}

	if instanceStorageConfigs != nil {
		instanceGroup.InstanceStorageConfigs = instanceStorageConfigs
	}

	// 如果指定了 azIndex，为 instance group 设置 OverrideVpcConfig
	if hyperPodInstance.AzIndex > 0 && ctx.VPC != nil {
		selectedSubnet := aws.SelectSubnetByAzIndex(hyperPodInstance.AzIndex, ctx.VPC, ctx.SubnetType)
		instanceGroup.OverrideVpcConfig = &awssagemaker.CfnCluster_VpcConfigProperty{
			Subnets:          &[]*string{selectedSubnet.SubnetId()},
			SecurityGroupIds: &[]*string{ctx.SecurityGroups.Default.SecurityGroupId()},
		}
	}

	// 如果指定了 ThreadsPerCore，设置该参数（只能是 1 或 2）
	threadsPerCore := hyperPodInstance.ThreadsPerCore
	if threadsPerCore == 0 {
		threadsPerCore = 1 // 默认值
	}
	if threadsPerCore != 1 && threadsPerCore != 2 {
		threadsPerCore = 1 // 无效值时使用默认值
	}
	instanceGroup.ThreadsPerCore = jsii.Number(float64(threadsPerCore))

	// Orchestrator configuration (EKS) - Get from dependency or direct ARN
	var orchestrator interface{}
	var eksClusterArn string
	
	// First try to get EKS ARN from dependency
	if hyperPodInstance.DependsOn != "" {
		magicToken, err := dependency.GetDependencyInfo(hyperPodInstance.DependsOn)
		if err == nil {
			properties, err := dependency.ExtractDependencyProperties(magicToken, "EKS")
			if err == nil {
				if arn, exists := properties["clusterArn"]; exists {
					if arnStr, ok := arn.(string); ok {
						eksClusterArn = arnStr
					}
				}
			}
		}
	}
	
	// Fallback to direct ARN if provided
	if eksClusterArn == "" && hyperPodInstance.EksClusterArn != "" {
		eksClusterArn = hyperPodInstance.EksClusterArn
	}
	
	// Always configure EKS orchestrator if EKS ARN is available
	if eksClusterArn != "" {
		orchestrator = &awssagemaker.CfnCluster_OrchestratorProperty{
			Eks: &awssagemaker.CfnCluster_ClusterOrchestratorEksConfigProperty{
				ClusterArn: jsii.String(eksClusterArn),
			},
		}
	}
	// If no EKS ARN provided, use default Slurm (no orchestrator)

	// VPC configuration from context
	var vpcConfig interface{}
	if ctx.VPC != nil {
		var selectedSubnet awsec2.ISubnet
		
		if hyperPodInstance.AzIndex > 0 {
			// 使用指定的可用区 - HyperPod instance group 需要单个 AZ
			selectedSubnet = aws.SelectSubnetByAzIndex(hyperPodInstance.AzIndex, ctx.VPC, ctx.SubnetType)
		} else {
			// 使用默认子网选择（第一个子网）
			subnets := ctx.VPC.SelectSubnets(&awsec2.SubnetSelection{
				SubnetType: ctx.SubnetType,
			})
			if len(*subnets.Subnets) > 0 {
				selectedSubnet = (*subnets.Subnets)[0]
			}
		}
		
		if selectedSubnet != nil {
			vpcConfig = &awssagemaker.CfnCluster_VpcConfigProperty{
				Subnets:          &[]*string{selectedSubnet.SubnetId()},
				SecurityGroupIds: &[]*string{ctx.SecurityGroups.Default.SecurityGroupId()},
			}
		}
	}

	// Create cluster properties
	clusterProps := &awssagemaker.CfnClusterProps{
		ClusterName:    jsii.String(clusterName),
		InstanceGroups: []interface{}{instanceGroup},
	}

	// Optional properties
	if orchestrator != nil {
		// Using EKS orchestrator - set continuous mode if not specified
		if hyperPodInstance.NodeProvisioningMode == "" {
			clusterProps.NodeProvisioningMode = jsii.String("Continuous")
		} else {
			clusterProps.NodeProvisioningMode = jsii.String(hyperPodInstance.NodeProvisioningMode)
		}
	} else if hyperPodInstance.NodeProvisioningMode != "" {
		// Using Slurm - only allow Standard mode
		if hyperPodInstance.NodeProvisioningMode == "Continuous" {
			// Override to Standard for Slurm
			clusterProps.NodeProvisioningMode = jsii.String("Standard")
		} else {
			clusterProps.NodeProvisioningMode = jsii.String(hyperPodInstance.NodeProvisioningMode)
		}
	}
	if hyperPodInstance.NodeRecovery != "" {
		clusterProps.NodeRecovery = jsii.String(hyperPodInstance.NodeRecovery)
	}
	
	// TODO: AutoScaling and ClusterRole support will be added when CDK is updated
	// Currently not supported in CDK version
	// if hyperPodInstance.AutoScalingMode != "" {
	//     autoScaling := &awssagemaker.CfnCluster_AutoScalingProperty{
	//         Mode: jsii.String(hyperPodInstance.AutoScalingMode),
	//     }
	//     if hyperPodInstance.AutoScalerType != "" {
	//         autoScaling.AutoScalerType = jsii.String(hyperPodInstance.AutoScalerType)
	//     }
	//     clusterProps.AutoScaling = autoScaling
	// }
	// if hyperPodInstance.ClusterRole != "" {
	//     clusterProps.ClusterRole = jsii.String(hyperPodInstance.ClusterRole)
	// }

	if orchestrator != nil {
		clusterProps.Orchestrator = orchestrator
	}
	if vpcConfig != nil {
		clusterProps.VpcConfig = vpcConfig
	}

	// Add Lustre configuration - but not with continuous mode
	if hyperPodInstance.CreateLustre != nil && *hyperPodInstance.CreateLustre && orchestrator == nil {
		// Only create Lustre with Slurm orchestrator (not EKS continuous mode)
		storageCapacity := hyperPodInstance.LustreStorageCapacity
		if storageCapacity == 0 {
			storageCapacity = 1200 // Default
		}
		throughput := hyperPodInstance.LustreThroughput
		if throughput == 0 {
			throughput = 250 // Default
		}

		environmentConfig := &awssagemaker.CfnCluster_EnvironmentConfigProperty{
			FSxLustreConfig: &awssagemaker.CfnCluster_FSxLustreConfigProperty{
				PerUnitStorageThroughput: jsii.Number(float64(throughput)),
				SizeInGiB:               jsii.Number(float64(storageCapacity)),
			},
		}

		restrictedInstanceGroup := &awssagemaker.CfnCluster_ClusterRestrictedInstanceGroupProperty{
			EnvironmentConfig: environmentConfig,
			ExecutionRole:     executionRole.RoleArn(),
			InstanceCount:     jsii.Number(float64(hyperPodInstance.InstanceCount)),
			InstanceGroupName: jsii.String(instanceGroupName + "-lustre"),
			InstanceType:      jsii.String(hyperPodInstance.InstanceType),
		}

		clusterProps.RestrictedInstanceGroups = []interface{}{restrictedInstanceGroup}
	}

	// Create the cluster
	cluster := awssagemaker.NewCfnCluster(ctx.Stack, jsii.String(hyperPodInstance.GetID()), clusterProps)

	// 添加对 EKS HyperPod 组件的依赖
	if hyperPodInstance.DependsOn != "" {
		// 直接使用完整的 DependsOn 格式
		if eksForge, exists := dependency.GlobalManager.Get(hyperPodInstance.DependsOn); exists {
			// 尝试获取 EKS forge 的 properties
			if eksForgeObj, ok := eksForge.(interface{ GetProperties() map[string]interface{} }); ok {
				properties := eksForgeObj.GetProperties()
				if hyperPodJob, exists := properties["hyperPodJob"]; exists {
					// 依赖 HyperPod 组件
					if dependable, ok := hyperPodJob.(constructs.IDependable); ok {
						cluster.Node().AddDependency(dependable)
					}
				}
			}
		}
	}

	// 如果有依赖的 EKS 集群，更新 inference operator 的 HyperPod 集群 ARN
	if hyperPodInstance.DependsOn != "" {
		updateInferenceOperator(hyperPodInstance, cluster)
	}

	return cluster
}

func (h *HyperPodForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// HyperPod doesn't need additional security group rules
	// EKS and SageMaker handle their own networking requirements
}

func (h *HyperPodForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	hyperPodInstance, ok := (*ctx.Instance).(*HyperPodInstanceConfig)
	if !ok {
		return
	}
	h.createOutputs(hyperPodInstance, ctx)
}

func (h *HyperPodForge) createOutputs(hyperPodInstance *HyperPodInstanceConfig, ctx *interfaces.ForgeContext) {
	clusterName := hyperPodInstance.ClusterName
	if clusterName == "" {
		clusterName = fmt.Sprintf("%s-hyperpod", hyperPodInstance.GetID())
	}

	cluster := h.cluster
	clusterArn := cluster.AttrClusterArn()

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("HyperPodClusterArn"+hyperPodInstance.GetID()), &awscdk.CfnOutputProps{
		Value:       clusterArn,
		Description: jsii.String("HyperPod Cluster ARN for " + hyperPodInstance.GetID()),
	})

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("HyperPodClusterName"+hyperPodInstance.GetID()), &awscdk.CfnOutputProps{
		Value:       jsii.String(clusterName),
		Description: jsii.String("HyperPod Cluster Name for " + hyperPodInstance.GetID()),
	})
}

func (h *HyperPodForge) MergeConfigs(defaults, instance config.InstanceConfig) config.InstanceConfig {
	merged := defaults.(*HyperPodInstanceConfig)
	hyperPodInstance := instance.(*HyperPodInstanceConfig)

	// Override with instance configuration values
	if hyperPodInstance.GetID() != "" {
		merged.ID = hyperPodInstance.GetID()
	}
	if hyperPodInstance.GetType() != "" {
		merged.Type = hyperPodInstance.GetType()
	}
	if hyperPodInstance.GetSubnet() != "" {
		merged.Subnet = hyperPodInstance.GetSubnet()
	}
	if hyperPodInstance.GetSecurityGroup() != "" {
		merged.SecurityGroup = hyperPodInstance.GetSecurityGroup()
	}
	if hyperPodInstance.AzIndex > 0 {
		merged.AzIndex = hyperPodInstance.AzIndex
	}
	if hyperPodInstance.ThreadsPerCore > 0 {
		merged.ThreadsPerCore = hyperPodInstance.ThreadsPerCore
	}
	if hyperPodInstance.ClusterName != "" {
		merged.ClusterName = hyperPodInstance.ClusterName
	}
	if hyperPodInstance.NodeProvisioningMode != "" {
		merged.NodeProvisioningMode = hyperPodInstance.NodeProvisioningMode
	}
	if hyperPodInstance.NodeRecovery != "" {
		merged.NodeRecovery = hyperPodInstance.NodeRecovery
	}
	if hyperPodInstance.AutoScalingMode != "" {
		merged.AutoScalingMode = hyperPodInstance.AutoScalingMode
	}
	if hyperPodInstance.AutoScalerType != "" {
		merged.AutoScalerType = hyperPodInstance.AutoScalerType
	}
	if hyperPodInstance.ClusterRole != "" {
		merged.ClusterRole = hyperPodInstance.ClusterRole
	}
	if hyperPodInstance.InstanceGroupName != "" {
		merged.InstanceGroupName = hyperPodInstance.InstanceGroupName
	}
	if hyperPodInstance.InstanceType != "" {
		merged.InstanceType = hyperPodInstance.InstanceType
	}
	if hyperPodInstance.InstanceCount > 0 {
		merged.InstanceCount = hyperPodInstance.InstanceCount
	}
	if hyperPodInstance.EbsVolumeSizeGb > 0 {
		merged.EbsVolumeSizeGb = hyperPodInstance.EbsVolumeSizeGb
	}
	if hyperPodInstance.UseDefaultLifecycle != nil {
		merged.UseDefaultLifecycle = hyperPodInstance.UseDefaultLifecycle
	}
	if hyperPodInstance.OnCreateScript != "" {
		merged.OnCreateScript = hyperPodInstance.OnCreateScript
	}
	if hyperPodInstance.SourceS3Uri != "" {
		merged.SourceS3Uri = hyperPodInstance.SourceS3Uri
	}
	if hyperPodInstance.ExecutionRolePolicies != "" {
		merged.ExecutionRolePolicies = hyperPodInstance.ExecutionRolePolicies
	}
	if hyperPodInstance.EksVersion != "" {
		merged.EksVersion = hyperPodInstance.EksVersion
	}
	if hyperPodInstance.EksClusterArn != "" {
		merged.EksClusterArn = hyperPodInstance.EksClusterArn
	}
	if hyperPodInstance.DependsOn != "" {
		merged.DependsOn = hyperPodInstance.DependsOn
	}
	if hyperPodInstance.CreateLustre != nil {
		merged.CreateLustre = hyperPodInstance.CreateLustre
	}
	if hyperPodInstance.LustreStorageCapacity > 0 {
		merged.LustreStorageCapacity = hyperPodInstance.LustreStorageCapacity
	}
	if hyperPodInstance.LustreThroughput > 0 {
		merged.LustreThroughput = hyperPodInstance.LustreThroughput
	}

	return merged
}

func (h *HyperPodForge) GetOutputs() []map[string]interface{} {
	var outputs []map[string]interface{}
	if h.cluster != nil {
		outputs = append(outputs, map[string]interface{}{
			"Key":   "HyperPodClusterArn",
			"Value": *h.cluster.AttrClusterArn(),
			"Type":  "String",
		})
	}
	return outputs
}
