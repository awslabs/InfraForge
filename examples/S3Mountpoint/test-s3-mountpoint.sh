#!/bin/bash

# S3 Mountpoint CSI Driver 测试脚本
# 使用方法: ./test-s3-mountpoint.sh [test-type]
# test-type: quick | full | cleanup

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查前提条件
check_prerequisites() {
    log_info "检查前提条件..."
    
    # 检查kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl 未安装"
        exit 1
    fi
    
    # 检查集群连接
    if ! kubectl cluster-info &> /dev/null; then
        log_error "无法连接到Kubernetes集群"
        exit 1
    fi
    
    # 检查S3 CSI Driver
    if ! kubectl get pods -n kube-system -l app.kubernetes.io/name=aws-mountpoint-s3-csi-driver | grep -q Running; then
        log_error "S3 CSI Driver未正常运行"
        exit 1
    fi
    
    # 检查存储资源
    if ! kubectl get pvc s3-sc-pvc &> /dev/null; then
        log_error "S3 PVC (s3-sc-pvc) 不存在"
        exit 1
    fi
    
    log_success "前提条件检查通过"
}

# 部署测试Pod
deploy_test_pod() {
    log_info "部署测试Pod..."
    
    if kubectl get pod s3-test-pod &> /dev/null; then
        log_warning "测试Pod已存在，删除旧的Pod..."
        kubectl delete pod s3-test-pod --ignore-not-found=true
        sleep 10
    fi
    
    kubectl apply -f s3-mountpoint-test-pod.yaml
    
    log_info "等待Pod启动..."
    kubectl wait --for=condition=Ready pod/s3-test-pod --timeout=300s
    
    log_success "测试Pod部署成功"
}

# 快速测试
quick_test() {
    log_info "执行快速测试..."
    
    # 检查挂载点
    log_info "检查S3挂载点..."
    kubectl exec s3-test-pod -- df -h | grep s3 || {
        log_error "S3未正确挂载"
        return 1
    }
    
    # 测试文件读取
    log_info "测试文件读取..."
    kubectl exec s3-test-pod -- ls -la /mnt/s3/ > /dev/null || {
        log_error "无法读取S3内容"
        return 1
    }
    
    # 测试文件写入
    log_info "测试文件写入..."
    TIMESTAMP=$(date +%Y%m%d-%H%M%S)
    kubectl exec s3-test-pod -- bash -c "echo 'Quick test - $TIMESTAMP' > /mnt/s3/quick-test-$TIMESTAMP.txt" || {
        log_warning "文件写入失败（可能是只读挂载）"
    }
    
    log_success "快速测试完成"
}

# 完整测试
full_test() {
    log_info "执行完整测试..."
    
    # 运行测试Job
    log_info "运行自动化测试Job..."
    kubectl delete job s3-mountpoint-test-job --ignore-not-found=true
    kubectl apply -f s3-mountpoint-test-pod.yaml
    
    # 等待Job完成
    log_info "等待测试Job完成..."
    kubectl wait --for=condition=complete job/s3-mountpoint-test-job --timeout=300s
    
    # 显示测试结果
    log_info "测试结果:"
    kubectl logs job/s3-mountpoint-test-job
    
    log_success "完整测试完成"
}

# 清理资源
cleanup() {
    log_info "清理测试资源..."
    
    kubectl delete pod s3-test-pod --ignore-not-found=true
    kubectl delete job s3-mountpoint-test-job --ignore-not-found=true
    
    log_success "清理完成"
}

# 显示状态
show_status() {
    log_info "S3 Mountpoint CSI Driver 状态:"
    
    echo "=== S3 CSI Driver Pods ==="
    kubectl get pods -n kube-system -l app.kubernetes.io/name=aws-mountpoint-s3-csi-driver
    
    echo -e "\n=== Mountpoint Pods ==="
    kubectl get pods -n mount-s3 2>/dev/null || echo "无mountpoint pods"
    
    echo -e "\n=== 存储资源 ==="
    kubectl get sc,pv,pvc | grep s3
    
    echo -e "\n=== 测试Pod ==="
    kubectl get pod s3-test-pod 2>/dev/null || echo "无测试Pod"
}

# 主函数
main() {
    local test_type=${1:-quick}
    
    echo "🚀 S3 Mountpoint CSI Driver 测试工具"
    echo "测试类型: $test_type"
    echo "=================================="
    
    case $test_type in
        "quick")
            check_prerequisites
            deploy_test_pod
            quick_test
            show_status
            ;;
        "full")
            check_prerequisites
            deploy_test_pod
            quick_test
            full_test
            show_status
            ;;
        "cleanup")
            cleanup
            ;;
        "status")
            show_status
            ;;
        *)
            echo "使用方法: $0 [quick|full|cleanup|status]"
            echo "  quick   - 快速测试（默认）"
            echo "  full    - 完整测试"
            echo "  cleanup - 清理资源"
            echo "  status  - 显示状态"
            exit 1
            ;;
    esac
    
    log_success "测试完成！"
}

# 执行主函数
main "$@"
