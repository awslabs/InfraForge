# ForgeManager 依赖管理机制

## 概述

ForgeManager 使用 GlobalManager 来管理 forge 实例之间的依赖关系。本文档说明依赖的存储、查找和使用机制。

## 依赖存储机制

### 存储位置
- **文件:**  `core/manager/forgemanager.go`
- **方法:**  `processForge()` 第141-143行

### 存储逻辑
```go
if trimIndex {
    dependency.GlobalManager.Store(aws.GetOriginalID(merged.GetID()), iforge)
} else {
    dependency.GlobalManager.Store(merged.GetID(), iforge)
}
```

### 存储的 Key
- **普通情况:**  使用配置中的 `id` 字段
- **EC2 多实例:**  使用 `aws.GetOriginalID()` 处理的 ID

### 存储的 Value
- 存储的是 **forge 对象本身**（如 `*EksForge`），不是资源对象

## 依赖查找机制

### 配置格式
```json
{
    "dependsOn": "EKS:eks"
}
```

### 解析逻辑
```go
// 解析 "EKS:eks" -> "eks"
parts := strings.Split(dependsOn, ":")
dependencyID := dependsOn
if len(parts) == 2 {
    dependencyID = parts[1]  // 取冒号后面的部分
}
```

### 查找步骤
1. 解析依赖字符串，提取实际的 ID
2. 从 GlobalManager 中查找对应的 forge 对象
3. 类型转换获取具体的 forge 类型
4. 从 forge 对象中获取实际的资源对象

## 实际案例：HyperPod 依赖 EKS

### 配置示例
```json
{
    "eks": {
        "instances": [
            {
                "id": "eks"
            }
        ]
    },
    "hyperpod": {
        "instances": [
            {
                "id": "hyperpod",
                "dependsOn": "EKS:eks"
            }
        ]
    }
}
```

### 依赖解析过程
1. **EKS 创建时:**  `GlobalManager.Store("eks", eksForgeObject)`
2. **HyperPod 查找时:**  
   - 解析 `"EKS:eks"` -> `"eks"`
   - 调用 `GlobalManager.Get("eks")`
   - 获得 `*EksForge` 对象
   - 调用 `eksForge.GetCluster()` 获得实际的 EKS 集群

### 代码示例
```go
// HyperPod 中的依赖查找
parts := strings.Split(hyperPodInstance.DependsOn, ":")
dependencyID := parts[1] // "eks"

if eksForge, exists := dependency.GlobalManager.Get(dependencyID); exists {
    if eksForgeObj, ok := eksForge.(*eksforge.EksForge); ok {
        eksCluster := eksForgeObj.GetCluster()
        // 使用 eksCluster 进行后续操作
    }
}
```

## 关键要点

1. **存储的是 forge 对象**，不是资源对象
2. **依赖格式:**  `"服务类型:实例ID"`，查找时只用实例ID部分
3. **类型转换:**  需要将 `interface{}` 转换为具体的 forge 类型
4. **资源获取:**  通过 forge 对象的方法获取实际资源

## 扩展新的依赖关系

### 步骤
1. 确保 forge 对象被正确存储到 GlobalManager
2. 在依赖方解析 `dependsOn` 字符串
3. 从 GlobalManager 获取 forge 对象
4. 类型转换并获取所需资源
5. 如需要，为 forge 添加 getter 方法（如 `GetCluster()`）

### 注意事项
- 依赖解析应该在资源创建的适当时机进行
- 确保依赖的 forge 已经完成创建
- 处理依赖不存在的情况
