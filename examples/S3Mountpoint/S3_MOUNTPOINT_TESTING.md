# S3 Mountpoint CSI Driver 测试指南

本文档提供了测试S3 Mountpoint CSI Driver功能的详细指南。

## 前提条件

1. EKS集群已部署并配置了S3 Mountpoint CSI Driver
2. 存在以下资源：
   - StorageClass: `s3-sc`
   - PersistentVolume: `s3-sc-pv`
   - PersistentVolumeClaim: `s3-sc-pvc`

## 快速测试

### 1. 部署测试Pod

```bash
kubectl apply -f examples/s3-mountpoint-test-pod.yaml
```

### 2. 检查Pod状态

```bash
# 检查Pod状态
kubectl get pods s3-test-pod -o wide

# 检查mountpoint pod
kubectl get pods -n mount-s3
```

### 3. 进入Pod进行测试

```bash
kubectl exec -it s3-test-pod -- /bin/bash
```

### 4. 在Pod内执行测试命令

```bash
# 检查挂载点
df -h | grep s3

# 查看S3内容
ls -la /mnt/s3/

# 测试文件读取
cat /mnt/s3/test-file.txt

# 测试文件写入
echo "Hello from K8s - $(date)" > /mnt/s3/k8s-test-$(date +%s).txt

# 验证写入
ls -la /mnt/s3/k8s-test-*
```

## 自动化测试

### 运行测试Job

```bash
# 运行自动化测试Job
kubectl apply -f examples/s3-mountpoint-test-pod.yaml

# 查看Job状态
kubectl get jobs s3-mountpoint-test-job

# 查看测试结果
kubectl logs job/s3-mountpoint-test-job
```

## 故障排除

### 1. Pod无法启动

```bash
# 检查Pod事件
kubectl describe pod s3-test-pod

# 检查S3 CSI Driver状态
kubectl get pods -n kube-system -l app.kubernetes.io/name=aws-mountpoint-s3-csi-driver
```

### 2. 挂载失败

```bash
# 检查S3 CSI Controller日志
kubectl logs -n kube-system -l app=s3-csi-controller

# 检查mountpoint pod日志
kubectl logs -n mount-s3 <mountpoint-pod-name>
```

### 3. 权限问题

```bash
# 检查ServiceAccount注解
kubectl get serviceaccount -n kube-system s3-csi-driver-sa -o yaml

# 检查RBAC权限
kubectl get clusterrole s3-csi-driver-additional-permissions
kubectl get clusterrolebinding s3-csi-driver-additional-binding
```

## 性能测试

### 基本性能测试

```bash
kubectl exec -it s3-test-pod -- /bin/bash -c "
# 目录列表性能
time ls -la /mnt/s3/ > /dev/null

# 文件读取性能
time cat /mnt/s3/large-file.txt > /dev/null

# 文件写入性能
time dd if=/dev/zero of=/mnt/s3/test-1mb.dat bs=1M count=1
"
```

## 清理资源

```bash
# 删除测试Pod
kubectl delete pod s3-test-pod

# 删除测试Job
kubectl delete job s3-mountpoint-test-job

# 清理测试文件（可选）
kubectl run cleanup --rm -i --tty --image=amazonlinux:2023 -- /bin/bash -c "
  # 这里需要手动挂载S3并清理测试文件
"
```

## 常见测试场景

### 1. 多Pod共享测试

创建多个Pod使用同一个PVC，验证共享访问：

```bash
# 创建第二个测试Pod
kubectl run s3-test-pod-2 --image=amazonlinux:2023 --command -- sleep 3600
kubectl patch pod s3-test-pod-2 -p '{"spec":{"volumes":[{"name":"s3-volume","persistentVolumeClaim":{"claimName":"s3-sc-pvc"}}],"containers":[{"name":"s3-test-pod-2","volumeMounts":[{"name":"s3-volume","mountPath":"/mnt/s3"}]}]}}'
```

### 2. 大文件传输测试

```bash
kubectl exec -it s3-test-pod -- /bin/bash -c "
# 创建大文件
dd if=/dev/zero of=/mnt/s3/large-test-file.dat bs=1M count=100

# 验证文件
ls -lh /mnt/s3/large-test-file.dat
"
```

### 3. 并发访问测试

```bash
kubectl exec -it s3-test-pod -- /bin/bash -c "
# 并发写入测试
for i in {1..10}; do
  echo 'Concurrent test $i - $(date)' > /mnt/s3/concurrent-test-$i.txt &
done
wait
ls -la /mnt/s3/concurrent-test-*
"
```

## 监控和日志

### 查看相关日志

```bash
# S3 CSI Controller日志
kubectl logs -n kube-system -l app=s3-csi-controller -f

# S3 CSI Node日志
kubectl logs -n kube-system -l app=s3-csi-node -f

# Mountpoint Pod日志
kubectl logs -n mount-s3 -l app=mountpoint-s3 -f
```

### 监控指标

```bash
# 检查Pod资源使用
kubectl top pods s3-test-pod

# 检查节点资源使用
kubectl top nodes
```

## 注意事项

1. **写入权限: ** 确保S3存储桶配置允许写入操作
2. **网络延迟: ** S3访问可能有网络延迟，这是正常的
3. **缓存行为: ** Mountpoint S3有自己的缓存机制
4. **并发限制: ** 注意S3的并发访问限制
5. **成本考虑: ** 频繁的S3操作会产生费用

## 支持的操作

✅ **支持的操作:** 
- 文件读取
- 文件写入
- 目录创建
- 文件删除
- 目录列表

❌ **不支持的操作:** 
- 文件权限修改
- 符号链接
- 硬链接
- 某些POSIX文件系统特性
