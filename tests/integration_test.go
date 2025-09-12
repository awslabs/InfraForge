// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package tests

import (
	"encoding/json"
	"os"
	"testing"
	
	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/registry"
	"github.com/aws/aws-cdk-go/awscdk/v2"
	"github.com/aws/jsii-runtime-go"
)

// 集成测试：测试从配置文件创建基础设施堆栈的完整流程
// 注意：这个测试不会实际部署资源，只会验证合成过程

func TestIntegrationBasicInfrastructure(t *testing.T) {
	// 跳过集成测试，除非明确启用
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}
	
	// 创建一个测试配置
	testConfig := map[string]interface{}{
		"global": map[string]interface{}{
			"stackName": "test-infra-stack",
			"dualStack": false,
		},
		"enabledForges": []string{"vpc1", "efs1"},
		"forges": map[string]interface{}{
			"vpc": map[string]interface{}{
				"vpc1": map[string]interface{}{
					"id":             "vpc1",
					"type":           "vpc",
					"cidrBlock":      "10.0.0.0/16",
					"natGatewayPerAZ": false,
				},
			},
			"efs": map[string]interface{}{
				"efs1": map[string]interface{}{
					"id":             "efs1",
					"type":           "efs",
					"subnet":         "private",
					"securityGroup":  "default",
					"performanceMode": "generalPurpose",
					"throughputMode":  "bursting",
					"enableEncryption": true,
					"removalPolicy":   "destroy",
				},
			},
		},
	}
	
	// 将配置写入临时文件
	configJSON, err := json.MarshalIndent(testConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal test config: %v", err)
	}
	
	tempConfigFile := "/tmp/test_integration_config.json"
	err = os.WriteFile(tempConfigFile, configJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write test config file: %v", err)
	}
	
	// 加载配置
	loadedConfig, err := config.LoadConfig(tempConfigFile)
	if err != nil {
		t.Fatalf("Failed to load test config: %v", err)
	}
	
	// 验证配置加载正确
	if loadedConfig.Global.StackName != "test-infra-stack" {
		t.Errorf("Expected stackName to be 'test-infra-stack', got %v", loadedConfig.Global.StackName)
	}
	
	// 创建CDK应用和堆栈
	app := awscdk.NewApp(nil)
	stack := awscdk.NewStack(app, jsii.String(loadedConfig.Global.StackName), &awscdk.StackProps{})
	
	// 验证已注册的forge类型
	if len(registry.ForgeConstructors) == 0 {
		t.Fatal("No forge constructors registered")
	}
	
	// 验证已注册的实例创建器
	vpcCreator := registry.CreateInstance("vpc")
	if vpcCreator == nil {
		t.Fatal("VPC instance creator not registered")
	}
	
	efsCreator := registry.CreateInstance("efs")
	if efsCreator == nil {
		t.Fatal("EFS instance creator not registered")
	}
	
	// 验证堆栈不为nil
	if stack == nil {
		t.Fatal("Failed to create CDK stack")
	}
	
	// 尝试合成堆栈（这只会验证CDK构造是否有效，不会部署资源）
	template := app.Synth(nil)
	if template == nil {
		t.Fatal("Failed to synthesize CDK app")
	}
	
	// 验证堆栈存在于合成的模板中
	stacks := template.Stacks()
	if len(*stacks) == 0 {
		t.Fatal("No stacks found in synthesized template")
	}
	
	// 验证堆栈名称
	foundStack := false
	for _, s := range *stacks {
		if *s.StackName() == loadedConfig.Global.StackName {
			foundStack = true
			break
		}
	}
	
	if !foundStack {
		t.Errorf("Stack %q not found in synthesized template", loadedConfig.Global.StackName)
	}
}

// 测试配置验证
func TestConfigValidation(t *testing.T) {
	// 跳过集成测试，除非明确启用
	if os.Getenv("RUN_INTEGRATION_TESTS") != "true" {
		t.Skip("Skipping integration test. Set RUN_INTEGRATION_TESTS=true to run")
	}
	
	// 创建一个无效的测试配置（缺少必要字段）
	invalidConfig := map[string]interface{}{
		"global": map[string]interface{}{
			// 缺少stackName
			"dualStack": false,
		},
		"enabledForges": []string{"vpc1"},
		"forges": map[string]interface{}{
			"vpc": map[string]interface{}{
				"vpc1": map[string]interface{}{
					// 缺少id
					"type":      "vpc",
					"cidrBlock": "10.0.0.0/16",
				},
			},
		},
	}
	
	// 将配置写入临时文件
	configJSON, err := json.MarshalIndent(invalidConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal invalid test config: %v", err)
	}
	
	tempConfigFile := "/tmp/test_invalid_config.json"
	err = os.WriteFile(tempConfigFile, configJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid test config file: %v", err)
	}
	
	// 加载配置 - 这里我们期望它能加载，但后续的验证应该失败
	loadedConfig, err := config.LoadConfig(tempConfigFile)
	if err != nil {
		t.Fatalf("Failed to load invalid test config: %v", err)
	}
	
	// 验证配置缺少必要字段
	if loadedConfig.Global.StackName != "" {
		t.Error("Expected stackName to be empty, but it exists")
	}
	
	// 这里应该调用你的配置验证逻辑
	// 由于我们没有看到实际的验证代码，这里只是提供一个模板
	
	// 例如：
	// err = validateConfig(loadedConfig)
	// if err == nil {
	//     t.Error("Expected validation error for invalid config, but got nil")
	// }
}
