# InfraForge 依赖管理系统

本目录包含 InfraForge 依赖管理系统的核心代码。该系统负责跟踪在 CDK 部署过程中创建的 AWS 资源，并提供一种标准化的方式，使不同的 Forge 服务能够访问这些资源的属性。

## 系统概述

依赖管理系统由以下组件组成：

1. **ForgeManager：** 一个中央注册表，用于存储 Forge 实例的引用
2. **GlobalManager：** 用于跨 Forge 通信的全局实例

该系统通过提供一种标准化的方式来直接从 Forge 实例访问资源属性，实现了 CDK 部署过程中不同 Forge 服务之间的无缝通信。

## 核心组件

### ForgeManager

`ForgeManager` 是一个中央注册表，用于存储 Forge 实例的引用。它提供了通过 ID 存储和检索 Forge 实例的方法。

```go
type ForgeManager struct {
    forges map[string]interface{} // 存储 Forge 实例
    mutex  sync.RWMutex
}
```

### 方法

- `Store(key string, forge interface{})：` 使用给定的键存储 Forge 实例
- `Get(key string) (interface{}, bool)：` 通过键检索 Forge 实例

### GlobalManager

提供了一个全局的 ForgeManager 实例用于跨 Forge 通信：

```go
var GlobalManager *ForgeManager
```

## 使用示例

```go
import "github.com/awslabs/InfraForge/core/dependency"

// 存储 forge 实例
dependency.GlobalManager.Store("vpc:main", vpcForge)

// 检索 forge 实例
if forge, exists := dependency.GlobalManager.Get("vpc:main"); exists {
    vpcForge := forge.(*vpc.VpcForge)
    // 使用 VPC forge
}
```

## 主要特性

- 使用互斥锁保护的线程安全操作
- 通过 interface{} 存储支持任何 forge 类型
- 在所有 forge 实现中全局可访问
- 简单的键值存储机制
// 存储 Forge 实例
dependency.GlobalManager.Store("my-resource-id", forgeInstance)

// 检索 Forge 实例
forge, exists := dependency.GlobalManager.Get("my-resource-id")

// 直接从 Forge 获取属性
properties, exists := dependency.GlobalManager.GetProperties("my-resource-id")
```

### Forge 属性

每个 Forge 实例都实现了 `GetProperties()` 方法，该方法返回其资源属性的映射。属性在 Forge 创建过程中保存，包含所有相关的资源信息。

```go
type Forge interface {
    GetProperties() map[string]interface{}
}
```

### 依赖解析

系统提供了用于解析资源之间依赖关系的工具。`GetDependencyInfo` 函数接受格式为 `<ResourceType>:<ResourceID>` 的依赖字符串，并返回资源属性的 JSON 表示：

```go
dependencyInfo, err := dependency.GetDependencyInfo("EFS:my-efs-resource")
```

对于多个依赖，您可以提供逗号分隔的列表：

```go
dependencyInfo, err := dependency.GetDependencyInfo("EFS:my-efs-resource,EC2:my-ec2-instance")
```

## 支持的资源类型

系统支持以下 AWS 资源类型：

- **VPC**：虚拟私有云资源
- **EC2**：EC2 实例及详细实例信息
- **ECS**：弹性容器服务集群
- **EKS**：弹性 Kubernetes 服务集群
- **EFS**：弹性文件系统资源
- **Lustre**：FSx for Lustre 文件系统
- **DS**：目录服务（Microsoft Active Directory）
- **RDS**：关系数据库服务实例和集群

## 与 CDK 部署的集成

在 CDK 部署过程中，DependencyManager 会在创建 Forge 实例时填充这些实例。每个 Forge 在创建过程中保存其资源属性。当 Forge 服务需要访问另一个资源的属性时，它可以使用依赖解析工具来获取所需的信息。

这种方法解耦了服务，允许它们在没有直接依赖的情况下进行交互，使系统更易于维护和更灵活。

## 使用示例

```go
// Forge 实例在创建过程中自动存储
// 获取依赖信息以在其他 Forge 中使用
dependencyInfo, err := dependency.GetDependencyInfo("EFS:my-efs")
if err != nil {
    return err
}

// 提取特定资源属性
efsProperties, err := dependency.ExtractDependencyProperties(dependencyInfo, "EFS")
if err != nil {
    return err
}

// 使用属性
fileSystemId := efsProperties["fileSystemId"].(string)
mountPoint := efsProperties["mountPoint"].(string)
```

## 架构优势

当前架构提供了几个优势：

1. **简化代码**：无需单独的资源处理器
2. **更好的性能**：属性在创建时计算一次
3. **集中逻辑**：所有资源属性逻辑都在 Forge 创建方法中
4. **易于维护**：资源属性的单一真实来源
5. **类型安全**：直接访问 Forge 方法和属性
