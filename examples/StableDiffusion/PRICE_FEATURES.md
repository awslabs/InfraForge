# 价格增强功能技术文档

> 📖 **用户指南:**  如需快速上手，请先查看 [`README.md`](./README.md)。本文档提供价格功能的详细技术实现和高级配置。

## 🆚 功能对比

| 功能 | 原版 | 价格增强版 |
|------|------|-----------|
| 性能测试 | ✅ | ✅ |
| OOM处理 | ✅ | ✅ |
| Pod监控 | ✅ | ✅ |
| **实时价格获取** | ❌ | ✅ |
| **成本计算** | ❌ | ✅ |
| **实例检测** | ❌ | ✅ |
| **价格缓存** | ❌ | ✅ |
| **成本分析报告** | ❌ | ✅ |

## 💰 价格功能架构

### 1. 价格获取层
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Spot Price    │    │  OnDemand Price │    │ Fallback Price  │
│   AWS EC2 API   │    │ AWS Pricing API │    │  Static Values  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │  Price Cache    │
                    │   (30 min)      │
                    └─────────────────┘
```

### 2. 实例检测层
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│      Pod        │    │      Node       │    │   Instance      │
│   kubectl get   │    │   kubectl get   │    │    Labels       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Instance Info   │
                    │ Type/Zone/LC    │
                    └─────────────────┘
```

### 3. 成本计算层
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│ Inference Time  │    │ Model Load Time │    │ Instance Price  │
│   (seconds)     │    │   (seconds)     │    │   ($/hour)      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────┐
                    │ Cost Calculator │
                    │ 1000 Images ($) │
                    └─────────────────┘
```

## 🔧 API调用详情

### 1. Spot价格获取
```bash
aws ec2 describe-spot-price-history \
    --region us-west-2 \
    --instance-types g5.xlarge \
    --product-descriptions "Linux/UNIX" \
    --max-items 1 \
    --query 'SpotPriceHistory[0].SpotPrice'
```

**返回示例:**  `"0.3012"`

### 2. 按需价格获取
```bash
aws pricing get-products \
    --service-code AmazonEC2 \
    --region us-east-1 \
    --filters \
        "Type=TERM_MATCH,Field=instanceType,Value=g5.xlarge" \
        "Type=TERM_MATCH,Field=location,Value=US West (Oregon)" \
        "Type=TERM_MATCH,Field=tenancy,Value=Shared" \
        "Type=TERM_MATCH,Field=operating-system,Value=Linux"
```

**返回示例:**  复杂JSON，需要解析获取价格

### 3. 实例信息获取
```bash
# 获取Pod所在节点
kubectl get pod $POD_NAME -o jsonpath='{.spec.nodeName}'

# 获取节点实例类型
kubectl get node $NODE_NAME -o jsonpath='{.metadata.labels.node\.kubernetes\.io/instance-type}'

# 获取生命周期类型
kubectl get node $NODE_NAME -o jsonpath='{.metadata.labels.karpenter\.sh/capacity-type}'
```

## 📊 成本计算公式

### 基础公式
```
单张照片时间 = 推理时间 ÷ 批次大小
1000张照片推理时间 = 单张照片时间 × 1000
总运行时间 = 1000张照片推理时间 + 模型加载时间
总运行小时 = 总运行时间 ÷ 3600
总成本 = 总运行小时 × 实例小时价格
```

### 示例计算
```
推理时间: 15.2秒
批次大小: 1
模型加载时间: 28.5秒
实例价格: $0.301/小时

单张照片时间 = 15.2 ÷ 1 = 15.2秒
1000张照片推理时间 = 15.2 × 1000 = 15200秒
总运行时间 = 15200 + 28.5 = 15228.5秒
总运行小时 = 15228.5 ÷ 3600 = 4.23小时
总成本 = 4.23 × $0.301 = $1.27
```

## 🎯 价格缓存机制

### 缓存策略
- **缓存时间:**  30分钟
- **缓存键:**  `{instance_type}_{lifecycle}_{zone}`
- **缓存内容:**  `{price},{price_type}`

### 缓存逻辑
```bash
cache_key="g5.xlarge_spot_us-west-2a"
current_time=$(date +%s)

# 检查缓存
if [[ -n "${PRICE_CACHE[$cache_key]}" ]]; then
    cache_time=${PRICE_CACHE_TIME[$cache_key]}
    if [[ $((current_time - cache_time)) -lt 1800 ]]; then
        # 使用缓存
        echo "${PRICE_CACHE[$cache_key]}"
        return
    fi
fi

# 获取新价格并缓存
price=$(get_current_spot_price "$instance_type" "$zone")
PRICE_CACHE[$cache_key]="$price,spot"
PRICE_CACHE_TIME[$cache_key]="$current_time"
```

## 📈 CSV输出格式详解

### 20个字段说明
```csv
test_name,model,requested_instance,actual_instance,lifecycle,zone,batch_size,inference_steps,resolution,precision,inference_time_s,time_per_image_s,gpu_memory_gb,spot_price_usd,ondemand_price_usd,actual_price_usd,cost_1000_images_spot_usd,cost_1000_images_ondemand_usd,cost_1000_images_actual_usd,status
```

| 序号 | 字段名 | 类型 | 说明 | 示例 |
|------|--------|------|------|------|
| 1 | test_name | string | 测试名称 | `sd-2-1-g5xlarge-b1-s20-float16` |
| 2 | model | string | 模型名称 | `SD2.1` |
| 3 | requested_instance | string | 请求的实例类型 | `g5.xlarge` |
| 4 | actual_instance | string | 实际分配的实例类型 | `g5.xlarge` |
| 5 | lifecycle | string | 实例生命周期 | `spot` |
| 6 | zone | string | 可用区 | `us-west-2a` |
| 7 | batch_size | int | 批次大小 | `1` |
| 8 | inference_steps | int | 推理步数 | `20` |
| 9 | resolution | string | 分辨率 | `1024x1024` |
| 10 | precision | string | 精度 | `float16` |
| 11 | inference_time_s | float | 推理时间(秒) | `15.2` |
| 12 | time_per_image_s | float | 单图时间(秒) | `15.2` |
| 13 | gpu_memory_gb | float | GPU内存使用(GB) | `18.5` |
| 14 | **spot_price_usd** | float | **Spot价格($/小时)** | `0.301` |
| 15 | **ondemand_price_usd** | float | **按需价格($/小时)** | `1.006` |
| 16 | **actual_price_usd** | float | **实际价格($/小时)** | `0.301` |
| 17 | **cost_1000_images_spot_usd** | float | **1000张Spot成本($)** | `1.28` |
| 18 | **cost_1000_images_ondemand_usd** | float | **1000张按需成本($)** | `4.27` |
| 19 | **cost_1000_images_actual_usd** | float | **1000张实际成本($)** | `1.28` |
| 20 | status | string | 测试状态 | `SUCCESS` |

### 🎯 关键优势

**完整价格对比:**  每个测试结果都包含Spot和按需价格，便于成本对比分析
**实际vs理论:**  记录实际启动的实例类型，同时提供理论价格对比
**节省计算:**  自动计算Spot相对于按需的节省金额和百分比
**多维度成本:**  提供三种成本计算（Spot、按需、实际）

## 🚨 错误处理

### 价格获取失败
```bash
# Spot价格API失败
if [[ "$spot_price" == "None" || -z "$spot_price" ]]; then
    # 使用预设价格
    price=$(get_fallback_price "$instance_type")
    price_type="fallback"
fi
```

### 实例信息获取失败
```bash
# Pod信息获取失败
if [[ "$actual_instance_type" == "unknown" ]]; then
    log_warning "⚠️ 无法获取实例信息，使用请求的实例类型"
    actual_instance_type="$instance_type"
    lifecycle="unknown"
    zone="unknown"
fi
```

### API权限不足
```bash
# 检查权限
aws ec2 describe-spot-price-history --instance-types g5.xlarge --max-items 1 2>&1 | grep -q "AccessDenied"
if [[ $? -eq 0 ]]; then
    log_error "❌ 缺少EC2权限，使用预设价格"
    price=$(get_fallback_price "$instance_type")
fi
```

## 💡 最佳实践

### 1. 价格获取优化
- ✅ 使用缓存避免频繁API调用
- ✅ 优先获取Spot价格（通常便宜60-90%）
- ✅ 提供后备价格确保功能可用
- ✅ 标注价格获取时间

### 2. 成本计算准确性
- ✅ 包含模型加载时间
- ✅ 基于实际实例类型计算
- ✅ 考虑批次大小影响
- ✅ 提供成本效率指标

### 3. 用户体验
- ✅ 清晰的价格类型标识
- ✅ Spot价格波动提醒
- ✅ 详细的成本分析报告
- ✅ 实例类型变化通知

### 4. 故障恢复
- ✅ API失败时的优雅降级
- ✅ 多层价格获取策略
- ✅ 详细的错误日志
- ✅ 用户友好的错误提示

## 🔮 未来扩展

### 可能的增强功能
1. **历史价格分析:**  分析价格趋势，推荐最佳运行时间
2. **成本预算控制:**  设置成本上限，自动选择实例类型
3. **多区域价格对比:**  对比不同区域的价格
4. **Reserved Instance支持:**  支持RI价格计算
5. **成本优化建议:**  基于历史数据提供优化建议

### 扩展示例
```bash
# 历史价格分析
--enable-price-history --history-days 7

# 成本预算控制
--max-cost-per-1000-images 2.0

# 多区域对比
--regions "us-west-2,us-east-1,eu-west-1"
```

通过价格增强功能，用户可以：
- 🎯 做出基于成本的明智决策
- 📊 优化大规模图像生成预算
- 💰 充分利用Spot实例节省成本
- 📈 分析不同配置的成本效益
