// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"fmt"
	"sync"

	"github.com/awslabs/InfraForge/core/partition"
	"github.com/awslabs/InfraForge/core/utils/types"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/jsii-runtime-go"
)

// 单例缓存，确保相同 KeyPair 只创建一次
var keyPairCache = make(map[string]awsec2.IKeyPair)
var keyPairMutex = sync.Mutex{}

// 检查 KeyPair 是否在 AWS 中存在
func keyPairExistsInAWS(keyPairName string) bool {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return false
	}
	
	ec2Client := ec2.NewFromConfig(cfg)
	_, err = ec2Client.DescribeKeyPairs(context.TODO(), &ec2.DescribeKeyPairsInput{
		KeyNames: []string{keyPairName},
	})
	
	return err == nil
}

// 恢复原始的简单 KeyPair 创建逻辑
func CreateOrGetKeyPair(stack awscdk.Stack, keyName, osType string) awsec2.IKeyPair {
	region := partition.DefaultRegion
	
	if osType != "windows" {
		osType = "linux"
	}
	
	cacheKey := fmt.Sprintf("%s-%s-%s", keyName, osType, region)
	
	keyPairMutex.Lock()
	defer keyPairMutex.Unlock()

	if cachedKeyPair, ok := keyPairCache[cacheKey]; ok {
		return cachedKeyPair
	}
	
	keyNameWithRegion := fmt.Sprintf("%s-%s-%s", keyName, osType, region)
	constructId := fmt.Sprintf("KeyPair-%s", types.HashString(keyNameWithRegion))
	
	var keyPair awsec2.KeyPair
	var iKeyPair awsec2.IKeyPair

	if osType == "windows" {
		keyPair = awsec2.NewKeyPair(stack, jsii.String(constructId), &awsec2.KeyPairProps{
			KeyPairName: jsii.String(keyNameWithRegion),
			Type: awsec2.KeyPairType_RSA,
		})
	} else {
		keyPair = awsec2.NewKeyPair(stack, jsii.String(constructId), &awsec2.KeyPairProps{
			KeyPairName: jsii.String(keyNameWithRegion),
			Type: awsec2.KeyPairType_ED25519,
			Format: awsec2.KeyPairFormat_PEM,
		})
	}

	iKeyPair = keyPair
	if keyPair != nil {
		keyPairCache[cacheKey] = iKeyPair
	}
	return iKeyPair
}
