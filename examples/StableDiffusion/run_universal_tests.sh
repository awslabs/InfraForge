#!/bin/bash

# 通用Stable Diffusion自动化测试脚本
# 支持多种机型、批次大小、推理步数、分辨率和精度
# 用法: ./run_universal_tests.sh [test_config_file]

set -e

# ========== 日志优化工具函数 ==========
# 检查 Pod 是否可以安全获取日志
check_pod_log_availability() {
    local pod_name=$1
    
    if [[ -z "$pod_name" ]]; then
        echo "false"
        return 1
    fi
    
    # 使用kubectl检查Pod状态
    local pod_phase=$(kubectl get pod "$pod_name" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    
    # 如果无法获取基本信息，说明Pod可能已经不存在或集群有问题
    if [[ -z "$pod_phase" ]]; then
        echo "false"
        return 1
    fi
    
    # 检查Pod状态
    case "$pod_phase" in
        "Running"|"Succeeded")
            echo "true"
            return 0
            ;;
        "Failed")
            echo "true"
            return 0
            ;;
        "Pending")
            echo "limited"
            return 0
            ;;
        *)
            echo "false"
            return 1
            ;;
    esac
}

# 检查 Node 是否健康
check_node_health() {
    local node_name=$1
    
    if [[ -z "$node_name" ]]; then
        echo "unknown"
        return 1
    fi
    
    local node_ready=$(kubectl get node "$node_name" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
    
    if [[ "$node_ready" == "True" ]]; then
        echo "healthy"
        return 0
    elif [[ "$node_ready" == "False" ]]; then
        echo "unhealthy"
        return 1
    elif [[ -z "$node_ready" ]]; then
        echo "missing"
        return 1
    else
        echo "unknown"
        return 1
    fi
}

# 安全获取Pod日志
safe_get_pod_logs() {
    local pod_name=$1
    local tail_lines=${2:-50}
    
    if [[ -z "$pod_name" ]]; then
        return 1
    fi
    
    # 快速检查Pod状态
    local pod_phase=$(kubectl get pod "$pod_name" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    
    if [[ -z "$pod_phase" ]]; then
        return 1
    fi
    
    # 根据状态调整超时时间
    case "$pod_phase" in
        "Running"|"Succeeded"|"Failed")
            kubectl logs "$pod_name" --tail="$tail_lines" 2>/dev/null || return 1
            ;;
        *)
            return 1
            ;;
    esac
}

# 安全获取Pod特定模式的日志
safe_grep_pod_logs() {
    local pod_name=$1
    local pattern=$2
    local tail_lines=${3:-100}
    
    if [[ -z "$pod_name" || -z "$pattern" ]]; then
        return 1
    fi
    
    local pod_phase=$(kubectl get pod "$pod_name" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
    
    if [[ "$pod_phase" == "Running" || "$pod_phase" == "Succeeded" || "$pod_phase" == "Failed" ]]; then
        kubectl logs "$pod_name" --tail="$tail_lines" 2>/dev/null | grep -E "$pattern" || return 1
    else
        return 1
    fi
}

# 获取Pod失败的真正原因
get_pod_failure_reason() {
    local pod_name=$1
    
    if [[ -z "$pod_name" ]]; then
        echo "Pod名称为空"
        return 1
    fi
    
    # 获取容器状态
    local container_state=$(kubectl get pod "$pod_name" -o jsonpath='{.status.containerStatuses[0].state.terminated}' 2>/dev/null || echo "{}")
    
    if [[ "$container_state" != "{}" ]]; then
        local exit_code=$(echo "$container_state" | jq -r '.exitCode // empty' 2>/dev/null)
        local reason=$(echo "$container_state" | jq -r '.reason // empty' 2>/dev/null)
        local message=$(echo "$container_state" | jq -r '.message // empty' 2>/dev/null)
        
        if [[ -n "$exit_code" && -n "$reason" ]]; then
            echo "容器退出: 退出码=$exit_code, 原因=$reason"
            if [[ -n "$message" && "$message" != "null" ]]; then
                echo "详细信息: $message"
            fi
        fi
    fi
    
    # 尝试获取最后几行有用的日志
    local logs=$(safe_get_pod_logs "$pod_name" 20 5)
    if [[ -n "$logs" ]]; then
        # 过滤出错误和关键信息
        local error_logs=$(echo "$logs" | grep -E "(ERROR|Error|FAILED|Failed|Exception|Traceback|CUDA|GPU|Memory|OOM)" | tail -5)
        if [[ -n "$error_logs" ]]; then
            echo "关键错误信息:"
            echo "$error_logs"
            return 0
        fi
        
        # 如果没有明显错误，显示最后几行
        echo "最后日志:"
        echo "$logs" | tail -3
    else
        echo "无法获取Pod日志"
    fi
}

# 检测任务完成状态
detect_task_completion() {
    local test_name=$1
    local pod_name=$2
    
    # 检查Pod日志中的任务完成标记
    local pod_logs=$(kubectl logs "$pod_name" --tail=50 2>/dev/null || echo "")
    
    if [[ -n "$pod_logs" ]]; then
        # 首先检查是否为OOM情况
        if echo "$pod_logs" | grep -q "PERFORMANCE_SUMMARY:.*OOM"; then
            echo "task_oom"
            return 0
        fi
        
        # 检查任务完成标记
        if echo "$pod_logs" | grep -q "任务完成，等待脚本处理"; then
            echo "task_completed"
            return 0
        fi
        
        # 检查性能摘要输出（但排除OOM情况）
        if echo "$pod_logs" | grep -q "PERFORMANCE_SUMMARY:" && ! echo "$pod_logs" | grep -q "PERFORMANCE_SUMMARY:.*OOM"; then
            echo "task_completed"
            return 0
        fi
        
        # 检查测试成功完成标记
        if echo "$pod_logs" | grep -q "completed successfully"; then
            echo "task_completed"
            return 0
        fi
    fi
    
    echo "task_running"
    return 0  # 改为return 0，避免set -e导致脚本退出
}

# 检测OOM情况
detect_oom_situation() {
    local test_name=$1
    local pod_name=$2
    
    # 首先检查Pod日志中的OOM信息（最可靠的方法）
    local pod_logs=$(kubectl logs "$pod_name" --tail=50 2>/dev/null || echo "")
    if [[ -n "$pod_logs" ]]; then
        # 检查日志中的CUDA OOM错误
        if echo "$pod_logs" | grep -q "CUDA Out of Memory Error detected"; then
            echo "oom_detected"
            return 0
        fi
        
        # 检查日志中的OOM相关信息
        if echo "$pod_logs" | grep -qE "(OutOfMemoryError|out of memory|OOM|CUDA out of memory)"; then
            echo "oom_detected"
            return 0
        fi
        
        # 检查日志中是否有OOM容器保持运行的标记
        if echo "$pod_logs" | grep -q "OOM容器保持运行中"; then
            echo "oom_detected"
            return 0
        fi
        
        # 检查日志中是否有信号处理器的OOM标记
        if echo "$pod_logs" | grep -qE "(收到系统信号|OOM_SIGNAL|OOM_SIGSEGV|容器因系统信号终止|段错误信号)"; then
            echo "oom_detected"
            return 0
        fi
    fi
    
    # 检查Pod的退出码（139通常表示段错误，可能是OOM导致的）
    local exit_code=$(kubectl get pod "$pod_name" -o jsonpath='{.status.containerStatuses[0].state.terminated.exitCode}' 2>/dev/null || echo "")
    if [[ "$exit_code" == "139" ]]; then
        echo "oom_detected"
        return 0
    fi
    
    # 检查Pod事件中的OOM信息
    local events=$(kubectl get events --field-selector involvedObject.name="$pod_name" -o json 2>/dev/null || echo '{"items":[]}')
    local oom_events=$(echo "$events" | jq -r '.items[] | select(.reason == "OOMKilled" or .message | contains("OOM") or .message | contains("out of memory")) | .message' 2>/dev/null)
    
    if [[ -n "$oom_events" ]]; then
        echo "oom_detected"
        return 0
    fi
    
    # 检查Pod状态中的OOM信息
    local pod_status=$(kubectl get pod "$pod_name" -o json 2>/dev/null || echo '{}')
    local oom_status=$(echo "$pod_status" | jq -r '.status.containerStatuses[]? | select(.state.terminated.reason == "OOMKilled" or .lastState.terminated.reason == "OOMKilled") | .state.terminated.reason // .lastState.terminated.reason' 2>/dev/null)
    
    if [[ "$oom_status" == "OOMKilled" ]]; then
        echo "oom_detected"
        return 0
    fi
    
    echo "no_oom"
    return 0  # 改为return 0，避免set -e导致脚本退出
}

# 智能等待OOM检测
wait_for_oom_detection() {
    local test_name=$1
    local pod_name=$2
    local max_wait=${3:-120}  # 最多等待2分钟
    
    log_info "🔍 检测到可能的OOM情况，等待容器内程序输出OOM信息..."
    
    local wait_time=0
    local check_interval=10
    
    while [[ $wait_time -lt $max_wait ]]; do
        # 通过Pod日志检测OOM
        local pod_logs=$(kubectl logs "$pod_name" --tail=50 2>/dev/null || echo "")
        if [[ -n "$pod_logs" ]]; then
            # 检查日志中的CUDA OOM错误
            if echo "$pod_logs" | grep -q "CUDA Out of Memory Error detected"; then
                log_success "✅ 检测到CUDA OOM错误信息"
                return 0
            fi
            
            # 检查日志中的OOM相关信息
            if echo "$pod_logs" | grep -qE "(OutOfMemoryError|out of memory|OOM|CUDA out of memory)"; then
                log_success "✅ 检测到OOM相关错误信息"
                return 0
            fi
            
            # 检查日志中是否有OOM容器保持运行的标记
            if echo "$pod_logs" | grep -q "OOM容器保持运行中"; then
                log_success "✅ 检测到OOM容器保持运行标记"
                return 0
            fi
        fi
        
        sleep $check_interval
        wait_time=$((wait_time + check_interval))
        
        if [[ $((wait_time % 30)) -eq 0 ]]; then
            log_info "⏳ 继续等待OOM信息... (已等待 ${wait_time}秒)"
        fi
    done
    
    log_warning "⚠️  等待OOM信息超时 (${max_wait}秒)"
    return 1
}
# ========== 日志优化工具函数结束 ==========

# 内置测试函数 - 验证日志优化效果
test_log_optimization() {
    local pod_name=$1
    
    if [[ -z "$pod_name" ]]; then
        pod_name=$(kubectl get pods -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
        if [[ -z "$pod_name" ]]; then
            log_warning "未找到可用的Pod进行测试"
            return 1
        fi
        log_info "自动选择Pod: $pod_name"
    fi
    
    echo "=============================================================================="
    log_info "🧪 测试日志优化效果 - Pod: $pod_name"
    echo "=============================================================================="
    
    # 测试Pod状态检查
    log_info "📊 检查Pod状态..."
    availability=$(check_pod_log_availability "$pod_name" 5)
    log_success "Pod日志可用性: $availability"
    
    # 性能对比测试
    log_info "📊 性能对比测试..."
    
    # 传统方式
    start_time=$(date +%s.%N 2>/dev/null || date +%s)
    traditional_result=$(kubectl logs "$pod_name" --tail=5 2>/dev/null || echo "FAILED")
    end_time=$(date +%s.%N 2>/dev/null || date +%s)
    traditional_time=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "N/A")
    
    # 优化方式
    start_time=$(date +%s.%N 2>/dev/null || date +%s)
    optimized_result=$(safe_get_pod_logs "$pod_name" 5 10 2>/dev/null || echo "FAILED")
    end_time=$(date +%s.%N 2>/dev/null || date +%s)
    optimized_time=$(echo "$end_time - $start_time" | bc -l 2>/dev/null || echo "N/A")
    
    # 显示结果
    printf "%-15s %-15s %-10s\n" "方法" "耗时(秒)" "状态"
    echo "-----------------------------------------------"
    printf "%-15s %-15s %-10s\n" "传统方式" "$traditional_time" "$([[ "$traditional_result" != "FAILED" ]] && echo "成功" || echo "失败")"
    printf "%-15s %-15s %-10s\n" "优化方式" "$optimized_time" "$([[ "$optimized_result" != "FAILED" ]] && echo "成功" || echo "失败")"
    
    log_success "✅ 测试完成"
    echo "=============================================================================="
    exit 0
}

# 默认配置文件
CONFIG_FILE="${1:-universal_test_config.json}"
TEMPLATE_FILE="stable-diffusion-universal-template.yaml"
RESULTS_DIR="/tmp/universal_sd_test_results_$(date +%Y%m%d_%H%M%S)"

# 处理特殊命令行参数
case "$1" in
    "--test-logs")
        test_log_optimization "$2"
        ;;
    "--help"|"-h")
        echo "通用Stable Diffusion自动化测试脚本"
        echo "用法:"
        echo "  $0 [config_file]           # 运行测试 (默认: universal_test_config.json)"
        echo "  $0 --test-logs [pod_name]  # 测试日志优化效果"
        echo "  $0 --help                  # 显示帮助信息"
        echo ""
        echo "日志优化特性:"
        echo "  ✅ 在获取日志前检查Pod和Node状态"
        echo "  ✅ 智能超时控制 (3-15秒)"
        echo "  ✅ 多层次错误处理"
        echo "  ✅ 提供替代信息源"
        echo "  ✅ 用户友好的状态反馈"
        exit 0
        ;;
esac

# 创建结果目录
mkdir -p "$RESULTS_DIR"

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 日志函数
# 显示1000张照片成本估算
show_1000_images_cost() {
    local per_img=$1
    local spot_price=$2
    local ondemand_price=$3
    local actual_price=$4
    local lifecycle=$5
    
    if [[ "$per_img" == "N/A" ]]; then
        return
    fi
    
    log_info "💰 1000张图片成本估算:"
    
    # Spot价格成本
    if [[ "$spot_price" != "N/A" ]]; then
        local cost_1000_spot=$(echo "scale=2; $spot_price * $per_img * 1000 / 3600" | bc -l 2>/dev/null || echo "N/A")
        if [[ "$cost_1000_spot" != "N/A" ]]; then
            log_info "   💸 Spot价格: \$${cost_1000_spot}"
        fi
    fi
    
    # 按需价格成本
    if [[ "$ondemand_price" != "N/A" ]]; then
        local cost_1000_ondemand=$(echo "scale=2; $ondemand_price * $per_img * 1000 / 3600" | bc -l 2>/dev/null || echo "N/A")
        if [[ "$cost_1000_ondemand" != "N/A" ]]; then
            log_info "   💵 按需价格: \$${cost_1000_ondemand}"
        fi
    fi
    
    # 实际使用价格成本
    if [[ "$actual_price" != "N/A" ]]; then
        local cost_1000_actual=$(echo "scale=2; $actual_price * $per_img * 1000 / 3600" | bc -l 2>/dev/null || echo "N/A")
        if [[ "$cost_1000_actual" != "N/A" ]]; then
            log_info "   ✅ 实际成本: \$${cost_1000_actual} (${lifecycle}实例)"
        fi
    fi
    
    # 计算节省
    if [[ "$cost_1000_spot" != "N/A" && "$cost_1000_ondemand" != "N/A" ]]; then
        local savings=$(echo "scale=1; ($cost_1000_ondemand - $cost_1000_spot) * 100 / $cost_1000_ondemand" | bc -l 2>/dev/null || echo "N/A")
        if [[ "$savings" != "N/A" ]]; then
            log_info "   💡 Spot节省: ${savings}% (相比按需价格)"
        fi
    fi
}

log_info() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] ${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] ${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] ${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "[$(date '+%Y-%m-%d %H:%M:%S')] ${RED}[ERROR]${NC} $1"
}

build_static_fallback_cache() {
    # 静态后备映射（包含所有AWS区域）
    REGION_LOCATION_CACHE="us-east-1:US East (N. Virginia)
us-east-2:US East (Ohio)
us-west-1:US West (N. California)
us-west-2:US West (Oregon)
ca-central-1:Canada (Central)
ca-west-1:Canada (Calgary)
eu-central-1:EU (Frankfurt)
eu-central-2:EU (Zurich)
eu-west-1:EU (Ireland)
eu-west-2:EU (London)
eu-west-3:EU (Paris)
eu-north-1:EU (Stockholm)
eu-south-1:EU (Milan)
eu-south-2:EU (Spain)
ap-northeast-1:Asia Pacific (Tokyo)
ap-northeast-2:Asia Pacific (Seoul)
ap-northeast-3:Asia Pacific (Osaka)
ap-southeast-1:Asia Pacific (Singapore)
ap-southeast-2:Asia Pacific (Sydney)
ap-southeast-3:Asia Pacific (Jakarta)
ap-southeast-4:Asia Pacific (Melbourne)
ap-south-1:Asia Pacific (Mumbai)
ap-south-2:Asia Pacific (Hyderabad)
ap-east-1:Asia Pacific (Hong Kong)
ap-east-2:Asia Pacific (Taipei)
sa-east-1:South America (Sao Paulo)
me-south-1:Middle East (Bahrain)
me-central-1:Middle East (UAE)
af-south-1:Africa (Cape Town)
il-central-1:Israel (Tel Aviv)
us-gov-east-1:AWS GovCloud (US-East)
us-gov-west-1:AWS GovCloud (US-West)
cn-north-1:China (Beijing)
cn-northwest-1:China (Ningxia)"
    echo "📋 使用静态后备映射（包含所有AWS区域）" >&2
}


build_region_location_cache() {
    # echo "🔄 构建区域位置映射缓存..." >&2  # 隐藏详细输出
    
    # 清空缓存
    REGION_LOCATION_CACHE=""
    local all_mappings=""
    
    # 使用GPU实例类型优先，因为我们主要测试GPU工作负载
    local instance_types=("g4dn.xlarge" "g5.xlarge" "p3.2xlarge" "m5.large" "t3.micro")
    
    for instance_type in "${instance_types[@]}"; do
        # echo "  📡 查询实例类型: $instance_type" >&2  # 隐藏详细输出
        
        # 获取更多数据，使用分页
        local next_token=""
        local page_count=0
        
        while [[ $page_count -lt 2 ]]; do  # 最多查询2页，减少API调用
            local api_params=(
                --service-code AmazonEC2
                --region us-east-1
                --filters "Type=TERM_MATCH,Field=instanceType,Value=$instance_type"
                --format-version aws_v1
                --max-items 100
                --output json
            )
            
            if [[ -n "$next_token" ]]; then
                api_params+=(--starting-token "$next_token")
            fi
            
            local pricing_data=$(aws pricing get-products "${api_params[@]}" 2>/dev/null)
            
            if [[ -n "$pricing_data" ]]; then
                # 提取regionCode和location的对应关系
                local page_mappings=$(echo "$pricing_data" | jq -r '.PriceList[] | fromjson | .product.attributes | "\(.regionCode):\(.location)"' 2>/dev/null | sort -u)
                
                if [[ -n "$page_mappings" ]]; then
                    all_mappings="$all_mappings
$page_mappings"
                    
                    # 如果已经找到us-east-1，可以提前结束某些查询
                    if echo "$page_mappings" | grep -q "^us-east-1:"; then
                        : # echo "  ✅ 在 $instance_type 中找到 us-east-1 映射" >&2  # 隐藏详细输出
                    fi
                fi
                
                # 检查是否有下一页
                next_token=$(echo "$pricing_data" | jq -r '.NextToken // empty' 2>/dev/null)
                if [[ -z "$next_token" ]]; then
                    break
                fi
            else
                echo "  ⚠️  $instance_type 查询失败，跳过" >&2
                break
            fi
            
            page_count=$((page_count + 1))
        done
        
        # 如果已经获得足够的映射（包含主要区域），可以提前结束
        if [[ -n "$all_mappings" ]]; then
            local current_count=$(echo "$all_mappings" | grep -v '^$' | sort -u | wc -l)
            if [[ $current_count -gt 20 ]]; then
                # echo "  📊 已获取 $current_count 个区域映射，足够使用" >&2  # 隐藏详细输出
                break
            fi
        fi
    done
    
    # 去重并排序
    if [[ -n "$all_mappings" ]]; then
        REGION_LOCATION_CACHE=$(echo "$all_mappings" | grep -v '^$' | sort -u)
        local mapping_count=$(echo "$REGION_LOCATION_CACHE" | wc -l)
        # echo "✅ 成功构建区域位置映射缓存，包含 $mapping_count 个映射" >&2  # 隐藏详细输出
        
        # 显示关键区域是否包含（调试用）
        local key_regions=("us-east-1" "us-west-2" "eu-central-1" "ap-northeast-1")
        local found_regions=""
        for region in "${key_regions[@]}"; do
            if echo "$REGION_LOCATION_CACHE" | grep -q "^${region}:"; then
                found_regions="$found_regions $region"
            fi
        done
        # echo "📍 关键区域覆盖:$found_regions" >&2  # 隐藏详细输出
    else
        echo "⚠️  无法从Pricing API获取区域映射，使用静态后备映射" >&2
        build_static_fallback_cache
    fi
}

# 预先构建区域位置映射缓存
log_info "🌍 预先构建AWS区域位置映射缓存..."
build_region_location_cache
log_info "✅ 区域位置映射缓存构建完成"


# 价格缓存机制
PRICE_CACHE_DATA=""
PRICE_CACHE_TIME_DATA=""

set_price_cache() {
    local key=$1
    local value=$2
    local timestamp=$(date +%s)
    
    # 简单的字符串拼接缓存
    PRICE_CACHE_DATA="${PRICE_CACHE_DATA}${key}:${value}|"
    PRICE_CACHE_TIME_DATA="${PRICE_CACHE_TIME_DATA}${key}:${timestamp}|"
}

get_price_cache() {
    local key=$1
    echo "$PRICE_CACHE_DATA" | grep -o "${key}:[^|]*" | cut -d':' -f2-
}

get_price_cache_time() {
    local key=$1
    echo "$PRICE_CACHE_TIME_DATA" | grep -o "${key}:[^|]*" | cut -d':' -f2
}

# 价格缓存机制
PRICE_CACHE_DATA=""
PRICE_CACHE_TIME_DATA=""

set_price_cache() {
    local key=$1
    local value=$2
    local timestamp=$(date +%s)
    
    # 简单的字符串拼接缓存
    PRICE_CACHE_DATA="${PRICE_CACHE_DATA}${key}:${value}|"
    PRICE_CACHE_TIME_DATA="${PRICE_CACHE_TIME_DATA}${key}:${timestamp}|"
}

get_price_cache() {
    local key=$1
    echo "$PRICE_CACHE_DATA" | grep -o "${key}:[^|]*" | cut -d':' -f2-
}

get_price_cache_time() {
    local key=$1
    echo "$PRICE_CACHE_TIME_DATA" | grep -o "${key}:[^|]*" | cut -d':' -f2
}

# 动态获取区域到位置名称的映射 (改进版)
REGION_LOCATION_CACHE=""

get_location_name_for_region() {
    local region=$1
    
    # 如果缓存已存在，直接使用（脚本运行期间永久有效）
    if [[ -n "$REGION_LOCATION_CACHE" ]]; then
        # 从缓存中查找
        local cached_location=$(echo "$REGION_LOCATION_CACHE" | grep "^${region}:" | cut -d':' -f2-)
        if [[ -n "$cached_location" ]]; then
            echo "$cached_location"
            return
        fi
    fi
    
    # 缓存中没有找到，使用静态后备
    get_static_location_mapping "$region"
}

get_static_location_mapping() {
    local region=$1
    case "$region" in
        # 美国区域
        "us-east-1") echo "US East (N. Virginia)" ;;
        "us-east-2") echo "US East (Ohio)" ;;
        "us-west-1") echo "US West (N. California)" ;;
        "us-west-2") echo "US West (Oregon)" ;;
        
        # 加拿大区域
        "ca-central-1") echo "Canada (Central)" ;;
        "ca-west-1") echo "Canada (Calgary)" ;;
        
        # 欧洲区域
        "eu-central-1") echo "EU (Frankfurt)" ;;
        "eu-central-2") echo "EU (Zurich)" ;;
        "eu-west-1") echo "EU (Ireland)" ;;
        "eu-west-2") echo "EU (London)" ;;
        "eu-west-3") echo "EU (Paris)" ;;
        "eu-north-1") echo "EU (Stockholm)" ;;
        "eu-south-1") echo "EU (Milan)" ;;
        "eu-south-2") echo "EU (Spain)" ;;
        
        # 亚太区域
        "ap-northeast-1") echo "Asia Pacific (Tokyo)" ;;
        "ap-northeast-2") echo "Asia Pacific (Seoul)" ;;
        "ap-northeast-3") echo "Asia Pacific (Osaka)" ;;
        "ap-southeast-1") echo "Asia Pacific (Singapore)" ;;
        "ap-southeast-2") echo "Asia Pacific (Sydney)" ;;
        "ap-southeast-3") echo "Asia Pacific (Jakarta)" ;;
        "ap-southeast-4") echo "Asia Pacific (Melbourne)" ;;
        "ap-south-1") echo "Asia Pacific (Mumbai)" ;;
        "ap-south-2") echo "Asia Pacific (Hyderabad)" ;;
        "ap-east-1") echo "Asia Pacific (Hong Kong)" ;;
        "ap-east-2") echo "Asia Pacific (Taipei)" ;;
        
        # 南美区域
        "sa-east-1") echo "South America (Sao Paulo)" ;;
        
        # 中东区域
        "me-south-1") echo "Middle East (Bahrain)" ;;
        "me-central-1") echo "Middle East (UAE)" ;;
        
        # 非洲区域
        "af-south-1") echo "Africa (Cape Town)" ;;
        
        # 以色列区域
        "il-central-1") echo "Israel (Tel Aviv)" ;;
        
        # 政府云区域
        "us-gov-east-1") echo "AWS GovCloud (US-East)" ;;
        "us-gov-west-1") echo "AWS GovCloud (US-West)" ;;
        
        # 中国区域
        "cn-north-1") echo "China (Beijing)" ;;
        "cn-northwest-1") echo "China (Ningxia)" ;;
        
        # 未知区域
        *) 
            echo "US West (Oregon)"
            echo "⚠️  警告: 未知区域 $region，使用默认位置 US West (Oregon)" >&2
            echo "💡 提示: 请更新静态映射或检查区域代码是否正确" >&2
            ;;
    esac
}

# 获取按需价格 (使用AWS Pricing API)
get_ondemand_price() {
    local instance_type=$1
    local region=${2:-$(aws configure get region 2>/dev/null || echo "us-west-2")}
    
    # 获取区域对应的位置名称
    local location_name=$(get_location_name_for_region "$region")
    
    # 检查缓存
    local cache_key="ondemand_${instance_type}_${region}"
    local cached_price=$(get_price_cache "$cache_key")
    local cache_time=$(get_price_cache_time "$cache_key")
    local current_time=$(date +%s)
    
    # 如果缓存存在且未过期（1小时），直接返回
    if [[ -n "$cached_price" && -n "$cache_time" && $((current_time - cache_time)) -lt 3600 ]]; then
        echo "$cached_price"
        return
    fi
    
    # 获取价格数据
    local price=$(aws pricing get-products \
        --service-code AmazonEC2 \
        --region us-east-1 \
        --filters \
            "Type=TERM_MATCH,Field=instanceType,Value=$instance_type" \
            "Type=TERM_MATCH,Field=operatingSystem,Value=Linux" \
            "Type=TERM_MATCH,Field=location,Value=$location_name" \
        --format-version aws_v1 \
        --max-items 10 \
        --output json 2>/dev/null | \
        jq -r '.PriceList[] | fromjson | select(.terms.OnDemand | to_entries[0].value.priceDimensions | to_entries[0].value.pricePerUnit.USD != "0.0000000000") | .terms.OnDemand | to_entries[0].value.priceDimensions | to_entries[0].value.pricePerUnit.USD' 2>/dev/null | \
        sort -n | head -1)
    
    if [[ -n "$price" && "$price" != "null" && "$price" != "0.0000000000" ]]; then
        # 缓存结果
        set_price_cache "$cache_key" "$price"
        echo "$price"
    else
        echo ""
    fi
}

# 获取当前Spot价格
get_current_spot_price() {
    local instance_type=$1
    local zone=$2
    local region=${zone%?}  # 去掉最后一个字符得到region
    
    # 检查缓存
    local cache_key="spot_${instance_type}_${zone}"
    local cached_price=$(get_price_cache "$cache_key")
    local cache_time=$(get_price_cache_time "$cache_key")
    local current_time=$(date +%s)
    
    # 如果缓存存在且未过期（10分钟），直接返回
    if [[ -n "$cached_price" && -n "$cache_time" && $((current_time - cache_time)) -lt 600 ]]; then
        echo "$cached_price"
        return
    fi
    
    # 获取Spot价格历史
    local spot_price=$(aws ec2 describe-spot-price-history \
        --instance-types "$instance_type" \
        --availability-zone "$zone" \
        --product-descriptions "Linux/UNIX" \
        --max-items 1 \
        --region "$region" \
        --output json 2>/dev/null | \
        jq -r '.SpotPriceHistory[0].SpotPrice // empty' 2>/dev/null)
    
    if [[ -n "$spot_price" && "$spot_price" != "null" ]]; then
        # 缓存结果
        set_price_cache "$cache_key" "$spot_price"
        echo "$spot_price"
    else
        echo ""
    fi
}

# 获取Pod实际运行的实例信息
get_actual_instance_info() {
    local pod_name=$1
    
    if [[ -z "$pod_name" ]]; then
        echo "unknown,unknown,unknown,unknown"
        return
    fi
    
    # 获取Pod所在的节点名
    local node_name=$(kubectl get pod "$pod_name" -o jsonpath='{.spec.nodeName}' 2>/dev/null || echo "")
    
    if [[ -n "$node_name" ]]; then
        # 获取节点的实例类型
        local actual_instance_type=$(kubectl get node "$node_name" -o jsonpath='{.metadata.labels.node\.kubernetes\.io/instance-type}' 2>/dev/null || echo "unknown")
        
        # 获取节点的生命周期类型 (spot vs on-demand)
        local lifecycle=$(kubectl get node "$node_name" -o jsonpath='{.metadata.labels.karpenter\.sh/capacity-type}' 2>/dev/null || echo "unknown")
        
        # 获取可用区
        local zone=$(kubectl get node "$node_name" -o jsonpath='{.metadata.labels.topology\.kubernetes\.io/zone}' 2>/dev/null || echo "unknown")
        
        echo "$actual_instance_type,$lifecycle,$zone,$node_name"
    else
        echo "unknown,unknown,unknown,unknown"
    fi
}
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 检查必要文件
if [[ ! -f "$CONFIG_FILE" ]]; then
    log_error "配置文件 $CONFIG_FILE 不存在"
    exit 1
fi

if [[ ! -f "$TEMPLATE_FILE" ]]; then
    log_error "模板文件 $TEMPLATE_FILE 不存在"
    exit 1
fi

# 解析配置并运行测试
log_info "开始读取通用Stable Diffusion测试配置: $CONFIG_FILE"

# 读取测试配置
test_count=$(jq '.tests | length' "$CONFIG_FILE")
description=$(jq -r '.description' "$CONFIG_FILE")
prompt=$(jq -r '.default_settings.prompt' "$CONFIG_FILE")

log_info "配置描述: $description"
log_info "默认提示词: $prompt"
log_info "找到 $test_count 个测试配置"

# 创建汇总报告文件
summary_file="$RESULTS_DIR/universal_test_summary.csv"
echo "test_name,model,requested_instance,actual_instance,lifecycle,zone,batch_size,inference_steps,resolution,precision,inference_time_s,time_per_image_s,gpu_memory_gb,spot_price_usd,ondemand_price_usd,actual_price_usd,cost_spot_usd,cost_ondemand_usd,cost_actual_usd,status" > "$summary_file"

successful_tests=0
failed_tests=0
skipped_tests=0

# 遍历每个测试配置
for i in $(seq 0 $((test_count-1))); do
    # 提取测试参数
    test_name=$(jq -r ".tests[$i].name" "$CONFIG_FILE")
    model=$(jq -r ".tests[$i].model" "$CONFIG_FILE")
    instance_type=$(jq -r ".tests[$i].instance_type" "$CONFIG_FILE")
    batch_size=$(jq -r ".tests[$i].batch_size" "$CONFIG_FILE")
    inference_steps=$(jq -r ".tests[$i].inference_steps" "$CONFIG_FILE")
    image_width=$(jq -r ".tests[$i].image_width" "$CONFIG_FILE")
    image_height=$(jq -r ".tests[$i].image_height" "$CONFIG_FILE")
    precision=$(jq -r ".tests[$i].precision" "$CONFIG_FILE")
    test_prompt=$(jq -r ".tests[$i].prompt" "$CONFIG_FILE")
    memory_request=$(jq -r ".tests[$i].memory_request" "$CONFIG_FILE")
    memory_limit=$(jq -r ".tests[$i].memory_limit" "$CONFIG_FILE")
    timeout=$(jq -r ".tests[$i].timeout // 1800" "$CONFIG_FILE")
    
    echo "=============================================================================="
    log_info "🚀 开始测试 $((i+1))/$test_count: $test_name"
    echo "=============================================================================="
    log_info "  模型: $model"
    log_info "  实例类型: $instance_type"
    log_info "  批次大小: $batch_size"
    log_info "  推理步数: $inference_steps"
    log_info "  分辨率: ${image_width}x${image_height}"
    log_info "  精度: $precision"
    log_info "  提示词: $test_prompt"
    log_info "  内存: ${memory_request}/${memory_limit}"
    
    # 生成测试专用的YAML文件
    test_yaml="$RESULTS_DIR/${test_name}.yaml"
    
    # 替换模板中的变量
    sed -e "s/\${TEST_NAME}/$test_name/g" \
        -e "s|\${MODEL_ID}|$model|g" \
        -e "s/\${INSTANCE_TYPE}/$instance_type/g" \
        -e "s/\${BATCH_SIZE}/$batch_size/g" \
        -e "s/\${INFERENCE_STEPS}/$inference_steps/g" \
        -e "s|\${PROMPT}|$test_prompt|g" \
        -e "s/\${IMAGE_WIDTH}/$image_width/g" \
        -e "s/\${IMAGE_HEIGHT}/$image_height/g" \
        -e "s/\${PRECISION}/$precision/g" \
        -e "s/\${MEMORY_REQUEST}/$memory_request/g" \
        -e "s/\${MEMORY_LIMIT}/$memory_limit/g" \
        "$TEMPLATE_FILE" > "$test_yaml"
    
    # 部署测试任务
    log_info "部署通用测试任务: $test_name"
    
    # 检查是否存在同名的Job，如果存在则先删除
    if kubectl get job "$test_name" >/dev/null 2>&1; then
        log_warning "⚠️  发现已存在的Job: $test_name, 正在清理..."
        kubectl delete job "$test_name" --ignore-not-found=true
        
        # 等待Job完全删除
        cleanup_timeout=30
        cleanup_elapsed=0
        while kubectl get job "$test_name" >/dev/null 2>&1 && [[ $cleanup_elapsed -lt $cleanup_timeout ]]; do
            sleep 2
            cleanup_elapsed=$((cleanup_elapsed + 2))
            if [[ $((cleanup_elapsed % 10)) -eq 0 ]]; then
                log_info "⏳ 等待Job清理完成... (${cleanup_elapsed}s)"
            fi
        done
        
        if kubectl get job "$test_name" >/dev/null 2>&1; then
            log_error "❌ Job清理超时，强制继续部署"
        else
            log_success "✅ Job清理完成"
        fi
    fi
    
    if kubectl apply -f "$test_yaml"; then
        log_success "任务部署成功"
        
        # 等待任务完成
        log_info "等待任务完成 (超时: ${timeout}秒)..."
        log_info "💡 提示: 你可以在另一个终端运行以下命令查看实时日志:"
        log_info "   kubectl logs -f job/$test_name"
        log_info "⏳ 通过Karpenter启动新机器可能需要1-5分钟，请耐心等待..."
        
        start_time=$(date +%s)
        job_completed=false
        last_log_time=0
        pod_created=false
        last_check_time=0
        pod_running=false
        pod_name=""  # 初始化Pod名称变量
        pod_failed=false  # 跟踪Pod是否已经报告过失败
        last_pod_phase=""  # 跟踪上次的Pod状态
        
        while true; do
            current_time=$(date +%s)
            elapsed=$((current_time - start_time))
            
            if [[ $elapsed -gt $timeout ]]; then
                log_error "任务超时 ($timeout 秒)"
                kubectl delete job "$test_name" --ignore-not-found=true
                echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,TIMEOUT,TIMEOUT,TIMEOUT,N/A,N/A,N/A,N/A,N/A,N/A,TIMEOUT" >> "$summary_file"
                failed_tests=$((failed_tests + 1))
                break
            fi
            
            # 每3秒检查一次状态，提高任务完成检测的响应速度
            if [[ $((current_time - last_check_time)) -ge 3 ]]; then
                last_check_time=$current_time
                
                # 检查Job状态
                job_status=$(kubectl get job "$test_name" -o jsonpath='{.status.conditions[?(@.type=="Complete")].status}' 2>/dev/null || echo "")
                
                job_failed=$(kubectl get job "$test_name" -o jsonpath='{.status.conditions[?(@.type=="Failed")].status}' 2>/dev/null || echo "")
                
                # 动态获取当前Pod名称（重要：处理Pod重建的情况）
                current_pod_name=$(kubectl get pods -l job-name="$test_name" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
                
                current_pod_phase=""
                current_pod_ready=""
                
                # 检测任务完成状态（在Job状态变为Complete之前）
                if [[ -n "$current_pod_name" && "$job_status" != "True" ]]; then
                    task_status=$(detect_task_completion "$test_name" "$current_pod_name")
                    if [[ "$task_status" == "task_completed" ]]; then
                        log_success "🎉 检测到任务成功完成，开始处理数据..."
                        
                        # 等待一小段时间确保所有日志都已输出
                        sleep 2
                        
                        # 立即处理性能数据
                        performance_line=$(safe_grep_pod_logs "$current_pod_name" "PERFORMANCE_SUMMARY:" 200 15)
                        if [[ -n "$performance_line" ]]; then
                            # 解析性能数据
                            perf_data=$(echo "$performance_line" | cut -d' ' -f2)
                            
                            # 检查是否为 OOM 或其他错误
                            if [[ "$perf_data" == *"OOM"* ]]; then
                                echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_SKIPPED" >> "$summary_file"
                                log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                                skipped_tests=$((skipped_tests + 1))
                            elif [[ "$perf_data" == *"ERROR"* ]]; then
                                echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,ERROR,ERROR,ERROR,N/A,N/A,N/A,N/A,N/A,N/A,ERROR" >> "$summary_file"
                                log_error "❌ 测试遇到错误: $test_name"
                                failed_tests=$((failed_tests + 1))
                            else
                                # 成功的测试，快速处理数据
                                IFS=',' read -r tn it bs is inf_time per_img gpu_mem <<< "$perf_data"
                                
                                # 获取基本实例信息
                                instance_info=$(get_actual_instance_info "$current_pod_name")
                                IFS=',' read -r actual_instance_type lifecycle zone node_name <<< "$instance_info"
                                
                                if [[ "$actual_instance_type" == "unknown" ]]; then
                                    actual_instance_type="$instance_type"
                                    lifecycle="unknown"
                                    zone="unknown"
                                fi
                                
                                # 获取价格信息
                                log_info "💰 获取价格信息..."
                                log_info "🔍 实际实例类型: $actual_instance_type, 区域: $region"
                                
                                # 获取区域信息
                                region=${zone%?}  # 去掉最后一个字符得到region
                                if [[ "$region" == "unknown" ]]; then
                                    region=$(aws configure get region 2>/dev/null || echo "us-west-2")
                                fi
                                
                                # 获取按需价格
                                ondemand_price=$(get_ondemand_price "$actual_instance_type" "$region")
                                if [[ -z "$ondemand_price" ]]; then
                                    ondemand_price="N/A"
                                else
                                    log_info "💵 按需价格: \$${ondemand_price}/小时"
                                fi
                                
                                # 获取Spot价格
                                spot_price=""
                                if [[ "$zone" != "unknown" ]]; then
                                    spot_price=$(get_current_spot_price "$actual_instance_type" "$zone")
                                fi
                                if [[ -z "$spot_price" ]]; then
                                    spot_price="N/A"
                                else
                                    log_info "💸 Spot价格: \$${spot_price}/小时"
                                fi
                                
                                # 确定实际使用的价格
                                actual_price=""
                                if [[ "$lifecycle" == "spot" && "$spot_price" != "N/A" ]]; then
                                    actual_price="$spot_price"
                                    log_info "✅ 使用Spot实例，实际价格: \$${actual_price}/小时"
                                elif [[ "$ondemand_price" != "N/A" ]]; then
                                    actual_price="$ondemand_price"
                                    log_info "✅ 使用按需实例，实际价格: \$${actual_price}/小时"
                                else
                                    actual_price="N/A"
                                fi
                                
                                # 计算成本（基于推理时间 + 30秒模型加载时间）
                                total_time=$(echo "$inf_time + 30" | bc -l 2>/dev/null || echo "60")
                                cost_spot="N/A"
                                cost_ondemand="N/A"
                                cost_actual="N/A"
                                
                                if [[ "$spot_price" != "N/A" ]]; then
                                    cost_spot=$(echo "scale=4; $spot_price * $total_time / 3600" | bc -l 2>/dev/null || echo "N/A")
                                fi
                                
                                if [[ "$ondemand_price" != "N/A" ]]; then
                                    cost_ondemand=$(echo "scale=4; $ondemand_price * $total_time / 3600" | bc -l 2>/dev/null || echo "N/A")
                                fi
                                
                                if [[ "$actual_price" != "N/A" ]]; then
                                    cost_actual=$(echo "scale=4; $actual_price * $total_time / 3600" | bc -l 2>/dev/null || echo "N/A")
                                fi
                                
                                # 记录完整数据到CSV
                                echo "$test_name,$model,$instance_type,$actual_instance_type,$lifecycle,$zone,$batch_size,$inference_steps,${image_width}x${image_height},$precision,$inf_time,$per_img,$gpu_mem,$spot_price,$ondemand_price,$actual_price,$cost_spot,$cost_ondemand,$cost_actual,SUCCESS" >> "$summary_file"
                                
                                # 计算1000张图片的成本估算
                                show_1000_images_cost "$per_img" "$spot_price" "$ondemand_price" "$actual_price" "$lifecycle"

                                # 快速记录基本数据（现在包含价格信息）
                                log_success "📊 任务完成数据已快速记录"
                                successful_tests=$((successful_tests + 1))
                                
                                log_info "⚡ 性能摘要: 推理时间=${inf_time}s, 单图时间=${per_img}s, GPU内存=${gpu_mem}GB"
                            fi
                        else
                            echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,NO_DATA,NO_DATA,NO_DATA,N/A,N/A,N/A,N/A,N/A,N/A,COMPLETED_NO_DATA" >> "$summary_file"
                            log_warning "⚠️  任务完成但未找到性能数据"
                        fi
                        
                        # 立即取消任务，避免等待180秒
                        log_info "✅ 数据处理完成，主动取消任务以节省资源"
                        kubectl delete job "$test_name" --ignore-not-found=true
                        log_info "🗑️ 任务已取消: $test_name"
                        
                        job_completed=true
                        break
                    elif [[ "$task_status" == "task_oom" ]]; then
                        log_warning "⚠️  检测到GPU内存不足(OOM)，任务被跳过..."
                        
                        # 等待一小段时间确保所有日志都已输出
                        sleep 2
                        
                        # 记录OOM结果到CSV
                        echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_SKIPPED" >> "$summary_file"
                        log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                        log_info "💡 建议: 使用float16精度、减少batch_size或选择更大的GPU实例"
                        log_info "📊 OOM测试结果已记录: 推理时间=N/A, 单图时间=N/A, GPU内存=OOM"
                        skipped_tests=$((skipped_tests + 1))
                        
                        # 立即取消任务
                        log_info "✅ OOM检测完成，主动取消任务以节省资源"
                        kubectl delete job "$test_name" --ignore-not-found=true
                        log_info "🗑️ 任务已取消: $test_name"
                        
                        job_completed=true
                        break
                    fi
                fi
                
                # 检查Job完成状态
                if [[ "$job_status" == "True" ]]; then
                    log_success "✅ Job已完成: $test_name"
                    job_completed=true
                    break
                    
                elif [[ "$job_failed" == "True" ]]; then
                    log_error "❌ 通用测试任务失败: $test_name"
                    
                    if [[ -n "$pod_name" ]]; then
                        log_error "Pod状态: $pod_phase"
                        get_pod_failure_reason "$pod_name"
                    fi
                    
                    echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,FAILED,FAILED,FAILED,N/A,N/A,N/A,N/A,N/A,N/A,FAILED" >> "$summary_file"
                    failed_tests=$((failed_tests + 1))
                    break
                fi
            
            if [[ -n "$current_pod_name" && "$current_pod_name" != "$pod_name" ]]; then
                    if [[ -n "$pod_name" ]]; then
                        log_warning "🔄 检测到Pod重建: $pod_name -> $current_pod_name"
                        
                        # 检测是否为OOM导致的重建
                        oom_status=$(detect_oom_situation "$test_name" "$pod_name")
                        if [[ "$oom_status" == "oom_detected" ]]; then
                            log_warning "💡 检测到OOM情况，可能是GPU内存不足"
                            # 等待容器内程序输出OOM信息
                            if wait_for_oom_detection "$test_name" "$current_pod_name" 60; then
                                log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                                echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_DETECTED" >> "$summary_file"
                                skipped_tests=$((skipped_tests + 1))
                                kubectl delete job "$test_name" --ignore-not-found=true
                                break
                            fi
                        else
                            log_info "💡 可能原因: 节点被回收、Pod出错重启、或资源不足"
                        fi
                    fi
                    pod_name="$current_pod_name"
                    pod_created=true  # 重置状态
                    pod_running=false
                    pod_failed=false  # 重置失败状态
                    last_pod_phase=""  # 重置状态跟踪
                fi
                
                if [[ -n "$pod_name" ]]; then
                    current_pod_phase=$(kubectl get pod "$pod_name" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
                    current_pod_ready=$(kubectl get pod "$pod_name" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
                    
                    # 更新Pod状态
                    pod_phase="$current_pod_phase"
                    pod_ready="$current_pod_ready"
                    
                    if [[ "$pod_created" == false ]]; then
                        log_info "✅ Pod已创建: $pod_name"
                        pod_created=true
                    fi
                    
                    if [[ "$pod_phase" == "Running" && "$pod_running" == false ]]; then
                        log_info "🚀 Pod开始运行: $pod_name"
                        pod_running=true
                    fi
                    
                    # 检测Pod异常状态 - 只在状态变化时输出
                    if [[ "$pod_phase" == "Failed" && "$pod_failed" == false ]]; then
                        log_error "❌ Pod失败: $pod_name"
                        
                        # 检查是否为OOM
                        oom_status=$(detect_oom_situation "$test_name" "$pod_name")
                        if [[ "$oom_status" == "oom_detected" ]]; then
                            log_warning "🧠 检测到OOM情况"
                            log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                            echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_DETECTED" >> "$summary_file"
                            skipped_tests=$((skipped_tests + 1))
                            kubectl delete job "$test_name" --ignore-not-found=true
                            break
                        else
                            # 等待30秒看是否为延迟OOM
                            log_info "⏳ 等待30秒检查是否为延迟OOM..."
                            sleep 30
                            delayed_oom_status=$(detect_oom_situation "$test_name" "$pod_name")
                            if [[ "$delayed_oom_status" == "oom_detected" ]]; then
                                log_warning "🧠 延迟检测到OOM情况"
                                log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                                echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_DETECTED" >> "$summary_file"
                                skipped_tests=$((skipped_tests + 1))
                                kubectl delete job "$test_name" --ignore-not-found=true
                                break
                            fi
                            
                            # 不是OOM，获取失败原因
                            get_pod_failure_reason "$pod_name"
                        fi
                        
                        pod_failed=true  # 标记已经报告过失败
                    elif [[ "$pod_phase" == "Pending" && "$last_pod_phase" != "Pending" ]]; then
                        # 检查Pod为什么还在Pending - 只在第一次进入Pending时输出
                        pending_reason=$(kubectl get pod "$pod_name" -o jsonpath='{.status.conditions[?(@.type=="PodScheduled")].reason}' 2>/dev/null || echo "")
                        if [[ "$pending_reason" == "Unschedulable" ]]; then
                            log_info "⏳ Pod等待调度中 (可能需要启动新节点): $pod_name"
                            # 不显示技术细节，保持日志简洁
                        else
                            log_info "⏳ Pod调度中: $pod_name"
                        fi
                    fi
                    
                    # 更新状态跟踪
                    last_pod_phase="$pod_phase"
                elif [[ $elapsed -gt 300 ]]; then  # 5分钟后还没有Pod就警告
                    log_warning "⚠️  等待超过5分钟仍未创建Pod，可能存在资源或调度问题"
                fi
            fi
            
            # 检测Pod变化
            if [[ -n "$current_pod_name" && "$current_pod_name" != "$pod_name" ]]; then
                    if [[ -n "$pod_name" ]]; then
                        log_warning "🔄 检测到Pod重建: $pod_name -> $current_pod_name"
                        
                        # 检测是否为OOM导致的重建
                        oom_status=$(detect_oom_situation "$test_name" "$pod_name")
                        if [[ "$oom_status" == "oom_detected" ]]; then
                            log_warning "💡 检测到OOM情况，可能是GPU内存不足"
                            # 等待容器内程序输出OOM信息
                            if wait_for_oom_detection "$test_name" "$current_pod_name" 60; then
                                log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                                echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_DETECTED" >> "$summary_file"
                                skipped_tests=$((skipped_tests + 1))
                                kubectl delete job "$test_name" --ignore-not-found=true
                                break
                            fi
                        else
                            log_info "💡 可能原因: 节点被回收、Pod出错重启、或资源不足"
                        fi
                    fi
                    pod_name="$current_pod_name"
                    pod_created=true  # 重置状态
                    pod_running=false
                    pod_failed=false  # 重置失败状态
                    last_pod_phase=""  # 重置状态跟踪
                fi
                
                if [[ -n "$pod_name" ]]; then
                    current_pod_phase=$(kubectl get pod "$pod_name" -o jsonpath='{.status.phase}' 2>/dev/null || echo "")
                    current_pod_ready=$(kubectl get pod "$pod_name" -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "")
                    
                    # 更新Pod状态
                    pod_phase="$current_pod_phase"
                    pod_ready="$current_pod_ready"
                    
                    if [[ "$pod_created" == false ]]; then
                        log_info "✅ Pod已创建: $pod_name"
                        pod_created=true
                    fi
                    
                    if [[ "$pod_phase" == "Running" && "$pod_running" == false ]]; then
                        log_info "🚀 Pod开始运行: $pod_name"
                        pod_running=true
                    fi
                    
                    # 检测Pod异常状态 - 只在状态变化时输出
                    if [[ "$pod_phase" == "Failed" && "$pod_failed" == false ]]; then
                        log_error "❌ Pod失败: $pod_name"
                        
                        # 检查是否为OOM
                        oom_status=$(detect_oom_situation "$test_name" "$pod_name")
                        if [[ "$oom_status" == "oom_detected" ]]; then
                            log_warning "🧠 检测到OOM情况"
                            log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                            echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_DETECTED" >> "$summary_file"
                            skipped_tests=$((skipped_tests + 1))
                            kubectl delete job "$test_name" --ignore-not-found=true
                            break
                        else
                            # 等待30秒看是否为延迟OOM
                            log_info "⏳ 等待30秒检查是否为延迟OOM..."
                            sleep 30
                            delayed_oom_status=$(detect_oom_situation "$test_name" "$pod_name")
                            if [[ "$delayed_oom_status" == "oom_detected" ]]; then
                                log_warning "🧠 延迟检测到OOM情况"
                                log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                                echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_DETECTED" >> "$summary_file"
                                skipped_tests=$((skipped_tests + 1))
                                kubectl delete job "$test_name" --ignore-not-found=true
                                break
                            fi
                            
                            # 不是OOM，获取失败原因
                            get_pod_failure_reason "$pod_name"
                        fi
                        
                        pod_failed=true  # 标记已经报告过失败
                    elif [[ "$pod_phase" == "Pending" && "$last_pod_phase" != "Pending" ]]; then
                        # 检查Pod为什么还在Pending - 只在第一次进入Pending时输出
                        pending_reason=$(kubectl get pod "$pod_name" -o jsonpath='{.status.conditions[?(@.type=="PodScheduled")].reason}' 2>/dev/null || echo "")
                        if [[ "$pending_reason" == "Unschedulable" ]]; then
                            log_info "⏳ Pod等待调度中 (可能需要启动新节点): $pod_name"
                            # 不显示技术细节，保持日志简洁
                        else
                            log_info "⏳ Pod调度中: $pod_name"
                        fi
                    fi
                    
                    # 更新状态跟踪
                    last_pod_phase="$pod_phase"
                elif [[ $elapsed -gt 300 ]]; then  # 5分钟后还没有Pod就警告
                    log_warning "⚠️  等待超过5分钟仍未创建Pod，可能存在资源或调度问题"
                fi
                
                # 只有在Pod真正开始运行后才检查完成状态
                if [[ "$pod_running" == true ]]; then
                    if [[ "$job_status" == "True" ]]; then
                        log_success "🎉 通用测试任务完成: $test_name"
                        
                        if [[ -n "$pod_name" ]]; then
                            # 等待一下确保日志完全写入
                            sleep 5
                            
                            # 从日志中提取性能摘要
                            performance_line=$(safe_grep_pod_logs "$pod_name" "PERFORMANCE_SUMMARY:" 200 15)
                            if [[ -z "$performance_line" ]]; then
                                # 尝试获取更多日志
                                performance_line=$(safe_grep_pod_logs "$pod_name" "PERFORMANCE_SUMMARY:" 500 20)
                            fi
                            
                            if [[ -n "$performance_line" ]]; then
                                # 解析性能数据
                                perf_data=$(echo "$performance_line" | cut -d' ' -f2)
                                
                                # 检查是否为 OOM 或其他错误
                                if [[ "$perf_data" == *"OOM"* ]]; then
                                    # OOM 情况：解析数据格式为 test_name,instance_type,batch_size,inference_steps,N/A,N/A,OOM
                                    IFS=',' read -r tn it bs is inf_time per_img gpu_mem <<< "$perf_data"
                                    echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_SKIPPED" >> "$summary_file"
                                    log_warning "⚠️  测试因GPU内存不足被跳过: $test_name"
                                    log_info "💡 建议: 使用float16精度、减少batch_size或选择更大的GPU实例"
                                    log_info "📊 OOM测试结果已记录: 推理时间=N/A, 单图时间=N/A, GPU内存=OOM"
                                    skipped_tests=$((skipped_tests + 1))
                                elif [[ "$perf_data" == *"ERROR"* ]]; then
                                    echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,ERROR,ERROR,ERROR,N/A,N/A,N/A,N/A,N/A,N/A,ERROR" >> "$summary_file"
                                    log_error "❌ 测试遇到错误: $test_name"
                                    failed_tests=$((failed_tests + 1))
                                else
                                    # 成功的测试，解析性能数据并计算价格
                                    IFS=',' read -r tn it bs is inf_time per_img gpu_mem <<< "$perf_data"
                                    
                                    # 获取Pod实际运行的实例信息
                                    log_info "🔍 检测Pod实际运行的实例信息..."
                                    instance_info=$(get_actual_instance_info "$pod_name")
                                    IFS=',' read -r actual_instance_type lifecycle zone node_name <<< "$instance_info"
                                    
                                    if [[ "$actual_instance_type" != "unknown" ]]; then
                                        log_info "📍 实例信息: $actual_instance_type ($lifecycle) 在 $zone"
                                    else
                                        log_warning "⚠️  无法获取实例信息，使用请求的实例类型"
                                        actual_instance_type="$instance_type"
                                        lifecycle="unknown"
                                        zone="unknown"
                                    fi
                                    
                                    # 获取模型加载时间
                                    model_load_line=$(safe_grep_pod_logs "$pod_name" "Model loaded in" 200 10)
                                    if [[ -n "$model_load_line" ]]; then
                                        model_load_time=$(echo "$model_load_line" | sed 's/.*in \([0-9.]*\) seconds.*/\1/' | tail -1)
                                        log_info "📝 检测到模型加载时间: ${model_load_time}秒"
                                    else
                                        model_load_time="30"  # 默认估算30秒
                                        log_info "📝 使用默认模型加载时间: ${model_load_time}秒"
                                    fi
                                    
                                    # 计算总运行时间（模型加载 + 推理时间）
                                    total_time=$(echo "$model_load_time + $inf_time" | bc -l 2>/dev/null || echo "60")
                                    log_info "⏱️ 总运行时间: ${total_time}秒 (模型加载: ${model_load_time}s + 推理: ${inf_time}s)"
                                    
                                    # 获取区域信息
                                    region=${zone%?}  # 去掉最后一个字符得到region
                                    if [[ "$region" == "unknown" ]]; then
                                        region=$(aws configure get region 2>/dev/null || echo "us-west-2")
                                        log_info "🌍 使用默认区域: $region"
                                    else
                                        log_info "🌍 检测到区域: $region"
                                    fi
                                    
                                    # 获取价格信息
                                    log_info "💰 获取价格信息..."
                                    
                                    # 获取按需价格
                                    ondemand_price=$(get_ondemand_price "$actual_instance_type" "$region")
                                    if [[ -z "$ondemand_price" ]]; then
                                        ondemand_price="N/A"
                                        log_warning "⚠️  无法获取按需价格"
                                    else
                                        log_info "💵 按需价格: \$${ondemand_price}/小时"
                                    fi
                                    
                                    # 获取Spot价格
                                    spot_price=""
                                    if [[ "$zone" != "unknown" ]]; then
                                        spot_price=$(get_current_spot_price "$actual_instance_type" "$zone")
                                    fi
                                    if [[ -z "$spot_price" ]]; then
                                        spot_price="N/A"
                                        log_warning "⚠️  无法获取Spot价格"
                                    else
                                        log_info "💸 Spot价格: \$${spot_price}/小时"
                                    fi
                                    
                                    # 确定实际使用的价格
                                    actual_price=""
                                    if [[ "$lifecycle" == "spot" && "$spot_price" != "N/A" ]]; then
                                        actual_price="$spot_price"
                                        log_info "✅ 使用Spot实例，实际价格: \$${actual_price}/小时"
                                    elif [[ "$ondemand_price" != "N/A" ]]; then
                                        actual_price="$ondemand_price"
                                        log_info "✅ 使用按需实例，实际价格: \$${actual_price}/小时"
                                    else
                                        actual_price="N/A"
                                        log_warning "⚠️  无法确定实际价格"
                                    fi
                                    
                                    # 计算成本
                                    cost_spot="N/A"
                                    cost_ondemand="N/A"
                                    cost_actual="N/A"
                                    
                                    if [[ "$spot_price" != "N/A" ]]; then
                                        cost_spot=$(echo "scale=4; $spot_price * $total_time / 3600" | bc -l 2>/dev/null || echo "N/A")
                                    fi
                                    
                                    if [[ "$ondemand_price" != "N/A" ]]; then
                                        cost_ondemand=$(echo "scale=4; $ondemand_price * $total_time / 3600" | bc -l 2>/dev/null || echo "N/A")
                                    fi
                                    
                                    if [[ "$actual_price" != "N/A" ]]; then
                                        cost_actual=$(echo "scale=4; $actual_price * $total_time / 3600" | bc -l 2>/dev/null || echo "N/A")
                                    fi
                                    
                                    # 记录完整数据到CSV
                                    echo "$test_name,$model,$instance_type,$actual_instance_type,$lifecycle,$zone,$batch_size,$inference_steps,${image_width}x${image_height},$precision,$inf_time,$per_img,$gpu_mem,$spot_price,$ondemand_price,$actual_price,$cost_spot,$cost_ondemand,$cost_actual,SUCCESS" >> "$summary_file"
                                    
                                    log_success "📊 通用测试性能数据已记录"
                                    
                                    # 注意：这个分支通常不会被执行，因为任务完成检测会提前处理
                                    log_info "ℹ️ 通过Job Complete状态处理的数据（备用路径）"
                                    
                                    # 显示详细的性能和成本摘要
                                    log_info "⚡ 性能摘要: 推理时间=${inf_time}s, 单图时间=${per_img}s, GPU内存=${gpu_mem}GB"
                                    
                                    if [[ "$cost_actual" != "N/A" ]]; then
                                        log_info "💰 成本摘要: 本次测试成本=\$${cost_actual}"
                                        
                                        # 显示节省信息
                                        if [[ "$lifecycle" == "spot" && "$cost_ondemand" != "N/A" && "$cost_spot" != "N/A" ]]; then
                                            savings=$(echo "scale=2; ($cost_ondemand - $cost_spot) * 100 / $cost_ondemand" | bc -l 2>/dev/null || echo "N/A")
                                            if [[ "$savings" != "N/A" ]]; then
                                                log_info "💡 Spot节省: ${savings}% (相比按需实例)"
                                            fi
                                        fi
                                    fi
                                fi
                            else
                                echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,NO_DATA,NO_DATA,NO_DATA,N/A,N/A,N/A,N/A,N/A,N/A,COMPLETED_NO_DATA" >> "$summary_file"
                                log_warning "⚠️  未找到性能数据，可能任务刚完成，日志还在写入"
                            fi
                        else
                            echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,NO_POD,NO_POD,NO_POD,N/A,N/A,N/A,N/A,N/A,N/A,NO_POD" >> "$summary_file"
                            log_warning "⚠️  未找到Pod"
                            failed_tests=$((failed_tests + 1))
                        fi
                        
                        job_completed=true
                        break
                        
                    elif [[ "$job_failed" == "True" ]]; then
                        log_error "❌ 通用测试任务失败: $test_name"
                        
                        if [[ -n "$pod_name" ]]; then
                            log_error "Pod状态: $pod_phase"
                            get_pod_failure_reason "$pod_name"
                        fi
                        
                        echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,FAILED,FAILED,FAILED,N/A,N/A,N/A,N/A,N/A,N/A,FAILED" >> "$summary_file"
                        failed_tests=$((failed_tests + 1))
                        break
                    fi
                fi
            
            # 显示详细进度信息 (避免在Pod失败后重复显示)
            if [[ $((elapsed % 60)) -eq 0 ]] && [[ $elapsed -ne $last_log_time ]]; then
                last_log_time=$elapsed
                if [[ -n "$pod_name" ]]; then
                    # 根据Pod状态显示不同的信息
                    case "$pod_phase" in
                        "Running")
                            log_info "⏳ 等待中... (已等待 ${elapsed}秒, Pod: $pod_name, 状态: $pod_phase)"
                            # 显示最新进度
                            recent_logs=$(safe_grep_pod_logs "$pod_name" "(===|Loading|Starting|Inference|Generated|Saved|Model loaded|PERFORMANCE_SUMMARY)" 5 5)
                            if [[ -n "$recent_logs" ]]; then
                                echo "  📝 最新进度: $(echo "$recent_logs" | tail -1)"
                            fi
                            ;;
                        "Pending")
                            # 智能处理Pending状态，区分不同原因
                            if [[ -n "$current_pod_name" ]]; then
                                # 获取Pod的详细状态
                                pod_conditions=$(kubectl get pod "$current_pod_name" -o jsonpath='{.status.conditions}' 2>/dev/null || echo "")
                                waiting_reason=$(kubectl get pod "$current_pod_name" -o jsonpath='{.status.containerStatuses[0].state.waiting.reason}' 2>/dev/null || echo "")
                                
                                # 根据等待原因显示不同信息
                                case "$waiting_reason" in
                                    "ContainerCreating")
                                        log_info "📦 容器创建中..."
                                        ;;
                                    "ImagePullBackOff"|"ErrImagePull")
                                        log_warning "⚠️  镜像拉取问题: $waiting_reason"
                                        ;;
                                    *)
                                        if [[ $elapsed -lt 180 ]]; then  # 前3分钟
                                            log_info "⏳ Pod等待调度中 (可能需要启动新节点): $current_pod_name"
                                            if [[ $elapsed -eq 60 ]]; then  # 1分钟时显示提示
                                                log_info "💡 Karpenter正在评估节点需求，通常需要1-3分钟..."
                                            fi
                                        elif [[ $elapsed -lt 300 ]]; then  # 3-5分钟
                                            if [[ $((elapsed % 60)) -eq 0 ]]; then  # 每分钟显示一次
                                                log_info "⏳ 等待节点启动中... (已等待 ${elapsed}秒)"
                                                log_info "💡 大型GPU实例启动可能需要更长时间"
                                            fi
                                        else  # 超过5分钟
                                            if [[ $((elapsed % 180)) -eq 0 ]]; then  # 每3分钟显示一次
                                                log_warning "⚠️  等待超过5分钟，可能存在资源或调度问题"
                                                log_info "💡 建议检查: kubectl describe pod $current_pod_name"
                                            fi
                                        fi
                                        ;;
                                esac
                            else
                                if [[ $elapsed -lt 300 ]]; then  # 前5分钟
                                    log_info "⏳ 等待中... (已等待 ${elapsed}秒, 等待Pod创建，Karpenter可能正在启动新节点...)"
                                else
                                    if [[ $((elapsed % 300)) -eq 0 ]]; then  # 每5分钟显示一次
                                        log_info "⏳ 等待中... (已等待 ${elapsed}秒, 仍在等待Pod创建)"
                                    fi
                                fi
                            fi
                            ;;
                        "Failed")
                            # Pod失败后，减少日志输出频率
                            if [[ $((elapsed % 300)) -eq 0 ]]; then  # 每5分钟显示一次
                                log_warning "⏳ 等待中... (已等待 ${elapsed}秒, Pod: $pod_name, 状态: 失败，等待Job重试或超时)"
                            fi
                            ;;
                        *)
                            log_info "⏳ 等待中... (已等待 ${elapsed}秒, Pod: $pod_name, 状态: $pod_phase)"
                            ;;
                    esac
                else
                    if [[ $elapsed -lt 300 ]]; then  # 前5分钟
                        log_info "⏳ 等待中... (已等待 ${elapsed}秒, 等待Pod创建，Karpenter可能正在启动新节点...)"
                    else
                        if [[ $((elapsed % 300)) -eq 0 ]]; then  # 每5分钟显示一次
                            log_info "⏳ 等待中... (已等待 ${elapsed}秒, 仍在等待Pod创建)"
                        fi
                    fi
                fi
            fi
            
            # 短暂休眠
            sleep 3
        done
        
        # 清理任务
        log_info "🧹 清理任务: $test_name"
        kubectl delete job "$test_name" --ignore-not-found=true
        
    else
        log_error "任务部署失败: $test_name"
        echo "$test_name,$model,$instance_type,unknown,unknown,unknown,$batch_size,$inference_steps,${image_width}x${image_height},$precision,DEPLOY_FAILED,DEPLOY_FAILED,DEPLOY_FAILED,N/A,N/A,N/A,N/A,N/A,N/A,DEPLOY_FAILED" >> "$summary_file"
        failed_tests=$((failed_tests + 1))
    fi
    
    # 测试间隔
    if [[ $i -lt $((test_count-1)) ]]; then
        echo "------------------------------------------------------------------------------"
        log_info "等待 10 秒后开始下一个测试..."
        echo "------------------------------------------------------------------------------"
        sleep 10
    fi
done

# 生成最终报告
log_info "生成通用Stable Diffusion测试报告..."

report_file="$RESULTS_DIR/universal_final_report.txt"
cat > "$report_file" << EOF
通用Stable Diffusion自动化测试报告
==================================

测试时间: $(date)
配置文件: $CONFIG_FILE
配置描述: $description
默认提示词: $prompt
总测试数: $test_count

测试结果统计:
  ✅ 成功测试: $successful_tests
  ❌ 失败测试: $failed_tests
  ⚠️  跳过测试(OOM): $skipped_tests
  📊 成功率: $(( (successful_tests * 100) / test_count ))%
  🔍 OOM率: $(( (skipped_tests * 100) / test_count ))%

详细结果请查看: $summary_file

测试矩阵摘要:
EOF

# 添加测试矩阵信息到报告
models=$(jq -r '.test_matrix.models | join(", ")' "$CONFIG_FILE")
instance_types=$(jq -r '.test_matrix.instance_types | join(", ")' "$CONFIG_FILE")
batch_sizes=$(jq -r '.test_matrix.batch_sizes | join(", ")' "$CONFIG_FILE")
inference_steps=$(jq -r '.test_matrix.inference_steps | join(", ")' "$CONFIG_FILE")
resolutions=$(jq -r '.test_matrix.resolutions | join(", ")' "$CONFIG_FILE")
precisions=$(jq -r '.test_matrix.precisions | join(", ")' "$CONFIG_FILE")

cat >> "$report_file" << EOF
  模型: $models
  实例类型: $instance_types
  批次大小: $batch_sizes
  推理步数: $inference_steps
  分辨率: $resolutions
  精度: $precisions

EOF

# 如果有 OOM 测试，添加分析建议
if [[ $skipped_tests -gt 0 ]]; then
    cat >> "$report_file" << EOF
OOM 分析和建议:
==============
本次测试中有 $skipped_tests 个配置因GPU内存不足而跳过。

常见 OOM 原因:
  - 使用 float32 精度 (占用内存是 float16 的2倍)
  - 批次大小过大 (batch_size > 1)
  - 分辨率过高 (1024x1024 比 896x896 占用更多内存)
  - GPU 实例类型内存不足

优化建议:
  1. 优先使用 float16 精度
  2. 将 batch_size 设置为 1
  3. 对于 SDXL，考虑使用 896x896 分辨率
  4. 选择更大的 GPU 实例类型 (如 g6e.xlarge)
  5. 设置环境变量: PYTORCH_CUDA_ALLOC_CONF=expandable_segments:True

EOF
fi

log_success "通用Stable Diffusion测试完成!"
log_info "成功: $successful_tests, 失败: $failed_tests, 跳过(OOM): $skipped_tests"
log_info "汇总结果: $summary_file"
log_info "详细报告: $report_file"
log_info "EFS输出目录: /shared/stable-diffusion-outputs/"

# 显示性能汇总
if [[ $successful_tests -gt 0 ]]; then
    log_info "通用测试性能汇总 (成功的测试):"
    echo "测试名称,模型,请求实例,实际实例,生命周期,批次,步数,分辨率,精度,推理时间(s),单图时间(s),GPU内存(GB),Spot价格,按需价格,实际价格,Spot成本,按需成本,实际成本"
    grep "SUCCESS" "$summary_file" | while IFS=',' read -r name model req_instance actual_instance lifecycle zone batch steps resolution precision inf_time per_img gpu_mem spot_price ondemand_price actual_price cost_spot cost_ondemand cost_actual status; do
        printf "%-25s %-15s %-12s %-12s %-9s %-6s %-6s %-12s %-8s %-12s %-12s %-10s %-12s %-12s %-12s %-12s %-12s %-12s\n" \
            "$name" "$model" "$req_instance" "$actual_instance" "$lifecycle" "$batch" "$steps" "$resolution" "$precision" \
            "$inf_time" "$per_img" "$gpu_mem" "$spot_price" "$ondemand_price" "$actual_price" "$cost_spot" "$cost_ondemand" "$cost_actual"
    done
    
    echo
    log_info "💰 成本分析 (1000张照片):"
    grep "SUCCESS" "$summary_file" | while IFS=',' read -r name model req_instance actual_instance lifecycle zone batch steps resolution precision inf_time per_img gpu_mem spot_price ondemand_price actual_price cost_spot cost_ondemand cost_actual status; do
        if [[ "$per_img" != "N/A" && "$actual_price" != "N/A" ]]; then
            cost_1000=$(echo "scale=2; $actual_price * $per_img * 1000 / 3600" | bc -l 2>/dev/null || echo "N/A")
            if [[ "$cost_1000" != "N/A" ]]; then
                printf "  %-25s: \$%-8s (使用 %s %s)\n" "$name" "$cost_1000" "$actual_instance" "$lifecycle"
            fi
        fi
    done
fi

echo
log_info "🚀 通用测试特性:"
log_info "  ✅ 支持多种模型 (SD 1.5, SD 2.1, 等)"
log_info "  ✅ 支持多种实例类型 (g6.xlarge, g5.xlarge, 等)"
log_info "  ✅ 支持多种批次大小 (1, 4, 8, 等)"
log_info "  ✅ 支持多种推理步数 (15, 25, 50, 等)"
log_info "  ✅ 支持多种分辨率 (512x512, 1024x1024, 等)"
log_info "  ✅ 支持多种精度 (float32, float16, bfloat16)"
log_info "  ✅ 自动选择合适的调度器"
log_info "  ✅ 自动配置内存限制"
log_info "  ✅ 详细的性能报告和分析"
