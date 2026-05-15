// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package batch

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/dependency"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/core/utils/types"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsbatch"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/jsii-runtime-go"
)

// BatchInstanceConfig 扁平化配置，逗号分隔多个资源，分号分隔同一 CE 内的多个实例类型
//
// 示例（两个 CE、两个 Queue、三个 JD）:
//   ceNames:         "cpu-pool,gpu-pool"
//   instanceTypes:   "c5.xlarge,c5.2xlarge;g4dn.xlarge"   ← ; 分隔不同 CE，, 分隔同 CE 内多实例类型
//   maxvCpus:        "1024,128"
//   queueNames:      "cpu-queue,gpu-queue"
//   queueCeRefs:     "cpu-pool,gpu-pool"
//   queuePriorities: "10,5"
//   jobDefNames:     "preprocess,simulate,visualize"
//   containerImage:  "img-a:1.0,img-b:2.0,img-c:3.0"
//   vcpus:           "2,4,8"
//   memory:          "4096,8192,16384"
type BatchInstanceConfig struct {
	config.BaseInstanceConfig

	// CE 配置（逗号分隔，按位置对应）
	CeNames                    string `json:"ceNames,omitempty"`
	InstanceTypes              string `json:"instanceTypes"`
	MaxvCpus                   string `json:"maxvCpus"`
	MinvCpus                   string `json:"minvCpus,omitempty"`
	AllocationStrategy         string `json:"allocationStrategy,omitempty"`
	SpotBidPercentage          string `json:"spotBidPercentage,omitempty"` // 逗号分隔，按 CE 位置对应；0 或空表示 On-Demand
	// 逗号分隔，按 CE 位置对应；可选值: ECS_AL2, ECS_AL2023, ECS_AL2_NVIDIA, ECS_AL2023_NVIDIA
	// 不填则由 CDK 自动选择（CPU: ECS_AL2，GPU: ECS_AL2_NVIDIA）
	ImageTypes                 string `json:"imageTypes,omitempty"`
	UseOptimalInstanceTypes    *bool  `json:"useOptimalInstanceTypes,omitempty"`
	UpdateToLatestImageVersion *bool  `json:"updateToLatestImageVersion,omitempty"`
	// 逗号分隔，按 CE 位置对应；CLUSTER/SPREAD/PARTITION，Multinode 建议 CLUSTER
	PlacementGroupStrategy     string `json:"placementGroupStrategy,omitempty"`
	// 关闭超线程（ThreadsPerCore=1），默认 false；FPU 密集型计算（如 FLUKA）可提升性能
	DisableHyperthreading      bool   `json:"disableHyperthreading,omitempty"`

	// 网络配置
	AzIndex string `json:"azIndex,omitempty"`

	// Queue 配置（逗号分隔，按位置对应）
	QueueNames       string `json:"queueNames,omitempty"`
	QueueCeRefs      string `json:"queueCeRefs,omitempty"`
	QueuePriorities  string `json:"queuePriorities,omitempty"`

	// Job Definition 配置（逗号分隔，按位置对应）
	JobDefNames    string `json:"jobDefNames,omitempty"`
	ContainerImage string `json:"containerImage,omitempty"`
	VCpus          string `json:"vcpus,omitempty"`
	Memory         string `json:"memory,omitempty"`
	// 逗号分隔，按 JD 位置对应；单位分钟，0 表示不设超时
	TimeoutMinutes string `json:"timeoutMinutes,omitempty"`

	// Multinode 配置
	JobDefinitionType string `json:"jobDefinitionType,omitempty"`
	NumNodes          string `json:"numNodes,omitempty"`
	MainNode          string `json:"mainNode,omitempty"`

	// IAM 配置
	InstanceRolePolicies string `json:"instanceRolePolicies,omitempty"`
	ServiceRolePolicies  string `json:"serviceRolePolicies,omitempty"`
	JobRolePolicies      string `json:"jobRolePolicies,omitempty"`

	// Retry 配置
	RetryAttempts int `json:"retryAttempts,omitempty"`

	// UserData 配置
	UserDataToken      string `json:"userDataToken,omitempty"`
	UserDataScriptPath string `json:"userDataScriptPath,omitempty"`
	S3Location         string `json:"s3Location,omitempty"`

	// 存储依赖
	DependsOn string `json:"dependsOn,omitempty"`
}

type namedJobDef struct {
	name string
	jd   awsbatch.IJobDefinition
}

type BatchForge struct {
	computeEnvironments map[string]awsbatch.ManagedEc2EcsComputeEnvironment
	jobQueues           map[string]awsbatch.JobQueue
	jobDefinitions      []namedJobDef
}

func (b *BatchForge) Create(ctx *interfaces.ForgeContext) interface{} {
	inst, ok := (*ctx.Instance).(*BatchInstanceConfig)
	if !ok {
		return nil
	}

	b.computeEnvironments = make(map[string]awsbatch.ManagedEc2EcsComputeEnvironment)
	b.jobQueues = make(map[string]awsbatch.JobQueue)

	b.createComputeEnvironments(inst, ctx)
	b.createJobQueues(inst, ctx)

	if inst.ContainerImage != "" {
		b.createJobDefinitions(inst, ctx)
	}

	return b
}

// ── Compute Environments ──────────────────────────────────────────────────────

func (b *BatchForge) createComputeEnvironments(inst *BatchInstanceConfig, ctx *interfaces.ForgeContext) {
	ceNames := splitComma(inst.CeNames)
	instanceTypeGroups := splitSemi(inst.InstanceTypes)
	maxvCpusList := splitComma(inst.MaxvCpus)
	minvCpusList := splitComma(inst.MinvCpus)
	azIndexList := splitComma(inst.AzIndex)
	spotBidList := splitComma(inst.SpotBidPercentage)
	imageTypeList := splitComma(inst.ImageTypes)
	placementGroupList := splitComma(inst.PlacementGroupStrategy)

	count := len(instanceTypeGroups)
	if count == 0 {
		return
	}

	ceNames = padNames(ceNames, count, inst.GetID()+"-ce")

	if len(maxvCpusList) != count {
		panic(fmt.Sprintf("batch %s: instanceTypes(%d) 与 maxvCpus(%d) 数量不一致", inst.GetID(), count, len(maxvCpusList)))
	}

	for i, instanceTypeGroup := range instanceTypeGroups {
		ceName := fmt.Sprintf("%s-%s-compute-env", inst.GetID(), ceNames[i])

		// , 分隔同一 CE 内的多个实例类型
		rawTypes := strings.Split(instanceTypeGroup, ",")
		instanceTypesList := make([]awsec2.InstanceType, len(rawTypes))
		for j, t := range rawTypes {
			instanceTypesList[j] = awsec2.NewInstanceType(jsii.String(strings.TrimSpace(t)))
		}

		maxvCpus, _ := strconv.Atoi(strings.TrimSpace(maxvCpusList[i]))
		minvCpus := 0
		if i < len(minvCpusList) && minvCpusList[i] != "" {
			minvCpus, _ = strconv.Atoi(strings.TrimSpace(minvCpusList[i]))
		}

		props := &awsbatch.ManagedEc2EcsComputeEnvironmentProps{
			ComputeEnvironmentName:     jsii.String(ceName),
			Vpc:                        ctx.VPC,
			InstanceTypes:              &instanceTypesList,
			UseOptimalInstanceClasses:  jsii.Bool(false),
			SecurityGroups:             &[]awsec2.ISecurityGroup{ctx.SecurityGroups.Default},
			MinvCpus:                   jsii.Number(minvCpus),
			MaxvCpus:                   jsii.Number(maxvCpus),
			UpdateToLatestImageVersion: jsii.Bool(types.GetBoolValue(inst.UpdateToLatestImageVersion, false)),
		}

		// 按 CE 位置设置 AMI 类型，GPU CE 必须指定 ECS_AL2023_NVIDIA 否则无法加入
		if i < len(imageTypeList) && imageTypeList[i] != "" {
			imageType := resolveImageType(imageTypeList[i])
			props.Images = &[]*awsbatch.EcsMachineImage{
				{ImageType: imageType},
			}
		}

		azIdx := 0
		if i < len(azIndexList) && azIndexList[i] != "" {
			azIdx, _ = strconv.Atoi(strings.TrimSpace(azIndexList[i]))
		}
		if azIdx > 0 {
			selectedSubnet := aws.SelectSubnetByAzIndex(azIdx, ctx.VPC, ctx.SubnetType)
			props.VpcSubnets = &awsec2.SubnetSelection{
				Subnets: &[]awsec2.ISubnet{selectedSubnet},
			}
		}

		if inst.InstanceRolePolicies != "" {
			instanceRoleId := fmt.Sprintf("%s-%s-instance-role", inst.GetID(), ceNames[i])
			instanceRole := aws.CreateRole(ctx.Stack, instanceRoleId, inst.InstanceRolePolicies, "ec2")
			props.InstanceRole = instanceRole
		}

		if inst.UserDataToken != "" {
			launchTemplate := b.createLaunchTemplateWithUserData(inst, ctx, ceNames[i])
			props.LaunchTemplate = launchTemplate
		}

		// 按 CE 位置取对应的 SpotBidPercentage；0 或空表示 On-Demand
		spotBid := 0
		if i < len(spotBidList) {
			if v, err := strconv.Atoi(spotBidList[i]); err == nil {
				spotBid = v
			}
		}
		if spotBid > 0 {
			props.Spot = jsii.Bool(true)
			props.SpotBidPercentage = jsii.Number(spotBid)
		}

		// AllocationStrategy 必须与 Spot/OD 匹配：Spot 策略不能用于 OD CE
		if inst.AllocationStrategy != "" {
			switch inst.AllocationStrategy {
			case "BEST_FIT":
				props.AllocationStrategy = awsbatch.AllocationStrategy_BEST_FIT
			case "BEST_FIT_PROGRESSIVE":
				props.AllocationStrategy = awsbatch.AllocationStrategy_BEST_FIT_PROGRESSIVE
			case "SPOT_CAPACITY_OPTIMIZED", "SPOT_PRICE_CAPACITY_OPTIMIZED":
				if spotBid > 0 {
					props.AllocationStrategy = awsbatch.AllocationStrategy_SPOT_PRICE_CAPACITY_OPTIMIZED
				} else {
					props.AllocationStrategy = awsbatch.AllocationStrategy_BEST_FIT_PROGRESSIVE
				}
			}
		}

		if inst.ServiceRolePolicies != "" {
			serviceRoleId := fmt.Sprintf("%s-%s-service-role", inst.GetID(), ceNames[i])
			serviceRole := aws.CreateRole(ctx.Stack, serviceRoleId, inst.ServiceRolePolicies, "batch")
			props.ServiceRole = serviceRole
		}

		// PlacementGroup：按 CE 位置对应；CLUSTER 适合 Multinode/MPI，保证同 AZ 高带宽
		if i < len(placementGroupList) && placementGroupList[i] != "" {
			pgName := fmt.Sprintf("%s-%s-pg", inst.GetID(), ceNames[i])
			pg := awsec2.NewPlacementGroup(ctx.Stack, jsii.String(pgName), &awsec2.PlacementGroupProps{
				PlacementGroupName: jsii.String(pgName),
				Strategy:           resolvePlacementGroupStrategy(placementGroupList[i]),
			})
			props.PlacementGroup = pg
		}

		ce := awsbatch.NewManagedEc2EcsComputeEnvironment(ctx.Stack, jsii.String(ceName), props)
		b.computeEnvironments[ceNames[i]] = ce
	}
}

func (b *BatchForge) createLaunchTemplateWithUserData(inst *BatchInstanceConfig, ctx *interfaces.ForgeContext, suffix string) awsec2.LaunchTemplate {
	templateName := fmt.Sprintf("%s-%s-batch-lt", inst.GetID(), suffix)

	magicToken, err := dependency.GetDependencyInfo(inst.DependsOn)
	if err != nil {
		fmt.Printf("Error getting dependency info: %v\n", err)
	}

	userDataGenerator := &aws.UserDataGenerator{
		OsType:             awsec2.OperatingSystemType_LINUX,
		ScriptPath:         "./userdata.sh",
		UserDataToken:      inst.UserDataToken,
		UserDataScriptPath: inst.UserDataScriptPath,
		MagicToken:         magicToken,
		S3Location:         inst.S3Location,
	}

	userData, err := userDataGenerator.GenerateMimeMultipartUserData()
	if err != nil {
		fmt.Printf("Error generating MIME multipart user data: %v\n", err)
		userData = awsec2.UserData_ForLinux(&awsec2.LinuxUserDataOptions{})
	}

	ltProps := &awsec2.LaunchTemplateProps{
		LaunchTemplateName: jsii.String(templateName),
		UserData:           userData,
	}

	lt := awsec2.NewLaunchTemplate(ctx.Stack, jsii.String(templateName), ltProps)

	// 关闭超线程：CpuOptions 需要同时指定 CoreCount 和 ThreadsPerCore，
	// 但 Batch CE 内混合多种实例类型时 CoreCount 各不相同，无法在 LaunchTemplate 层面统一设置。
	// 如需禁用超线程，应在 CE 内只放同一种实例类型，或在容器内通过 numactl/taskset 控制。

	return lt
}

// ── Job Queues ────────────────────────────────────────────────────────────────

func (b *BatchForge) createJobQueues(inst *BatchInstanceConfig, ctx *interfaces.ForgeContext) {
	queueNames := splitComma(inst.QueueNames)
	queueCeRefs := splitComma(inst.QueueCeRefs)
	queuePriorities := splitComma(inst.QueuePriorities)

	// 若未指定 queueNames，为每个 CE 自动生成一个同名 Queue
	if len(queueNames) == 0 {
		for ceName := range b.computeEnvironments {
			queueNames = append(queueNames, ceName)
			queueCeRefs = append(queueCeRefs, ceName)
			queuePriorities = append(queuePriorities, "10")
		}
	}

	count := len(queueNames)
	if len(queueCeRefs) != count {
		panic(fmt.Sprintf("batch %s: queueNames(%d) 与 queueCeRefs(%d) 数量不一致", inst.GetID(), count, len(queueCeRefs)))
	}

	for i, queueName := range queueNames {
		fullQueueName := fmt.Sprintf("%s-%s", inst.GetID(), queueName)

		priority := 10
		if i < len(queuePriorities) {
			if p, err := strconv.Atoi(strings.TrimSpace(queuePriorities[i])); err == nil {
				priority = p
			}
		}

		ceRef := strings.TrimSpace(queueCeRefs[i])
		ce, ok := b.computeEnvironments[ceRef]
		if !ok {
			panic(fmt.Sprintf("batch %s: queue %s 引用了不存在的 CE %s", inst.GetID(), queueName, ceRef))
		}

		queue := awsbatch.NewJobQueue(ctx.Stack, jsii.String(fullQueueName), &awsbatch.JobQueueProps{
			JobQueueName: jsii.String(fullQueueName),
			Priority:     jsii.Number(priority),
			ComputeEnvironments: &[]*awsbatch.OrderedComputeEnvironment{
				{
					ComputeEnvironment: ce,
					Order:              jsii.Number(1),
				},
			},
		})

		b.jobQueues[queueName] = queue
	}
}

// ── Job Definitions ───────────────────────────────────────────────────────────

func (b *BatchForge) createJobDefinitions(inst *BatchInstanceConfig, ctx *interfaces.ForgeContext) {
	nameList := parseJobDefNames(inst.JobDefNames, inst.GetID(), len(splitComma(inst.JobDefNames)))
	count := len(nameList)
	images := broadcastString(inst.ContainerImage, count)
	vcpuList := parseIntListFromString(inst.VCpus, count, 1)
	memoryList := parseIntListFromString(inst.Memory, count, 512)
	timeoutList := parseIntListFromString(inst.TimeoutMinutes, count, 0)
	numNodesList := parseIntListFromString(inst.NumNodes, count, 2)
	mainNodeList := parseIntListFromString(inst.MainNode, count, 0)

	if len(images) != count || len(vcpuList) != count || len(memoryList) != count {
		panic(fmt.Sprintf("batch %s: jobDefNames(%d), containerImage(%d), vcpus(%d), memory(%d) 数量不一致",
			inst.GetID(), count, len(images), len(vcpuList), len(memoryList)))
	}

	for i, image := range images {
		var jd awsbatch.IJobDefinition
		if inst.JobDefinitionType == "multinode" {
			jd = b.createMultinodeJobDefinition(inst, ctx, nameList[i], image, vcpuList[i], memoryList[i], timeoutList[i], numNodesList[i], mainNodeList[i])
		} else {
			jd = b.createContainerJobDefinition(inst, ctx, nameList[i], image, vcpuList[i], memoryList[i], timeoutList[i])
		}
		b.jobDefinitions = append(b.jobDefinitions, namedJobDef{name: nameList[i], jd: jd})
	}
}

func (b *BatchForge) createContainerJobDefinition(inst *BatchInstanceConfig, ctx *interfaces.ForgeContext, jobDefName, image string, vcpus, memory, timeoutMinutes int) awsbatch.EcsJobDefinition {
	execRoleId := fmt.Sprintf("%s-exec-role", jobDefName)
	execRole := aws.CreateRole(ctx.Stack, execRoleId, "service-role/AmazonECSTaskExecutionRolePolicy", "ecs-tasks")

	containerProps := &awsbatch.EcsEc2ContainerDefinitionProps{
		Image:         awsecs.ContainerImage_FromRegistry(jsii.String(image), &awsecs.RepositoryImageProps{}),
		Cpu:           jsii.Number(vcpus),
		Memory:        awscdk.Size_Mebibytes(jsii.Number(memory)),
		ExecutionRole: execRole,
	}

	if inst.JobRolePolicies != "" {
		jobRoleId := fmt.Sprintf("%s-job-role", jobDefName)
		jobRole := aws.CreateRole(ctx.Stack, jobRoleId, inst.JobRolePolicies, "ecs-tasks")
		containerProps.JobRole = jobRole
	}

	volumes := b.createStorageVolumes(inst)
	if len(volumes) > 0 {
		containerProps.Volumes = &volumes
	}

	containerDef := awsbatch.NewEcsEc2ContainerDefinition(ctx.Stack, jsii.String(jobDefName+"-container"), containerProps)

	jobDefProps := &awsbatch.EcsJobDefinitionProps{
		JobDefinitionName: jsii.String(jobDefName),
		Container:         containerDef,
	}

	if inst.RetryAttempts > 0 {
		jobDefProps.RetryAttempts = jsii.Number(inst.RetryAttempts)
		jobDefProps.RetryStrategies = &[]awsbatch.RetryStrategy{
			awsbatch.RetryStrategy_Of(awsbatch.Action_RETRY, awsbatch.Reason_SPOT_INSTANCE_RECLAIMED()),
			awsbatch.RetryStrategy_Of(awsbatch.Action_EXIT, awsbatch.Reason_NON_ZERO_EXIT_CODE()),
		}
	}

	if timeoutMinutes > 0 {
		jobDefProps.Timeout = awscdk.Duration_Minutes(jsii.Number(timeoutMinutes))
	}

	return awsbatch.NewEcsJobDefinition(ctx.Stack, jsii.String(jobDefName), jobDefProps)
}

func (b *BatchForge) createMultinodeJobDefinition(inst *BatchInstanceConfig, ctx *interfaces.ForgeContext, jobDefName, image string, vcpus, memory, timeoutMinutes, numNodes, mainNode int) awsbatch.IJobDefinition {

	execRoleId := fmt.Sprintf("%s-multinode-exec-role", jobDefName)
	execRole := aws.CreateRole(ctx.Stack, execRoleId, "service-role/AmazonECSTaskExecutionRolePolicy", "ecs-tasks")

	containerProps := &awsbatch.EcsEc2ContainerDefinitionProps{
		Image:         awsecs.ContainerImage_FromRegistry(jsii.String(image), &awsecs.RepositoryImageProps{}),
		Cpu:           jsii.Number(vcpus),
		Memory:        awscdk.Size_Mebibytes(jsii.Number(memory)),
		ExecutionRole: execRole,
	}

	if inst.JobRolePolicies != "" {
		jobRoleId := fmt.Sprintf("%s-multinode-job-role", jobDefName)
		jobRole := aws.CreateRole(ctx.Stack, jobRoleId, inst.JobRolePolicies, "ecs-tasks")
		containerProps.JobRole = jobRole
	}

	volumes := b.createStorageVolumes(inst)
	if len(volumes) > 0 {
		containerProps.Volumes = &volumes
	}

	mainContainer := awsbatch.NewEcsEc2ContainerDefinition(ctx.Stack, jsii.String(jobDefName+"-main-container"), containerProps)
	workerContainer := awsbatch.NewEcsEc2ContainerDefinition(ctx.Stack, jsii.String(jobDefName+"-worker-container"), containerProps)

	containers := []*awsbatch.MultiNodeContainer{
		{Container: mainContainer, StartNode: jsii.Number(mainNode), EndNode: jsii.Number(mainNode)},
		{Container: workerContainer, StartNode: jsii.Number(mainNode + 1), EndNode: jsii.Number(numNodes - 1)},
	}

	return awsbatch.NewMultiNodeJobDefinition(ctx.Stack, jsii.String(jobDefName), &awsbatch.MultiNodeJobDefinitionProps{
		JobDefinitionName: jsii.String(jobDefName),
		MainNode:          jsii.Number(mainNode),
		Containers:        &containers,
	})
}

func (b *BatchForge) createStorageVolumes(inst *BatchInstanceConfig) []awsbatch.EcsVolume {
	var volumes []awsbatch.EcsVolume
	if inst.DependsOn == "" {
		return volumes
	}

	for _, dep := range splitComma(inst.DependsOn) {
		mountPoint, err := dependency.GetMountPoint(dep)
		if err != nil {
			fmt.Printf("Error getting mount point for %s: %v\n", dep, err)
			continue
		}

		parts := strings.Split(dep, ":")
		if len(parts) != 2 {
			continue
		}
		resourceType := strings.ToLower(parts[0])
		resourceId := parts[1]

		if resourceType == "efs" || resourceType == "fsx" || resourceType == "lustre" {
			volumes = append(volumes, awsbatch.EcsVolume_Host(&awsbatch.HostVolumeOptions{
				Name:          jsii.String(fmt.Sprintf("%s-volume", resourceId)),
				HostPath:      jsii.String(mountPoint),
				ContainerPath: jsii.String(mountPoint),
			}))
		}
	}
	return volumes
}

// ── MergeConfigs ──────────────────────────────────────────────────────────────

func (b *BatchForge) MergeConfigs(defaults, instance config.InstanceConfig) config.InstanceConfig {
	src := defaults.(*BatchInstanceConfig)
	merged := *src
	inst := instance.(*BatchInstanceConfig)

	if inst.GetID() != "" {
		merged.ID = inst.GetID()
	}
	if inst.GetType() != "" {
		merged.Type = inst.GetType()
	}
	if inst.GetSubnet() != "" {
		merged.Subnet = inst.GetSubnet()
	}
	if inst.GetSecurityGroup() != "" {
		merged.SecurityGroup = inst.GetSecurityGroup()
	}
	if inst.CeNames != "" {
		merged.CeNames = inst.CeNames
	}
	if inst.ImageTypes != "" {
		merged.ImageTypes = inst.ImageTypes
	}
	if inst.InstanceTypes != "" {
		merged.InstanceTypes = inst.InstanceTypes
	}
	if inst.MaxvCpus != "" {
		merged.MaxvCpus = inst.MaxvCpus
	}
	if inst.MinvCpus != "" {
		merged.MinvCpus = inst.MinvCpus
	}
	if inst.AllocationStrategy != "" {
		merged.AllocationStrategy = inst.AllocationStrategy
	}
	if inst.SpotBidPercentage != "" {
		merged.SpotBidPercentage = inst.SpotBidPercentage
	}
	if inst.AzIndex != "" {
		merged.AzIndex = inst.AzIndex
	}
	if inst.QueueNames != "" {
		merged.QueueNames = inst.QueueNames
	}
	if inst.QueueCeRefs != "" {
		merged.QueueCeRefs = inst.QueueCeRefs
	}
	if inst.QueuePriorities != "" {
		merged.QueuePriorities = inst.QueuePriorities
	}
	if inst.JobDefNames != "" {
		merged.JobDefNames = inst.JobDefNames
	}
	if inst.ContainerImage != "" {
		merged.ContainerImage = inst.ContainerImage
	}
	if inst.VCpus != "" {
		merged.VCpus = inst.VCpus
	}
	if inst.Memory != "" {
		merged.Memory = inst.Memory
	}
	if inst.JobDefinitionType != "" {
		merged.JobDefinitionType = inst.JobDefinitionType
	}
	if inst.NumNodes != "" {
		merged.NumNodes = inst.NumNodes
	}
	if inst.MainNode != "" {
		merged.MainNode = inst.MainNode
	}
	if inst.InstanceRolePolicies != "" {
		merged.InstanceRolePolicies = inst.InstanceRolePolicies
	}
	if inst.ServiceRolePolicies != "" {
		merged.ServiceRolePolicies = inst.ServiceRolePolicies
	}
	if inst.JobRolePolicies != "" {
		merged.JobRolePolicies = inst.JobRolePolicies
	}
	if inst.RetryAttempts != 0 {
		merged.RetryAttempts = inst.RetryAttempts
	}
	if inst.UserDataToken != "" {
		merged.UserDataToken = inst.UserDataToken
	}
	if inst.UserDataScriptPath != "" {
		merged.UserDataScriptPath = inst.UserDataScriptPath
	}
	if inst.S3Location != "" {
		merged.S3Location = inst.S3Location
	}
	if inst.DependsOn != "" {
		merged.DependsOn = inst.DependsOn
	}
	if inst.UseOptimalInstanceTypes != nil {
		merged.UseOptimalInstanceTypes = inst.UseOptimalInstanceTypes
	}
	if inst.UpdateToLatestImageVersion != nil {
		merged.UpdateToLatestImageVersion = inst.UpdateToLatestImageVersion
	}
	if inst.PlacementGroupStrategy != "" {
		merged.PlacementGroupStrategy = inst.PlacementGroupStrategy
	}
	if inst.DisableHyperthreading {
		merged.DisableHyperthreading = inst.DisableHyperthreading
	}
	if inst.TimeoutMinutes != "" {
		merged.TimeoutMinutes = inst.TimeoutMinutes
	}

	return &merged
}

func (b *BatchForge) ConfigureRules(ctx *interfaces.ForgeContext) {}

func (b *BatchForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	inst := (*ctx.Instance).(*BatchInstanceConfig)

	for name, ce := range b.computeEnvironments {
		awscdk.NewCfnOutput(ctx.Stack, jsii.String("BatchCE"+inst.GetID()+name), &awscdk.CfnOutputProps{
			Value:       ce.ComputeEnvironmentArn(),
			Description: jsii.String("Batch Compute Environment ARN: " + name),
		})
	}

	for name, queue := range b.jobQueues {
		awscdk.NewCfnOutput(ctx.Stack, jsii.String("BatchQueue"+inst.GetID()+name), &awscdk.CfnOutputProps{
			Value:       queue.JobQueueArn(),
			Description: jsii.String("Batch Job Queue ARN: " + name),
		})
		awscdk.NewCfnOutput(ctx.Stack, jsii.String("BatchQueueName"+inst.GetID()+name), &awscdk.CfnOutputProps{
			Value:       queue.JobQueueName(),
			Description: jsii.String("Batch Job Queue Name: " + name),
		})
	}

	for _, entry := range b.jobDefinitions {
		awscdk.NewCfnOutput(ctx.Stack, jsii.String("BatchJD"+entry.name), &awscdk.CfnOutputProps{
			Value:       entry.jd.JobDefinitionArn(),
			Description: jsii.String("Batch Job Definition ARN: " + entry.name),
		})
	}
}

// ── 辅助函数 ──────────────────────────────────────────────────────────────────

// splitSemi 用 ; 分隔不同 CE（仅 instanceTypes 使用）
func splitSemi(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ";")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// splitComma 用 , 分隔多个成员（其他所有多值字段）
func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

// padNames 若 names 数量不足 count，用 prefix-0/1/2 补齐
func padNames(names []string, count int, prefix string) []string {
	result := make([]string, count)
	for i := range result {
		if i < len(names) {
			result[i] = names[i]
		} else {
			result[i] = fmt.Sprintf("%s-%d", prefix, i)
		}
	}
	return result
}

// broadcastString 单值则广播到 count 个，多值直接返回
func broadcastString(s string, count int) []string {
	parts := splitComma(s)
	if len(parts) == 1 {
		result := make([]string, count)
		for i := range result {
			result[i] = parts[0]
		}
		return result
	}
	return parts
}

// parseIntListFromString 解析逗号分隔的整数列表；单值则广播，空串用 defaultVal 广播
func parseIntListFromString(s string, count int, defaultVal int) []int {
	result := make([]int, count)
	parts := splitComma(s)

	for i := range result {
		val := defaultVal
		if len(parts) == 1 {
			if v, err := strconv.Atoi(parts[0]); err == nil {
				val = v
			}
		} else if i < len(parts) {
			if v, err := strconv.Atoi(parts[i]); err == nil {
				val = v
			}
		}
		result[i] = val
	}
	return result
}

// resolvePlacementGroupStrategy 将字符串映射到 PlacementGroupStrategy
func resolvePlacementGroupStrategy(s string) awsec2.PlacementGroupStrategy {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CLUSTER":
		return awsec2.PlacementGroupStrategy_CLUSTER
	case "PARTITION":
		return awsec2.PlacementGroupStrategy_PARTITION
	case "SPREAD":
		return awsec2.PlacementGroupStrategy_SPREAD
	default:
		return awsec2.PlacementGroupStrategy_CLUSTER
	}
}

// resolveImageType 将字符串映射到 EcsMachineImageType
func resolveImageType(s string) awsbatch.EcsMachineImageType {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "ECS_AL2_NVIDIA":
		return awsbatch.EcsMachineImageType_ECS_AL2_NVIDIA
	case "ECS_AL2023":
		return awsbatch.EcsMachineImageType_ECS_AL2023
	// TODO: 升级 CDK 版本后启用 ECS_AL2023_NVIDIA
	// case "ECS_AL2023_NVIDIA":
	// 	return awsbatch.EcsMachineImageType_ECS_AL2023_NVIDIA
	default: // ECS_AL2
		return awsbatch.EcsMachineImageType_ECS_AL2
	}
}

// parseJobDefNames 生成 JD 名字列表
func parseJobDefNames(jobDefNames, baseID string, count int) []string {
	names := splitComma(jobDefNames)
	result := make([]string, count)
	for i := range result {
		if i < len(names) {
			result[i] = fmt.Sprintf("%s-%s-job-def", baseID, names[i])
		} else {
			result[i] = fmt.Sprintf("%s-job-def-%d", baseID, i)
		}
	}
	return result
}
