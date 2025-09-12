// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package aws

import (
	"fmt"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
)

// SelectSubnetByAzIndex 根据 azIndex 选择单个子网
// azIndex: 用户指定的可用区索引（从1开始），0表示使用默认（第一个子网）
func SelectSubnetByAzIndex(azIndex int, vpc awsec2.IVpc, subnetType awsec2.SubnetType) awsec2.ISubnet {
	if azIndex > 0 {
		// 使用指定的可用区
		availabilityZones := GetAvailabilityZones()
		azIdx := azIndex - 1 // 转换为0基索引
		
		if azIdx >= 0 && azIdx < len(availabilityZones) {
			selectedAZ := availabilityZones[azIdx]
			subnetSelection := &awsec2.SubnetSelection{
				SubnetType:        subnetType,
				AvailabilityZones: &[]*string{&selectedAZ},
			}
			
			selectedSubnets := vpc.SelectSubnets(subnetSelection)
			subnets := *selectedSubnets.Subnets
			
			if len(subnets) > 0 {
				return subnets[0]
			}
		}
		
		fmt.Printf("警告: 可用区索引 %d 无效，使用默认选择\n", azIndex)
	}
	
	// 默认选择第一个子网
	allSubnets := *vpc.SelectSubnets(&awsec2.SubnetSelection{
		SubnetType: subnetType,
	}).Subnets
	
	if len(allSubnets) == 0 {
		panic(fmt.Sprintf("没有找到类型为 %v 的子网", subnetType))
	}
	
	return allSubnets[0]
}

// SelectSubnetIdByAzIndex 根据 azIndex 选择单个子网ID（用于 ParallelCluster）
func SelectSubnetIdByAzIndex(azIndex int, vpc awsec2.IVpc, subnetType awsec2.SubnetType) string {
	subnet := SelectSubnetByAzIndex(azIndex, vpc, subnetType)
	return *subnet.SubnetId()
}
