# S3 Mountpoint fio 性能测试指南

本文档详细介绍如何使用fio对S3 Mountpoint进行性能测试。

## 概述

S3 Mountpoint支持fio测试，但由于其基于网络的特性，性能特征与本地存储有所不同：

### ✅ **适合的测试场景**
- 大文件顺序读写
- 批量数据传输
- 并发访问测试
- 吞吐量测试

### ⚠️ **需要注意的特性**
- 网络延迟影响小文件随机访问
- 写入操作需要上传到S3
- 缓存机制影响重复读取
- 并发限制由S3服务决定

## 快速开始

### 1. 基础性能测试
```bash
./examples/test-s3-fio.sh basic
```

### 2. 高级性能测试
```bash
./examples/test-s3-fio.sh advanced
```

### 3. 交互式测试
```bash
./examples/test-s3-fio.sh interactive
```

## 测试类型详解

### 基础测试 (basic)

包含以下测试项目：
- **顺序写入:**  1GB文件，1MB块大小
- **顺序读取:**  读取写入的文件
- **随机写入:**  100MB文件，4K块大小
- **随机读取:**  随机读取写入的文件

**适用场景:**  快速了解S3 Mountpoint基本性能

### 高级测试 (advanced)

包含以下测试项目：
- **吞吐量测试:**  大块顺序写入 (16MB块)
- **延迟测试:**  小块随机读取 (4K块)
- **并发测试:**  多线程混合读写

**适用场景:**  深入了解不同负载下的性能表现

### 压力测试 (stress)

包含以下测试项目：
- **长时间写入:**  10分钟持续写入
- **高并发混合:**  8线程混合读写，5分钟

**适用场景:**  测试系统稳定性和持续性能

### 自动化基准测试 (job)

运行完整的自动化测试Job，包括：
- 系统信息收集
- 多种测试场景
- 自动清理

**适用场景:**  无人值守的性能基准测试

## 自定义fio配置

### 创建自定义配置文件

```ini
# my-s3-test.fio
[global]
ioengine=sync
direct=0
directory=/mnt/s3/fio-test
group_reporting=1

[large-file-test]
rw=write
bs=32M
size=5G
numjobs=1

[small-file-test]
rw=randwrite
bs=4K
size=100M
numjobs=4
```

### 运行自定义测试

```bash
./examples/test-s3-fio.sh custom my-s3-test.fio
```

## 性能优化建议

### 1. 块大小优化

```bash
# 大文件传输 - 使用大块
fio --bs=16M --rw=write --size=1G

# 小文件操作 - 使用小块
fio --bs=4K --rw=randwrite --size=100M
```

### 2. 并发优化

```bash
# 适度并发 (2-8个job)
fio --numjobs=4 --rw=write

# 避免过度并发 (可能导致S3限流)
```

### 3. 测试模式选择

```bash
# 顺序访问 - 最佳性能
fio --rw=write  # 或 --rw=read

# 随机访问 - 受网络延迟影响
fio --rw=randwrite  # 或 --rw=randread

# 混合访问 - 模拟真实负载
fio --rw=randrw --rwmixread=70
```

## 性能基准参考

### 典型性能指标

| 测试类型 | 块大小 | 预期吞吐量 | 预期延迟 |
|---------|--------|------------|----------|
| 顺序写入 | 1MB | 100-500 MB/s | 低 |
| 顺序读取 | 1MB | 200-800 MB/s | 低 |
| 随机写入 | 4K | 1-10 MB/s | 高 |
| 随机读取 | 4K | 5-50 MB/s | 中等 |

*注意: 实际性能取决于网络条件、S3区域、实例类型等因素*

### 影响性能的因素

1. **网络带宽:**  限制最大吞吐量
2. **S3区域:**  延迟和吞吐量差异
3. **实例类型:**  CPU和网络性能
4. **并发数:**  过多可能导致限流
5. **块大小:**  影响效率和延迟

## 故障排除

### 常见问题

#### 1. fio安装失败
```bash
# 手动安装
kubectl exec s3-fio-test-pod -- yum install -y fio
```

#### 2. 权限错误
```bash
# 检查S3挂载权限
kubectl exec s3-fio-test-pod -- ls -la /mnt/s3/
```

#### 3. 性能异常低
```bash
# 检查网络连接
kubectl exec s3-fio-test-pod -- ping -c 3 s3.amazonaws.com

# 检查S3区域配置
kubectl exec s3-fio-test-pod -- env | grep AWS
```

#### 4. 测试文件残留
```bash
# 清理测试文件
kubectl exec s3-fio-test-pod -- rm -rf /mnt/s3/fio-test/*
```

### 调试命令

```bash
# 检查Pod状态
kubectl describe pod s3-fio-test-pod

# 查看mountpoint pod日志
kubectl logs -n mount-s3 -l app=mountpoint-s3

# 检查S3挂载状态
kubectl exec s3-fio-test-pod -- df -h /mnt/s3
```

## 高级用法

### 1. 多节点并发测试

```yaml
# 创建多个测试Pod在不同节点
apiVersion: apps/v1
kind: Deployment
metadata:
  name: s3-fio-multi-node
spec:
  replicas: 3
  selector:
    matchLabels:
      app: s3-fio-multi
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app: s3-fio-multi
            topologyKey: kubernetes.io/hostname
```

### 2. 长期性能监控

```bash
# 运行长期测试并记录结果
kubectl exec s3-fio-test-pod -- bash -c "
while true; do
  echo '=== $(date) ===' >> /mnt/s3/perf-log.txt
  fio --name=monitor --rw=write --bs=1M --size=100M --filename=/mnt/s3/monitor-test --output-format=json >> /mnt/s3/perf-log.txt
  sleep 300
done
"
```

### 3. 与其他存储对比

```bash
# 同时测试EFS和S3性能
kubectl exec s3-fio-test-pod -- bash -c "
echo 'S3 Mountpoint:'
fio --name=s3-test --rw=write --bs=1M --size=1G --filename=/mnt/s3/test

echo 'EFS:'
fio --name=efs-test --rw=write --bs=1M --size=1G --filename=/mnt/efs/test
"
```

## 最佳实践

1. **测试前预热:**  先进行小规模测试预热缓存
2. **合理设置大小:**  避免创建过大的测试文件
3. **监控资源使用:**  关注CPU、内存、网络使用情况
4. **清理测试数据:**  及时清理避免S3存储费用
5. **记录测试条件:**  记录网络、实例类型等环境信息

## 成本考虑

- **S3请求费用:**  大量小文件操作会产生请求费用
- **数据传输费用:**  跨区域传输可能产生费用
- **存储费用:**  测试文件会占用S3存储空间
- **计算费用:**  长时间测试会消耗EKS节点资源

建议在测试后及时清理测试数据，并选择合适的测试规模。
