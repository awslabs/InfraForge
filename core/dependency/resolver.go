// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package dependency

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// GetDependencyInfo 获取多个依赖信息
func GetDependencyInfo(dependency string) (string, error) {
	if len(dependency) == 0 {
		return "", nil
	}

	// 存储所有依赖资源的信息
	dependencyInfos := make(map[string]*ResourceInfo)

	// 按逗号分割多个依赖
	dependencies := strings.Split(dependency, ",")

	// 处理每个依赖
	for _, dep := range dependencies {
		// 清理可能存在的空格
		dep = strings.TrimSpace(dep)

		parts := strings.Split(dep, ":")
		if len(parts) != 2 {
			return "", fmt.Errorf("%w: invalid dependency format '%s'", ErrInvalidFormat, dep)
		}

		resourceType := parts[0]
		resourceID := parts[1]

		// 直接从 ForgeManager 获取属性（使用完整的 "type:id" 格式）
		properties, exists := GlobalManager.GetProperties(dep)
		if !exists {
			return "", fmt.Errorf("%w: resource '%s' not found", ErrResourceNotFound, dep)
		}

		// 创建资源信息并存储
		dependencyInfos[dep] = &ResourceInfo{
			Type:       resourceType,
			Id:         resourceID,
			Properties: properties,
		}
	}

	// 创建包含所有依赖信息的结构
	result := struct {
		Dependencies map[string]*ResourceInfo `json:"dependencies"`
	}{
		Dependencies: dependencyInfos,
	}

	// 转换为 JSON
	jsonData, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("error marshaling dependency info: %w", err)
	}

	return string(jsonData), nil
}

func GetMountPoint(dependency string) (string, error) {
	magicToken, err := GetDependencyInfo(dependency)
	if err != nil {
		return "", err
	}

	// 解析 JSON 到新的结构体
	var response DependenciesResponse
	err = json.Unmarshal([]byte(magicToken), &response)
	if err != nil {
		return "", err
	}

	// 从正确的路径获取 mountPoint
	resourceInfo, exists := response.Dependencies[dependency]
	if !exists {
		return "", fmt.Errorf("dependency %s not found", dependency)
	}

	mountPoint, ok := resourceInfo.Properties["mountPoint"].(string)
	if !ok {
		return "", errors.New("mountPoint not found or not a string")
		
	}

	return mountPoint, nil
}

// ExtractDependencyProperties 从magicToken中提取指定类型的依赖属性
func ExtractDependencyProperties(magicTokenStr string, resourceType string) (map[string]interface{}, error) {
	var magicToken map[string]interface{}
	err := json.Unmarshal([]byte(magicTokenStr), &magicToken)
	if err != nil {
		return nil, fmt.Errorf("failed to parse magic token: %v", err)
	}

	dependencies, ok := magicToken["dependencies"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid magic token format: dependencies not found")
	}

	for _, dep := range dependencies {
		depMap, ok := dep.(map[string]interface{})
		if !ok {
			continue
		}

		if depType, ok := depMap["type"].(string); ok && depType == resourceType {
			if properties, ok := depMap["properties"].(map[string]interface{}); ok {
				return properties, nil
			}
		}
	}

	return nil, fmt.Errorf("%s dependency not found", resourceType)
}
