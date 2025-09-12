// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package efs

import (
	"testing"
	
	"github.com/awslabs/InfraForge/core/config"
)

// We're using the actual EfsInstanceConfig and EfsForge from efs.go

func TestEfsInstanceConfig(t *testing.T) {
	// 创建一个EFS实例配置
	config := EfsInstanceConfig{
		BaseInstanceConfig: config.BaseInstanceConfig{
			ID:            "test-efs",
			Type:          "efs",
			Subnet:        "private",
			SecurityGroup: "default",
		},
		RemovePolicy: "DESTROY",
	}
	
	// 验证基本字段
	if config.GetID() != "test-efs" {
		t.Errorf("Expected ID to be 'test-efs', got %q", config.GetID())
	}
	
	if config.GetType() != "efs" {
		t.Errorf("Expected Type to be 'efs', got %q", config.GetType())
	}
	
	// 验证EFS特定字段
	if config.RemovePolicy != "DESTROY" {
		t.Errorf("Expected RemovePolicy to be 'DESTROY', got %q", config.RemovePolicy)
	}
}

func TestEfsForge_MergeConfigs(t *testing.T) {
	// Skip this test for now
	t.Skip("Skipping test for EfsForge.MergeConfigs - implement when ready")
}

func TestEfsForge_Create(t *testing.T) {
	// Skip this test for now
	t.Skip("Skipping test for EfsForge.Create - implement when ready")
}
