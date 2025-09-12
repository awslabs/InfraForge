// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"testing"
	"reflect"
	
	"github.com/awslabs/InfraForge/core/interfaces"
	"github.com/awslabs/InfraForge/core/config"
	"github.com/awslabs/InfraForge/forges/aws/vpc"
)

func TestRegisterForge(t *testing.T) {
	// 清理测试前的注册表
	oldForgeConstructors := ForgeConstructors
	ForgeConstructors = make(map[string]func() interfaces.Forge)
	defer func() { ForgeConstructors = oldForgeConstructors }()

	// 测试注册一个新的forge
	testForgeType := "test-forge"
	testConstructor := func() interfaces.Forge { return &vpc.VpcForge{} }
	
	RegisterForge(testForgeType, testConstructor)
	
	// 验证forge已被正确注册
	if _, exists := ForgeConstructors[testForgeType]; !exists {
		t.Errorf("Expected forge type %q to be registered", testForgeType)
	}
	
	// 测试注册nil构造函数时应该panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when registering nil constructor, but no panic occurred")
		}
	}()
	RegisterForge("nil-forge", nil)
}

func TestRegisterInstanceCreator(t *testing.T) {
	// 清理测试前的注册表
	oldInstanceCreators := instanceCreators
	instanceCreators = make(map[string]func() config.InstanceConfig)
	defer func() { instanceCreators = oldInstanceCreators }()

	// 测试注册一个新的实例创建器
	testType := "test-instance"
	testCreator := func() config.InstanceConfig { return &vpc.VpcInstanceConfig{} }
	
	RegisterInstanceCreator(testType, testCreator)
	
	// 验证实例创建器已被正确注册
	if _, exists := instanceCreators[testType]; !exists {
		t.Errorf("Expected instance creator for type %q to be registered", testType)
	}
	
	// 测试重复注册同一类型应该panic
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected panic when registering duplicate instance creator, but no panic occurred")
		}
	}()
	RegisterInstanceCreator(testType, testCreator)
}

func TestCreateInstance(t *testing.T) {
	// 清理测试前的注册表
	oldInstanceCreators := instanceCreators
	instanceCreators = make(map[string]func() config.InstanceConfig)
	defer func() { instanceCreators = oldInstanceCreators }()

	// 注册一个测试实例创建器
	testType := "test-instance"
	testCreator := func() config.InstanceConfig { return &vpc.VpcInstanceConfig{} }
	RegisterInstanceCreator(testType, testCreator)
	
	// 测试创建已注册类型的实例
	instance := CreateInstance(testType)
	if instance == nil {
		t.Errorf("Expected non-nil instance for registered type %q", testType)
	}
	
	// 验证实例类型
	if _, ok := instance.(*vpc.VpcInstanceConfig); !ok {
		t.Errorf("Expected instance of type *vpc.VpcInstanceConfig, got %v", reflect.TypeOf(instance))
	}
	
	// 测试创建未注册类型的实例
	unknownInstance := CreateInstance("unknown-type")
	if unknownInstance != nil {
		t.Errorf("Expected nil instance for unregistered type, got %v", unknownInstance)
	}
}
