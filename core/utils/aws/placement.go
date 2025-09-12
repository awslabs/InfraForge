// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/aws/jsii-runtime-go"
)

// 单例缓存，确保相同 PlacementGroup 只创建一次
var placementGroupCache = make(map[string]awsec2.IPlacementGroup)
var placementGroupMutex = sync.Mutex{}

// 检查 PlacementGroup 是否在 AWS 中存在
func placementGroupExistsInAWS(pgName string) bool {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return false
	}

	ec2Client := ec2.NewFromConfig(cfg)
	_, err = ec2Client.DescribePlacementGroups(context.TODO(), &ec2.DescribePlacementGroupsInput{
		GroupNames: []string{pgName},
	})

	return err == nil
}

// 单例缓存的 Get or Create PlacementGroup
func CreateOrGetPlacementGroup(stack awscdk.Stack, pgName string, strategy string) awsec2.IPlacementGroup {
	if pgName == "" {
		return nil
	}
	
	fullPgName := fmt.Sprintf("%s-%s", *awscdk.Aws_STACK_NAME(), pgName)
	pgKey := fmt.Sprintf("%s-%s", fullPgName, strategy)
	pgHash := types.HashString(pgKey)
	
	placementGroupMutex.Lock()
	defer placementGroupMutex.Unlock()
	
	if cachedPG, exists := placementGroupCache[pgHash]; exists {
		return cachedPG
	}
	
	constructId := fmt.Sprintf("PlacementGroup-%s", pgHash)
	var placementGroup awsec2.IPlacementGroup
	
	if placementGroupExistsInAWS(fullPgName) {
		placementGroup = awsec2.PlacementGroup_FromPlacementGroupName(stack, jsii.String(constructId), jsii.String(fullPgName))
	} else {
		placementGroup = awsec2.NewPlacementGroup(stack, jsii.String(constructId), &awsec2.PlacementGroupProps{
			PlacementGroupName: jsii.String(fullPgName),
			Strategy:          awsec2.PlacementGroupStrategy(strategy),
		})
	}
	
	placementGroupCache[pgHash] = placementGroup
	return placementGroup
}
