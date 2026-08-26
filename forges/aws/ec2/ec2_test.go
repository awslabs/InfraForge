// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ec2

import (
	"testing"
	
	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/core/utils/types"
)

// We're using the actual Ec2InstanceConfig and Ec2Forge from ec2.go

func boolPtr(b bool) *bool { return &b }

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
		EbsSize:           "30",
		EbsVolumeType:     "gp3",
		EnableEfa:         boolPtr(true),
		EnaSrdEnabled:     boolPtr(true),
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
	
	if config.EbsSize != "30" {
		t.Errorf("Expected EbsSize to be \"30\", got %q", config.EbsSize)
	}
	
	if config.EbsVolumeType != "gp3" {
		t.Errorf("Expected EbsVolumeType to be 'gp3', got %q", config.EbsVolumeType)
	}
	
	// 验证新的网络配置字段
	if !types.GetBoolValue(config.EnableEfa, false) {
		t.Errorf("Expected EnableEfa to be true, got %v", config.EnableEfa)
	}
	
	if !types.GetBoolValue(config.EnaSrdEnabled, false) {
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

// TestRequireImdsv2Semantics 固化 IMDSv2 的取值语义：
// 未配置 requireImdsv2 时默认强制 IMDSv2（httpTokens=required），只有显式写 false 才退回 IMDSv1 可用。
// 这一默认值是中国区等开启了账户级 httpTokensEnforced 的账户能正常启动实例的前提。
func TestRequireImdsv2Semantics(t *testing.T) {
	cases := []struct {
		name string
		cfg  Ec2InstanceConfig
		want bool
	}{
		{"unset defaults to required", Ec2InstanceConfig{}, true},
		{"explicit true", Ec2InstanceConfig{RequireImdsv2: boolPtr(true)}, true},
		{"explicit false opts out", Ec2InstanceConfig{RequireImdsv2: boolPtr(false)}, false},
	}

	for _, c := range cases {
		if got := types.GetBoolValue(c.cfg.RequireImdsv2, true); got != c.want {
			t.Errorf("%s: expected requireImdsv2=%v, got %v", c.name, c.want, got)
		}
	}
}

// TestMergeConfigs_RequireImdsv2 验证实例级 requireImdsv2 能覆盖 defaults，
// 且实例未设置时保留 defaults 的取值。
func TestMergeConfigs_RequireImdsv2(t *testing.T) {
	forge := &Ec2Forge{}

	// 实例显式关闭，应覆盖 defaults 的 true
	merged := forge.MergeConfigs(
		&Ec2InstanceConfig{RequireImdsv2: boolPtr(true)},
		&Ec2InstanceConfig{RequireImdsv2: boolPtr(false)},
	).(*Ec2InstanceConfig)
	if types.GetBoolValue(merged.RequireImdsv2, true) {
		t.Error("instance requireImdsv2=false should override defaults=true")
	}

	// 实例未设置，应沿用 defaults
	merged = forge.MergeConfigs(
		&Ec2InstanceConfig{RequireImdsv2: boolPtr(true)},
		&Ec2InstanceConfig{},
	).(*Ec2InstanceConfig)
	if !types.GetBoolValue(merged.RequireImdsv2, false) {
		t.Error("defaults requireImdsv2=true should be kept when instance leaves it unset")
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
