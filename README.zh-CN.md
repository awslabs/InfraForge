# InfraForge - AWS 基础设施即配置框架

[English](README.md) | [中文](README.zh-CN.md)

InfraForge 是一个创新的基础设施即配置（IaC）框架，彻底改变了组织部署和管理 AWS 资源的方式。该企业级解决方案基于 AWS CDK 和 Go 构建，通过其模块化的"forge"组件系统将复杂的云架构转换为简单的 JSON 配置。

## 项目概述

InfraForge 允许您使用配置驱动的方法定义、部署和管理复杂的 AWS 基础设施。该框架抽象了 AWS CloudFormation 和 CDK 的复杂性，为常见的基础设施模式提供了更高级的接口。

该系统被设计为一个支持任何 AWS 服务和解决方案的综合平台，通过其模块化架构允许持续优化和增强。InfraForge 通过实现快速部署、成本降低和运营简化来提供显著的商业价值。

## 架构

![InfraForge 架构](docs/architecture.svg)

InfraForge 架构由四个关键组件组成：

- **🤖 Amazon Q CLI：** 自然语言理解和意图处理
- **🧠 MCP 服务器：** 自学习解决方案发现和智能指导生成
- **⚙️ InfraForge 引擎：** 模块化部署编排和依赖管理
- **📋 解决方案模板：** 配置驱动的基础设施模式和最佳实践

### 主要特点

- **模块化架构：** 基础设施组件被组织为可以组合在一起的"forges"
- **配置驱动：** 通过 JSON 配置文件定义基础设施
- **多资源支持：** 支持各种 AWS 服务，包括 DS、EC2、ECS、EKS、EFS、FSx Lustre 等
- **跨堆栈引用：** 资源可以相互引用和依赖
- **灵活的部署选项：** 部署整个堆栈或单个组件
- **Amazon Q 集成：** 可选的 MCP 服务器，用于对话式基础设施管理

## 项目结构

```
InfraForge/
├── cmd/                  # 命令行界面代码
│   └── infraforge/       # 主要 CLI 应用程序和可执行文件
├── configs/              # 解决方案特定的配置文件
│   ├── batch/            # AWS Batch 解决方案
│   ├── bench/            # 基准测试解决方案
│   ├── directoryservice/ # 目录服务解决方案
│   ├── ec2/              # EC2 解决方案
│   ├── ecs/              # ECS 解决方案
│   ├── eks/              # EKS 解决方案
│   ├── enclave/          # Enclave 解决方案
│   ├── hyperpod/         # SageMaker HyperPod 解决方案
│   ├── kafka/            # Kafka 解决方案
│   ├── kudu/             # Kudu 解决方案
│   ├── kubernetes/       # Kubernetes 解决方案
│   ├── netbench/         # 网络基准测试解决方案
│   ├── parallelcluster/  # AWS ParallelCluster 解决方案
│   ├── rds/              # RDS 解决方案
│   ├── redroid/          # Redroid 解决方案
│   └── web3/             # Web3 解决方案
├── core/                 # 核心框架功能
├── forges/               # 基础设施组件实现
│   ├── aws/              # AWS 特定的 forge 实现
│   │   ├── batch/        # AWS Batch forges
│   │   ├── ds/           # 目录服务 forges
│   │   ├── ec2/          # EC2 实例 forges
│   │   ├── ecs/          # ECS 集群和服务 forges
│   │   ├── eks/          # EKS 集群 forges
│   │   ├── hyperpod/     # SageMaker HyperPod forges
│   │   ├── iam/          # IAM 角色和策略 forges
│   │   ├── lambda/       # Lambda 函数 forges
│   │   ├── parallelcluster/ # AWS ParallelCluster forges
│   │   ├── rds/          # RDS 数据库 forges
│   │   ├── storage/      # 存储相关 forges（EFS、FSx）
│   │   └── vpc/          # VPC 和网络 forges
│   ├── desktop/          # 桌面环境 forges
│   ├── kubernetes/       # Kubernetes 相关 forges
│   └── monitoring/       # 监控和可观测性 forges
├── registry/             # Forge 注册表和管理
├── scripts/              # 用户数据脚本和模板
├── tools/                # 实用工具和脚本
├── docs/                 # 文档
├── examples/             # 示例配置和用法
└── tests/                # 测试用例和测试实用程序
```

## 配置

InfraForge 使用 JSON 配置文件来定义基础设施。主要配置文件是 `config.json`，它定义了：

- 全局设置，如堆栈名称
- 要部署的已启用 forges
- 不同 forge 类型的资源配置

特定解决方案的配置存储在 `configs/` 目录中，命名约定为 `config_<solution>.json`。要使用特定的解决方案配置，请在部署前将其复制到 `cmd/infraforge/config.json`。

配置结构示例：

```json
{
    "global": {
        "stackName": "aws-infra-forge",
        "dualStack": true
    },
    "enabledForges": [
        "efs1",
        "ecs1",
        "ds1",
        "windows2022",
        "ubuntu2204",
        "al2023"
    ],
    "forges": {
        "vpc": { ... },
        "ds":  { ... },
        "efs": { ... },
        "lustre": { ... },
        "ecs": { ... },
        "eks": { ... },
        "ec2": { ... }
    }
}
```

## 支持的 Forge 类型

InfraForge 目前支持以下 forge 类型：

- **VPC：** 具有公共、私有和隔离子网的网络基础设施
- **EC2：** 具有各种操作系统选项的虚拟机（Amazon Linux、Ubuntu、Windows、CentOS）
- **ECS：** 具有 Fargate 和 EC2 启动类型的容器编排
- **EKS：** 托管 Kubernetes 集群，支持 Karpenter
- **AWS Batch：** 托管批处理计算服务
- **AWS ParallelCluster：** HPC 集群管理
- **SageMaker HyperPod：** 分布式机器学习训练
- **RDS：** 托管关系数据库服务
- **EFS：** 用于共享存储的弹性文件系统
- **FSx Lustre：** 用于计算工作负载的高性能文件系统
- **Lambda：** 无服务器函数
- **IAM：** 身份和访问管理资源
- **Directory Service：** 托管 Microsoft Active Directory

## 入门指南

### 先决条件

- Go 1.23 或更高版本
- AWS CDK CLI
- 配置了适当凭证的 AWS CLI
- Node.js（CDK 所需）

### 安装

1. 克隆仓库：
   ```
   git clone https://github.com/awslabs/InfraForge.git
   cd InfraForge
   ```

2. 安装依赖项：
   ```
   go mod download
   ```

3. 构建应用程序：
   ```
   cd cmd/infraforge
   go build
   ```

### 使用方法

1. 从 `configs/` 目录中选择一个解决方案配置：
   ```
   # 例如，使用 ParallelCluster 解决方案：
   cp configs/parallelcluster/config_parallelcluster.json cmd/infraforge/config.json
   
   # 或者使用基准测试解决方案：
   cp configs/bench/config_bench.json cmd/infraforge/config.json
   ```

2. 运行引导脚本：
   ```
   ./bootstrap.sh
   ```

3. 部署您的基础设施：
   ```
   cd cmd/infraforge
   ./deploy.sh
   ```

4. 销毁基础设施：
   ```
   ./destroy.sh
   ```

## Amazon Q 集成（可选）

使用 Amazon Q 进行对话式基础设施管理：

1. 构建 MCP 服务器：
   ```bash
   cd tools/mcp/
   go build
   sudo cp infraforge_mcp_server /usr/local/bin/
   sudo chmod +x /usr/local/bin/infraforge_mcp_server
   ```

2. 将 MCP 服务器添加到 Q CLI：
   ```bash
   q mcp add --force --name infraforge --command infraforge_mcp_server --timeout 7200000
   ```

3. 准备工作目录：
   ```bash
   cd cmd/infraforge
   cp -r ../../configs .
   ```

4. 启动带有 InfraForge 工具的 Amazon Q Chat：
   ```bash
   q chat --trust-tools=fs_read,@infraforge/getDeploymentStatus,@infraforge/getStackOutputs,@infraforge/getOperationManual,@infraforge/listTemplates
   ```

4. 使用对话命令：
   ```
   > 列出可用模板
   > 部署 ParallelCluster 集群
   > 检查部署状态
   ```

详细使用方法请参见[用户指南](docs/user-guide.md)。

## 管理解决方案配置

InfraForge 支持在 `configs/` 目录中按类别组织的多个解决方案配置：

- `batch/`：AWS Batch 解决方案
- `bench/`：基准测试解决方案
- `directoryservice/`：目录服务解决方案
- `ec2/`：EC2 解决方案
- `ecs/`：ECS 解决方案
- `eks/`：EKS 解决方案
- `enclave/`：Enclave 解决方案
- `hyperpod/`：SageMaker HyperPod 解决方案
- `kafka/`：Kafka 解决方案
- `kudu/`：Kudu 解决方案
- `kubernetes/`：Kubernetes 解决方案
- `netbench/`：网络基准测试解决方案
- `parallelcluster/`：AWS ParallelCluster 解决方案
- `rds/`：RDS 解决方案
- `redroid/`：Redroid 解决方案
- `web3/`：Web3 解决方案

创建新解决方案：
1. 在 `configs/` 中确定适当的类别目录或创建一个新目录
2. 创建一个名为 `config_<solution>.json` 的新配置文件
3. 复制并修改现有配置或从头开始
4. 要部署，请将您的解决方案配置复制到 `cmd/infraforge/config.json`

## 有用的命令

- `cdk deploy`：将堆栈部署到您的默认 AWS 账户/区域
- `cdk diff`：比较已部署的堆栈与当前状态
- `cdk synth`：生成合成的 CloudFormation 模板
- `go test`：运行单元测试
- `cdk --app ./infraforge deploy`：使用 InfraForge CDK 应用程序部署

## 贡献

欢迎贡献！请随时提交拉取请求。

## 许可证

该项目根据 [Apache License 2.0](LICENSE) 许可 - 有关详细信息，请参阅 LICENSE 文件。
