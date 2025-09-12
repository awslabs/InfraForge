// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package interfaces

import (
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/awslabs/InfraForge/core/config"
)

// ForgeContext 包含 Forge 操作所需的所有上下文信息
type ForgeContext struct {
	Stack          awscdk.Stack
	Instance       *config.InstanceConfig
	VPC            awsec2.IVpc
	SubnetType     awsec2.SubnetType
	Dependencies   map[string]interface{}
	DualStack      bool
	SecurityGroups *SecurityGroups
}

// SecurityGroups 包含所有安全组
type SecurityGroups struct {
	Default  awsec2.SecurityGroup
	Public   awsec2.SecurityGroup
	Private  awsec2.SecurityGroup
	Isolated awsec2.SecurityGroup
}

type Forge interface {
	Create(ctx *ForgeContext) interface{}
	ConfigureRules(ctx *ForgeContext)
	MergeConfigs(defaults, instance config.InstanceConfig) config.InstanceConfig
	CreateOutputs(ctx *ForgeContext)
}
