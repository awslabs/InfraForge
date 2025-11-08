// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package manager

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/dependency"
	"github.com/awslabs/InfraForge/core/partition"
	"github.com/awslabs/InfraForge/core/utils/aws"
	"github.com/awslabs/InfraForge/forges/aws/ec2"
	"github.com/awslabs/InfraForge/forges/aws/eks"
	"github.com/awslabs/InfraForge/forges/aws/iam"
	"github.com/awslabs/InfraForge/forges/aws/parallelcluster"
	"github.com/awslabs/InfraForge/forges/aws/vpc"
	"github.com/awslabs/InfraForge/registry"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
)

type ForgeManager struct {
	stack         awscdk.Stack
	vpc           awsec2.IVpc
	securityGroups *interfaces.SecurityGroups
	subnetTypeMap map[string]awsec2.SubnetType
	dualStack     bool
}

func NewForgeManager(stack awscdk.Stack, dualStack bool) *ForgeManager {
	// 初始化默认 IAM 策略
	partition.DefaultManagedPolicy = iam.CreateDCVLicensingPolicy(stack)
	iam.CreateDCVOutputs(stack, partition.DefaultManagedPolicy)

	return &ForgeManager{
		stack:     stack,
		dualStack: dualStack,
		subnetTypeMap: map[string]awsec2.SubnetType{
			"public":    awsec2.SubnetType_PUBLIC,
			"private":   awsec2.SubnetType_PRIVATE_WITH_EGRESS,
			"isolated":  awsec2.SubnetType_PRIVATE_ISOLATED,
		},
	}
}

func (fm *ForgeManager) CreateVPC(infraConfig *config.Config) error {
	vpcForge := &vpc.VpcForge{}
	
	// 创建 VPC 实例配置
	vpcInst := registry.CreateInstance("vpc")
	
	// 检查是否有 VPC instances，如果没有则使用 defaults
	var vpcConfig json.RawMessage
	if len(infraConfig.Forges["vpc"].Instances) > 0 {
		vpcConfig = infraConfig.Forges["vpc"].Instances[0]
	} else {
		vpcConfig = infraConfig.Forges["vpc"].Defaults
	}
	
	if err := json.Unmarshal(vpcConfig, vpcInst); err != nil {
		return fmt.Errorf("error parsing VPC config: %v", err)
	}
	
	// 创建 VPC 上下文
	vpcCtx := &interfaces.ForgeContext{
		Stack:     fm.stack,
		Instance:  &vpcInst,
		DualStack: fm.dualStack,
	}

	// 创建 VPC
	ivpc := vpcForge.Create(vpcCtx)
	vpcForgeResult := ivpc.(*vpc.VpcForge)
	fm.vpc = vpcForgeResult.GetVpc()

	// 创建安全组
	publicSG, privateSG, isolatedSG := vpc.CreateSecurityGroups(fm.stack, fm.vpc, fm.dualStack)
	
	fm.securityGroups = &interfaces.SecurityGroups{
		Default:  privateSG,
		Public:   publicSG,
		Private:  privateSG,
		Isolated: isolatedSG,
	}

	// 更新 VPC 上下文并配置规则
	vpcCtx.SecurityGroups = fm.securityGroups
	vpcForge.ConfigureRules(vpcCtx)
	vpcForge.CreateOutputs(vpcCtx)

	return nil
}

func (fm *ForgeManager) CreateForge(instanceId string, infraConfig *config.Config) error {
	for typ, forgeConfig := range infraConfig.Forges {
		for _, rawInst := range forgeConfig.Instances {
			inst := registry.CreateInstance(typ)
			if err := json.Unmarshal(rawInst, inst); err != nil {
				return fmt.Errorf("error parsing config file: %v", err)
			}

			if inst.GetID() == instanceId {
				return fm.processForge(typ, forgeConfig.Defaults, inst)
			}
		}
	}
	return fmt.Errorf("forge instance %s not found", instanceId)
}

func (fm *ForgeManager) processForge(typ string, rawDefaults json.RawMessage, inst config.InstanceConfig) error {
	constructor, ok := registry.ForgeConstructors[typ]
	if !ok {
		return fmt.Errorf("unknown forge type: %s", typ)
	}
	forge := constructor()

	defaults := registry.CreateInstance(typ)
	if err := json.Unmarshal(rawDefaults, defaults); err != nil {
		return fmt.Errorf("error parsing defaults: %v", err)
	}

	merged := forge.MergeConfigs(defaults, inst)

	// 按需创建共享资源（简化版）
	fm.createSharedResourcesForInstance(merged)

	// 创建 ForgeContext
	ctx := fm.createForgeContext(merged)

	// 执行 forge 操作
	iforge := forge.Create(ctx)
	if iforge == nil {
		return fmt.Errorf("failed to create forge for %s", merged.GetID())
	}
	
	// 存储依赖，使用 "type:id" 格式
	trimIndex := false
	if ec2Inst, ok := merged.(*ec2.Ec2InstanceConfig); ok {
		trimIndex = ec2Inst.InstanceCount > 1
	}
	
	// 使用 "type:id" 格式存储，将 forge key 转换为大写以匹配依赖格式
	// 例如：forge key "lustre" -> "LUSTRE:fsx" 匹配依赖中的 "LUSTRE:fsx"
	var storeKey string
	if trimIndex {
		storeKey = fmt.Sprintf("%s:%s", strings.ToUpper(typ), aws.GetOriginalID(merged.GetID()))
	} else {
		storeKey = fmt.Sprintf("%s:%s", strings.ToUpper(typ), merged.GetID())
	}
	dependency.GlobalManager.Store(storeKey, iforge)
	
	forge.ConfigureRules(ctx)
	forge.CreateOutputs(ctx)

	return nil
}

// createSharedResourcesForInstance 为单个实例创建所需的共享资源
func (fm *ForgeManager) createSharedResourcesForInstance(instance config.InstanceConfig) {
	switch inst := instance.(type) {
	case *ec2.Ec2InstanceConfig:
		keyName := inst.KeyName
		if keyName == "" {
			keyName = *awscdk.Aws_STACK_NAME()
		}
		aws.CreateOrGetKeyPair(fm.stack, keyName, inst.OsType)
		
		if inst.PlacementGroup != "" {
			aws.CreateOrGetPlacementGroup(fm.stack, inst.PlacementGroup, inst.PlacementGroupStrategy)
		}
		if inst.Policies != "" {
			aws.CreateOrGetInstanceProfile(fm.stack, inst.Policies)
		}
		
	case *eks.EksInstanceConfig:
		keyName := inst.KeyName
		if keyName == "" {
			keyName = *awscdk.Aws_STACK_NAME()
		}
		aws.CreateOrGetKeyPair(fm.stack, keyName, inst.OsType)
		
	case *parallelcluster.ParallelClusterInstanceConfig:
		keyName := inst.KeyName
		if keyName == "" {
			keyName = *awscdk.Aws_STACK_NAME()
		}
		aws.CreateOrGetKeyPair(fm.stack, keyName, "Linux")
	}
}

func (fm *ForgeManager) createForgeContext(merged config.InstanceConfig) *interfaces.ForgeContext {
	subnetType := fm.getSubnetType(merged.GetSubnet())
	
	// 根据实例配置动态选择默认安全组
	var defaultSG awsec2.SecurityGroup
	switch merged.GetSecurityGroup() {
	case "public":
		defaultSG = fm.securityGroups.Public
	case "isolated":
		defaultSG = fm.securityGroups.Isolated
	default:
		defaultSG = fm.securityGroups.Private
	}

	// 为这个特定的 forge 创建定制的 SecurityGroups
	forgeSecurityGroups := &interfaces.SecurityGroups{
		Default:  defaultSG,  // 根据配置动态设置
		Public:   fm.securityGroups.Public,
		Private:  fm.securityGroups.Private,
		Isolated: fm.securityGroups.Isolated,
	}

	dependencies := map[string]interface{}{
		"public":    fm.securityGroups.Public,
		"private":   fm.securityGroups.Private,
		"isolated":  fm.securityGroups.Isolated,
	}

	return &interfaces.ForgeContext{
		Stack:          fm.stack,
		Instance:       &merged,
		VPC:            fm.vpc,
		SubnetType:     subnetType,
		SecurityGroups: forgeSecurityGroups,  // 使用动态设置的安全组
		Dependencies:   dependencies,
		DualStack:      fm.dualStack,
	}
}

func (fm *ForgeManager) getSecurityGroup(sgType string) awsec2.SecurityGroup {
	switch sgType {
	case "public":
		return fm.securityGroups.Public
	case "private":
		return fm.securityGroups.Private
	case "isolated":
		return fm.securityGroups.Isolated
	default:
		return fm.securityGroups.Private
	}
}

func (fm *ForgeManager) getSubnetType(subnet string) awsec2.SubnetType {
	if subnetType, ok := fm.subnetTypeMap[subnet]; ok {
		return subnetType
	}
	return awsec2.SubnetType_PRIVATE_WITH_EGRESS
}
