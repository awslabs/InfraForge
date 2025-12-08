// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package parallelcluster

import (
	"fmt"
	"strings"


	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/security"
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/core/partition"
	"github.com/awslabs/InfraForge/core/dependency"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	//"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/jsii-runtime-go"
)

// ParallelClusterInstanceConfig defines the configuration for a ParallelCluster instance
type ParallelClusterInstanceConfig struct {
	config.BaseInstanceConfig
	// Required parameters
	KeyName            string `json:"keyName"`

	// Optional parameters with defaults
	ClusterName        string `json:"clusterName"`
	Version            string `json:"version"`
	HeadNodeType       string `json:"headNodeType"`
	ComputeNodeType    string `json:"computeNodeType"`
	AzIndex            int    `json:"azIndex"`
	OsType             string `json:"osType"`
	DiskSize           int    `json:"diskSize,omitempty"`
	MinSize            int    `json:"minSize,omitempty"`
	MaxSize            int    `json:"maxSize,omitempty"`
	UserDataToken      string `json:"userDataToken"`
	UserDataScriptPath string `json:"userDataScriptPath"`
	
	// Compute resource configuration
	DisableSimultaneousMultithreading *bool  `json:"disableSimultaneousMultithreading,omitempty"`
	AllocationStrategy                string `json:"allocationStrategy,omitempty"`
	SpotAllocationStrategy            string `json:"spotAllocationStrategy,omitempty"`
	ScalingStrategy                   string `json:"scalingStrategy,omitempty"`

	// EFA configuration for CPU queue
	EnableEfa          *bool  `json:"enableEfa,omitempty"`
	
	// Placement group configuration for CPU queue
	PlacementGroupEnabled *bool  `json:"placementGroupEnabled,omitempty"`
	
	// Database configuration for Slurm accounting (auto-enabled if RDS dependency exists)
	DatabaseName          string `json:"databaseName,omitempty"`  // 数据库名称，需要手动指定
	PlacementGroupId      string `json:"placementGroupId,omitempty"`
	PgAzIndex            int    `json:"pgAzIndex,omitempty"`

	// GPU queue configuration
	EnableGpuQueue     *bool  `json:"enableGpuQueue,omitempty"`
	GpuInstanceType    string `json:"gpuInstanceType,omitempty"`
	GpuMinSize         int    `json:"gpuMinSize,omitempty"`
	GpuMaxSize         int    `json:"gpuMaxSize,omitempty"`
	
	// EFA configuration for GPU queue
	GpuEnableEfa       *bool  `json:"gpuEnableEfa,omitempty"`
	
	// Placement group configuration for GPU queue
	GpuPlacementGroupEnabled *bool  `json:"gpuPlacementGroupEnabled,omitempty"`
	GpuPgAzIndex            int    `json:"gpuPgAzIndex,omitempty"`

	// NICE DCV configuration
	EnableDcv          *bool  `json:"enableDcv,omitempty"`
	DcvPort            int    `json:"dcvPort,omitempty"`

	// Timeout settings
	HeadNodeBootstrapTimeout   int `json:"headNodeBootstrapTimeout,omitempty"`
	
	// 端口配置
	AllowedPorts     string `json:"allowedPorts,omitempty"`      // 允许的端口配置
	AllowedPortsIpv6 string `json:"allowedPortsIpv6,omitempty"`  // IPv6端口配置
	ComputeNodeBootstrapTimeout int `json:"computeNodeBootstrapTimeout,omitempty"`

	// Additional configuration
	Policies  string `json:"policies,omitempty"`
	DependsOn string `json:"dependsOn"`
}

// ParallelClusterForge implements the Forge interface for AWS ParallelCluster
type ParallelClusterForge struct {
	cluster awscdk.CustomResource
}

// Create implements the Forge interface
func (f *ParallelClusterForge) Create(ctx *interfaces.ForgeContext) interface{} {
	pcInstance, ok := (*ctx.Instance).(*ParallelClusterInstanceConfig)
	if !ok {
		fmt.Printf("Error: Failed to cast instance to ParallelClusterInstanceConfig\n")
		return nil
	}

	if pcInstance.KeyName == "" {
		pcInstance.KeyName = "pcluster"
	}

	// Get or create key pair
	aws.CreateOrGetKeyPair(ctx.Stack, pcInstance.KeyName, "Linux")
	//keyPair := aws.CreateOrGetKeyPair(ctx.Stack, pcInstance.KeyName, "Linux")

	// 获取当前区域的所有可用区
	// 移除未使用的 availabilityZones 变量
	// availabilityZones := ctx.VPC.AvailabilityZones()

	// 创建子网选择对象
	headNodeSelection := &awsec2.SubnetSelection{
		SubnetType: ctx.SubnetType,
	}

	computeNodeSelection := &awsec2.SubnetSelection{
		SubnetType: awsec2.SubnetType_PRIVATE_WITH_EGRESS,
	}

	// 使用 VPC 的 SelectSubnets 方法获取子网
	selectedHeadNodeSubnets := ctx.VPC.SelectSubnets(headNodeSelection)
	selectedComputeNodeSubnets := ctx.VPC.SelectSubnets(computeNodeSelection)

	// 获取子网切片
	headNodeSubnets := selectedHeadNodeSubnets.Subnets
	computeNodeSubnets := selectedComputeNodeSubnets.Subnets

	// 首先解引用指针，获取切片
	subnets := *computeNodeSubnets

	// 创建一个字符串切片来存储所有子网 ID
	computeNodeSubnetIds := make([]string, len(subnets))
	for i, subnet := range subnets {
		computeNodeSubnetIds[i] = *subnet.SubnetId()  // 假设 SubnetId() 返回 *string
	}

	// 确保有可用的子网
	if len(*headNodeSubnets) == 0 {
		// 处理没有子网的情况

		panic("No subnets available for head node")
	}

	// 使用统一的子网选择函数选择头节点子网
	headNodeSubnet := aws.SelectSubnetByAzIndex(pcInstance.AzIndex, ctx.VPC, ctx.SubnetType)

	// 这里我们使用传入的sg作为头节点安全组，使用ctx.Dependencies中的private安全组作为计算节点安全组
	// 假设ctx.Dependencies中包含了securityGroups
	var computeNodeSg awsec2.SecurityGroup
	if privateSg, ok := ctx.Dependencies["private"].(awsec2.SecurityGroup); ok {
		computeNodeSg = privateSg
	} else {
		// 如果没有找到预定义的私有安全组，则使用传入的sg作为备选
		computeNodeSg = ctx.SecurityGroups.Default
	}

	// Create the ParallelCluster provider stack
	providerStack := awscdk.NewNestedStack(ctx.Stack, jsii.String(fmt.Sprintf("%s-provider", pcInstance.ID)), &awscdk.NestedStackProps{})

	/*
	// Create the provider resource
	providerResource := awscdk.NewCfnCustomResource(providerStack, jsii.String(fmt.Sprintf("%s-provider-resource", pcInstance.ID)), &awscdk.CfnCustomResourceProps{
		ServiceToken: jsii.String(fmt.Sprintf(
			"https://${AWS::Region}-aws-parallelcluster.s3.${AWS::Region}.${AWS::URLSuffix}/parallelcluster/%s/templates/custom_resource/cluster.yaml",
			pcInstance.Version,
		)),
		ServiceTimeout: jsii.Number(1800),
	})
	*/

	// 创建Provider Stack作为嵌套堆栈
	var templateUrl string
	if partition.DefaultPartition == "aws-cn" {
		templateUrl = fmt.Sprintf(
			"https://%s-aws-parallelcluster.s3.%s.amazonaws.com.cn/parallelcluster/%s/templates/custom_resource/cluster.yaml",
			partition.DefaultRegion,
			partition.DefaultRegion,
			pcInstance.Version,
		)
	} else {
		templateUrl = fmt.Sprintf(
			"https://%s-aws-parallelcluster.s3.%s.amazonaws.com/parallelcluster/%s/templates/custom_resource/cluster.yaml",
			partition.DefaultRegion,
			partition.DefaultRegion,
			pcInstance.Version,
		)
	}

	// Create ParallelCluster provider resource with Lambda execution role
	// AdditionalIamPolicies are attached to the Lambda role (not HeadNode) to allow:
	// - AmazonSSMManagedInstanceCore: Required for ParallelCluster operations
	// - IAMFullAccess: Required to attach/detach IAM policies to HeadNode role
	providerResource := awscdk.NewCfnStack(providerStack, jsii.String(fmt.Sprintf("%s-provider-resource", pcInstance.ID)), &awscdk.CfnStackProps{
		TemplateUrl: jsii.String(templateUrl),
		Parameters: &map[string]*string{
			"AdditionalIamPolicies": jsii.String(fmt.Sprintf("arn:%s:iam::aws:policy/AmazonSSMManagedInstanceCore,arn:%s:iam::aws:policy/IAMFullAccess", partition.DefaultPartition, partition.DefaultPartition)),
		},
	})

	// Create IAM role for ParallelCluster
	/*
	clusterRole := awsiam.NewRole(ctx.Stack, jsii.String(fmt.Sprintf("%s-role", pcInstance.ID)), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), &awsiam.ServicePrincipalOpts{}),
		ManagedPolicies: &[]awsiam.IManagedPolicy{
			awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String("AmazonSSMManagedInstanceCore")),
		},
		Description: jsii.String("Role for ParallelCluster instances"),
	})


	if pcInstance.AdditionalPolicies != "" {
		aws.AddManagedPolicies(clusterRole, pcInstance.AdditionalPolicies)
	}
	*/

	// Create the cluster configuration
	clusterConfig := map[string]interface{}{
		"Image": map[string]interface{}{
			"Os": pcInstance.OsType,
		},
		"HeadNode": map[string]interface{}{
			"InstanceType": pcInstance.HeadNodeType,
			"Networking": map[string]interface{}{
				"SubnetId": headNodeSubnet.SubnetId(),
				"SecurityGroups": []string{*ctx.SecurityGroups.Default.SecurityGroupId()}, // 使用传入的安全组
			},
			"Ssh": map[string]interface{}{
				"KeyName": fmt.Sprintf("%s-linux-%s", pcInstance.KeyName, partition.DefaultRegion),
			},
			"Iam": map[string]interface{}{
				"AdditionalIamPolicies": buildAdditionalIamPolicies(pcInstance.Policies, types.GetBoolValue(pcInstance.EnableDcv, false), partition.DefaultPartition),
			},
			"LocalStorage": map[string]interface{}{
				"RootVolume": map[string]interface{}{
					"Size": pcInstance.DiskSize,
				},
			},
			"CustomActions": map[string]interface{}{
				"OnNodeConfigured": map[string]interface{}{
					"Script": getOnNodeConfiguredScriptPath(pcInstance),
					"Args": []string{pcInstance.UserDataToken},
				},
			},
		},
		"Scheduling": map[string]interface{}{
			"Scheduler": "slurm",
			"SlurmQueues": getSlurmQueues(pcInstance, computeNodeSubnetIds, computeNodeSg, ctx),
		},
	}
	
	// 添加 ScalingStrategy 配置
	if pcInstance.ScalingStrategy != "" {
		scheduling := clusterConfig["Scheduling"].(map[string]interface{})
		scheduling["ScalingStrategy"] = pcInstance.ScalingStrategy
	}
	
	// 添加 NICE DCV 配置
	if types.GetBoolValue(pcInstance.EnableDcv, false) {
		dcvConfig := map[string]interface{}{
			"Enabled": true,
		}
		
		// 设置 DCV 端口
		if pcInstance.DcvPort > 1024 {
			dcvConfig["Port"] = pcInstance.DcvPort
		} else {
			dcvConfig["Port"] = 8443 // 默认端口
		}
		
		// DCV 通过 allowedPorts 参数控制访问，不需要单独的 IP 配置
		// 用户需要在 allowedPorts 中包含 DCV 端口（通常是 8443）
		
		// 将 DCV 配置添加到 HeadNode 配置中
		headNode := clusterConfig["HeadNode"].(map[string]interface{})
		headNode["Dcv"] = dcvConfig
	}

	// Add timeout settings if provided
	if pcInstance.HeadNodeBootstrapTimeout > 0 || pcInstance.ComputeNodeBootstrapTimeout > 0 {
		timeouts := make(map[string]interface{})
		
		if pcInstance.HeadNodeBootstrapTimeout > 0 {
			timeouts["HeadNodeBootstrapTimeout"] = pcInstance.HeadNodeBootstrapTimeout
		}
		
		if pcInstance.ComputeNodeBootstrapTimeout > 0 {
			timeouts["ComputeNodeBootstrapTimeout"] = pcInstance.ComputeNodeBootstrapTimeout
		}
		
		if len(timeouts) > 0 {
			clusterConfig["DevSettings"] = map[string]interface{}{
				"Timeouts": timeouts,
			}
		}
	}


	magicToken, err := dependency.GetDependencyInfo(pcInstance.DependsOn)
	//fmt.Printf("*** MagicToken = %s", magicToken)
	if err != nil {
		fmt.Printf("Error getting dependency info: %v\n", err)
	}

	// 处理Directory Service配置
	dsProperties, err := dependency.ExtractDependencyProperties(magicToken, "DS")
	if err == nil {
		directoryConfig := buildDirectoryConfig(dsProperties)
		if directoryConfig != nil {
			clusterConfig["DirectoryService"] = directoryConfig
		}
	}

	// 处理RDS数据库配置
	rdsProperties, err := dependency.ExtractDependencyProperties(magicToken, "RDS")
	if err == nil {
		databaseConfig := buildDatabaseConfig(rdsProperties, pcInstance.DatabaseName)
		if databaseConfig != nil {
			// Database配置应该在Scheduling.SlurmSettings下
			scheduling := clusterConfig["Scheduling"].(map[string]interface{})
			if slurmSettings, ok := scheduling["SlurmSettings"].(map[string]interface{}); ok {
				slurmSettings["Database"] = databaseConfig
			} else {
				scheduling["SlurmSettings"] = map[string]interface{}{
					"Database": databaseConfig,
				}
			}
		}
	}

	addEfsToClusterConfig(clusterConfig, magicToken)

	addLustreToClusterConfig(clusterConfig, magicToken)

	// Create the ParallelCluster custom resource
	//serviceTokenRef := providerResource.GetAtt(jsii.String("ServiceToken"), awscdk.ResolutionTypeHint_STRING)
	serviceTokenRef := providerResource.GetAtt(jsii.String("Outputs.ServiceToken"), awscdk.ResolutionTypeHint_STRING)

	// 生成一个有效的集群名称
	clusterName := pcInstance.ClusterName
	if clusterName == "" {
		// 确保生成的名称以字母开头，只包含字母、数字和连字符
		clusterName = fmt.Sprintf("%s", pcInstance.ID)
	}

	f.cluster = awscdk.NewCustomResource(ctx.Stack, jsii.String(pcInstance.ID), &awscdk.CustomResourceProps{
		ServiceToken: serviceTokenRef.ToString(),
		Properties: &map[string]interface{}{
			"ClusterName":          clusterName,
			"ClusterConfiguration": clusterConfig,
		},
	})

	// Add dependency on the provider stack
	f.cluster.Node().AddDependency(providerStack)

	return f.cluster
}

// ConfigureRules implements the Forge interface
func (f *ParallelClusterForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// 检查是否有自定义端口配置
	if ctx.Instance != nil {
		pcInstance, ok := (*ctx.Instance).(*ParallelClusterInstanceConfig)
		if ok && (pcInstance.AllowedPorts != "" || pcInstance.AllowedPortsIpv6 != "") {
			// 使用通用端口规则处理函数
			security.ApplyPortRules(ctx.SecurityGroups.Public, pcInstance.AllowedPorts, pcInstance.AllowedPortsIpv6, ctx.DualStack)
			
			// ParallelCluster 特有的内部通信规则
			security.AddAllTrafficIngressRule(ctx.SecurityGroups.Private, ctx.SecurityGroups.Private, "Allow all traffic within private security group")
			security.AddAllTrafficIngressRule(ctx.SecurityGroups.Private, ctx.SecurityGroups.Public, "Allow all traffic from head node to compute nodes")
			security.AddAllTrafficIngressRule(ctx.SecurityGroups.Public, ctx.SecurityGroups.Private, "Allow all traffic from compute nodes to head node")
			security.ConfigureEFASecurityRules(ctx.SecurityGroups.Private, "parallelcluster")
			return
		}
	}

}

// CreateOutputs implements the Forge interface
func (f *ParallelClusterForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	pcInstance, ok := (*ctx.Instance).(*ParallelClusterInstanceConfig)
	if !ok || f.cluster == nil {
		return
	}

	// Export the head node IP
	awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("%s-head-node-ip", pcInstance.ID)), &awscdk.CfnOutputProps{
		Value:       f.cluster.GetAtt(jsii.String("headNode.privateIpAddress")).ToString(),
		Description: jsii.String(fmt.Sprintf("The private IP address of the ParallelCluster Head Node for %s", pcInstance.ID)),
	})

	// Export the head node instance ID
	awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("%s-head-node-id", pcInstance.ID)), &awscdk.CfnOutputProps{
		Value:       f.cluster.GetAtt(jsii.String("headNode.instanceId")).ToString(),
		Description: jsii.String(fmt.Sprintf("The Instance ID of the ParallelCluster Head Node for %s", pcInstance.ID)),
	})

	// Export the System Manager URL
	awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("%s-ssm-url", pcInstance.ID)), &awscdk.CfnOutputProps{
		Value: jsii.String(fmt.Sprintf(
			"https://%s.console.aws.amazon.com/systems-manager/session-manager/%s?region=%s",
			partition.DefaultRegion,
			*f.cluster.GetAtt(jsii.String("headNode.instanceId")).ToString(),
			partition.DefaultRegion,
		)),
		Description: jsii.String(fmt.Sprintf("URL to access the Head Node via System Manager for %s", pcInstance.ID)),
	})

	// Export validation messages
	awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("%s-validation-messages", pcInstance.ID)), &awscdk.CfnOutputProps{
		Value:       f.cluster.GetAtt(jsii.String("validationMessages")).ToString(),
		Description: jsii.String(fmt.Sprintf("Any warnings from cluster create or update operations for %s", pcInstance.ID)),
	})

	// 生成一个有效的集群名称
	clusterName := pcInstance.ClusterName
	if clusterName == "" {
		// 确保生成的名称以字母开头，只包含字母、数字和连字符
		clusterName = fmt.Sprintf("%s", pcInstance.ID)
	}

	// Export cluster name
	awscdk.NewCfnOutput(ctx.Stack, jsii.String(fmt.Sprintf("%s-cluster-name", pcInstance.ID)), &awscdk.CfnOutputProps{
		Value:       jsii.String(fmt.Sprintf("%s", clusterName)),
		Description: jsii.String(fmt.Sprintf("ParallelCluster name for %s", pcInstance.ID)),
	})
}

// MergeConfigs implements the Forge interface
func (f *ParallelClusterForge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
	// 从默认配置中复制基本字段
	merged := defaults.(*ParallelClusterInstanceConfig)

	// 从实例配置中覆盖基本字段
	parallelClusterInstance := instance.(*ParallelClusterInstanceConfig)
	if instance != nil {
		if parallelClusterInstance.GetID() != "" {
			merged.ID = parallelClusterInstance.GetID()
		}
		if parallelClusterInstance.Type != "" {
			merged.Type = parallelClusterInstance.GetType()
		}
		if parallelClusterInstance.Subnet != "" {
			merged.Subnet = parallelClusterInstance.GetSubnet()
		}
		if parallelClusterInstance.SecurityGroup != "" {
			merged.SecurityGroup = parallelClusterInstance.GetSecurityGroup()
		}
	}

	if parallelClusterInstance.KeyName != "" {
		merged.KeyName = parallelClusterInstance.KeyName
	}

	if parallelClusterInstance.Version != "" {
		merged.Version = parallelClusterInstance.Version
	}

	if parallelClusterInstance.HeadNodeType != "" {
		merged.HeadNodeType = parallelClusterInstance.HeadNodeType
	}

	if parallelClusterInstance.AzIndex != 0 {
		merged.AzIndex = parallelClusterInstance.AzIndex
	}

	if parallelClusterInstance.ComputeNodeType != "" {
		merged.ComputeNodeType = parallelClusterInstance.ComputeNodeType
	}

	if parallelClusterInstance.OsType != "" {
		merged.OsType = parallelClusterInstance.OsType
	}

	if parallelClusterInstance.ClusterName != "" {
		merged.ClusterName = parallelClusterInstance.ClusterName
	}

	if parallelClusterInstance.UserDataToken != "" {
		merged.UserDataToken = parallelClusterInstance.UserDataToken
	}

	if parallelClusterInstance.UserDataScriptPath != "" {
		merged.UserDataScriptPath = parallelClusterInstance.UserDataScriptPath
	}

	if parallelClusterInstance.DiskSize != 0 {
		merged.DiskSize = parallelClusterInstance.DiskSize
	}

	if parallelClusterInstance.MinSize != 0 {
		merged.MinSize = parallelClusterInstance.MinSize
	}

	if parallelClusterInstance.MaxSize != 0 {
		merged.MaxSize = parallelClusterInstance.MaxSize
	}
	
	// 合并新添加的字段
	merged.DisableSimultaneousMultithreading = parallelClusterInstance.DisableSimultaneousMultithreading
	
	if parallelClusterInstance.AllocationStrategy != "" {
		merged.AllocationStrategy = parallelClusterInstance.AllocationStrategy
	}
	
	if parallelClusterInstance.SpotAllocationStrategy != "" {
		merged.SpotAllocationStrategy = parallelClusterInstance.SpotAllocationStrategy
	}
	
	if parallelClusterInstance.ScalingStrategy != "" {
		merged.ScalingStrategy = parallelClusterInstance.ScalingStrategy
	}

	// 合并 EFA 配置 (CPU 队列)
	if parallelClusterInstance.EnableEfa != nil {
		merged.EnableEfa = parallelClusterInstance.EnableEfa
	}
	
	// 合并 CPU 队列的放置组配置
	if parallelClusterInstance.PlacementGroupEnabled != nil {
		merged.PlacementGroupEnabled = parallelClusterInstance.PlacementGroupEnabled
	}
	
	if parallelClusterInstance.PlacementGroupId != "" {
		merged.PlacementGroupId = parallelClusterInstance.PlacementGroupId
	}
	
	if parallelClusterInstance.PgAzIndex != 0 {
		merged.PgAzIndex = parallelClusterInstance.PgAzIndex
	}

	if parallelClusterInstance.Policies != "" {
		merged.Policies = parallelClusterInstance.Policies
	}

	if parallelClusterInstance.DependsOn != "" {
		merged.DependsOn = parallelClusterInstance.DependsOn
	}

	if parallelClusterInstance.HeadNodeBootstrapTimeout > 1200 {
		merged.HeadNodeBootstrapTimeout = parallelClusterInstance.HeadNodeBootstrapTimeout
	}

	if parallelClusterInstance.ComputeNodeBootstrapTimeout > 1200 {
		merged.ComputeNodeBootstrapTimeout = parallelClusterInstance.ComputeNodeBootstrapTimeout
	}

	if parallelClusterInstance.EnableGpuQueue != nil {
		merged.EnableGpuQueue = parallelClusterInstance.EnableGpuQueue
	}

	if parallelClusterInstance.GpuInstanceType != "" {
		merged.GpuInstanceType = parallelClusterInstance.GpuInstanceType
	}

	if parallelClusterInstance.GpuMinSize != 0 {
		merged.GpuMinSize = parallelClusterInstance.GpuMinSize
	}

	if parallelClusterInstance.GpuMaxSize != 0 {
		merged.GpuMaxSize = parallelClusterInstance.GpuMaxSize
	}
	
	// 合并 GPU 队列的 EFA 配置
	if parallelClusterInstance.GpuEnableEfa != nil {
		merged.GpuEnableEfa = parallelClusterInstance.GpuEnableEfa
	}
	
	// 合并 GPU 队列的放置组配置
	if parallelClusterInstance.GpuPlacementGroupEnabled != nil {
		merged.GpuPlacementGroupEnabled = parallelClusterInstance.GpuPlacementGroupEnabled
	}
	
	if parallelClusterInstance.GpuPgAzIndex != 0 {
		merged.GpuPgAzIndex = parallelClusterInstance.GpuPgAzIndex
	}

	if parallelClusterInstance.EnableDcv != nil {
		merged.EnableDcv = parallelClusterInstance.EnableDcv
	}

	if parallelClusterInstance.DcvPort > 0 {
		merged.DcvPort = parallelClusterInstance.DcvPort
	}

	if parallelClusterInstance.DcvPort != 0 {
		merged.DcvPort = parallelClusterInstance.DcvPort
	}

	return merged
}

// 解析 magic_token
// buildDirectoryConfig 根据DS属性构建DirectoryService配置
func buildDirectoryConfig(dsProperties map[string]interface{}) map[string]interface{} {
	domainName, _ := dsProperties["name"].(string)
	secretARN, _ := dsProperties["secretARN"].(string)
	shortName, _ := dsProperties["shortName"].(string)

	if domainName == "" || secretARN == "" || shortName == "" {
		return nil
	}

	// 处理DNS IP地址
	ldapUrls := []string{}
	if dnsIPs, ok := dsProperties["attrDnsIpAddresses"].([]interface{}); ok {
		for _, ip := range dnsIPs {
			if ipStr, ok := ip.(string); ok {
				ldapUrls = append(ldapUrls, fmt.Sprintf("ldap://%s", ipStr))
			}
		}
	}

	// 构造域名组件
	domainComponents := []string{}
	for _, part := range strings.Split(domainName, ".") {
		domainComponents = append(domainComponents, fmt.Sprintf("DC=%s", part))
	}
	dcPath := strings.Join(domainComponents, ",")

	// 构造DomainReadOnlyUser
	domainReadOnlyUser := fmt.Sprintf("CN=ReadOnlyUser,OU=Users,OU=%s,%s", shortName, dcPath)

	return map[string]interface{}{
		"GenerateSshKeysForUsers": true,
		"DomainName":             dcPath,
		"DomainAddr":             strings.Join(ldapUrls, ","),
		"PasswordSecretArn":      secretARN,
		"DomainReadOnlyUser":     domainReadOnlyUser,
		"AdditionalSssdConfigs": map[string]interface{}{
			"ldap_auth_disable_tls_never_use_in_production": "True",
			"ldap_id_mapping": "False",
		},
	}
}

// buildDatabaseConfig 根据RDS属性构建Database配置
func buildDatabaseConfig(rdsProperties map[string]interface{}, databaseName string) map[string]interface{} {
	uri, _ := rdsProperties["endpoint"].(string)
	username, _ := rdsProperties["username"].(string)
	secretArn, _ := rdsProperties["secretArn"].(string)

	if uri == "" || username == "" || secretArn == "" {
		return nil
	}

	databaseConfig := map[string]interface{}{
		"Uri":               uri,
		"UserName":          username,
		"PasswordSecretArn": secretArn,
	}

	// 只有明确指定了databaseName才添加，否则使用默认
	if databaseName != "" {
		databaseConfig["DatabaseName"] = databaseName
	}

	return databaseConfig
}

func addEfsToClusterConfig(clusterConfig map[string]interface{}, magicTokenStr string) error {
	efsProperties, err := dependency.ExtractDependencyProperties(magicTokenStr, "EFS")
	if err != nil {
		return err
	}

	efsConfig := buildEfsConfig(efsProperties)
	if efsConfig == nil {
		return fmt.Errorf("failed to build EFS config")
	}

	// 添加SharedStorage配置
	sharedStorage, ok := clusterConfig["SharedStorage"].([]map[string]interface{})
	if !ok {
		sharedStorage = []map[string]interface{}{}
	}

	sharedStorage = append(sharedStorage, efsConfig)
	clusterConfig["SharedStorage"] = sharedStorage

	return nil
}

func addLustreToClusterConfig(clusterConfig map[string]interface{}, magicTokenStr string) error {
	lustreProperties, err := dependency.ExtractDependencyProperties(magicTokenStr, "LUSTRE")
	if err != nil {
		return err
	}

	lustreConfig := buildLustreConfig(lustreProperties)
	if lustreConfig == nil {
		return fmt.Errorf("failed to build Lustre config")
	}

	// 添加SharedStorage配置
	sharedStorage, ok := clusterConfig["SharedStorage"].([]map[string]interface{})
	if !ok {
		sharedStorage = []map[string]interface{}{}
	}

	sharedStorage = append(sharedStorage, lustreConfig)
	clusterConfig["SharedStorage"] = sharedStorage

	return nil
}

// buildEfsConfig 根据EFS属性构建EFS配置
func buildEfsConfig(efsProperties map[string]interface{}) map[string]interface{} {
	fileSystemId, _ := efsProperties["fileSystemId"].(string)
	mountPoint, _ := efsProperties["mountPoint"].(string)

	if fileSystemId == "" || mountPoint == "" {
		return nil
	}

	return map[string]interface{}{
		"MountDir":    mountPoint,
		"Name":        "efs",
		"StorageType": "Efs",
		"EfsSettings": map[string]interface{}{
			"FileSystemId": fileSystemId,
		},
	}
}

// buildLustreConfig 根据Lustre属性构建Lustre配置
func buildLustreConfig(lustreProperties map[string]interface{}) map[string]interface{} {
	fileSystemId, _ := lustreProperties["fileSystemId"].(string)
	mountPoint, _ := lustreProperties["mountPoint"].(string)

	if fileSystemId == "" || mountPoint == "" {
		return nil
	}

	return map[string]interface{}{
		"MountDir":    mountPoint,
		"Name":        "InfraForgeFsx",
		"StorageType": "FsxLustre",
		"FsxLustreSettings": map[string]interface{}{
			"FileSystemId": fileSystemId,
		},
	}
}

// 解析magicToken JSON字符串
func getOnNodeConfiguredScriptPath(pcInstance *ParallelClusterInstanceConfig) string {
	// If no token is provided, return empty string
	if pcInstance.UserDataToken == "" {
		return ""
	}

	// If a custom script path directory is provided, use it to construct the full path
	if pcInstance.UserDataScriptPath != "" {
		return fmt.Sprintf("%s/user_data_%s.sh", pcInstance.UserDataScriptPath, pcInstance.UserDataToken)
	}

	// Otherwise use the default path with the token
	return fmt.Sprintf("https://aws-hpc-builder.s3.amazonaws.com/project/user_data/user_data_%s.sh", pcInstance.UserDataToken)
}

// buildAdditionalIamPolicies 构建 IAM 策略列表
func buildAdditionalIamPolicies(policies string, enableDcv bool, partition string) []map[string]interface{} {
	policyList := []map[string]interface{}{}
	addedPolicies := make(map[string]bool)

	// 如果启用 DCV，自动添加 SSM 策略（DCV 需要 SSM 支持）
	if enableDcv {
		ssmPolicy := fmt.Sprintf("arn:%s:iam::aws:policy/AmazonSSMManagedInstanceCore", partition)
		policyList = append(policyList, map[string]interface{}{
			"Policy": ssmPolicy,
		})
		addedPolicies[ssmPolicy] = true
		addedPolicies["AmazonSSMManagedInstanceCore"] = true
	}

	// 添加用户指定的策略
	if policies != "" {
		policyNames := strings.Split(policies, ",")
		for _, policyName := range policyNames {
			policyName = strings.TrimSpace(policyName)
			if policyName == "" {
				continue
			}
			
			// Build full ARN for the policy
			var policyArn string
			if strings.HasPrefix(policyName, "arn:") {
				policyArn = policyName
			} else if strings.Contains(policyName, "/") {
				// service-role/PolicyName format
				policyArn = fmt.Sprintf("arn:%s:iam::aws:policy/%s", partition, policyName)
			} else {
				// Simple policy name
				policyArn = fmt.Sprintf("arn:%s:iam::aws:policy/%s", partition, policyName)
			}
			
			// 避免重复添加
			if !addedPolicies[policyArn] && !addedPolicies[policyName] {
				policyList = append(policyList, map[string]interface{}{
					"Policy": policyArn,
				})
				addedPolicies[policyArn] = true
				addedPolicies[policyName] = true
			}
		}
	}

	return policyList
}

// getSlurmQueues 函数用于根据配置生成Slurm队列配置
func getSlurmQueues(pcInstance *ParallelClusterInstanceConfig, computeNodeSubnetIds []string, computeNodeSg awsec2.SecurityGroup, ctx *interfaces.ForgeContext) []map[string]interface{} {
	// 处理逗号分隔的实例类型列表
	getInstancesConfig := func(instanceTypeStr string) []map[string]interface{} {
		instanceTypes := strings.Split(instanceTypeStr, ",")
		instances := make([]map[string]interface{}, len(instanceTypes))
		
		for i, instanceType := range instanceTypes {
			instances[i] = map[string]interface{}{
				"InstanceType": strings.TrimSpace(instanceType),
			}
		}
		
		return instances
	}

	// 创建计算资源配置（支持多实例类型）
	createComputeResources := func(instanceType string, minSize, maxSize int, disableSMT *bool, queueType string) []map[string]interface{} {
		instanceTypes := strings.Split(instanceType, ",")
		computeResources := []map[string]interface{}{}

		// 1. 如果有多个实例类型，创建混合资源
		if len(instanceTypes) > 1 {
			mixedComputeResource := map[string]interface{}{
				"Name":                              "auto",
				"Instances":                         getInstancesConfig(instanceType),
				"MinCount":                          minSize,
				"MaxCount":                          maxSize,
				"DisableSimultaneousMultithreading": disableSMT,
			}
			computeResources = append(computeResources, mixedComputeResource)
		}

		// 2. 为每个实例类型创建单独资源
		for _, instType := range instanceTypes {
			instType = strings.TrimSpace(instType)
			if instType == "" {
				continue
			}

			instanceFamily := strings.Split(instType, ".")[0]

			singleComputeResource := map[string]interface{}{
				"Name":                              instanceFamily,
				"Instances":                         getInstancesConfig(instType),
				"MinCount":                          0,
				"MaxCount":                          maxSize,
				"DisableSimultaneousMultithreading": disableSMT,
			}
			computeResources = append(computeResources, singleComputeResource)
		}

		// 3. 如果只有一个实例类型，也创建主资源
		if len(instanceTypes) == 1 {
			instType := strings.TrimSpace(instanceTypes[0])

			mainComputeResource := map[string]interface{}{
				"Name":                              "auto",
				"Instances":                         getInstancesConfig(instType),
				"MinCount":                          minSize,
				"MaxCount":                          maxSize,
				"DisableSimultaneousMultithreading": disableSMT,
			}
			computeResources = append(computeResources, mainComputeResource)
		}

		return computeResources
	}

	// 为 CPU 队列选择子网
	var cpuSubnetIds []string
	if types.GetBoolValue(pcInstance.PlacementGroupEnabled, false) {
		// 如果启用了放置组，只使用指定的可用区
		azIndex := pcInstance.AzIndex
		if pcInstance.PgAzIndex > 0 {
			azIndex = pcInstance.PgAzIndex
		}
		
		// 使用统一的子网选择函数
		subnetId := aws.SelectSubnetIdByAzIndex(azIndex, ctx.VPC, awsec2.SubnetType_PRIVATE_WITH_EGRESS)
		cpuSubnetIds = []string{subnetId}
	} else {
		// 如果没有启用放置组，使用所有子网
		cpuSubnetIds = computeNodeSubnetIds
	}

	// 创建 CPU 队列的网络配置
	cpuNetworkingConfig := map[string]interface{}{
		"SubnetIds":      cpuSubnetIds,
		"SecurityGroups": []string{*computeNodeSg.SecurityGroupId()},
	}
	
	// 为 CPU 队列添加放置组配置（如果启用）
	if types.GetBoolValue(pcInstance.PlacementGroupEnabled, false) {
		placementGroup := map[string]interface{}{
			"Enabled": true,
		}
		
		// 如果指定了放置组 ID，则使用它
		if pcInstance.PlacementGroupId != "" {
			placementGroup["Id"] = pcInstance.PlacementGroupId
		}
		
		cpuNetworkingConfig["PlacementGroup"] = placementGroup
	}
	
	// 创建 CPU 计算资源配置
	cpuComputeResources := createComputeResources(
		pcInstance.ComputeNodeType,
		pcInstance.MinSize,
		pcInstance.MaxSize,
		pcInstance.DisableSimultaneousMultithreading,
		"cpu",
	)
	
	// 为 CPU 队列添加 EFA 配置（如果启用）
	if types.GetBoolValue(pcInstance.EnableEfa, false) {
		for i := range cpuComputeResources {
			cpuComputeResources[i]["Efa"] = map[string]interface{}{
				"Enabled": true,
			}
		}
	}
	
	// 创建 CPU 按需队列
	cpuQueue := map[string]interface{}{
		"Name": "cpu",
		"ComputeResources": cpuComputeResources,
		"Networking": cpuNetworkingConfig,
	}
	
	// 设置分配策略
	if pcInstance.AllocationStrategy != "" {
		cpuQueue["AllocationStrategy"] = pcInstance.AllocationStrategy
	}
	
	queues := []map[string]interface{}{cpuQueue}

	// 创建 CPU Spot 计算资源配置
	cpuSpotComputeResources := createComputeResources(
		pcInstance.ComputeNodeType,
		0,
		pcInstance.MaxSize,
		pcInstance.DisableSimultaneousMultithreading,
		"cpu-spot",
	)
	
	// 为 CPU Spot 队列添加 EFA 配置（如果启用）
	if types.GetBoolValue(pcInstance.EnableEfa, false) {
		for i := range cpuSpotComputeResources {
			cpuSpotComputeResources[i]["Efa"] = map[string]interface{}{
				"Enabled": true,
			}
		}
	}
	
	// 创建 CPU Spot 队列
	cpuSpotQueue := map[string]interface{}{
		"Name": "cpu-spot",
		"ComputeResources": cpuSpotComputeResources,
		"Networking": cpuNetworkingConfig, // 使用与 CPU 队列相同的网络配置
		"CapacityType": "SPOT",
	}
	
	// 为 Spot 队列设置专门的分配策略
	if pcInstance.SpotAllocationStrategy != "" {
		cpuSpotQueue["AllocationStrategy"] = pcInstance.SpotAllocationStrategy
	} else if pcInstance.AllocationStrategy != "" {
		// 如果没有指定Spot专用策略，则使用通用策略
		cpuSpotQueue["AllocationStrategy"] = pcInstance.AllocationStrategy
	}
	
	queues = append(queues, cpuSpotQueue)

	// 如果启用了GPU队列，添加GPU队列配置
	if types.GetBoolValue(pcInstance.EnableGpuQueue, false) {
		// 设置默认值
		gpuInstanceType := "g4dn.xlarge" // 默认GPU实例类型
		if pcInstance.GpuInstanceType != "" {
			gpuInstanceType = pcInstance.GpuInstanceType
		}

		gpuMinSize := 0 // 默认最小节点数
		if pcInstance.GpuMinSize > 0 {
			gpuMinSize = pcInstance.GpuMinSize
		}

		gpuMaxSize := 4 // 默认最大节点数
		if pcInstance.GpuMaxSize > 0 {
			gpuMaxSize = pcInstance.GpuMaxSize
		}

		// 为 GPU 队列选择子网
		var gpuSubnetIds []string
		if types.GetBoolValue(pcInstance.GpuPlacementGroupEnabled, false) {
			// 如果启用了放置组，只使用指定的可用区
			azIndex := pcInstance.AzIndex
			if pcInstance.GpuPgAzIndex > 0 {
				azIndex = pcInstance.GpuPgAzIndex
			}
			
			// 使用统一的子网选择函数
			subnetId := aws.SelectSubnetIdByAzIndex(azIndex, ctx.VPC, awsec2.SubnetType_PRIVATE_WITH_EGRESS)
			gpuSubnetIds = []string{subnetId}
		} else {
			// 如果没有启用放置组，使用所有子网
			gpuSubnetIds = computeNodeSubnetIds
		}

		// 创建 GPU 队列的网络配置
		gpuNetworkingConfig := map[string]interface{}{
			"SubnetIds":      gpuSubnetIds,
			"SecurityGroups": []string{*computeNodeSg.SecurityGroupId()},
		}
		
		// 为 GPU 队列添加放置组配置（如果启用）
		if types.GetBoolValue(pcInstance.GpuPlacementGroupEnabled, false) {
			gpuPlacementGroup := map[string]interface{}{
				"Enabled": true,
			}
			
			gpuNetworkingConfig["PlacementGroup"] = gpuPlacementGroup
		}

		// 创建 GPU 计算资源配置
		gpuComputeResources := createComputeResources(
			gpuInstanceType,
			gpuMinSize,
			gpuMaxSize,
			pcInstance.DisableSimultaneousMultithreading,
			"gpu",
		)
		
		// 为 GPU 队列添加 EFA 配置（如果启用）
		if types.GetBoolValue(pcInstance.GpuEnableEfa, false) {
			for i := range gpuComputeResources {
				gpuComputeResources[i]["Efa"] = map[string]interface{}{
					"Enabled": true,
				}
			}
		}

		// 添加 GPU 按需队列
		gpuQueue := map[string]interface{}{
			"Name": "gpu",
			"ComputeResources": gpuComputeResources,
			"Networking": gpuNetworkingConfig,
		}
		
		// 设置分配策略
		if pcInstance.AllocationStrategy != "" {
			gpuQueue["AllocationStrategy"] = pcInstance.AllocationStrategy
		}

		queues = append(queues, gpuQueue)
		
		// 创建 GPU Spot 计算资源配置
		gpuSpotComputeResources := createComputeResources(
			gpuInstanceType,
			0,
			gpuMaxSize,
			pcInstance.DisableSimultaneousMultithreading,
			"gpu-spot",
		)
		
		// 为 GPU Spot 队列添加 EFA 配置（如果启用）
		if types.GetBoolValue(pcInstance.GpuEnableEfa, false) {
			for i := range gpuSpotComputeResources {
				gpuSpotComputeResources[i]["Efa"] = map[string]interface{}{
					"Enabled": true,
				}
			}
		}
		
		// 添加 GPU Spot 队列
		gpuSpotQueue := map[string]interface{}{
			"Name": "gpu-spot",
			"ComputeResources": gpuSpotComputeResources,
			"Networking": gpuNetworkingConfig, // 使用与 GPU 队列相同的网络配置
			"CapacityType": "SPOT",
		}
		
		// 为 GPU Spot 队列设置专门的分配策略
		if pcInstance.SpotAllocationStrategy != "" {
			gpuSpotQueue["AllocationStrategy"] = pcInstance.SpotAllocationStrategy
		} else if pcInstance.AllocationStrategy != "" {
			// 如果没有指定Spot专用策略，则使用通用策略
			gpuSpotQueue["AllocationStrategy"] = pcInstance.AllocationStrategy
		}
		
		queues = append(queues, gpuSpotQueue)
	}

	return queues
}
