// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"
)

func TestBaseInstanceConfig(t *testing.T) {
	// 创建一个基本实例配置
	config := BaseInstanceConfig{
		ID:            "test-instance",
		Type:          "test-type",
		Subnet:        "public",
		SecurityGroup: "default",
	}
	
	// 测试getter方法
	if config.GetID() != "test-instance" {
		t.Errorf("Expected ID to be 'test-instance', got %q", config.GetID())
	}
	
	if config.GetType() != "test-type" {
		t.Errorf("Expected Type to be 'test-type', got %q", config.GetType())
	}
	
	if config.GetSubnet() != "public" {
		t.Errorf("Expected Subnet to be 'public', got %q", config.GetSubnet())
	}
	
	if config.GetSecurityGroup() != "default" {
		t.Errorf("Expected SecurityGroup to be 'default', got %q", config.GetSecurityGroup())
	}
}

// 创建一个测试配置结构体
type TestConfig struct {
	Global struct {
		StackName string `json:"stackName"`
		DualStack bool   `json:"dualStack"`
	} `json:"global"`
	EnabledForges []string `json:"enabledForges"`
	Forges        struct {
		Vpc map[string]json.RawMessage `json:"vpc"`
		Ec2 map[string]json.RawMessage `json:"ec2"`
	} `json:"forges"`
}

func TestLoadConfig(t *testing.T) {
	// 创建一个测试配置JSON
	configJSON := `{
		"global": {
			"stackName": "test-stack",
			"dualStack": true
		},
		"enabledForges": ["vpc1", "ec2-1"],
		"forges": {
			"vpc": {
				"vpc1": {
					"id": "vpc1",
					"type": "vpc",
					"cidrBlock": "10.0.0.0/16",
					"natGatewayPerAZ": false
				}
			},
			"ec2": {
				"ec2-1": {
					"id": "ec2-1",
					"type": "ec2",
					"instanceType": "t3.micro",
					"ami": "ami-12345678"
				}
			}
		}
	}`
	
	// 创建一个临时配置文件
	tempFile := "/tmp/test_config.json"
	
	// 写入测试配置
	err := os.WriteFile(tempFile, []byte(configJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// 加载配置
	config, err := LoadConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	// 验证配置内容
	if config.Global.StackName != "test-stack" {
		t.Errorf("Expected StackName to be 'test-stack', got %q", config.Global.StackName)
	}
	
	if !config.Global.DualStack {
		t.Errorf("Expected DualStack to be true")
	}
	
	expectedEnabledForges := []string{"vpc1", "ec2-1"}
	if !reflect.DeepEqual(config.EnabledForges, expectedEnabledForges) {
		t.Errorf("Expected EnabledForges to be %v, got %v", expectedEnabledForges, config.EnabledForges)
	}
	
	// 验证Forges配置存在
	if len(config.Forges) == 0 {
		t.Errorf("Expected Forges config to exist")
	}
}

func TestWriteConfigFile(t *testing.T) {
	// 创建一个测试配置
	config := Config{}
	config.Global.StackName = "test-stack"
	config.Global.DualStack = true
	config.EnabledForges = []string{"vpc1", "ec2-1"}
	
	// 将配置转换为JSON
	configJSON, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config to JSON: %v", err)
	}
	
	// 写入配置文件
	tempFile := "/tmp/test_write_config.json"
	err = os.WriteFile(tempFile, configJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}
	
	// 读取配置文件
	loadedConfig, err := LoadConfig(tempFile)
	if err != nil {
		t.Fatalf("Failed to load written config file: %v", err)
	}
	
	// 验证配置内容
	if loadedConfig.Global.StackName != "test-stack" {
		t.Errorf("Expected StackName to be 'test-stack', got %q", loadedConfig.Global.StackName)
	}
	
	if !loadedConfig.Global.DualStack {
		t.Errorf("Expected DualStack to be true")
	}
	
	expectedEnabledForges := []string{"vpc1", "ec2-1"}
	if !reflect.DeepEqual(loadedConfig.EnabledForges, expectedEnabledForges) {
		t.Errorf("Expected EnabledForges to be %v, got %v", expectedEnabledForges, loadedConfig.EnabledForges)
	}
}
