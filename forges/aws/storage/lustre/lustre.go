// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lustre 

import (
	"strings"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/security"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/forges/aws/storage/utils"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsfsx"
	"github.com/aws/jsii-runtime-go"
)

type LustreInstanceConfig struct {
	config.BaseInstanceConfig
	AzIndex                  int      `json:"azIndex,omitempty"`
	DataCompressionType      string   `json:"dataCompressionType,omitempty"`
	DeploymentType           string   `json:"deploymentType,omitempty"`
	FileSystemVersion        string   `json:"fileSystemVersion,omitempty"`
	PerUnitStorageThroughput float64  `json:"perUnitStorageThroughput,omitempty"`
	RemovalPolicy            string   `json:"removalPolicy,omitempty"`
	StorageCapacityGiB       int      `json:"size,omitempty"`
}

type LustreForge struct {
        lustre     awsfsx.LustreFileSystem
        properties map[string]interface{}
}

func (l *LustreForge) Create(ctx *interfaces.ForgeContext) interface{} {
	lustreInstance, ok := (*ctx.Instance).(*LustreInstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return nil
	}
	fileSystemName := "FsxLustreFileSystem-"+lustreInstance.GetID()

	var perUnitStorageThroughput *float64

	deploymentType := parseDeploymentType(lustreInstance.DeploymentType)
	if deploymentType ==  awsfsx.LustreDeploymentType_PERSISTENT_1 || deploymentType == awsfsx.LustreDeploymentType_PERSISTENT_2 {
		perUnitStorageThroughput = &lustreInstance.PerUnitStorageThroughput
	} else {
		perUnitStorageThroughput = nil
	}

        lustreConfiguration := &awsfsx.LustreConfiguration{
                DeploymentType: deploymentType,
		DataCompressionType: parseDataCompressionType(lustreInstance.DataCompressionType),
		PerUnitStorageThroughput: perUnitStorageThroughput,
        }

	// 使用统一的子网选择函数
	selectedSubnet := aws.SelectSubnetByAzIndex(lustreInstance.AzIndex, ctx.VPC, ctx.SubnetType)

	fileSystemVersion := utils.ParseLustreVersion(lustreInstance.FileSystemVersion)

	fileSystem := awsfsx.NewLustreFileSystem(ctx.Stack, jsii.String(fileSystemName), &awsfsx.LustreFileSystemProps{
		LustreConfiguration:   lustreConfiguration,
		StorageCapacityGiB:    jsii.Number(lustreInstance.StorageCapacityGiB),
		Vpc:                   ctx.VPC,
		VpcSubnet:             selectedSubnet,
		SecurityGroup:         ctx.SecurityGroups.Default,
		FileSystemTypeVersion: fileSystemVersion,
                RemovalPolicy:         parseRemovalPolicy(lustreInstance.RemovalPolicy),
        })

	l.lustre = fileSystem
	
	// 保存 Lustre 属性
	if l.properties == nil {
		l.properties = make(map[string]interface{})
	}
	l.properties["fileSystemId"] = fileSystem.FileSystemId()
	l.properties["dnsName"] = fileSystem.DnsName()
	l.properties["mountName"] = fileSystem.MountName()
	l.properties["mountPoint"] = "/" + lustreInstance.GetID()  // 挂载点
	l.properties["storageCapacityGiB"] = lustreInstance.StorageCapacityGiB

        return l
}

func (l *LustreForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	lustreInstance, ok := (*ctx.Instance).(*LustreInstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return
	}

	awscdk.NewCfnOutput(ctx.Stack, jsii.String("LustreFileSystem"+lustreInstance.GetID()), &awscdk.CfnOutputProps{
		Value:       l.lustre.FileSystemId(),
		Description: jsii.String("Lustre File System ID"),
	})
}

// 获取 Lustre 文件系统
func (l *LustreForge) GetLustreFileSystem() awsfsx.LustreFileSystem {
    return l.lustre
}

// 获取存储容量
func (l *LustreForge) GetStorageCapacityGiB() int {
    if capacity, ok := l.properties["storageCapacityGiB"].(int); ok {
        return capacity
    }
    return 0
}

func (l *LustreForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// 为 Lustre 配置特定的入站规则
	// 打开 Lustre 端口 988, 1018-1023
	// 使用安全的方法添加规则，避免重复
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Public, 988, "Allow Lustre access from public subnet")
	security.AddTcpRangeIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Public, 1018, 1023, "Allow Lustre access from public subnet")
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Private, 988, "Allow Lustre access from private subnet")
	security.AddTcpRangeIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Private, 1018, 1023, "Allow Lustre access from private subnet")

	// 为 Lustre 配置特定的入站规则
	// 打开 Lustre 端口 988, 1018-1023
        //defaultSG.AddIngressRule(publicSG, awsec2.Port_TcpRange(jsii.Number(988), jsii.Number(988)), jsii.String("Allow Lustre access from public subnet"), nil)
        //defaultSG.AddIngressRule(publicSG, awsec2.Port_TcpRange(jsii.Number(1018), jsii.Number(1023)), jsii.String("Allow Lustre access from public subnet"), nil)
        //defaultSG.AddIngressRule(privateSG, awsec2.Port_TcpRange(jsii.Number(988), jsii.Number(988)), jsii.String("Allow Lustre access from private subnet"), nil)
        //defaultSG.AddIngressRule(privateSG, awsec2.Port_TcpRange(jsii.Number(1018), jsii.Number(1023)), jsii.String("Allow Lustre access from private subnet"), nil)

}

func (l *LustreForge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
	// 从默认配置中复制基本字段
	merged := defaults.(*LustreInstanceConfig)

	// 从实例配置中覆盖基本字段
        lustreInstance := instance.(*LustreInstanceConfig)
	if instance != nil {
		if lustreInstance.GetID() != "" {
			merged.ID = lustreInstance.GetID()
		}
		if lustreInstance.Type != "" {
			merged.Type = lustreInstance.GetType()
		}
		if lustreInstance.Subnet != "" {
			merged.Subnet = lustreInstance.GetSubnet()
		}
		if lustreInstance.SecurityGroup != "" {
			merged.SecurityGroup = lustreInstance.GetSecurityGroup()
		}
	}

	if lustreInstance.StorageCapacityGiB > 0 {
		merged.StorageCapacityGiB = lustreInstance.StorageCapacityGiB
	} else { 
		merged.StorageCapacityGiB = 1200
	}
	if lustreInstance.DataCompressionType != "" {
		merged.DataCompressionType = lustreInstance.DataCompressionType
	}
	if lustreInstance.DeploymentType != "" {
		merged.DeploymentType = lustreInstance.DeploymentType
	}
	if lustreInstance.RemovalPolicy != "" {
		merged.RemovalPolicy = lustreInstance.RemovalPolicy
	}
	if lustreInstance.FileSystemVersion != "" {
		merged.FileSystemVersion = lustreInstance.FileSystemVersion
	}
	if lustreInstance.PerUnitStorageThroughput > 0 { 
		merged.PerUnitStorageThroughput = lustreInstance.PerUnitStorageThroughput
	}

	return merged
}

func parseDeploymentType(input string) awsfsx.LustreDeploymentType {
	// 将输入转换为小写
	lowered := strings.ToLower(input)

	// 移除所有非字母数字字符
	cleaned := strings.ReplaceAll(lowered, "_", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	// 将已知的变体转换为目标格式
	switch cleaned {
	case "scratch1": 
		return awsfsx.LustreDeploymentType_SCRATCH_1
	case "scratch2":
		return awsfsx.LustreDeploymentType_SCRATCH_2
	case "persistent1":
		return awsfsx.LustreDeploymentType_PERSISTENT_1
	case "persistent2":
		return awsfsx.LustreDeploymentType_PERSISTENT_2
	default:
		return awsfsx.LustreDeploymentType_SCRATCH_2
	}
}

func parseDataCompressionType(input string) awsfsx.LustreDataCompressionType {
	// 将输入转换为小写
	lowered := strings.ToLower(input)

	// 移除所有非字母数字字符
	cleaned := strings.ReplaceAll(lowered, "_", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	// 将已知的变体转换为目标格式
	switch cleaned {
	case "none": 
		return awsfsx.LustreDataCompressionType_NONE
	case "lz4":
		return awsfsx.LustreDataCompressionType_LZ4
	default:
		return awsfsx.LustreDataCompressionType_NONE
	}
}

func parseRemovalPolicy(input string) awscdk.RemovalPolicy {
	// 将输入转换为小写
	lowered := strings.ToLower(input)

	// 移除所有非字母数字字符
	cleaned := strings.ReplaceAll(lowered, "_", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	// 将已知的变体转换为目标格式
	switch cleaned {
	case "destroy": 
		return awscdk.RemovalPolicy_DESTROY
	case "retain": 
		return awscdk.RemovalPolicy_RETAIN
	default:
		return awscdk.RemovalPolicy_DESTROY
	}
}
func (l *LustreForge) GetProperties() map[string]interface{} {
	return l.properties
}
