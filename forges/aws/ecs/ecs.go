// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ecs

import (
	"log"
	"fmt"
	"strings"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/security"
	"github.com/awslabs/InfraForge/core/dependency"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/awslabs/InfraForge/forges/aws/ecs/utils"
	"github.com/awslabs/InfraForge/core/partition"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsautoscaling"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awslogs"
	"github.com/aws/jsii-runtime-go"
)

type EcsInstanceConfig struct {
	config.BaseInstanceConfig
	UserDataToken            string `json:"userDataToken,omitempty"`
	UserDataScriptPath       string `json:"userDataScriptPath,omitempty"`
	AmiHardwareType          string `json:"amiHardwareType,omitempty"`
	DependsOn                string `json:"dependsOn,omitempty"`
	CpuCount                 int    `json:"cpuCount"`
	FargateCpuCount          int    `json:"fargateCpuCount"`
	GpuCount                 int    `json:"gpuCount"`
	Image                    string `json:"image"`
	MemoryMiB                int    `json:"memoryMiB"`
	FargateMemoryMiB         int    `json:"fargateMemoryMiB"`
	EbsIops                  int    `json:"ebsIops,omitempty"`
	EbsSize                  int    `json:"ebsSize,omitempty"`
	EbsThroughput            int    `json:"ebsThroughput,omitempty"`
	EbsVolumeType            string `json:"ebsVolumeType,omitempty"`
	EbsOptimized             *bool  `json:"ebsOptimized,omitempty"`
	ContainerInsights        string `json:"containerInsights,omitempty"`
	InstanceTypes            string `json:"instanceTypes"`
	MinCapacity              int    `json:"minCapacity,omitempty"`
	MaxCapacity              int    `json:"maxCapacity,omitempty"`
	NetworkMode              string `json:"networkMode,omitempty"`
	OnDemandPercentage       int    `json:"onDemandPercentage"`
	Policies                 string `json:"policies"`
	HealthCheck              string `json:"healthCheck,omitempty"`
	Interval                 int    `json:"interval,omitempty"`
	Retries                  int    `json:"retries,omitempty"`
	StartPeriod              int    `json:"startPeriod,omitempty"`
	Timeout                  int    `json:"timeout,omitempty"`
	EnableRestartPolicy      *bool  `json:"enableRestartPolicy,omitempty"`
	RestartAttemptPeriod     int    `json:"restartAttemptPeriod,omitempty"`
	TaskTypes                string `json:"taskTypes"`
	TaskRolePolicies         string `json:"taskRolePolicies,omitempty"`
	ExecutionRolePolicies    string `json:"executionRolePolicies,omitempty"`
	
	
}

type EcsForge struct {
	ecs      awsecs.Cluster
	properties map[string]interface{}
}

type TaskDefinitionConfig struct {
	EcsInstance *EcsInstanceConfig
	Actions     []*string
	Resources   []*string
	Command     []*string
}

// parseContainerInsights 将字符串配置转换为 ContainerInsights 枚举值
func parseContainerInsights(insightsConfig string) awsecs.ContainerInsights {
	// 将配置字符串转为小写以便不区分大小写比较
	configLower := strings.ToLower(insightsConfig)

	switch configLower {
	case "enhanced":
		return awsecs.ContainerInsights_ENHANCED
	case "enabled":
		return awsecs.ContainerInsights_ENABLED
	case "disabled":
		return awsecs.ContainerInsights_DISABLED
	default:
		// 默认值，如果配置不是有效值则禁用
		log.Printf("Warning: Invalid ContainerInsights value '%s', using DISABLED as default", insightsConfig)
		return awsecs.ContainerInsights_DISABLED
	}
}

func (e *EcsForge) Create(ctx *interfaces.ForgeContext) interface{} {
	ecsInstance, ok := (*ctx.Instance).(*EcsInstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return nil
	}

	cluster := awsecs.NewCluster(ctx.Stack, jsii.String(ecsInstance.GetID()), &awsecs.ClusterProps{
		ContainerInsightsV2: parseContainerInsights(ecsInstance.ContainerInsights),
		Vpc: ctx.VPC,
	})

	magicToken, err := dependency.GetDependencyInfo(ecsInstance.DependsOn)
	if err != nil {
		fmt.Printf("Error getting dependency info: %v\n", err)
	}

	userDataGenerator := &aws.UserDataGenerator{
		OsType:             awsec2.OperatingSystemType_LINUX,
		ScriptPath:         "./userdata.sh",
		UserDataToken:      ecsInstance.UserDataToken,
		UserDataScriptPath: ecsInstance.UserDataScriptPath,
		MagicToken:         magicToken,
	}

	userData, err := userDataGenerator.GenerateUserData()
	if err != nil {
		// 处理错误
		fmt.Errorf("Generate user data: %v", err)
	}

	role := awsiam.NewRole(ctx.Stack, jsii.String("ecsRole"), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), nil),
	})

	// 将托管策略添加到实例角色
	aws.AddManagedPolicies(role, ecsInstance.Policies)

	amiHardwareType := aws.ParseAmiHardwareType(ecsInstance.AmiHardwareType)
	var machineImage awsec2.IMachineImage
	// 到目前为止: 2025.03.09 Amazon Linux 2023 GPU 优化镜像，检查实例类型，使用 AL2 的 GPU 镜像
	/*
	if amiHardwareType == awsecs.AmiHardwareType_GPU {
		machineImage = awsecs.EcsOptimizedImage_AmazonLinux2(amiHardwareType, nil)
	} else {
		machineImage = awsecs.EcsOptimizedImage_AmazonLinux2023(amiHardwareType, nil)
	}
	*/

	// Doc: // https://aws.amazon.com/about-aws/whats-new/2025/03/amazon-ecs-gpu-optimized-ami-linux-2023/
	// 2025.03.10 ECS 正式推出 GPU-Optimized AMI for Amazon Linux 2023
	machineImage = awsecs.EcsOptimizedImage_AmazonLinux2023(amiHardwareType, nil)

	// CDK 的合成阶段，Token 还没有被解析成实际的 AMI ID, 不能通过 machineImage.GetImage(stack) 获得 machineImageConfig
	// ECS 所有 AMI(bottlerocket|Amazon Linux 2/2023) 的默认根设备名称是 /dev/xvda, 直接使用
	deviceName := "/dev/xvda" 

	// 创建配置（ECS 只支持单磁盘，转换为字符串）
	ebsConfig := &aws.EbsConfig{
		VolumeTypes:   ecsInstance.EbsVolumeType,
		Iops:          fmt.Sprintf("%d", ecsInstance.EbsIops),
		Sizes:         fmt.Sprintf("%d", ecsInstance.EbsSize),
		Throughputs:   fmt.Sprintf("%d", ecsInstance.EbsThroughput),
		Optimized:     types.GetBoolValue(ecsInstance.EbsOptimized, false),
		RootDevice:    deviceName,
	}

	// 调用函数
	blockDevices, err := aws.CreateEbsBlockDevices(ebsConfig)
	if err != nil {
		// 处理错误
		return fmt.Errorf("failed to create block devices: %w", err)
	}

	launchTemplate := awsec2.NewLaunchTemplate(ctx.Stack, jsii.String("ecs-instance"), &awsec2.LaunchTemplateProps{
		//InstanceType: awsec2.NewInstanceType(jsii.String("c7g.xlarge")),
		MachineImage: machineImage,
		RequireImdsv2: jsii.Bool(true),
		BlockDevices:  &blockDevices,
		SecurityGroup: ctx.SecurityGroups.Default,
		Role: role,
		UserData: userData,
	})

	overrides := aws.ParseInstanceTypeOverrides(ecsInstance.InstanceTypes)

	autoScalingGroup := awsautoscaling.NewAutoScalingGroup(ctx.Stack, jsii.String("ASG"), &awsautoscaling.AutoScalingGroupProps{
		//DesiredCapacity: jsii.Number(2),
		Vpc: ctx.VPC,
		MixedInstancesPolicy: &awsautoscaling.MixedInstancesPolicy{
			InstancesDistribution: &awsautoscaling.InstancesDistribution{
				// 设置基础容量使用按需实例(可选)
				OnDemandBaseCapacity: jsii.Number(0),

				// 关键参数:设置基础容量之外的按需实例百分比为 80%				
				OnDemandPercentageAboveBaseCapacity: jsii.Number(ecsInstance.OnDemandPercentage),

				// 按需实例的分配策略
				OnDemandAllocationStrategy: awsautoscaling.OnDemandAllocationStrategy_PRIORITIZED, // 使用优先级策略

				// 竞价实例的分配策略
				SpotAllocationStrategy: awsautoscaling.SpotAllocationStrategy_CAPACITY_OPTIMIZED, // 使用容量优化策略

				// 其他可选配置
				// SpotMaxPrice: jsii.String(""), // 留空表示使用按需实例价格作为竞价实例最高价
				// SpotInstancePools: jsii.Number(2), // 仅在使用 LOWEST_PRICE 策略时有效
			},
			LaunchTemplate: launchTemplate,
			LaunchTemplateOverrides: &overrides,
		},
		//MaxInstanceLifetime: awscdk.Duration_Seconds(jsii.Number(604800)),
		// 配置缩容策略
		TerminationPolicies: &[]awsautoscaling.TerminationPolicy{
			awsautoscaling.TerminationPolicy_OLDEST_INSTANCE,
		},
		NewInstancesProtectedFromScaleIn: jsii.Bool(false),
		MinCapacity: jsii.Number(ecsInstance.MinCapacity),  // 确保设置最小容量
		MaxCapacity: jsii.Number(ecsInstance.MaxCapacity), // 设置最大容量
		MaxInstanceLifetime: nil,
	})

	capacityProvider := awsecs.NewAsgCapacityProvider(ctx.Stack, jsii.String("AsgCapacityProvider"), &awsecs.AsgCapacityProviderProps{
		AutoScalingGroup: autoScalingGroup,
		EnableManagedScaling: jsii.Bool(true),
		TargetCapacityPercent: jsii.Number(80),
		EnableManagedTerminationProtection: jsii.Bool(false),
		//	MachineImageType: awsecs.MachineImageType_AMAZON_LINUX_2,
	})

	cluster.AddAsgCapacityProvider(capacityProvider,  &awsecs.AddAutoScalingGroupCapacityOptions{
		//CanContainersAccessInstanceRole: jsii.Bool(true),
	})

	healthCheckCommands := strings.Split(ecsInstance.HealthCheck, ",")
	command := []*string{
		jsii.String(healthCheckCommands[0]),
		jsii.String(healthCheckCommands[1]),
	}

	var actions = []*string{
		jsii.String("logs:CreateLogGroup"),
		jsii.String("logs:CreateLogStream"),
		jsii.String("logs:PutLogEvents"),
		jsii.String("logs:DescribeLogStreams"),
	}

	// 定义资源ARN常量 - 限制到AWS ECS日志组
	var resources = []*string{
		jsii.String(fmt.Sprintf("arn:%s:logs:%s:%s:log-group:/aws/ecs/%s/%s-*", 
			partition.DefaultPartition, *awscdk.Aws_REGION(), *awscdk.Aws_ACCOUNT_ID(), *ctx.Stack.StackName(), ecsInstance.GetID())),
	}


	taskConfig := TaskDefinitionConfig{
		EcsInstance: ecsInstance,
		Actions: actions,
		Resources: resources,
		Command: command,
	}

	createTaskDefinitions(ctx.Stack, taskConfig)

	/*
	var securityGroups []awsec2.ISecurityGroup
	securityGroups = append(securityGroups, awsec2.ISecurityGroup(ctx.SecurityGroups.Default))


	taskRole := taskDefinition.TaskRole()
	aws.AddManagedPolicies(taskRole, ecsInstance.TaskRolePolicies)

	executionRole := taskDefinition.ExecutionRole()
	aws.AddManagedPolicies(executionRole, ecsInstance.ExecutionRolePolicies)

	// 如果指定 DesiredCount > 1， stack 构建不会结束
	if ecsInstance.ServiceCount > 0 {
		subnetSelection := &awsec2.SubnetSelection{SubnetType: ctx.SubnetType}
		awsecs.NewEc2Service(ctx.Stack, jsii.String("ecsService"), &awsecs.Ec2ServiceProps{
			Cluster: cluster,
			DesiredCount: jsii.Number(ecsInstance.ServiceCount),
			TaskDefinition: taskDefinition,
			VpcSubnets: subnetSelection,
			SecurityGroups: &securityGroups,
		})
	}
	*/
	e.ecs = cluster
	
	// 保存 ECS 属性
	if e.properties == nil {
		e.properties = make(map[string]interface{})
	}
	e.properties["clusterName"] = cluster.ClusterName()
	e.properties["clusterArn"] = cluster.ClusterArn()

	return e
}

func (e *EcsForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	ecsInstance, ok := (*ctx.Instance).(*EcsInstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return
	}
	awscdk.NewCfnOutput(ctx.Stack, jsii.String(*ctx.Stack.StackName()+ecsInstance.GetID()), &awscdk.CfnOutputProps{
		Value:       e.ecs.ClusterName(),
		Description: jsii.String("ECS cluster name"),
	})
}

func (e *EcsForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// 为 EC2 配置特定的入站规则
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Public, 22, "Allow EC2 SSH access from public subnet")
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Private, 22, "Allow EC2 SSH access from private subnet")

	security.AddTcpIngressRuleFromAnyIp(ctx.SecurityGroups.Public, 22, "Allow EC2 SSH access from 0.0.0.0/0")
	if ctx.DualStack {
		security.AddTcpIngressRuleFromAnyIpv6(ctx.SecurityGroups.Public, 22, "Allow EC2 SSH access from 0.0.0.0/0")
	}
}

func (e *EcsForge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
	// 从默认配置中复制基本字段
	merged := defaults.(*EcsInstanceConfig)

	// 从实例配置中覆盖基本字段
	ecsInstance := instance.(*EcsInstanceConfig)
	if instance != nil {
		if ecsInstance.GetID() != "" {
			merged.ID = ecsInstance.GetID()
		}
		if ecsInstance.Type != "" {
			merged.Type = ecsInstance.GetType()
		}
		if ecsInstance.Subnet != "" {
			merged.Subnet = ecsInstance.GetSubnet()
		}
		if ecsInstance.SecurityGroup != "" {
			merged.SecurityGroup = ecsInstance.GetSecurityGroup()
		}
	}

	// Merge EC2 自己的字段

	if ecsInstance.UserDataToken != "" {
		merged.UserDataToken = ecsInstance.UserDataToken
	}
	if ecsInstance.UserDataScriptPath != "" {
		merged.UserDataScriptPath = ecsInstance.UserDataScriptPath
	}
	if ecsInstance.AmiHardwareType != "" {
		merged.AmiHardwareType = ecsInstance.AmiHardwareType
	}
	if ecsInstance.DependsOn != "" {
		merged.DependsOn = ecsInstance.DependsOn
	}
	if ecsInstance.CpuCount != 0 {
		merged.CpuCount = ecsInstance.CpuCount
	}
	if ecsInstance.FargateCpuCount != 0 {
		merged.FargateCpuCount = ecsInstance.FargateCpuCount
	}
	if ecsInstance.GpuCount != 0 {
		merged.GpuCount = ecsInstance.GpuCount
	}
	if ecsInstance.MemoryMiB != 0 {
		merged.MemoryMiB = ecsInstance.MemoryMiB
	}
	if ecsInstance.NetworkMode != "" {
		merged.NetworkMode = ecsInstance.NetworkMode
	}
	if ecsInstance.FargateMemoryMiB != 0 {
		merged.FargateMemoryMiB = ecsInstance.FargateMemoryMiB
	}
	if ecsInstance.EbsIops != 0 {
		merged.EbsIops = ecsInstance.EbsIops
	}
	if ecsInstance.EbsSize != 0 {
		merged.EbsSize = ecsInstance.EbsSize
	}
	if ecsInstance.EbsThroughput != 0 {
		merged.EbsThroughput = ecsInstance.EbsThroughput
	}
	if ecsInstance.EbsVolumeType != "" {
		merged.EbsVolumeType = ecsInstance.EbsVolumeType
	}
	if ecsInstance.EbsOptimized != nil {
		merged.EbsOptimized = ecsInstance.EbsOptimized
	}
	if ecsInstance.ContainerInsights != "" {
		merged.ContainerInsights = ecsInstance.ContainerInsights
	}
	if ecsInstance.Image != "" {
		merged.Image = ecsInstance.Image
	}
	if ecsInstance.InstanceTypes != "" {
		merged.InstanceTypes = ecsInstance.InstanceTypes
	}
	if ecsInstance.Policies != "" {
		merged.Policies = ecsInstance.Policies
	}
	if ecsInstance.ExecutionRolePolicies != "" {
		merged.ExecutionRolePolicies = ecsInstance.ExecutionRolePolicies
	}
	if ecsInstance.TaskTypes != "" {
		merged.TaskTypes = ecsInstance.TaskTypes
	}
	if ecsInstance.TaskRolePolicies != "" {
		merged.TaskRolePolicies = ecsInstance.TaskRolePolicies
	}
	if ecsInstance.HealthCheck != "" {
		merged.HealthCheck = ecsInstance.HealthCheck
	}
	if ecsInstance.Interval != 0 {
		merged.Interval = ecsInstance.Interval
	}
	if ecsInstance.Retries != 0 {
		merged.Retries = ecsInstance.Retries
	}
	if ecsInstance.StartPeriod != 0 {
		merged.StartPeriod = ecsInstance.StartPeriod
	}
	if ecsInstance.Timeout != 0 {
		merged.Timeout = ecsInstance.Timeout
	}
	if ecsInstance.EnableRestartPolicy != nil {
		merged.EnableRestartPolicy = ecsInstance.EnableRestartPolicy
	}
	if ecsInstance.RestartAttemptPeriod != 0 {
		merged.RestartAttemptPeriod = ecsInstance.RestartAttemptPeriod
	}
	if ecsInstance.MinCapacity != 0 {
		merged.MinCapacity = ecsInstance.MinCapacity
	}
	if ecsInstance.MaxCapacity != 0 {
		merged.MaxCapacity = ecsInstance.MaxCapacity
	}
	if ecsInstance.OnDemandPercentage != 0 {
		merged.OnDemandPercentage = ecsInstance.OnDemandPercentage
	}

	return merged
}

func createTaskDefinitions(stack awscdk.Stack, config TaskDefinitionConfig) {
	// 分割任务类型字符串
	taskTypes := strings.Split(config.EcsInstance.TaskTypes, ",")

	// 定义接口类型
	type TaskDefinitionInterface interface {
		AddToTaskRolePolicy(statement awsiam.PolicyStatement)
		AddToExecutionRolePolicy(statement awsiam.PolicyStatement)
		AddContainer(id *string, props *awsecs.ContainerDefinitionOptions) awsecs.ContainerDefinition
		TaskRole() awsiam.IRole
		ExecutionRole() awsiam.IRole
	}

	mountPointPath, err := dependency.GetMountPoint(config.EcsInstance.DependsOn)
	if err != nil {
		fmt.Printf("Error mount point: %v\n", err)
		mountPointPath = "/data"
	}

	//mountPointPath = mountPointPath

	// 先定义单个 volume
	volume := &awsecs.Volume{
		Name: jsii.String("hostVolume"),
		Host: &awsecs.Host{
			SourcePath: jsii.String(mountPointPath),
		},
	}

	// 创建 volumes 切片
	volumes :=  &[]*awsecs.Volume{volume}

	networkMode := utils.ParseNetworkMode(config.EcsInstance.NetworkMode)
	// 遍历每个类型创建对应的任务定义
	for _, taskType := range taskTypes {
		taskType = strings.TrimSpace(taskType) // 移除可能的空格

		var taskDef TaskDefinitionInterface // 使用接口类型
		var gpuTaskDef TaskDefinitionInterface // 使用接口类型

		gpuCount := config.EcsInstance.GpuCount
		cpuCount := config.EcsInstance.CpuCount
		memoryMiB := config.EcsInstance.MemoryMiB

		// 声明任务定义ID变量
		var taskDefId, gpuTaskDefId string
		
		// 为所有任务类型设置ID
		gpuTaskDefId = fmt.Sprintf("%s-%s-GpuTaskDef", config.EcsInstance.GetID(), strings.ToUpper(taskType))
		taskDefId = fmt.Sprintf("%s-%s-TaskDef", config.EcsInstance.GetID(), strings.ToUpper(taskType))


		// 定义单个 MountPoint
		mountPoint := &awsecs.MountPoint{
			SourceVolume:  jsii.String("hostVolume"),
			ContainerPath: jsii.String(mountPointPath),
			ReadOnly:      jsii.Bool(false),
		}
		// 创建 MountPoints 切片
		//mountPoints := &[]*awsecs.MountPoint{mountPoint}


		// 根据类型创建不同的任务定义
		switch strings.ToLower(taskType) {
		case "ec2":
			if config.EcsInstance.GpuCount > 0 {
				gpuTaskDef = awsecs.NewEc2TaskDefinition(stack, jsii.String(gpuTaskDefId), &awsecs.Ec2TaskDefinitionProps{
					NetworkMode: awsecs.NetworkMode_AWS_VPC,
					Volumes: volumes,
				})
			}
			gpuCount = 0
			taskDef = awsecs.NewEc2TaskDefinition(stack, jsii.String(taskDefId), &awsecs.Ec2TaskDefinitionProps{
				NetworkMode: awsecs.NetworkMode_AWS_VPC,
				Volumes: volumes,
			})
		case "fargate":
			cpuCount = config.EcsInstance.FargateCpuCount
			memoryMiB = config.EcsInstance.FargateMemoryMiB
			mountPoint = nil
			taskDef = awsecs.NewFargateTaskDefinition(stack, jsii.String("FargateTaskDef"), &awsecs.FargateTaskDefinitionProps{
				Cpu: jsii.Number(cpuCount),
				MemoryLimitMiB: jsii.Number(memoryMiB),
			})
			gpuCount = 0
		case "external":
			if config.EcsInstance.GpuCount > 0 {
				gpuTaskDef = awsecs.NewExternalTaskDefinition(stack, jsii.String("ExternalGpuTaskDef"), &awsecs.ExternalTaskDefinitionProps{
					NetworkMode: networkMode,
					Volumes: volumes,
				})
			}
			gpuCount = 0
			taskDef = awsecs.NewExternalTaskDefinition(stack, jsii.String("ExternalTaskDef"), &awsecs.ExternalTaskDefinitionProps{
				NetworkMode: networkMode,
				Volumes: volumes,
			})
		default:
			log.Printf("Unsupported task type: %s", taskType)
			continue
		}


		if config.EcsInstance.GpuCount > 0 && strings.ToLower(taskType) != "fargate" {
			// 添加策略
			gpuTaskDef.AddToTaskRolePolicy(
				awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
					Actions:   &config.Actions,
					Resources: &config.Resources,
				}))

				gpuTaskDef.AddToExecutionRolePolicy(
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Actions:   &config.Actions,
						Resources: &config.Resources,
					}))

					// 添加容器
					gpuContainer := gpuTaskDef.AddContainer(jsii.String("DefaultGpuContainer"), &awsecs.ContainerDefinitionOptions{
						Image:                awsecs.ContainerImage_FromRegistry(jsii.String(config.EcsInstance.Image), nil),
						Cpu:                  jsii.Number(cpuCount),
						GpuCount:             jsii.Number(config.EcsInstance.GpuCount),
						EnableRestartPolicy:  jsii.Bool(types.GetBoolValue(config.EcsInstance.EnableRestartPolicy, false)),
						RestartAttemptPeriod: awscdk.Duration_Seconds(jsii.Number(config.EcsInstance.RestartAttemptPeriod)),
						HealthCheck: &awsecs.HealthCheck{
							Command: &config.Command,
							Interval: awscdk.Duration_Seconds(jsii.Number(config.EcsInstance.Interval)),
							Retries: jsii.Number(config.EcsInstance.Retries),
							StartPeriod: awscdk.Duration_Seconds(jsii.Number(config.EcsInstance.StartPeriod)),
							Timeout: awscdk.Duration_Seconds(jsii.Number(config.EcsInstance.Timeout)),
						},
						MemoryLimitMiB: jsii.Number(memoryMiB),
						Logging: awsecs.NewAwsLogDriver(&awsecs.AwsLogDriverProps{
							LogGroup:        awslogs.NewLogGroup(stack, jsii.String(fmt.Sprintf("%s-LogGroup", gpuTaskDefId)), &awslogs.LogGroupProps{
								LogGroupName: jsii.String(fmt.Sprintf("/aws/ecs/%s/%s", *stack.StackName(), gpuTaskDefId)),
								Retention:    awslogs.RetentionDays_ONE_WEEK,
								RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
							}),
							StreamPrefix:    jsii.String("task"),
							Mode:           awsecs.AwsLogDriverMode_NON_BLOCKING,
							MaxBufferSize:  awscdk.Size_Mebibytes(jsii.Number(25)),
						}),
					})
					if mountPoint != nil {
						gpuContainer.AddMountPoints(mountPoint)
					}
				}


				// 添加策略
				taskDef.AddToTaskRolePolicy(
					awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
						Actions:   &config.Actions,
						Resources: &config.Resources,
					}))

					taskDef.AddToExecutionRolePolicy(
						awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
							Actions:   &config.Actions,
							Resources: &config.Resources,
						}))

						taskRole := taskDef.TaskRole()
						aws.AddManagedPolicies(taskRole, config.EcsInstance.TaskRolePolicies)

						executionRole := taskDef.ExecutionRole()
						aws.AddManagedPolicies(executionRole, config.EcsInstance.ExecutionRolePolicies)

						// 添加容器
						container := taskDef.AddContainer(jsii.String("DefaultContainer"), &awsecs.ContainerDefinitionOptions{
							Image:                awsecs.ContainerImage_FromRegistry(jsii.String(config.EcsInstance.Image), nil),
							Cpu:                  jsii.Number(cpuCount),
							GpuCount:             jsii.Number(gpuCount),
							EnableRestartPolicy:  jsii.Bool(types.GetBoolValue(config.EcsInstance.EnableRestartPolicy, false)),
							RestartAttemptPeriod: awscdk.Duration_Seconds(jsii.Number(config.EcsInstance.RestartAttemptPeriod)),
							HealthCheck: &awsecs.HealthCheck{
								Command: &config.Command,
								Interval: awscdk.Duration_Seconds(jsii.Number(config.EcsInstance.Interval)),
								Retries: jsii.Number(config.EcsInstance.Retries),
								StartPeriod: awscdk.Duration_Seconds(jsii.Number(config.EcsInstance.StartPeriod)),
								Timeout: awscdk.Duration_Seconds(jsii.Number(config.EcsInstance.Timeout)),
							},
							MemoryLimitMiB: jsii.Number(memoryMiB),
							Logging: awsecs.NewAwsLogDriver(&awsecs.AwsLogDriverProps{
								LogGroup:        awslogs.NewLogGroup(stack, jsii.String(fmt.Sprintf("%s-LogGroup", taskDefId)), &awslogs.LogGroupProps{
									LogGroupName: jsii.String(fmt.Sprintf("/aws/ecs/%s/%s", *stack.StackName(), taskDefId)),
									Retention:    awslogs.RetentionDays_ONE_WEEK,
									RemovalPolicy: awscdk.RemovalPolicy_DESTROY,
								}),
								StreamPrefix:    jsii.String("task"),
								Mode:           awsecs.AwsLogDriverMode_NON_BLOCKING,
								MaxBufferSize:  awscdk.Size_Mebibytes(jsii.Number(25)),
							}),
						})
						if mountPoint != nil {
							container.AddMountPoints(mountPoint)
						}
					}
				}

func (e *EcsForge) GetProperties() map[string]interface{} {
	return e.properties
}
