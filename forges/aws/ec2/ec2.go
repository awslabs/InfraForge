// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ec2

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
	"github.com/aws/aws-cdk-go/awscdk/v2/awsssm"
	"github.com/aws/jsii-runtime-go"
)

type Ec2InstanceConfig struct {
	config.BaseInstanceConfig
	AzIndex                  int    `json:"azIndex,omitempty"`
	InstanceCount            int    `json:"instanceCount,omitempty"`
	Debug                    *bool  `json:"debug,omitempty"`
	// 简化：移除所有 shared 冗余字段，只保留实例字段
	KeyName                  string `json:"keyName,omitempty"`
	Policies                 string `json:"policies,omitempty"`
	PlacementGroup           string `json:"placementGroup,omitempty"`
	PlacementGroupStrategy   string `json:"placementGroupStrategy,omitempty"`
	
	// 移除的冗余字段：
	// SharedKeyName            string `json:"sharedKeyName"`
	// SharedPolicies           string `json:"sharedPolicies"`
	// SharedPlacementGroup     string `json:"sharedPlacementGroup"`
	// SharedPGStrategy         string `json:"sharedPGStrategy"`
	DetailedMonitoring       *bool  `json:"detailedMonitoring,omitempty"`
	DependsOn                string `json:"dependsOn,omitempty"`
	EbsDeviceName            string `json:"ebsDeviceName,omitempty"`
	EbsIops                  int    `json:"ebsIops,omitempty"`
	EbsSize                  int    `json:"ebsSize,omitempty"`
	EbsThroughput            int    `json:"ebsThroughput,omitempty"`
	EbsVolumeType            string `json:"ebsVolumeType,omitempty"`
	EbsOptimized             *bool  `json:"ebsOptimized,omitempty"`
	EnclaveEnabled           *bool  `json:"enclaveEnabled,omitempty"`
	EnableEfa                *bool  `json:"enableEfa,omitempty"`
	EnaSrdEnabled            *bool  `json:"enaSrdEnabled,omitempty"`
	NetworkCardCount         int    `json:"networkCardCount,omitempty"`  // 使用的网卡数量，用于多网卡实例类型
	PurchaseOption           string `json:"purchaseOption,omitempty"`    // 购买选项：od, spot
	SpotMaxPrice             string `json:"spotMaxPrice,omitempty"`      // Spot实例最高价格
	CapacityBlockId          string `json:"capacityBlockId,omitempty"`   // Capacity Block ID（独立配置）
	AllowedPorts             string `json:"allowedPorts,omitempty"`      // 允许的端口配置
	AllowedPortsIpv6         string `json:"allowedPortsIpv6,omitempty"`  // IPv6端口配置
	InstanceType             string `json:"instanceType"`
	OsArch                   string `json:"osArch"`
	OsImage                  string `json:"osImage,omitempty"`
	OsName                   string `json:"osName,omitempty"`
	OsType                   string `json:"osType,omitempty"`
	OsVersion                string `json:"osVersion,omitempty"`
	S3Location               string `json:"s3Location,omitempty"`
	RequireImdsv2            *bool  `json:"requireImdsv2,omitempty"`
	UserDataToken            string `json:"userDataToken,omitempty"`
	UserDataScriptPath       string `json:"userDataScriptPath,omitempty"`
	StoreInstanceInfo        *bool  `json:"storeInstanceInfo,omitempty"`
}

/*
   # OSType
   # https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2@v2.171.1/awsec2#OperatingSystemType
   const (
	OperatingSystemType_LINUX   OperatingSystemType = "LINUX"
	OperatingSystemType_WINDOWS OperatingSystemType = "WINDOWS"
	// Used when the type of the operating system is not known (for example, for imported Auto-Scaling Groups).
	OperatingSystemType_UNKNOWN OperatingSystemType = "UNKNOWN"
        )

   # MachineImage
   # https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2@v2.171.1/awsec2#MachineImageConfig

   type IMachineImage interface {
	// Return the image to use in the given context.
	GetImage(scope constructs.Construct) *MachineImageConfig
        }

   type MachineImageConfig struct {
	// The AMI ID of the image to use.
	ImageId *string `field:"required" json:"imageId" yaml:"imageId"`
	// Operating system type for this image.
	OsType OperatingSystemType `field:"required" json:"osType" yaml:"osType"`
	// Initial UserData for this image.
	UserData UserData `field:"required" json:"userData" yaml:"userData"`
        }

   var userData userData


   machineImageConfig := &MachineImageConfig{
	ImageId: jsii.String("imageId"),
	OsType: awscdk.Aws_ec2.OperatingSystemType_LINUX,
	UserData: userData,
        }

*/

type Ec2Forge struct {
	ec2Instances []awsec2.Instance
	properties   map[string]interface{}
}

// EFAConfig EFA 网络接口配置
type EFAConfig struct {
	SubnetSelection *awsec2.SubnetSelection  // 可选，指定子网选择
	DeviceIndex    *string                   // 可选，指定设备索引
	Description    *string                   // 可选，网络接口描述
	// 其他可能的配置项
}

func (e *Ec2Forge) Create(ctx *interfaces.ForgeContext) interface{} {
	ec2Instance, ok := (*ctx.Instance).(*Ec2InstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return nil
	}

	var instances []awsec2.Instance
	var origId string
	var pg awsec2.IPlacementGroup
        origId = ec2Instance.GetID()

	// 简化：直接使用 PlacementGroup 字段
	if ec2Instance.PlacementGroup != "" {
		// 使用共享的 PlacementGroup 管理函数
		pg = aws.CreateOrGetPlacementGroup(ctx.Stack, ec2Instance.PlacementGroup, strings.ToUpper(ec2Instance.PlacementGroupStrategy))
	} else {
		pg = nil
	}
	
	if ec2Instance.InstanceCount > 1 && types.GetBoolValue(ec2Instance.StoreInstanceInfo, false) {
		// 创建 SSM Parameter 来存储实例数量
		awsssm.NewStringParameter(ctx.Stack, jsii.String(fmt.Sprintf("%s-count", origId)), &awsssm.StringParameterProps{
			ParameterName: jsii.String(fmt.Sprintf("/infraforge/ec2/%s/instanceCount", origId)),
			StringValue:   jsii.String(fmt.Sprintf("%d", ec2Instance.InstanceCount)),
		})

		instances = make([]awsec2.Instance, ec2Instance.InstanceCount)
		for i := 0; i < ec2Instance.InstanceCount; i++ {
			ec2Instance.SetID(fmt.Sprintf("%s.%d", origId, i + 1))
			instances[i] = createEc2Instance(ctx.Stack, ec2Instance, ctx.VPC, ctx.SubnetType, pg, ctx.SecurityGroups.Default, ctx.Dependencies, ctx.DualStack) 

			// 可选：为每个实例单独创建一个参数
			awsssm.NewStringParameter(ctx.Stack, jsii.String(fmt.Sprintf("%s-hostInfo-%d", origId, i)), &awsssm.StringParameterProps{
				ParameterName: jsii.String(fmt.Sprintf("/infraforge/ec2/%s/instance-%d", origId, i + 1)),
				StringValue: awscdk.Fn_Join(jsii.String(","), &[]*string{
					awscdk.Token_AsString(instances[i].InstancePrivateDnsName(), &awscdk.EncodingOptions{}),
					awscdk.Token_AsString(instances[i].InstancePrivateIp(), &awscdk.EncodingOptions{}),
				}),
			})
		}
	} else {
		instances = make([]awsec2.Instance, 1)
		instances[0] = createEc2Instance(ctx.Stack, ec2Instance, ctx.VPC, ctx.SubnetType, pg, ctx.SecurityGroups.Default, ctx.Dependencies, ctx.DualStack) 
	}

	e.ec2Instances = instances
	
	// 保存 EC2 属性
	if e.properties == nil {
		e.properties = make(map[string]interface{})
	}
	
	// 基本属性
	e.properties["instanceCount"] = len(instances)
	e.properties["instanceType"] = ec2Instance.InstanceType
	e.properties["osName"] = ec2Instance.OsName
	e.properties["osVersion"] = ec2Instance.OsVersion
	
	// 实例详细信息
	instanceDetails := make([]map[string]interface{}, 0, len(instances))
	for _, instance := range instances {
		instanceInfo := map[string]interface{}{
			"instanceId":             instance.InstanceId(),
			"instancePrivateIp":      instance.InstancePrivateIp(),
			"instancePrivateDnsName": instance.InstancePrivateDnsName(),
		}
		instanceDetails = append(instanceDetails, instanceInfo)
	}
	e.properties["instances"] = instanceDetails
	
	return e
}

func createEc2Instance(stack awscdk.Stack, ec2Instance *Ec2InstanceConfig, vpc awsec2.IVpc, subnetType awsec2.SubnetType, pg awsec2.IPlacementGroup, defaultSG awsec2.SecurityGroup, dependencies map[string]interface{}, dualStack bool) awsec2.Instance {
	amiOwner, amiName := aws.GetAMIInfo(partition.DefaultPartition, ec2Instance.OsName, ec2Instance.OsVersion, ec2Instance.OsArch)

	var amiArch string
	if ec2Instance.OsArch == "aarch64" {
		amiArch = "arm64"
	} else {
		amiArch = ec2Instance.OsArch
	}

	lookup := aws.ForgeAMILookup{
		AmiOwner:   amiOwner,
		AmiName:    amiName,
		AmiArch:    amiArch,
	}

	if ec2Instance.OsImage == "" {
		osImage, err := lookup.FindAMI()
		if err != nil {
			fmt.Println("Error:", err)
			return nil
		}
		ec2Instance.OsImage = osImage
	}

	deviceName := ec2Instance.EbsDeviceName
	var err error
	if ec2Instance.EbsDeviceName == "" {
		deviceName, err = aws.DescribeAMI(ec2Instance.OsImage)
		if err != nil {
			deviceName = "/dev/sda1"
		}
	}

	if types.GetBoolValue(ec2Instance.Debug, false) {
		fmt.Println("AMIInfo:", ec2Instance.OsImage, amiOwner,ec2Instance.OsArch, amiName)
	}

	var iKeyPair awsec2.IKeyPair

	// 确保使用与预创建时相同的 KeyName 逻辑
	keyName := ec2Instance.KeyName
	if keyName == "" {
		keyName = *awscdk.Aws_STACK_NAME()
	}
	iKeyPair = aws.CreateOrGetKeyPair(stack, keyName, ec2Instance.OsType)
	// 创建配置
	ebsConfig := &aws.EbsConfig{
		VolumeType:    ec2Instance.EbsVolumeType,
		Iops:          ec2Instance.EbsIops,
		Size:          ec2Instance.EbsSize,
		//Throughput:    ec2Instance.EbsThroughput,
		Optimized:     types.GetBoolValue(ec2Instance.EbsOptimized, false),
		DeviceName:    deviceName,
	}

	if types.GetBoolValue(ec2Instance.Debug, false) {
		fmt.Println("EBSInfo:", ec2Instance.EbsVolumeType, ec2Instance.EbsSize, ec2Instance.EbsIops, ec2Instance.EbsThroughput, deviceName)
	}
	
	// 调用函数
	blockDevices, err := aws.CreateEbsBlockDevices(ebsConfig)
	if err != nil {
		// 处理错误
		fmt.Errorf("failed to create block devices: %w", err)
		return  nil
	}

	magicToken, err := dependency.GetDependencyInfo(ec2Instance.DependsOn)
	if err != nil {
		fmt.Printf("Error getting dependency info: %v\n", err)
	}

	// 使用统一的子网选择函数
	selectedSubnet := aws.SelectSubnetByAzIndex(ec2Instance.AzIndex, vpc, subnetType)

	instanceProps := &awsec2.InstanceProps{
		Vpc:                vpc,
		InstanceType:       awsec2.NewInstanceType(jsii.String(ec2Instance.InstanceType)),
		EbsOptimized:       jsii.Bool(types.GetBoolValue(ec2Instance.EbsOptimized, false)),
		EnclaveEnabled:     jsii.Bool(types.GetBoolValue(ec2Instance.EnclaveEnabled, false)),
		DetailedMonitoring: jsii.Bool(types.GetBoolValue(ec2Instance.DetailedMonitoring, false)),
		MachineImage: &aws.ForgeAMIConfig{
			OsImage:           ec2Instance.OsImage,
			OsType:            ec2Instance.OsType,
			UserDataToken:     ec2Instance.UserDataToken,
			UserDataScriptPath: ec2Instance.UserDataScriptPath,
			MagicToken:        magicToken,
			S3Location:        ec2Instance.S3Location,
		},
		KeyPair:        iKeyPair,
		SecurityGroup:  defaultSG,
		VpcSubnets:     &awsec2.SubnetSelection{Subnets: &[]awsec2.ISubnet{selectedSubnet}},
		BlockDevices:   &blockDevices,
		PlacementGroup: pg,
	}

	// 简化：直接使用 policies 字段
	if ec2Instance.Policies != "" {
		instanceProfile := aws.CreateOrGetInstanceProfile(stack, ec2Instance.Policies)
		instanceProps.InstanceProfile = instanceProfile
	}

	// 首先定义一个变量来存储实例属性

	// 创建实例
	inst := awsec2.NewInstance(stack, jsii.String(ec2Instance.GetID()), instanceProps)

	/*
	inst := awsec2.NewInstance(stack, jsii.String(ec2Instance.GetID()), &awsec2.InstanceProps{
		Vpc: vpc,
		InstanceType: awsec2.NewInstanceType(jsii.String(ec2Instance.InstanceType)),
		EbsOptimized: jsii.Bool(types.GetBoolValue(ec2Instance.EbsOptimized, false)),
		EnclaveEnabled: jsii.Bool(types.GetBoolValue(ec2Instance.EnclaveEnabled, false)),
		DetailedMonitoring: jsii.Bool(types.GetBoolValue(ec2Instance.DetailedMonitoring, false)),
		MachineImage: &aws.ForgeAMIConfig{
			OsImage:           ec2Instance.OsImage,
			OsType:            ec2Instance.OsType,
			UserDataToken:     ec2Instance.UserDataToken,
			UserDataScriptPath: ec2Instance.UserDataScriptPath,
			MagicToken:        magicToken,
			S3Location:        ec2Instance.S3Location,
		},
		KeyPair: iKeyPair,
		SecurityGroup: defaultSG,
		VpcSubnets: &awsec2.SubnetSelection{SubnetType: subnetType},
		BlockDevices: &blockDevices,
		PlacementGroup: pg,
	},
	)
	*/

	// 获取第一个块设备
	//ebsVolume := blockDevices[0].Volume.(awsec2.BlockDeviceVolume)

	// 获取 EBS 设备属性
	//ebsDeviceProps := ebsVolume.EbsDevice()

	//throughput := *ebsDeviceProps.Throughput

	// 检查是否需要创建启动模板
	needsLaunchTemplate := types.GetBoolValue(ec2Instance.EnableEfa, false) || types.GetBoolValue(ec2Instance.EnaSrdEnabled, false) || ec2Instance.EbsThroughput > 125 || ec2Instance.NetworkCardCount > 1 || ec2Instance.PurchaseOption == "spot" || ec2Instance.CapacityBlockId != ""

	if needsLaunchTemplate {
		// 获取原始EC2实例的L1构造
		cfnInstance := inst.Node().DefaultChild().(awsec2.CfnInstance)

		// 创建启动模板数据
		launchTemplateData := &awsec2.CfnLaunchTemplate_LaunchTemplateDataProperty{}

		// 配置购买选项（Spot 和 Capacity Block 互斥）
		if ec2Instance.PurchaseOption == "spot" {
			// Spot 实例配置
			spotOptions := &awsec2.CfnLaunchTemplate_SpotOptionsProperty{}
			
			// 设置最高价格（如果指定）
			if ec2Instance.SpotMaxPrice != "" {
				spotOptions.MaxPrice = jsii.String(ec2Instance.SpotMaxPrice)
			}
			
			launchTemplateData.InstanceMarketOptions = &awsec2.CfnLaunchTemplate_InstanceMarketOptionsProperty{
				MarketType:  jsii.String("spot"),
				SpotOptions: spotOptions,
			}
		} else if ec2Instance.CapacityBlockId != "" {
			// Capacity Block 配置（仅在非 Spot 模式下）
			launchTemplateData.CapacityReservationSpecification = &awsec2.CfnLaunchTemplate_CapacityReservationSpecificationProperty{
				CapacityReservationTarget: &awsec2.CfnLaunchTemplate_CapacityReservationTargetProperty{
					CapacityReservationId: jsii.String(ec2Instance.CapacityBlockId),
				},
			}
		}

		// 如果需要配置网络接口（EFA、EnaSrd或多网卡）
		if types.GetBoolValue(ec2Instance.EnableEfa, false) || types.GetBoolValue(ec2Instance.EnaSrdEnabled, false) || ec2Instance.NetworkCardCount > 1 {
			// 获取子网ID和安全组IDs
			subnetId := cfnInstance.SubnetId()
			securityGroupIds := cfnInstance.SecurityGroupIds()

			// 删除原始网络属性
			cfnInstance.AddDeletionOverride(jsii.String("Properties.SubnetId"))
			cfnInstance.AddDeletionOverride(jsii.String("Properties.SecurityGroupIds"))

			// 确定网卡数量，默认为1
			cardCount := ec2Instance.NetworkCardCount
			if cardCount <= 0 {
				cardCount = 1
			}

			// 创建多个网络接口，每个网卡一个接口
			networkInterfaces := make([]interface{}, cardCount)
			for i := 0; i < cardCount; i++ {
				var deviceIndex int
				var networkCardIndex int
				
				if i == 0 {
					// 主网络接口：NetworkCardIndex=0, DeviceIndex=0
					deviceIndex = 0
					networkCardIndex = 0
				} else {
					// 其他网络接口：NetworkCardIndex=i, DeviceIndex=1
					deviceIndex = 1
					networkCardIndex = i
				}

				networkInterface := &awsec2.CfnLaunchTemplate_NetworkInterfaceProperty{
					DeviceIndex:      jsii.Number(deviceIndex),
					NetworkCardIndex: jsii.Number(networkCardIndex),
					SubnetId:         subnetId,
					Groups:           securityGroupIds,
					DeleteOnTermination: jsii.Bool(true),
				}

				// 如果启用EFA，所有网卡都配置为EFA类型
				if types.GetBoolValue(ec2Instance.EnableEfa, false) {
					if i == 0 {
						// 主接口使用 "efa"
						networkInterface.InterfaceType = jsii.String("efa")
					} else {
						// 其他接口使用 "efa-only"（如果支持的话，否则使用 "efa"）
						networkInterface.InterfaceType = jsii.String("efa-only")
					}
				}

				// 如果启用EnaSrd，所有网卡都启用EnaSrd
				if types.GetBoolValue(ec2Instance.EnaSrdEnabled, false) {
					networkInterface.EnaSrdSpecification = &awsec2.CfnLaunchTemplate_EnaSrdSpecificationProperty{
						EnaSrdEnabled: jsii.Bool(true),
						EnaSrdUdpSpecification: &awsec2.CfnLaunchTemplate_EnaSrdUdpSpecificationProperty{
							EnaSrdUdpEnabled: jsii.Bool(true),
						},
					}
				}

				networkInterfaces[i] = networkInterface
			}

			launchTemplateData.NetworkInterfaces = networkInterfaces
		}

		// 如果需要配置高EBS吞吐量
		if ec2Instance.EbsThroughput > 125 {
			// 创建仅包含Throughput设置的EBS配置
			ebsOverride := &awsec2.CfnLaunchTemplate_EbsProperty{
				Throughput: jsii.Number(ec2Instance.EbsThroughput),
			}

			// 创建块设备映射，只覆盖需要的属性
			blockDeviceMapping := &awsec2.CfnLaunchTemplate_BlockDeviceMappingProperty{
				DeviceName: jsii.String(deviceName),
				Ebs: ebsOverride,
			}

			launchTemplateData.BlockDeviceMappings = []interface{}{blockDeviceMapping}
		}

		// 创建启动模板
		launchTemplate := awsec2.NewCfnLaunchTemplate(stack, jsii.String(ec2Instance.GetID()+"LaunchTemplate"), &awsec2.CfnLaunchTemplateProps{
			LaunchTemplateName: jsii.String(ec2Instance.GetID() + "-launch-template"),
			LaunchTemplateData: launchTemplateData,
		})

		// 使用AddPropertyOverride设置启动模板属性
		cfnInstance.AddPropertyOverride(jsii.String("LaunchTemplate"), map[string]interface{}{
			"LaunchTemplateId": launchTemplate.Ref(),
			"Version": launchTemplate.AttrLatestVersionNumber(), // 使用启动模板的最新版本号属性
		})
	}

	// 简化：移除复杂的策略比较逻辑，策略已在 InstanceProfile 中处理

	if partition.DefaultManagedPolicy != nil {
		inst.Role().AddManagedPolicy(partition.DefaultManagedPolicy)
	}

	return inst
}

func (e *Ec2Forge) CreateOutputs(ctx *interfaces.ForgeContext) {
	ec2Instance, ok := (*ctx.Instance).(*Ec2InstanceConfig)
	if !ok {
		return
	}

	// 收集所有实例ID
	var instanceIds []string
	for _, inst := range e.ec2Instances {
		instanceIds = append(instanceIds, *inst.InstanceId())
	}

	// 将实例ID列表转换为逗号分隔的字符串
	idList := strings.Join(instanceIds, ",")

	// 创建单个输出，包含所有实例ID
	awscdk.NewCfnOutput(ctx.Stack, jsii.String("ElasticCloudCompute"+aws.GetOriginalID(ec2Instance.GetID())), &awscdk.CfnOutputProps{
		Value:       jsii.String(idList),
		Description: jsii.String("List of all Elastic Cloud Compute IDs"),
	})
}


func (e *Ec2Forge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// 检查是否有自定义端口配置
	if ctx.Instance != nil {
		ec2Instance, ok := (*ctx.Instance).(*Ec2InstanceConfig)
		if ok && (ec2Instance.AllowedPorts != "" || ec2Instance.AllowedPortsIpv6 != "") {
			// 使用通用端口规则处理函数
			security.ApplyPortRules(ctx.SecurityGroups.Public, ec2Instance.AllowedPorts, ec2Instance.AllowedPortsIpv6, ctx.DualStack)
			
			// 如果启用了EFA，配置EFA规则
			if types.GetBoolValue(ec2Instance.EnableEfa, false) {
				security.ConfigureEFASecurityRules(ctx.SecurityGroups.Default, "ec2-efa")
				if ctx.SecurityGroups.Default != ctx.SecurityGroups.Private {
					security.ConfigureEFASecurityRules(ctx.SecurityGroups.Private, "ec2-efa-private")
				}
			}
			return // 使用自定义配置，跳过所有默认规则
		}
	}
}

func (e *Ec2Forge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
	// 从默认配置中复制基本字段
	merged := defaults.(*Ec2InstanceConfig)

	// 从实例配置中覆盖基本字段
	ec2Instance := instance.(*Ec2InstanceConfig)
	if instance != nil {
		if ec2Instance.GetID() != "" {
			merged.ID = ec2Instance.GetID()
		}
		if ec2Instance.Type != "" {
			merged.Type = ec2Instance.GetType()
		}
		if ec2Instance.Subnet != "" {
			merged.Subnet = ec2Instance.GetSubnet()
		}
		if ec2Instance.SecurityGroup != "" {
			merged.SecurityGroup = ec2Instance.GetSecurityGroup()
		}
	}

	// Merge EC2 自己的字段

	if ec2Instance.AzIndex != 0 {
		merged.AzIndex = ec2Instance.AzIndex
	}
	if ec2Instance.InstanceCount != 0 {
		merged.InstanceCount = ec2Instance.InstanceCount
	}
	if ec2Instance.Debug != nil {
		merged.Debug = ec2Instance.Debug
	}
	// 移除所有 Shared 字段的合并逻辑，因为已经简化为单一字段
	if ec2Instance.DetailedMonitoring != nil {
		merged.DetailedMonitoring = ec2Instance.DetailedMonitoring
	}
	if ec2Instance.DependsOn != "" {
		merged.DependsOn = ec2Instance.DependsOn
	}
	if ec2Instance.EbsDeviceName != "" {
		merged.EbsDeviceName = ec2Instance.EbsDeviceName
	}
	if ec2Instance.EbsIops != 0 {
		merged.EbsIops = ec2Instance.EbsIops
	}
	if ec2Instance.EbsSize != 0 {
		merged.EbsSize = ec2Instance.EbsSize
	}
	if ec2Instance.EbsThroughput != 0 {
		merged.EbsThroughput = ec2Instance.EbsThroughput
	}
	if ec2Instance.EbsVolumeType != "" {
		merged.EbsVolumeType = ec2Instance.EbsVolumeType
	}
	if ec2Instance.EbsOptimized != nil {
		merged.EbsOptimized = ec2Instance.EbsOptimized
	}
	if ec2Instance.EnclaveEnabled != nil {
		merged.EnclaveEnabled = ec2Instance.EnclaveEnabled
	}
	if ec2Instance.EnableEfa != nil {
		merged.EnableEfa = ec2Instance.EnableEfa
	}
	if ec2Instance.EnaSrdEnabled != nil {
		merged.EnaSrdEnabled = ec2Instance.EnaSrdEnabled
	}
	if ec2Instance.NetworkCardCount != 0 {
		merged.NetworkCardCount = ec2Instance.NetworkCardCount
	}
	if ec2Instance.PurchaseOption != "" {
		merged.PurchaseOption = ec2Instance.PurchaseOption
	}
	if ec2Instance.SpotMaxPrice != "" {
		merged.SpotMaxPrice = ec2Instance.SpotMaxPrice
	}
	if ec2Instance.CapacityBlockId != "" {
		merged.CapacityBlockId = ec2Instance.CapacityBlockId
	}
	if ec2Instance.InstanceType != "" {
		merged.InstanceType = ec2Instance.InstanceType
	}
	if ec2Instance.KeyName != "" {
		merged.KeyName = ec2Instance.KeyName
	}
	if ec2Instance.OsImage != "" {
		merged.OsImage = ec2Instance.OsImage
	}
	if ec2Instance.OsName != "" {
		merged.OsName = ec2Instance.OsName
	}
	if ec2Instance.OsArch!= "" {
		merged.OsArch = ec2Instance.OsArch
	}
	if ec2Instance.OsType != "" {
		merged.OsType = ec2Instance.OsType
	}
	if ec2Instance.OsVersion != "" {
		merged.OsVersion = ec2Instance.OsVersion
	}
	if ec2Instance.PlacementGroup != "" {
		merged.PlacementGroup = ec2Instance.PlacementGroup 
	}
	if ec2Instance.PlacementGroupStrategy != "" {
		merged.PlacementGroupStrategy = ec2Instance.PlacementGroupStrategy 
	}
	if ec2Instance.Policies != "" {
		merged.Policies = ec2Instance.Policies
	}
	if ec2Instance.InstanceType != "" {
		merged.InstanceType = ec2Instance.InstanceType
	}
	if ec2Instance.KeyName != "" {
		merged.KeyName = ec2Instance.KeyName
	}
	if ec2Instance.OsImage != "" {
		merged.OsImage = ec2Instance.OsImage
	}
	if ec2Instance.OsName != "" {
		merged.OsName = ec2Instance.OsName
	}
	if ec2Instance.OsArch!= "" {
		merged.OsArch = ec2Instance.OsArch
	}
	if ec2Instance.OsType != "" {
		merged.OsType = ec2Instance.OsType
	}
	if ec2Instance.OsVersion != "" {
		merged.OsVersion = ec2Instance.OsVersion
	}
	if ec2Instance.PlacementGroup != "" {
		merged.PlacementGroup = ec2Instance.PlacementGroup 
	}
	if ec2Instance.PlacementGroupStrategy != "" {
		merged.PlacementGroupStrategy = ec2Instance.PlacementGroupStrategy 
	}
	if ec2Instance.Policies != "" {
		merged.Policies = ec2Instance.Policies
	}
	if ec2Instance.S3Location != "" {
		merged.S3Location = ec2Instance.S3Location
	}
	if ec2Instance.StoreInstanceInfo != nil {
		merged.StoreInstanceInfo = ec2Instance.StoreInstanceInfo
	}
	if ec2Instance.RequireImdsv2 != nil {
		merged.RequireImdsv2 = ec2Instance.RequireImdsv2
	}
	if ec2Instance.UserDataToken != "" {
		merged.UserDataToken = ec2Instance.UserDataToken
	}
	if ec2Instance.UserDataScriptPath != "" {
		merged.UserDataScriptPath = ec2Instance.UserDataScriptPath
	}

	return merged
}
func (e *Ec2Forge) GetProperties() map[string]interface{} {
	return e.properties
}
