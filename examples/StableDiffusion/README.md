# Stable Diffusion 性能测试系统 (价格增强版)

一个统一的 Stable Diffusion 性能测试系统，支持标准 SD 和 SDXL 模型的自动化测试。

**🆕 价格增强功能:**  动态获取AWS实例价格，计算1000张照片成本，支持Spot和按需价格对比分析。

## 🚀 核心文件

| 文件 | 说明 |
|------|------|
| `generate_universal_test_config.py` | 统一配置生成器 |
| `run_universal_tests.sh` | 测试执行脚本 (价格增强版) |
| `stable-diffusion-universal-template.yaml` | Kubernetes 任务模板 (增强版) |
| `r.sh` | 示例测试脚本 |
| `PRICE_FEATURES.md` | 📋 **价格功能详细技术文档** |

## 🎯 测试模式

### 1. 对比模式 (推荐)
```bash
python3 generate_universal_test_config.py \
    --mode comparison \
    --models "stabilityai/stable-diffusion-2-1" "stabilityai/stable-diffusion-xl-base-1.0" \
    --instance-types "g6e.xlarge" "g5.xlarge" \
    --batch-sizes 1 4 \
    --precisions "float16" "float32" \
    --output "comparison_test.json"

./run_universal_tests.sh comparison_test.json
```

### 2. SDXL 专用模式
```bash
python3 generate_universal_test_config.py \
    --mode sdxl_only \
    --instance-types "g6e.xlarge" \
    --output "sdxl_test.json"
```

### 3. 标准 SD 专用模式
```bash
python3 generate_universal_test_config.py \
    --mode sd_only \
    --instance-types "g5.xlarge" \
    --batch-sizes 1 4 \
    --output "sd_test.json"
```

### 4. 实例优化模式
```bash
python3 generate_universal_test_config.py \
    --mode instance_optimized \
    --instance-types "g6e.xlarge" "g5.xlarge" "g4dn.xlarge" \
    --output "instance_test.json"
```

### 5. 混合模式 (默认)
```bash
python3 generate_universal_test_config.py \
    --mode mixed \
    --output "mixed_test.json"
```

## ⚙️ 对比模式内存配置

对比模式支持可配置的内存参数，确保公平对比：

```bash
# 16GB 内存机器 (默认)
--comparison-memory-request "12Gi" --comparison-memory-limit "15Gi"

# 32GB 内存机器 (推荐)
--comparison-memory-request "24Gi" --comparison-memory-limit "30Gi"

# 保守配置
--comparison-memory-request "8Gi" --comparison-memory-limit "12Gi"
```

## 🔧 主要参数

| 参数 | 说明 | 示例 |
|------|------|------|
| `--mode` | 测试模式 | `comparison`, `sdxl_only`, `sd_only` |
| `--models` | 模型列表 | `"stabilityai/stable-diffusion-2-1"` |
| `--instance-types` | 实例类型 | `"g6e.xlarge" "g5.xlarge"` |
| `--batch-sizes` | 批次大小 | `1 4` |
| `--inference-steps` | 推理步数 | `15 25 50` |
| `--resolutions` | 分辨率 | `"512x512" "1024x1024"` |
| `--precisions` | 精度 | `"float16" "float32"` |
| `--prompt` | 提示词 | `"a photo of an astronaut"` |

## 🎯 实例类型建议

| 实例类型 | CPU 内存 | GPU 内存 | Spot价格范围 | 按需价格 | SDXL 适用性 | 成本效率 |
|----------|----------|----------|-------------|----------|-------------|----------|
| **g6e.xlarge** | 32GB | 48GB | $0.30-0.60 | $1.212 | ✅ 最佳 | 高性能 |
| **g5.xlarge** | 16GB | 24GB | $0.25-0.50 | $1.006 | ✅ 良好 | 平衡 |
| **g6.xlarge** | 16GB | 24GB | $0.20-0.40 | $0.758 | ✅ 良好 | 高性价比 |
| **g4dn.xlarge** | 16GB | 16GB | $0.15-0.30 | $0.526 | ⚠️ 不推荐 | 最便宜 |

## 🚨 OOM 处理

系统会自动检测和处理 GPU 内存不足 (OOM) 的情况：

- ✅ 自动跳过 OOM 测试
- ✅ 记录详细的 OOM 信息
- ✅ 提供优化建议
- ✅ 在结果中标记为 `OOM_SKIPPED`
- ✅ **新增:**  SIGSEGV段错误检测 (GPU OOM常见原因)
- ✅ **新增:**  多种OOM检测模式 (OOM_SIGNAL, OOM_SIGSEGV)

## ✨ 价格增强功能

### 🔍 动态价格获取
- ✅ **实时Spot价格:**  通过AWS API获取当前Spot价格
- ✅ **按需价格:**  通过AWS Pricing API获取按需实例价格
- ✅ **实例检测:**  自动检测Pod实际运行的实例类型
- ✅ **价格缓存:**  避免重复API调用，脚本运行期间有效
- ✅ **后备价格:**  API失败时使用预设价格

### 💰 成本计算
- ✅ **1000张照片成本:**  基于实际推理时间和实例价格
- ✅ **三重成本计算:**  Spot、按需、实际成本对比
- ✅ **模型加载时间:**  包含模型加载开销
- ✅ **成本效率:**  计算每秒每张的成本
- ✅ **节省分析:**  自动计算Spot相对按需的节省

### 📊 增强报告
- ✅ **20字段CSV:**  包含完整价格和成本信息
- ✅ **成本分析:**  最低成本和最高性能配置对比
- ✅ **价格统计:**  Spot vs 按需实例使用统计
- ✅ **实时价格:**  标注价格获取时间

### 🔧 系统改进 🆕
- ✅ **智能OOM检测:**  区分任务成功完成和OOM情况
- ✅ **信号处理增强:**  支持SIGSEGV段错误检测 (GPU OOM常见)
- ✅ **Job冲突处理:**  自动清理已存在的同名Job
- ✅ **代码重构:**  消除重复代码，优化缓存机制

> 📋 **详细技术文档:**  查看 [`PRICE_FEATURES.md`](./PRICE_FEATURES.md) 了解价格功能的完整技术细节、API调用方式、缓存机制等。

## 📋 增强版CSV输出格式

```csv
test_name,model,requested_instance,actual_instance,lifecycle,zone,batch_size,inference_steps,resolution,precision,inference_time_s,time_per_image_s,gpu_memory_gb,spot_price_usd,ondemand_price_usd,actual_price_usd,cost_1000_images_spot_usd,cost_1000_images_ondemand_usd,cost_1000_images_actual_usd,status
sd-test,SD2.1,g5.xlarge,g5.xlarge,spot,us-west-2a,1,20,1024x1024,float16,15.2,15.2,18.5,0.301,1.006,0.301,1.28,4.27,1.28,SUCCESS
sdxl-test,SDXL,g5.xlarge,g6.xlarge,spot,us-west-2b,1,20,1024x1024,float16,25.8,25.8,22.1,0.245,0.758,0.245,1.96,6.08,1.96,SUCCESS
```

### 关键字段说明
| 字段 | 说明 |
|------|------|
| `requested_instance` | 请求的实例类型 |
| `actual_instance` | 实际分配的实例类型 |
| `lifecycle` | 实例生命周期 (spot/ondemand) |
| `spot_price_usd` | Spot价格 ($/小时) |
| `ondemand_price_usd` | 按需价格 ($/小时) |
| `actual_price_usd` | 实际价格 ($/小时) |
| `cost_1000_images_spot_usd` | 1000张Spot成本 ($) |
| `cost_1000_images_ondemand_usd` | 1000张按需成本 ($) |
| `cost_1000_images_actual_usd` | 1000张实际成本 ($) |

## 💡 价格功能使用示例

### 控制台输出示例
```
🔍 检测Pod实际运行的实例信息...
📍 实例信息: g5.xlarge (spot) 在 us-west-2a
💰 获取完整价格信息 (Spot + 按需)...
💰 Spot价格: $0.301/小时
💰 按需价格: $1.006/小时
💰 实际价格: $0.301/小时 (spot)
💡 Spot节省: 70.1% (相比按需价格)
📝 检测到模型加载时间: 28.5秒
⚡ 性能摘要: 推理时间=15.2s, 单图时间=15.2s, GPU内存=18.5GB
💰 成本分析 (1000张照片):
   Spot成本: $1.28
   按需成本: $4.27
   实际成本: $1.28 (spot实例)
📈 成本效率: Spot $0.000084/秒/张, 按需 $0.000281/秒/张
⚠️ Spot价格会波动，实际成本可能变化 ±30%
```

### 成本分析报告示例
```
💰 成本分析 (1000张照片):
🥇 最低Spot成本配置: sd-2-1-g4dnxlarge-b1-s20-float16
   Spot成本: $0.95
   按需成本: $3.16
   节省: $2.21 (69.9%)
   实例: g4dn.xlarge (spot)
   性能: 18.2s/张

🚀 最高性能配置: sdxl-g6exlarge-b1-s20-float16
   性能: 12.8s/张
   Spot成本: $2.34
   按需成本: $7.78
   实例: g6e.xlarge (spot)

📊 平均成本: Spot $1.67, 按需 $5.52 (1000张照片)
💡 平均Spot节省: 69.8%
```

## ⚙️ 价格API配置

### 前置要求
```bash
# 确保AWS CLI已配置
aws configure list

# 确保有以下权限
# - ec2:DescribeSpotPriceHistory
# - pricing:GetProducts

# 安装依赖 (如果需要)
# bc: 用于浮点数计算
# jq: 用于JSON解析
```

### 支持的实例类型价格
- ✅ g6e.xlarge, g6e.2xlarge
- ✅ g5.xlarge, g5.2xlarge  
- ✅ g6.xlarge, g6.2xlarge
- ✅ g4dn.xlarge, g4dn.2xlarge

## 📊 测试结果

测试完成后会生成：

1. **CSV 汇总:**  `universal_test_summary.csv` (20个字段)
2. **详细报告:**  `universal_final_report.txt` (包含成本分析)
3. **测试输出:**  `/shared/stable-diffusion-outputs/`

### 结果格式
```csv
test_name,model,requested_instance,actual_instance,lifecycle,zone,batch_size,inference_steps,resolution,precision,inference_time_s,time_per_image_s,gpu_memory_gb,spot_price_usd,ondemand_price_usd,actual_price_usd,cost_1000_images_spot_usd,cost_1000_images_ondemand_usd,cost_1000_images_actual_usd,status
sdxl-test,SDXL,g6e.xlarge,g6e.xlarge,spot,us-west-2a,1,20,1024x1024,float16,15.2,15.2,18.5,0.364,1.212,0.364,1.54,5.13,1.54,SUCCESS
oom-test,SDXL,g5.xlarge,g5.xlarge,spot,us-west-2b,4,20,1024x1024,float32,N/A,N/A,OOM,N/A,N/A,N/A,N/A,N/A,N/A,OOM_SKIPPED
```

## 🚨 价格注意事项

1. **Spot价格波动:**  价格每几分钟变化，实际成本可能 ±30%
2. **实例类型变化:**  Karpenter可能分配不同的实例类型
3. **区域差异:**  不同可用区价格可能不同
4. **API限制:**  价格API有调用频率限制
5. **缓存机制:**  价格缓存在脚本运行期间有效，避免频繁调用

## 💡 使用技巧

1. **对比测试:**  使用 `comparison` 模式确保公平对比
2. **内存优化:**  优先使用 `float16` 精度和 `batch_size=1`
3. **实例选择:**  SDXL 推荐 `g6e.xlarge`，标准 SD 可用 `g5.xlarge`
4. **成本优化:**  查看CSV中的Spot成本列找到最便宜配置
5. **OOM 处理:**  系统会自动处理，无需担心测试中断

## 🔍 示例命令

```bash
# 查看帮助
python3 generate_universal_test_config.py --help

# 运行示例测试 (包含价格分析)
./r.sh

# 运行自定义测试
./run_universal_tests.sh your_config.json

# 查看结果
cat results/universal_test_summary.csv
cat results/universal_final_report.txt | grep -A 20 "成本分析"
```

## 🔧 故障排除

### 价格获取失败
```bash
# 检查AWS权限
aws ec2 describe-spot-price-history --instance-types g5.xlarge --max-items 1

# 检查Pricing API权限
aws pricing get-products --service-code AmazonEC2 --region us-east-1 --max-items 1
```

### 实例信息获取失败
```bash
# 检查Pod状态
kubectl get pods -l job-name=your-test-name

# 检查节点标签
kubectl get nodes --show-labels
```

> 🔧 **详细故障排除:**  查看 [`PRICE_FEATURES.md`](./PRICE_FEATURES.md) 获取完整的故障排除指南、API调用示例和错误处理机制。

通过价格增强功能，你可以：
- 🎯 找到最具成本效益的配置
- 📊 比较不同实例类型的真实成本
- 💰 基于实时价格做出决策
- 📈 优化大规模图像生成的预算
- 🔍 量化Spot实例的节省效果
