// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"fmt"
	"strings"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/awslabs/InfraForge/core/dependency"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsbatch"
	"github.com/aws/jsii-runtime-go"
)

type BatchInstanceConfig struct {
	config.BaseInstanceConfig
	
	// Compute Environment 配置
	InstanceTypes           string `json:"instanceTypes"`                    // "m5.large,c5.xlarge"
	UseOptimalInstanceTypes *bool  `json:"useOptimalInstanceTypes,omitempty"` // 是否使用 optimal 实例类型
	MinvCpus                int    `json:"minvCpus,omitempty"`
	MaxvCpus                int    `json:"maxvCpus"`
	DesiredvCpus            int    `json:"desiredvCpus,omitempty"`
	AllocationStrategy         string `json:"allocationStrategy,omitempty"`
	SpotBidPercentage          int    `json:"spotBidPercentage,omitempty"`
	UpdateToLatestImageVersion *bool  `json:"updateToLatestImageVersion,omitempty"`
	
	// 网络配置
	AzIndex              int    `json:"azIndex,omitempty"`  // 0=所有AZ，>0=特定AZ
	
	// Job Queue 配置
	QueuePriority        int    `json:"queuePriority,omitempty"`
	
	// Job Definition 配置
	JobDefinitionName    string `json:"jobDefinitionName,omitempty"`
	JobDefinitionType    string `json:"jobDefinitionType,omitempty"`    // container/multinode
	ContainerImage       string `json:"containerImage,omitempty"`
	VCpus               int    `json:"vcpus,omitempty"`
	Memory              int    `json:"memory,omitempty"`
	
	// Multinode 配置
	NumNodes            int    `json:"numNodes,omitempty"`
	MainNode            int    `json:"mainNode,omitempty"`
	
	// IAM 配置
	InstanceRolePolicies string `json:"instanceRolePolicies,omitempty"` // EC2 实例角色策略
	ServiceRolePolicies  string `json:"serviceRolePolicies,omitempty"`  // Batch 服务角色策略
	JobRolePolicies      string `json:"jobRolePolicies,omitempty"`      // 作业角色策略
	
	// UserData 配置（复用 EC2 的能力）
	UserDataToken       string `json:"userDataToken,omitempty"`        // 如 "nas" 自动挂载存储
	UserDataScriptPath  string `json:"userDataScriptPath,omitempty"`   // 自定义脚本路径
	S3Location          string `json:"s3Location,omitempty"`           // 自定义脚本位置
	
	// 存储依赖（通过 DependsOn 和 MagicToken 传递）
	DependsOn           string `json:"dependsOn,omitempty"`            // "efs1,fsx1" 等
}

type BatchForge struct {
	computeEnvironment awsbatch.ManagedEc2EcsComputeEnvironment
	jobQueue          awsbatch.JobQueue
	jobDefinition     awsbatch.IJobDefinition  // 使用接口类型支持不同的 JobDefinition
}

// 辅助函数已移动到 types.GetIntValue

func (b *BatchForge) Create(ctx *interfaces.ForgeContext) interface{} {
	batchInstance, ok := (*ctx.Instance).(*BatchInstanceConfig)
	if !ok {
		return nil
	}

	// 创建 Compute Environment
	b.computeEnvironment = b.createComputeEnvironment(batchInstance, ctx)
	
	// 创建 Job Queue
	b.jobQueue = b.createJobQueue(batchInstance, ctx)
	
	// 创建 Job Definition (如果配置了)
	if batchInstance.ContainerImage != "" {
		b.jobDefinition = b.createJobDefinition(batchInstance, ctx)
	}

	return b
}

func (b *BatchForge) createComputeEnvironment(batchInstance *BatchInstanceConfig, ctx *interfaces.ForgeContext) awsbatch.ManagedEc2EcsComputeEnvironment {
	// 创建或获取 Instance Profile
	var instanceProfile awsiam.IInstanceProfile
	if batchInstance.InstanceRolePolicies != "" {
		instanceProfile = aws.CreateOrGetInstanceProfile(ctx.Stack, batchInstance.InstanceRolePolicies)
	}

	// 解析实例类型
	instanceTypes := strings.Split(batchInstance.InstanceTypes, ",")
	instanceTypesList := make([]awsec2.InstanceType, len(instanceTypes))
	for i, instanceType := range instanceTypes {
		// 直接使用完整的实例类型字符串，如 "m5.large"
		instanceTypesList[i] = awsec2.NewInstanceType(jsii.String(strings.TrimSpace(instanceType)))
	}

	computeEnvName := fmt.Sprintf("%s-compute-env", batchInstance.GetID())
	
	props := &awsbatch.ManagedEc2EcsComputeEnvironmentProps{
		ComputeEnvironmentName:    jsii.String(computeEnvName),
		Vpc:                      ctx.VPC,
		InstanceTypes:            &instanceTypesList,
		SecurityGroups:           &[]awsec2.ISecurityGroup{ctx.SecurityGroups.Default},
		MinvCpus:                 jsii.Number(batchInstance.MinvCpus),
		MaxvCpus:                 jsii.Number(batchInstance.MaxvCpus),
		UseOptimalInstanceClasses: jsii.Bool(types.GetBoolValue(batchInstance.UseOptimalInstanceTypes, false)),
		UpdateToLatestImageVersion: jsii.Bool(types.GetBoolValue(batchInstance.UpdateToLatestImageVersion, false)),
	}

	// 子网选择：azIndex > 0 使用特定 AZ，否则使用所有 AZ
	if batchInstance.AzIndex > 0 {
		selectedSubnet := aws.SelectSubnetByAzIndex(batchInstance.AzIndex, ctx.VPC, ctx.SubnetType)
		props.VpcSubnets = &awsec2.SubnetSelection{
			Subnets: &[]awsec2.ISubnet{selectedSubnet},
		}
	}

	// 设置 Instance Profile
	if instanceProfile != nil {
		// 从 IInstanceProfile 获取 Role
		if role, ok := instanceProfile.(awsiam.IRole); ok {
			props.InstanceRole = role
		}
	}

	// 创建启动模板（复用 EC2 的 UserData 逻辑）
	if batchInstance.UserDataToken != "" {
		launchTemplate := b.createLaunchTemplateWithUserData(batchInstance, ctx)
		props.LaunchTemplate = launchTemplate
	}

	// 设置分配策略
	if batchInstance.AllocationStrategy != "" {
		switch batchInstance.AllocationStrategy {
		case "BEST_FIT":
			props.AllocationStrategy = awsbatch.AllocationStrategy_BEST_FIT
		case "BEST_FIT_PROGRESSIVE":
			props.AllocationStrategy = awsbatch.AllocationStrategy_BEST_FIT_PROGRESSIVE
		case "SPOT_CAPACITY_OPTIMIZED":
			props.AllocationStrategy = awsbatch.AllocationStrategy_SPOT_CAPACITY_OPTIMIZED
		}
	}

	// 设置 Spot 竞价百分比
	if batchInstance.SpotBidPercentage > 0 {
		props.Spot = jsii.Bool(true)
		props.SpotBidPercentage = jsii.Number(batchInstance.SpotBidPercentage)
	}

	// 设置 Service Role
	if batchInstance.ServiceRolePolicies != "" {
		serviceRoleId := fmt.Sprintf("%s-service-role", batchInstance.GetID())
		serviceRole := aws.CreateRole(ctx.Stack, serviceRoleId, batchInstance.ServiceRolePolicies, "batch")
		props.ServiceRole = serviceRole
	}

	return awsbatch.NewManagedEc2EcsComputeEnvironment(ctx.Stack, jsii.String(computeEnvName), props)
}

func (b *BatchForge) createLaunchTemplateWithUserData(batchInstance *BatchInstanceConfig, ctx *interfaces.ForgeContext) awsec2.LaunchTemplate {
	templateName := fmt.Sprintf("%s-batch-compute", batchInstance.GetID())
	
	// 获取依赖信息（MagicToken）
	magicToken, err := dependency.GetDependencyInfo(batchInstance.DependsOn)
	if err != nil {
		fmt.Printf("Error getting dependency info: %v\n", err)
	}

	// 使用 MIME multipart 格式的 UserData（专为 Batch Launch Template 设计）
	userDataGenerator := &aws.UserDataGenerator{
		OsType:             awsec2.OperatingSystemType_LINUX,
		ScriptPath:         "./userdata.sh",
		UserDataToken:      batchInstance.UserDataToken,
		UserDataScriptPath: batchInstance.UserDataScriptPath,
		MagicToken:         magicToken,
		S3Location:         batchInstance.S3Location,
	}

	userData, err := userDataGenerator.GenerateMimeMultipartUserData()
	if err != nil {
		fmt.Printf("Error generating MIME multipart user data: %v\n", err)
		// 创建空的 UserData 作为后备
		userData = awsec2.UserData_ForLinux(&awsec2.LinuxUserDataOptions{})
	}

	return awsec2.NewLaunchTemplate(ctx.Stack, jsii.String(templateName), &awsec2.LaunchTemplateProps{
		LaunchTemplateName: jsii.String(templateName),
		UserData:          userData,
	})
}

func (b *BatchForge) createJobQueue(batchInstance *BatchInstanceConfig, ctx *interfaces.ForgeContext) awsbatch.JobQueue {
	queueName := fmt.Sprintf("%s-job-queue", batchInstance.GetID())
	
	priority := 1
	if batchInstance.QueuePriority > 0 {
		priority = batchInstance.QueuePriority
	}

	return awsbatch.NewJobQueue(ctx.Stack, jsii.String(queueName), &awsbatch.JobQueueProps{
		JobQueueName: jsii.String(queueName),
		Priority:     jsii.Number(priority),
		ComputeEnvironments: &[]*awsbatch.OrderedComputeEnvironment{
			{
				ComputeEnvironment: b.computeEnvironment,
				Order:             jsii.Number(1),
			},
		},
	})
}

func (b *BatchForge) createJobDefinition(batchInstance *BatchInstanceConfig, ctx *interfaces.ForgeContext) awsbatch.IJobDefinition {
	jobDefName := batchInstance.JobDefinitionName
	if jobDefName == "" {
		jobDefName = fmt.Sprintf("%s-job-def", batchInstance.GetID())
	}

	jobDefType := batchInstance.JobDefinitionType
	if jobDefType == "" {
		jobDefType = "container"
	}

	if jobDefType == "multinode" {
		return b.createMultinodeJobDefinition(batchInstance, ctx, jobDefName)
	}

	return b.createContainerJobDefinition(batchInstance, ctx, jobDefName)
}

func (b *BatchForge) createContainerJobDefinition(batchInstance *BatchInstanceConfig, ctx *interfaces.ForgeContext, jobDefName string) awsbatch.EcsJobDefinition {
	containerProps := &awsbatch.EcsEc2ContainerDefinitionProps{
		Image:  awsecs.ContainerImage_FromRegistry(jsii.String(batchInstance.ContainerImage), &awsecs.RepositoryImageProps{}),
		Cpu:    jsii.Number(types.GetIntValue(&batchInstance.VCpus, 1)),
		Memory: awscdk.Size_Mebibytes(jsii.Number(types.GetIntValue(&batchInstance.Memory, 512))),
	}

	// 设置 Job Role
	if batchInstance.JobRolePolicies != "" {
		jobRoleId := fmt.Sprintf("%s-container-job-role", batchInstance.GetID())
		jobRole := aws.CreateRole(ctx.Stack, jobRoleId, batchInstance.JobRolePolicies, "ecs-tasks")
		containerProps.JobRole = jobRole
	}

	// 处理存储挂载
	volumes := b.createStorageVolumes(batchInstance)
	if len(volumes) > 0 {
		containerProps.Volumes = &volumes
	}

	containerDef := awsbatch.NewEcsEc2ContainerDefinition(ctx.Stack, jsii.String(jobDefName+"-container"), containerProps)

	return awsbatch.NewEcsJobDefinition(ctx.Stack, jsii.String(jobDefName), &awsbatch.EcsJobDefinitionProps{
		JobDefinitionName: jsii.String(jobDefName),
		Container:        containerDef,
	})
}

func (b *BatchForge) createStorageVolumes(batchInstance *BatchInstanceConfig) []awsbatch.EcsVolume {
	var volumes []awsbatch.EcsVolume
	
	if batchInstance.DependsOn == "" {
		return volumes
	}

	// 解析每个依赖
	dependencies := strings.Split(batchInstance.DependsOn, ",")
	for _, dep := range dependencies {
		dep = strings.TrimSpace(dep)
		
		// 直接获取挂载点
		mountPoint, err := dependency.GetMountPoint(dep)
		if err != nil {
			fmt.Printf("Error getting mount point for %s: %v\n", dep, err)
			continue
		}

		// 解析依赖格式: "EFS:efs" -> resourceType="EFS", resourceId="efs"
		parts := strings.Split(dep, ":")
		if len(parts) != 2 {
			continue
		}
		resourceType := strings.ToLower(parts[0])
		resourceId := parts[1]

		// 根据资源类型创建相应的卷 - 使用 Host 卷映射已挂载的目录
		if resourceType == "efs" || resourceType == "fsx" || resourceType == "lustre" {
			hostVolume := awsbatch.EcsVolume_Host(&awsbatch.HostVolumeOptions{
				Name:      jsii.String(fmt.Sprintf("%s-volume", resourceId)),
				HostPath: jsii.String(mountPoint),
				ContainerPath: jsii.String(mountPoint),
			})
			volumes = append(volumes, hostVolume)
		}
	}

	return volumes
}

func (b *BatchForge) createMultinodeJobDefinition(batchInstance *BatchInstanceConfig, ctx *interfaces.ForgeContext, jobDefName string) awsbatch.IJobDefinition {
	numNodes := types.GetIntValue(&batchInstance.NumNodes, 2)
	mainNode := types.GetIntValue(&batchInstance.MainNode, 0)

	// 容器属性
	containerProps := &awsbatch.EcsEc2ContainerDefinitionProps{
		Image:  awsecs.ContainerImage_FromRegistry(jsii.String(batchInstance.ContainerImage), &awsecs.RepositoryImageProps{}),
		Cpu:    jsii.Number(types.GetIntValue(&batchInstance.VCpus, 1)),
		Memory: awscdk.Size_Mebibytes(jsii.Number(types.GetIntValue(&batchInstance.Memory, 512))),
	}

	// 设置 Job Role
	if batchInstance.JobRolePolicies != "" {
		jobRoleId := fmt.Sprintf("%s-multinode-job-role", batchInstance.GetID())
		jobRole := aws.CreateRole(ctx.Stack, jobRoleId, batchInstance.JobRolePolicies, "ecs-tasks")
		containerProps.JobRole = jobRole
	}

	// 处理存储挂载
	volumes := b.createStorageVolumes(batchInstance)
	if len(volumes) > 0 {
		containerProps.Volumes = &volumes
	}

	// 创建主节点容器
	mainContainer := awsbatch.NewEcsEc2ContainerDefinition(ctx.Stack, jsii.String(jobDefName+"-main-container"), containerProps)

	// 创建工作节点容器
	workerContainer := awsbatch.NewEcsEc2ContainerDefinition(ctx.Stack, jsii.String(jobDefName+"-worker-container"), containerProps)

	// 创建 MultiNode 容器配置
	containers := []*awsbatch.MultiNodeContainer{
		{
			Container: mainContainer,
			StartNode: jsii.Number(mainNode),
			EndNode:   jsii.Number(mainNode),
		},
		{
			Container: workerContainer,
			StartNode: jsii.Number(mainNode + 1),
			EndNode:   jsii.Number(numNodes - 1),
		},
	}

	return awsbatch.NewMultiNodeJobDefinition(ctx.Stack, jsii.String(jobDefName), &awsbatch.MultiNodeJobDefinitionProps{
		JobDefinitionName: jsii.String(jobDefName),
		MainNode:         jsii.Number(mainNode),
		Containers:       &containers,
	})
}

func (b *BatchForge) MergeConfigs(defaults, instance config.InstanceConfig) config.InstanceConfig {
	merged := defaults.(*BatchInstanceConfig)  // 直接引用默认配置
	batchInstance := instance.(*BatchInstanceConfig)
	// 用实例配置覆盖默认配置
	if batchInstance.GetID() != "" {
		merged.ID = batchInstance.GetID()
	}
	if batchInstance.GetType() != "" {
		merged.Type = batchInstance.GetType()
	}
	if batchInstance.GetSubnet() != "" {
		merged.Subnet = batchInstance.GetSubnet()
	}
	if batchInstance.GetSecurityGroup() != "" {
		merged.SecurityGroup = batchInstance.GetSecurityGroup()
	}
	if batchInstance.InstanceTypes != "" {
		merged.InstanceTypes = batchInstance.InstanceTypes
	}
	if batchInstance.MaxvCpus != 0 {
		merged.MaxvCpus = batchInstance.MaxvCpus
	}
	if batchInstance.ContainerImage != "" {
		merged.ContainerImage = batchInstance.ContainerImage
	}
	if batchInstance.InstanceRolePolicies != "" {
		merged.InstanceRolePolicies = batchInstance.InstanceRolePolicies
	}
	if batchInstance.UserDataToken != "" {
		merged.UserDataToken = batchInstance.UserDataToken
	}
	if batchInstance.UserDataScriptPath != "" {
		merged.UserDataScriptPath = batchInstance.UserDataScriptPath
	}
	if batchInstance.S3Location != "" {
		merged.S3Location = batchInstance.S3Location
	}
	if batchInstance.UseOptimalInstanceTypes != nil {
		merged.UseOptimalInstanceTypes = batchInstance.UseOptimalInstanceTypes
	}
	if batchInstance.UpdateToLatestImageVersion != nil {
		merged.UpdateToLatestImageVersion = batchInstance.UpdateToLatestImageVersion
	}

	return merged
}

func (b *BatchForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// Batch 通常不需要特殊的安全组规则
}

func (b *BatchForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	batchInstance := (*ctx.Instance).(*BatchInstanceConfig)
	
	awscdk.NewCfnOutput(ctx.Stack, jsii.String("BatchComputeEnvironment"+batchInstance.GetID()), &awscdk.CfnOutputProps{
		Value:       b.computeEnvironment.ComputeEnvironmentArn(),
		Description: jsii.String("Batch Compute Environment ARN for " + batchInstance.GetID()),
	})

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("BatchJobQueue"+batchInstance.GetID()), &awscdk.CfnOutputProps{
		Value:       b.jobQueue.JobQueueArn(),
		Description: jsii.String("Batch Job Queue ARN for " + batchInstance.GetID()),
	})

	if b.jobDefinition != nil {
		awscdk.NewCfnOutput(ctx.Stack, jsii.String("BatchJobDefinition"+batchInstance.GetID()), &awscdk.CfnOutputProps{
			Value:       b.jobDefinition.JobDefinitionArn(),
			Description: jsii.String("Batch Job Definition ARN for " + batchInstance.GetID()),
		})
	}
}
