// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package dependency

import (
	"sync"
)

// ForgeManager 管理 Forge 创建的资源
type ForgeManager struct {
	forges map[string]interface{} // 存储 Forge 实例
	mutex  sync.RWMutex
}

// NewForgeManager 创建新的 Forge 管理器
func NewForgeManager() *ForgeManager {
	return &ForgeManager{
		forges: make(map[string]interface{}),
	}
}

// Store 存储 Forge 实例，使用 "type:id" 格式的 key
func (fm *ForgeManager) Store(key string, forge interface{}) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	fm.forges[key] = forge
}

// Get 获取 Forge 实例，支持 "type:id" 格式的 key
func (fm *ForgeManager) Get(key string) (interface{}, bool) {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()
	forge, exists := fm.forges[key]
	return forge, exists
}

// GetProperties 获取 Forge 的属性（如果支持）
// GetProperties 获取 Forge 实例的属性，支持 "type:id" 格式的 key
func (fm *ForgeManager) GetProperties(key string) (map[string]interface{}, bool) {
	forge, exists := fm.Get(key)
	if !exists {
		return nil, false
	}
	
	// 尝试调用 GetProperties 方法
	if propertiesGetter, ok := forge.(interface{ GetProperties() map[string]interface{} }); ok {
		return propertiesGetter.GetProperties(), true
	}
	
	return nil, false
}

var GlobalManager *ForgeManager

func init() {
	GlobalManager = NewForgeManager()
}
