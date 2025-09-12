// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package rds

import (
	"testing"

	"github.com/awslabs/InfraForge/core/config"
)

func TestMergeConfigs(t *testing.T) {
	defaults := &RdsInstanceConfig{
		BaseInstanceConfig: config.BaseInstanceConfig{
			ID:   "default-rds",
			Type: "rds",
		},
		Engine:       "mysql",
		InstanceType: "db.t3.micro",
	}

	instance := &RdsInstanceConfig{
		BaseInstanceConfig: config.BaseInstanceConfig{
			ID: "test-rds",
		},
		Engine:          "postgres",
		ClusterMode:     true,
		ReaderInstances: 2,
	}

	merged := (&RdsForge{}).MergeConfigs(defaults, instance).(*RdsInstanceConfig)

	if merged.GetID() != "test-rds" {
		t.Errorf("Expected ID 'test-rds', got '%s'", merged.GetID())
	}
	if merged.Engine != "postgres" {
		t.Errorf("Expected Engine 'postgres', got '%s'", merged.Engine)
	}
	if merged.InstanceType != "db.t3.micro" {
		t.Errorf("Expected InstanceType 'db.t3.micro', got '%s'", merged.InstanceType)
	}
	if !merged.ClusterMode {
		t.Error("Expected ClusterMode to be true")
	}
	if merged.ReaderInstances != 2 {
		t.Errorf("Expected ReaderInstances 2, got %d", merged.ReaderInstances)
	}
}

func TestIsAuroraEngine(t *testing.T) {
	tests := []struct {
		engine   string
		expected bool
	}{
		{"aurora-mysql", true},
		{"aurora-postgresql", true},
		{"mysql", false},
		{"postgres", false},
	}

	for _, tt := range tests {
		result := isAuroraEngine(tt.engine)
		if result != tt.expected {
			t.Errorf("isAuroraEngine(%s) = %v, expected %v", tt.engine, result, tt.expected)
		}
	}
}

func TestGetDefaultPort(t *testing.T) {
	tests := []struct {
		engine         string
		configuredPort int
		expected       int
	}{
		{"mysql", 0, 3306},
		{"postgres", 0, 5432},
		{"aurora-mysql", 0, 3306},
		{"aurora-postgresql", 0, 5432},
		{"mysql", 3307, 3307}, // 配置的端口优先
	}

	for _, tt := range tests {
		result := getDefaultPort(tt.engine, tt.configuredPort)
		if result != tt.expected {
			t.Errorf("getDefaultPort(%s, %d) = %d, expected %d", tt.engine, tt.configuredPort, result, tt.expected)
		}
	}
}
