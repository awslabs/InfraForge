// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package parallelcluster

import (
	"testing"

	"github.com/awslabs/InfraForge/core/config"
)

func TestMergeConfigs(t *testing.T) {
	forge := NewParallelClusterForge()

	defaults := &ParallelClusterInstanceConfig{
		BaseInstanceConfig: config.BaseInstanceConfig{
			ID:            "default-id",
			Type:          "PARALLELCLUSTER",
			Subnet:        "private",
			SecurityGroup: "private",
		},
		KeyName:          "default-key",
		SharedKeyName:    "default-shared-key",
		Version:          "3.14.0",
		HeadNodeType:     "t3.medium",
		ComputeNodeType:  "t3.micro",
		Os:               "alinux2",
		ClusterName:      "default-cluster",
		DiskSize:         35,
		MinSize:          0,
		MaxSize:          10,
		PublicCIDR:       "10.0.0.0/24",
		PrivateCIDR:      "10.0.16.0/20",
		AdditionalPolicies: []string{"policy1", "policy2"},
	}

	instance := &ParallelClusterInstanceConfig{
		BaseInstanceConfig: config.BaseInstanceConfig{
			ID: "custom-id",
		},
		KeyName:         "custom-key",
		HeadNodeType:    "t3.large",
		ComputeNodeType: "c5.large",
		DiskSize:        50,
		MaxSize:         20,
		AdditionalPolicies: []string{"policy3"},
	}

	merged := forge.MergeConfigs(defaults, instance)
	mergedConfig, ok := merged.(*ParallelClusterInstanceConfig)
	if !ok {
		t.Fatalf("Failed to cast merged config to ParallelClusterInstanceConfig")
	}

	// Check that values were merged correctly
	if mergedConfig.ID != "custom-id" {
		t.Errorf("Expected ID to be 'custom-id', got '%s'", mergedConfig.ID)
	}

	if mergedConfig.KeyName != "custom-key" {
		t.Errorf("Expected KeyName to be 'custom-key', got '%s'", mergedConfig.KeyName)
	}

	if mergedConfig.Version != "3.14.0" {
		t.Errorf("Expected Version to be '3.14.0', got '%s'", mergedConfig.Version)
	}

	if mergedConfig.HeadNodeType != "t3.large" {
		t.Errorf("Expected HeadNodeType to be 't3.large', got '%s'", mergedConfig.HeadNodeType)
	}

	if mergedConfig.ComputeNodeType != "c5.large" {
		t.Errorf("Expected ComputeNodeType to be 'c5.large', got '%s'", mergedConfig.ComputeNodeType)
	}

	if mergedConfig.Os != "alinux2" {
		t.Errorf("Expected Os to be 'alinux2', got '%s'", mergedConfig.Os)
	}
	
	if mergedConfig.DiskSize != 50 {
		t.Errorf("Expected DiskSize to be 50, got %d", mergedConfig.DiskSize)
	}
	
	if mergedConfig.MaxSize != 20 {
		t.Errorf("Expected MaxSize to be 20, got %d", mergedConfig.MaxSize)
	}

	if len(mergedConfig.AdditionalPolicies) != 1 || mergedConfig.AdditionalPolicies[0] != "policy3" {
		t.Errorf("Expected AdditionalPolicies to be ['policy3'], got %v", mergedConfig.AdditionalPolicies)
	}
}

func TestGetters(t *testing.T) {
	config := &ParallelClusterInstanceConfig{
		BaseInstanceConfig: config.BaseInstanceConfig{
			ID:            "test-id",
			Type:          "PARALLELCLUSTER",
			Subnet:        "private",
			SecurityGroup: "private",
		},
		KeyName:         "test-key",
		HeadNodeType:    "t3.large",
		ComputeNodeType: "c5.large",
	}

	if config.GetID() != "test-id" {
		t.Errorf("Expected GetID() to return 'test-id', got '%s'", config.GetID())
	}

	if config.GetType() != "PARALLELCLUSTER" {
		t.Errorf("Expected GetType() to return 'PARALLELCLUSTER', got '%s'", config.GetType())
	}

	if config.GetSubnet() != "private" {
		t.Errorf("Expected GetSubnet() to return 'private', got '%s'", config.GetSubnet())
	}

	if config.GetSecurityGroup() != "private" {
		t.Errorf("Expected GetSecurityGroup() to return 'private', got '%s'", config.GetSecurityGroup())
	}
}
