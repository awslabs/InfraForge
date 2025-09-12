# InfraForge 新 Forge 开发指南

## 🎯 概述

本指南提供了在 InfraForge 框架中开发新 Forge 组件的分步说明。Forge 是管理特定 AWS 服务的模块化基础设施组件。

## 📁 项目结构

### 1. 创建 Forge 目录结构
```
forges/aws/<service>/
├── <service>.go          # 主要实现文件
├── <service>_test.go     # 单元测试
└── README.md            # 文档
```

## 🏗️ 实现步骤

### 2. 定义配置结构体

```go
package <service>

import (
    "github.com/aws-samples/infraforge/core/config"
)

type <Service>InstanceConfig struct {
    config.BaseInstanceConfig
    
    // 服务特定的必需字段
    RequiredField    string `json:"requiredField"`           // 必需字段
    
    // 服务特定的可选字段
    OptionalField    string `json:"optionalField,omitempty"` // 可选字符串字段
    BoolField        *bool  `json:"boolField,omitempty"`     // 布尔字段（使用指针）
    IntField         int    `json:"intField,omitempty"`      // 整数字段
}
```

### 3. 研究并使用 AWS CDK 文档完善设计

初步设计完成后，使用 AWS CDK Go 文档来完善实现：

1. **访问 CDK 文档:**  https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/
2. **查找服务包:**  导航到特定的 AWS 服务（如 `awsbatch`、`awsrds`、`awsec2`）
3. **下载文档:**  将相关文档页面保存到本地作为参考
4. **注意包命名:**  CDK 包使用 `aws<service>` 格式（如 `batch.xxx` 变成 `awsbatch.xxx`）
5. **研究资源属性:**  查看可用的属性、方法和配置选项
6. **完善配置:**  添加遗漏的字段，修正属性名称，增强功能

**CDK 包 URL 示例：**
- AWS Batch: https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/awsbatch
- AWS RDS: https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/awsrds
- AWS EC2: https://pkg.go.dev/github.com/aws/aws-cdk-go/awscdk/v2/awsec2

**为什么先设计后参考？**
- 初步设计捕获业务需求和使用场景
- 文档审查增加技术深度和完整性
- 防止文档优先方法导致的过度简化
- 确保解决方案解决实际场景

### 4. 实现 Forge 接口

```go
type <Service>Forge struct {
    // 存储创建的资源引用
    resource    <AwsResource>
    properties  map[string]interface{}  // 存储资源属性供依赖使用
}

// 必需的接口方法
func (f *<Service>Forge) Create(ctx *interfaces.ForgeContext) error
func (f *<Service>Forge) MergeConfigs(defaults, instance config.InstanceConfig) config.InstanceConfig
func (f *<Service>Forge) ConfigureRules(ctx *interfaces.ForgeContext)
func (f *<Service>Forge) CreateOutputs(ctx *interfaces.ForgeContext)

// 如果资源会被其他资源依赖，需要实现此方法
func (f *<Service>Forge) GetProperties() map[string]interface{}
```

### 4. 实现 Create 方法

```go
func (f *<Service>Forge) Create(ctx *interfaces.ForgeContext) error {
    instance := ctx.Instance.(*<Service>InstanceConfig)
    
    // 1. 合并配置
    merged := f.MergeConfigs(ctx.DefaultConfig, instance).(*<Service>InstanceConfig)
    
    // 2. 初始化属性映射
    if f.properties == nil {
        f.properties = make(map[string]interface{})
    }
    
    // 3. 创建 AWS 资源
    resourceId := fmt.Sprintf("%s-%s", merged.GetID(), "<service>")
    props := &aws<Service>.<Resource>Props{
        // 配置资源属性
        RequiredProperty: jsii.String(merged.RequiredField),
        OptionalProperty: jsii.String(utils.GetStringValue(&merged.OptionalField, "default")),
        BoolProperty:     jsii.Bool(utils.GetBoolValue(merged.BoolField, false)),
        IntProperty:      jsii.Number(utils.GetIntValue(&merged.IntField, 0)),
    }
    
    // 处理服务特定逻辑
    // 在此添加您的服务创建逻辑
    
    return resource
    
    f.resource = aws<Service>.New<Resource>(ctx.Stack, jsii.String(resourceId), props)
    
    // 4. 存储资源属性供其他资源依赖
    f.properties["resourceId"] = *f.resource.ResourceId()
    f.properties["endpoint"] = *f.resource.Endpoint()
    // 根据服务类型添加其他相关属性
    
    // 5. 注册到依赖管理器
    dependency.GlobalManager.SetProperties(merged.GetID(), f.properties)
    
    return nil
}
```

### 5. 实现 MergeConfigs 方法

```go
func (f *<Service>Forge) MergeConfigs(defaults, instance config.InstanceConfig) config.InstanceConfig {
    merged := defaults.(*<Service>InstanceConfig)  // 从默认配置开始（直接引用）
    serviceInstance := instance.(*<Service>InstanceConfig)
    
    // 用实例配置值覆盖默认配置
    if serviceInstance.GetID() != "" {
        merged.ID = serviceInstance.GetID()
    }
    if serviceInstance.GetType() != "" {
        merged.Type = serviceInstance.GetType()
    }
    if serviceInstance.GetSubnet() != "" {
        merged.Subnet = serviceInstance.GetSubnet()
    }
    if serviceInstance.GetSecurityGroup() != "" {
        merged.SecurityGroup = serviceInstance.GetSecurityGroup()
    }
    
    // 字符串字段 - 如果不为空则覆盖
    if serviceInstance.OptionalField != "" {
        merged.OptionalField = serviceInstance.OptionalField
    }
    if serviceInstance.InstanceRolePolicies != "" {
        merged.InstanceRolePolicies = serviceInstance.InstanceRolePolicies
    }
    if serviceInstance.UserDataToken != "" {
        merged.UserDataToken = serviceInstance.UserDataToken
    }
    
    // 布尔指针字段 - 如果已设置（不为 nil）则覆盖
    if serviceInstance.BoolField != nil {
        merged.BoolField = serviceInstance.BoolField
    }
    
    // 整数字段 - 如果大于 0 则覆盖
    if serviceInstance.IntField > 0 {
        merged.IntField = serviceInstance.IntField
    }
    
    return merged  // 返回直接引用，不是 &merged
}
```

### 6. 实现其他必需方法

```go
func (f *<Service>Forge) ConfigureRules(ctx *interfaces.ForgeContext) {
    // 配置安全组规则（如果需要）
    // 示例:
    // ctx.SecurityGroups.Default.AddIngressRule(
    //     awsec2.Peer_AnyIpv4(),
    //     awsec2.Port_Tcp(jsii.Number(80)),
    //     jsii.String("Allow HTTP"),
    // )
}

func (f *<Service>Forge) CreateOutputs(ctx *interfaces.ForgeContext) {
    // 创建 CloudFormation 输出
    // 示例:
    // awscdk.NewCfnOutput(ctx.Stack, jsii.String("ResourceEndpoint"), &awscdk.CfnOutputProps{
    //     Value: f.resource.Endpoint(),
    //     Description: jsii.String("资源端点"),
    // })
}

// 如果资源会被用作依赖，实现此方法
func (f *<Service>Forge) GetProperties() map[string]interface{} {
    return f.properties
}
```

### 7. 注册 Forge

在 `registry/registry.go` 中添加：

```go
import (
    "<service>" "github.com/aws-samples/infraforge/forges/aws/<service>"
)

func init() {
    RegisterForge("<service>", func() interfaces.Forge {
        return &<service>.<Service>Forge{}
    })
}
```

## 📋 配置示例

### 8. 配置结构最佳实践

InfraForge 使用结构化配置格式，通过 `defaults` 和 `instances` 部分来减少配置重复：

```json
{
  "global": {
    "stackName": "my-infrastructure",
    "description": "基础设施部署描述"
  },
  "enabledForges": ["instance1", "instance2"],
  "forges": {
    "<service>": {
      "defaults": {
        "type": "SERVICE_TYPE",
        "subnet": "private",
        "security": "private",
        "commonField1": "defaultValue1",
        "commonField2": "defaultValue2",
        "booleanField": true,
        "numericField": 100
      },
      "instances": [
        {
          "id": "instance1"
        },
        {
          "id": "instance2",
          "commonField1": "overrideValue1"
        }
      ]
    }
  }
}
```

**关键原则：**
- **默认配置部分:**  包含所有通用配置参数
- **最小实例配置:**  实例只需要 `id` 和任何覆盖值
- **类型约定:**  服务类型使用大写（如 "RDS", "HYPERPOD", "EC2"）
- **标准值:**  子网/安全组使用 "private", "public", "isolated"

**重要说明：**
- **enabledForges:**  要部署的特定实例 ID 列表（如 `["instance1", "instance2"]`）
- **VPC:**  始终作为基础层自动创建 - 无需在 enabledForges 中指定
- **实例 ID:**  必须在 `enabledForges` 和 `forges` 中的实际实例定义之间匹配

## 🔧 工具函数

### 9. 常用工具函数

```go
// 处理默认值
utils.GetStringValue(&config.Field, "default")
utils.GetIntValue(&config.Field, 0)
utils.GetBoolValue(config.BoolField, false)

// 创建 IAM 角色
role := aws.CreateRole(ctx.Stack, roleId, policies, servicePrincipal)

// 处理依赖
if config.DependsOn != "" {
    magicToken, err := dependency.GetDependencyInfo(config.DependsOn)
    mountPoint, err := dependency.GetMountPoint(config.DependsOn)
    properties, err := dependency.ExtractDependencyProperties(magicToken, resourceType)
}
```

## 📊 常见属性示例

### 存储服务（EFS、FSx）
```go
f.properties["fileSystemId"] = *f.fileSystem.FileSystemId()
f.properties["mountPoint"] = "/mnt/efs"
```

### 数据库服务（RDS、ElastiCache）
```go
f.properties["endpoint"] = *f.database.Endpoint()
f.properties["port"] = *f.database.Port()
f.properties["username"] = username
f.properties["databaseName"] = databaseName
```

### 计算服务（EC2、ECS）
```go
f.properties["instanceId"] = *f.instance.InstanceId()
f.properties["privateIp"] = *f.instance.InstancePrivateIp()
f.properties["publicIp"] = *f.instance.InstancePublicIp()
```

### 网络服务（VPC、SecurityGroup）
```go
f.properties["vpcId"] = *f.vpc.VpcId()
f.properties["securityGroupId"] = *f.securityGroup.SecurityGroupId()
```

## 🧪 测试

### 10. 单元测试模板

```go
package <service>

import (
    "testing"
    "github.com/aws-samples/infraforge/core/config"
)

func TestCreate<Service>(t *testing.T) {
    config := &<Service>InstanceConfig{
        BaseInstanceConfig: config.BaseInstanceConfig{
            ID: "test-<service>",
            Type: "<service>",
        },
        RequiredField: "test-value",
        OptionalField: "test-optional",
    }
    
    forge := &<Service>Forge{}
    
    // 测试配置合并
    defaultConfig := &<Service>InstanceConfig{
        BaseInstanceConfig: config.BaseInstanceConfig{
            ID: "default-<service>",
            Type: "<service>",
        },
        OptionalField: "default-optional",
    }
    
    merged := forge.MergeConfigs(defaultConfig, config).(*<Service>InstanceConfig)
    
    if merged.RequiredField != "test-value" {
        t.Errorf("期望 RequiredField 为 'test-value'，实际为 '%s'", merged.RequiredField)
    }
    
    if merged.OptionalField != "test-optional" {
        t.Errorf("期望 OptionalField 为 'test-optional'，实际为 '%s'", merged.OptionalField)
    }
}

func TestMergeConfigs(t *testing.T) {
    // 测试配置合并逻辑
    // 添加全面的测试用例
}
```

## 🎯 最佳实践

### 11. 开发指导原则

1. **字段类型：** 布尔字段使用 `*bool` 以支持正确的配置合并
2. **默认值：** 使用 `utils.Get*Value()` 函数处理默认值
3. **依赖管理：** 使用 `dependency` 包处理资源依赖
4. **错误处理：** 返回有意义的错误消息和上下文
5. **测试：** 为每个 Forge 编写全面的单元测试
6. **文档：** 为配置字段和使用示例添加清晰的文档
7. **资源命名：** 对 AWS 资源使用一致的命名模式
8. **安全性：** 遵循 AWS 安全最佳实践和最小权限原则

### 12. 常见模式

- **IAM 角色：** 创建具有最小必需权限的服务特定 IAM 角色
- **安全组：** 仅配置必要的入站/出站规则
- **依赖：** 正确处理资源依赖和循环引用
- **属性：** 存储相关资源属性供其他资源使用
- **配置：** 支持必需和可选配置字段
- **验证：** 在资源创建前验证配置参数

## 🚀 部署

### 13. 测试你的 Forge

1. 将你的 forge 添加到配置中的 `enabledForges`
2. 运行 `go build` 编译
3. 使用 `./deploy.sh` 部署
4. 验证资源是否正确创建
5. 测试与其他 forge 的依赖解析

### 14. 集成

完成 forge 后：
- 添加全面的文档
- 创建示例配置
- 添加集成测试
- 提交代码审查
- 更新主 README 以包含新 forge 功能

遵循本指南可确保你的新 Forge 与 InfraForge 框架无缝集成，并遵循既定的模式和最佳实践。
