// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ec2

import (
	"testing"
	
	"github.com/awslabs/InfraForge/core/config"
)

// We're using the actual Ec2InstanceConfig and Ec2Forge from ec2.go

func TestEc2InstanceConfig(t *testing.T) {
	// 创建一个EC2实例配置
	config := Ec2InstanceConfig{
		BaseInstanceConfig: config.BaseInstanceConfig{
			ID:            "test-ec2",
			Type:          "ec2",
			Subnet:        "public",
			SecurityGroup: "default",
		},
		InstanceType:      "t3.micro",
		OsImage:           "ami-12345678",
		KeyName:           "test-key",
		UserDataToken:     "some-token",
		EbsSize:           30,
		EbsVolumeType:     "gp3",
		EnableEfa:         true,
		EnaSrdEnabled:     true,
		NetworkCardCount:  2,
		PurchaseOption:    "spot",
		SpotMaxPrice:      "0.10",
		CapacityBlockId:   "cr-1234567890abcdef0",
	}
	
	// 验证基本字段
	if config.GetID() != "test-ec2" {
		t.Errorf("Expected ID to be 'test-ec2', got %q", config.GetID())
	}
	
	if config.GetType() != "ec2" {
		t.Errorf("Expected Type to be 'ec2', got %q", config.GetType())
	}
	
	// 验证EC2特定字段
	if config.InstanceType != "t3.micro" {
		t.Errorf("Expected InstanceType to be 't3.micro', got %q", config.InstanceType)
	}
	
	if config.OsImage != "ami-12345678" {
		t.Errorf("Expected OsImage to be 'ami-12345678', got %q", config.OsImage)
	}
	
	if config.KeyName != "test-key" {
		t.Errorf("Expected KeyName to be 'test-key', got %q", config.KeyName)
	}
	
	if config.EbsSize != 30 {
		t.Errorf("Expected EbsSize to be 30, got %d", config.EbsSize)
	}
	
	if config.EbsVolumeType != "gp3" {
		t.Errorf("Expected EbsVolumeType to be 'gp3', got %q", config.EbsVolumeType)
	}
	
	// 验证新的网络配置字段
	if !config.EnableEfa {
		t.Errorf("Expected EnableEfa to be true, got %v", config.EnableEfa)
	}
	
	if !config.EnaSrdEnabled {
		t.Errorf("Expected EnaSrdEnabled to be true, got %v", config.EnaSrdEnabled)
	}
	
	if config.NetworkCardCount != 2 {
		t.Errorf("Expected NetworkCardCount to be 2, got %d", config.NetworkCardCount)
	}
	
	// 验证购买选项字段
	if config.PurchaseOption != "spot" {
		t.Errorf("Expected PurchaseOption to be 'spot', got %q", config.PurchaseOption)
	}
	
	if config.SpotMaxPrice != "0.10" {
		t.Errorf("Expected SpotMaxPrice to be '0.10', got %q", config.SpotMaxPrice)
	}
	
	if config.CapacityBlockId != "cr-1234567890abcdef0" {
		t.Errorf("Expected CapacityBlockId to be 'cr-1234567890abcdef0', got %q", config.CapacityBlockId)
	}
}

func TestEc2Forge_MergeConfigs(t *testing.T) {
	// Skip this test for now
	t.Skip("Skipping test for Ec2Forge.MergeConfigs - implement when ready")
}

func TestEc2Forge_Create(t *testing.T) {
	// Skip this test for now
	t.Skip("Skipping test for Ec2Forge.Create - implement when ready")
}
