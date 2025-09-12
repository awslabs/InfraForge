// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

func GetAvailabilityZones() []string {
	// 创建 AWS 配置
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// 创建 EC2 服务客户端
	ec2Client := ec2.NewFromConfig(cfg)

	// 获取可用区列表
	output, err := ec2Client.DescribeAvailabilityZones(context.TODO(), &ec2.DescribeAvailabilityZonesInput{})
	if err != nil {
		log.Fatalf("Failed to describe availability zones: %v", err)
	}

	// 构建可用区名称列表
	var availabilityZones []string
	for _, az := range output.AvailabilityZones {
		if az.ZoneName != nil {
			availabilityZones = append(availabilityZones, *az.ZoneName)
		}
	}

	return availabilityZones
}

