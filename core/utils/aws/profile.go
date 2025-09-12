// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/awslabs/InfraForge/core/partition"
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/jsii-runtime-go"
)

// 单例缓存，确保相同策略只创建一次
var instanceProfileCache = make(map[string]awsiam.IInstanceProfile)
var instanceProfileMutex = sync.Mutex{}

// 检查 InstanceProfile 是否在 AWS 中存在
func instanceProfileExistsInAWS(profileName string) bool {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return false
	}
	
	iamClient := iam.NewFromConfig(cfg)
	_, err = iamClient.GetInstanceProfile(context.TODO(), &iam.GetInstanceProfileInput{
		InstanceProfileName: &profileName,
	})
	
	return err == nil
}

// 单例缓存的 Get or Create InstanceProfile
func CreateOrGetInstanceProfile(stack awscdk.Stack, policyString string) awsiam.IInstanceProfile {
	if policyString == "" {
		return nil
	}
	
	sortedPolicies := SortPolicies(policyString)
	policyHash := types.HashString(sortedPolicies)
	
	instanceProfileMutex.Lock()
	defer instanceProfileMutex.Unlock()
	
	if cachedProfile, exists := instanceProfileCache[policyHash]; exists {
		return cachedProfile
	}
	
	profileName := fmt.Sprintf("%s-InstanceProfile-%s-%s", *awscdk.Aws_STACK_NAME(), partition.DefaultRegion, policyHash)
	constructId := fmt.Sprintf("InstanceProfile-%s", policyHash)
	
	var instanceProfile awsiam.IInstanceProfile
	
	if instanceProfileExistsInAWS(profileName) {
		instanceProfile = awsiam.InstanceProfile_FromInstanceProfileName(stack, jsii.String(constructId), jsii.String(profileName))
	} else {
		roleName := fmt.Sprintf("%s-InstanceRole-%s-%s", *awscdk.Aws_STACK_NAME(), partition.DefaultRegion, policyHash)
		roleConstructId := fmt.Sprintf("Role-%s", policyHash)
		
		role := awsiam.NewRole(stack, jsii.String(roleConstructId), &awsiam.RoleProps{
			AssumedBy: awsiam.NewServicePrincipal(jsii.String("ec2.amazonaws.com"), nil),
			RoleName:  jsii.String(roleName),
		})
		
		AddManagedPolicies(role, sortedPolicies)
		
		instanceProfile = awsiam.NewInstanceProfile(stack, jsii.String(constructId), &awsiam.InstanceProfileProps{
			InstanceProfileName: jsii.String(profileName),
			Role:                role,
		})
	}
	
	instanceProfileCache[policyHash] = instanceProfile
	return instanceProfile
}

// SortPolicies 对策略名称进行排序，用于创建唯一的缓存键
func SortPolicies(policyString string) string {
	if policyString == "" {
		return ""
	}
	
	policies := strings.Split(policyString, ",")
	for i, policy := range policies {
		policies[i] = strings.TrimSpace(policy)
	}

	// 简单的冒泡排序
	n := len(policies)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if policies[j] > policies[j+1] {
				policies[j], policies[j+1] = policies[j+1], policies[j]
			}
		}
	}
	
	return strings.Join(policies, ",")
}
