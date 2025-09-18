# InfraForge 用户指南

## 🚀 安装

### 前置要求
- Go 1.23 或更高版本
- 已配置适当凭证的 AWS CLI
- AWS CDK CLI 已安装

### 安装步骤

1. **解压并构建**
   ```bash
   tar xvf InfraForge.tar.gz
   cd InfraForge/cmd/infraforge
   go build
   chmod +x infraforge
   cp -a ../../configs .
   ```

2. **可选：构建 MCP 服务器用于 Amazon Q 集成**
   ```bash
   cd ../../tools/mcp/
   go build
   sudo cp infraforge_mcp_server /usr/local/bin/
   sudo chmod +x /usr/local/bin/infraforge_mcp_server
   
   # 将 MCP 服务器添加到 Q CLI
   q mcp add --force --name infraforge --command infraforge_mcp_server --timeout 7200000
   ```

## 🛠️ 基本使用

### 1. 初始化 CDK 环境
```bash
# 复制所需配置
cp configs/parallelcluster/config_parallelcluster.json config.json

# 引导 CDK（每个区域一次性设置）
cdk bootstrap --app ./infraforge --force --require-approval never
```

### 2. 部署基础设施
```bash
# 使用当前配置部署
cdk deploy --app ./infraforge
```

### 3. 更新基础设施
```bash
# 根据需要修改 config.json，然后重新部署
cdk deploy --app ./infraforge
```

### 4. 销毁基础设施
```bash
cdk destroy --app ./infraforge
```

## 💬 可选：Amazon Q Chat 集成

如果您想使用 InfraForge 与 Amazon Q Chat 进行对话式基础设施管理：

### 1. 初始化环境
```bash
./bootstrap.sh
```

### 2. 启动带有 InfraForge 工具的 Amazon Q Chat
```bash
q chat --trust-tools=fs_read,@infraforge/getDeploymentStatus,@infraforge/getStackOutputs,@infraforge/getOperationManual,@infraforge/listTemplates
```

### 3. 可用工具
连接后，您将可以访问这些 InfraForge 工具：

| 工具 | 权限 | 描述 |
|------|------|------|
| `@infraforge/deployInfra` | 不受信任 | 从模板部署基础设施 |
| `@infraforge/getDeploymentStatus` | 受信任 | 检查部署状态 |
| `@infraforge/getOperationManual` | 受信任 | 获取操作说明 |
| `@infraforge/getStackOutputs` | 受信任 | 检索堆栈输出 |
| `@infraforge/listTemplates` | 受信任 | 列出可用配置模板 |

### 4. Amazon Q 使用示例

**性能测试设置：**
```
帮我启动 c6i.xlarge 和 c7g.xlarge 实例来运行性能测试，
测试模块包括 c2clat、pts，其中 pts 包含子模块 stream、
mbw、byte 和 stress-ng。测试后，将结果保存到 s3://aws-infra-forge。
```

**Amazon Q 将会：**
1. 使用 `listTemplates` 列出可用模板
2. 使用 `getOperationManual` 获取操作手册
3. 使用 `fs_read` 读取模板配置
4. 使用 `deployInfra` 部署定制化基础设施

## 📋 可用模板

InfraForge 包含针对各种用例的预配置模板：

### 基准测试
- `configs/bench/config_bench.json` - 通用基准测试设置
- `configs/bench/config_sysbench.json` - 系统基准测试，包含 c2clat、pts 模块

### 高性能计算
- `configs/parallelcluster/config_parallelcluster.json` - AWS ParallelCluster 设置

### 容器计算
- `configs/batch/config_batch.json` - AWS Batch 容器工作负载

### 专业工作负载
- `configs/kudu/config_kudu.json` - Apache Kudu 部署
- `configs/enclave/config_enclave.json` - AWS Nitro Enclaves
- `configs/web3/config_agave.json` - Web3 区块链节点

### 网络
- `configs/netbench/config_netbench.json` - 网络性能测试
- `configs/netbench/config_locust_redis.json` - Redis 负载测试

## 🔧 配置定制

### 基本结构
```json
{
    "global": {
        "stackName": "my-infrastructure",
        "description": "自定义基础设施部署"
    },
    "enabledForges": ["ec2instance1", "efs1", "batch1"],
    "forges": {
        "ec2": {
            "instances": [
                {
                    "id": "ec2instance1",
                    "instanceType": "c6i.xlarge",
                    "userDataToken": "sysbench:modules=c2clat"
                }
            ]
        },
        "efs": {
            "instances": [
                {
                    "id": "efs1"
                }
            ]
        }
    }
}
```

### 关键配置要点

- **enabledForges: ** 要部署的特定实例 ID 列表（如 `["ec2instance1", "efs1", "batch1"]`）
- **VPC: ** 始终作为基础层自动创建 - 无需在 enabledForges 中指定
- **实例 ID: ** 必须在 `enabledForges` 和 `forges` 中的实际实例定义之间匹配

### 常用参数
- **instanceType: ** EC2 实例类型（如 `c6i.xlarge`、`c7g.xlarge`）
- **userDataToken: ** 自动软件安装和配置
- **dependsOn: ** 资源依赖（如 `"EFS:efs1,LUSTRE:lustre1"`）

## 📊 监控和输出

### 检查部署状态
```bash
# 手动检查
aws cloudformation describe-stacks --stack-name aws-infra-forge

# 或通过 Amazon Q Chat（如果使用集成）
> 检查部署状态
```

### 获取堆栈输出
```bash
# 手动检查
aws cloudformation describe-stacks --stack-name aws-infra-forge --query 'Stacks[0].Outputs'

# 或通过 Amazon Q Chat（如果使用集成）
> 显示堆栈输出
```

## 🔍 故障排除

### 常见问题

1. **需要 CDK 引导**
   ```bash
   cdk bootstrap --app ./infraforge --force --require-approval never
   ```

2. **权限被拒绝**
   ```bash
   chmod +x infraforge
   ```

3. **AWS 凭证**
   ```bash
   aws configure
   # 或
   export AWS_PROFILE=your-profile
   ```

### 获取帮助
- 查看 `configs/` 目录中的可用模板
- 检查 AWS CloudFormation 控制台了解部署详情
- 使用 Amazon Q Chat 集成获得对话式帮助（可选）

## 🎯 最佳实践

1. **从模板开始: ** 使用现有模板作为起点
2. **小规模测试: ** 首先部署简单配置
3. **监控资源: ** 检查 AWS 成本和资源使用情况
4. **及时清理: ** 测试完成后销毁资源
5. **版本控制: ** 将配置文件保存在版本控制中

## 📚 下一步

- 探索 `configs/` 目录中的可用模板
- 为您的特定用例定制配置
- 与 CI/CD 管道集成以实现自动化部署
- 尝试可选的 Amazon Q Chat 集成进行对话式基础设施管理
- 为项目贡献新模板和 forge 类型
