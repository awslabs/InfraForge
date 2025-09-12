// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"strings"

	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

// AddManagedPolicies 添加多个 AWS 托管策略到角色
// policyNames 支持单个策略名或用逗号分隔的多个策略名
func AddManagedPolicies(role awsiam.IRole, policyNames string) {
	// 分割策略名字符串
	policies := strings.Split(policyNames, ",")

	// 遍历并添加每个策略
	for _, policyName := range policies {
		// 去除空白字符
		policyName = strings.TrimSpace(policyName)
		if policyName != "" {
			role.AddManagedPolicy(awsiam.ManagedPolicy_FromAwsManagedPolicyName(jsii.String(policyName)))
		}
	}
}

// CreateRole 创建带有指定策略的角色
func CreateRole(scope constructs.Construct, roleId string, policyNames string, servicePrincipal string) awsiam.IRole {
	role := awsiam.NewRole(scope, jsii.String(roleId), &awsiam.RoleProps{
		AssumedBy: awsiam.NewServicePrincipal(jsii.String(servicePrincipal+".amazonaws.com"), nil),
	})
	
	if policyNames != "" {
		AddManagedPolicies(role, policyNames)
	}
	
	return role
}

