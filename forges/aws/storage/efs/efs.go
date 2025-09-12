// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package efs

import (
	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/security"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsefs"
	"github.com/aws/jsii-runtime-go"
)

type EfsInstanceConfig struct {
	config.BaseInstanceConfig
	RemovePolicy string `json:"removePolicy,omitempty"`
}

type EfsForge struct {
	efs        awsefs.FileSystem
	properties map[string]interface{}
}

func (e *EfsForge) Create(ctx *interfaces.ForgeContext) interface{} {
	efsInstance, ok := (*ctx.Instance).(*EfsInstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return nil
	}

	fileSystem := awsefs.NewFileSystem(ctx.Stack, jsii.String(efsInstance.GetID()), &awsefs.FileSystemProps{
		Vpc:                     ctx.VPC,
		RemovalPolicy:           awscdk.RemovalPolicy_DESTROY,
		SecurityGroup:           ctx.SecurityGroups.Default,
		VpcSubnets:              &awsec2.SubnetSelection{SubnetType: ctx.SubnetType},
		AllowAnonymousAccess:    jsii.Bool(true),
		PerformanceMode:         awsefs.PerformanceMode_GENERAL_PURPOSE,
		ThroughputMode:          awsefs.ThroughputMode_BURSTING,
		Encrypted:               jsii.Bool(true),
		FileSystemName:          jsii.String("AWS-Infra-Elastic-FileSystem"),
        })

	e.efs = fileSystem
	
	// 保存 EFS 属性
	if e.properties == nil {
		e.properties = make(map[string]interface{})
	}
	e.properties["fileSystemId"] = fileSystem.FileSystemId()
	e.properties["fileSystemArn"] = fileSystem.FileSystemArn()
	e.properties["mountPoint"] = "/" + efsInstance.GetID()  // 挂载点

        return e
}

func (e *EfsForge) CreateOutputs(ctx *interfaces.ForgeContext) {
	efsInstance, ok := (*ctx.Instance).(*EfsInstanceConfig)
	if !ok {
		// 处理类型断言失败的情况
		return
	}
	awscdk.NewCfnOutput(ctx.Stack, jsii.String("ElasticFileSystem"+efsInstance.GetID()), &awscdk.CfnOutputProps{
		Value:       e.efs.FileSystemId(),
		Description: jsii.String("Elastic File System ID"),
	})
}

func (e *EfsForge) ConfigureRules(ctx *interfaces.ForgeContext) {
	// 为 EFS 配置特定的入站规则
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Public, 2049, "Allow EFS access from public subnet")
	security.AddTcpIngressRule(ctx.SecurityGroups.Default, ctx.SecurityGroups.Private, 2049, "Allow EFS access from private subnet")

	//ctx.SecurityGroups.Default.AddIngressRule(ctx.SecurityGroups.Public, awsec2.Port_Tcp(jsii.Number(2049)), jsii.String("Allow EFS access from public subnet"), nil)
	//ctx.SecurityGroups.Default.AddIngressRule(ctx.SecurityGroups.Private, awsec2.Port_Tcp(jsii.Number(2049)), jsii.String("Allow EFS access from private subnet"), nil)

}

func (e *EfsForge) MergeConfigs(defaults config.InstanceConfig, instance config.InstanceConfig) config.InstanceConfig {
	// 从默认配置中复制基本字段
	merged := defaults.(*EfsInstanceConfig)

	// 从实例配置中覆盖基本字段
        efsInstance := instance.(*EfsInstanceConfig)
	if instance != nil {
		if efsInstance.GetID() != "" {
			merged.ID = efsInstance.GetID()
		}
		if efsInstance.Type != "" {
			merged.Type = efsInstance.GetType()
		}
		if efsInstance.Subnet != "" {
			merged.Subnet = efsInstance.GetSubnet()
		}
		if efsInstance.SecurityGroup != "" {
			merged.SecurityGroup = efsInstance.GetSecurityGroup()
		}
	}

	// 处理 RemovePolicy 字段
	if efsInstance.RemovePolicy != "" {
		merged.RemovePolicy = efsInstance.RemovePolicy
	}

	if merged.RemovePolicy == "" {
		merged.RemovePolicy = "RETAIN"
	}

	return merged
}

func (e *EfsForge) GetProperties() map[string]interface{} {
	return e.properties
}
