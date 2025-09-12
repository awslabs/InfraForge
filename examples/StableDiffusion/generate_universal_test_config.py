#!/usr/bin/env python3
"""
通用Stable Diffusion测试配置生成器
支持多种机型、批次大小、推理步数、分辨率和精度
支持标准SD模型和SDXL模型的智能配置
"""

import json
import argparse
import itertools
from datetime import datetime

def generate_test_config(
    models=None,
    instance_types=None,
    batch_sizes=None,
    inference_steps=None,
    resolutions=None,
    precisions=None,
    prompt="a photo of an astronaut riding a horse on mars",
    output_file="universal_test_config.json",
    test_mode="mixed",
    comparison_memory_request="12Gi",
    comparison_memory_limit="15Gi",
    comparison_timeout=2400
):
    """生成通用测试配置"""
    
    # 根据测试模式设置默认值
    if test_mode == "sdxl_only":
        # SDXL专用模式
        if models is None:
            models = ["stabilityai/stable-diffusion-xl-base-1.0"]
        if instance_types is None:
            instance_types = ["g6e.xlarge", "g5.xlarge", "g6.xlarge"]
        if batch_sizes is None:
            batch_sizes = [1]
        if inference_steps is None:
            inference_steps = [20, 30, 50]
        if resolutions is None:
            resolutions = ["1024x1024", "1152x896", "896x1152"]
        if precisions is None:
            precisions = ["float16"]
    elif test_mode == "sd_only":
        # 标准SD专用模式
        if models is None:
            models = ["stabilityai/stable-diffusion-2-1"]
        if instance_types is None:
            instance_types = ["g5.xlarge", "g6.xlarge"]
        if batch_sizes is None:
            batch_sizes = [1, 4]
        if inference_steps is None:
            inference_steps = [15, 25, 50]
        if resolutions is None:
            resolutions = ["512x512", "1024x1024"]
        if precisions is None:
            precisions = ["float16", "float32"]
    elif test_mode == "comparison":
        # 对比模式
        if models is None:
            models = ["stabilityai/stable-diffusion-2-1", "stabilityai/stable-diffusion-xl-base-1.0"]
        if instance_types is None:
            instance_types = ["g6e.xlarge", "g5.xlarge", "g6.xlarge"]
        if batch_sizes is None:
            batch_sizes = [1]
        if inference_steps is None:
            inference_steps = [20, 30]
        if resolutions is None:
            resolutions = ["1024x1024"]
        if precisions is None:
            precisions = ["float16"]
    elif test_mode == "instance_optimized":
        # 实例优化模式 - 针对不同实例类型的最佳配置
        if models is None:
            models = ["stabilityai/stable-diffusion-xl-base-1.0"]
        if instance_types is None:
            instance_types = ["g6e.xlarge", "g5.xlarge", "g6.xlarge", "g4dn.xlarge"]
        if batch_sizes is None:
            batch_sizes = [1]
        if inference_steps is None:
            inference_steps = [20]
        if resolutions is None:
            # 根据实例类型动态调整分辨率
            resolutions = ["1024x1024", "896x896"]  # 包含备用分辨率
        if precisions is None:
            precisions = ["float16"]
    else:
        # 混合模式（默认）
        if models is None:
            models = ["stabilityai/stable-diffusion-2-1"]
        if instance_types is None:
            instance_types = ["g5.xlarge"]
        if batch_sizes is None:
            batch_sizes = [1, 4]
        if inference_steps is None:
            inference_steps = [15, 25, 50]
        if resolutions is None:
            resolutions = ["1024x1024"]
        if precisions is None:
            precisions = ["float32"]
    
    # 生成所有组合
    combinations = list(itertools.product(
        models, instance_types, batch_sizes, inference_steps, resolutions, precisions
    ))
    
    tests = []
    for i, (model, instance, batch, steps, resolution, precision) in enumerate(combinations):
        # 解析分辨率
        width, height = map(int, resolution.split('x'))
        
        # 检查是否为SDXL模型
        is_sdxl = 'stable-diffusion-xl' in model or 'sdxl' in model.lower()
        
        # 智能配置验证和调整
        adjusted_config = validate_and_adjust_config(
            model, instance, batch, steps, width, height, precision, is_sdxl, test_mode,
            comparison_memory_request, comparison_memory_limit, comparison_timeout
        )
        
        if adjusted_config is None:
            continue  # 跳过不兼容的配置
        
        model, instance, batch, steps, width, height, precision, memory_request, memory_limit, timeout = adjusted_config
        
        # 生成简短的测试名称
        model_short = model.split('/')[-1].replace('stable-diffusion-', 'sd-').replace('.', '-')
        if 'xl' in model_short:
            model_short = model_short.replace('sd-xl-', 'sdxl-')
        instance_short = instance.replace('.', '')
        test_name = f"{model_short}-{instance_short}-b{batch}-s{steps}-{precision}"
        
        # 确保名称不超过50个字符
        if len(test_name) > 50:
            test_name = test_name[:50]
        
        test = {
            "name": test_name,
            "description": f"{model} test: {instance}, batch={batch}, steps={steps}, {width}x{height}, {precision}",
            "model": model,
            "model_type": "SDXL" if is_sdxl else "Standard SD",
            "instance_type": instance,
            "batch_size": batch,
            "inference_steps": steps,
            "image_width": width,
            "image_height": height,
            "precision": precision,
            "prompt": prompt,
            "memory_request": memory_request,
            "memory_limit": memory_limit,
            "timeout": timeout
        }
        tests.append(test)
    
    # 创建配置文件
    config = {
        "description": f"Universal Stable Diffusion test configuration - {test_mode} mode",
        "version": "1.0",
        "test_mode": test_mode,
        "generated_by": "generate_universal_test_config.py",
        "generated_at": datetime.now().isoformat(),
        "total_tests": len(tests),
        "instance_specifications": {
            "g6e.xlarge": {"cpu_memory": "32GB", "gpu_memory": "48GB", "sdxl_status": "EXCELLENT"},
            "g5.xlarge": {"cpu_memory": "16GB", "gpu_memory": "24GB", "sdxl_status": "GOOD"},
            "g6.xlarge": {"cpu_memory": "16GB", "gpu_memory": "24GB", "sdxl_status": "GOOD"},
            "g4dn.xlarge": {"cpu_memory": "16GB", "gpu_memory": "16GB", "sdxl_status": "NOT_RECOMMENDED"}
        },
        "default_settings": {
            "prompt": prompt,
            "timeout": 1800,
            "method": "universal_intelligent"
        },
        "test_matrix": {
            "models": models,
            "instance_types": instance_types,
            "batch_sizes": batch_sizes,
            "inference_steps": inference_steps,
            "resolutions": resolutions,
            "precisions": precisions
        },
        "tests": tests
    }
    
    # 保存配置文件
    with open(output_file, 'w', encoding='utf-8') as f:
        json.dump(config, f, indent=2, ensure_ascii=False)
    
    print(f"✅ 生成了 {len(tests)} 个测试配置 ({test_mode} 模式)")
    print(f"📁 配置文件保存到: {output_file}")
    print(f"🔧 测试矩阵:")
    print(f"   模型: {', '.join(models)}")
    print(f"   实例类型: {', '.join(instance_types)}")
    print(f"   批次大小: {', '.join(map(str, batch_sizes))}")
    print(f"   推理步数: {', '.join(map(str, inference_steps))}")
    print(f"   分辨率: {', '.join(resolutions)}")
    print(f"   精度: {', '.join(precisions)}")
    print(f"   提示词: {prompt}")
    
    # 显示模式特定信息
    if test_mode == "sdxl_only":
        print(f"\n🎯 SDXL专用模式特性:")
        print(f"   ✅ 针对SDXL优化的实例类型")
        print(f"   ✅ 强制使用float16精度")
        print(f"   ✅ 原生1024x1024分辨率")
        print(f"   ✅ 智能内存配置")
    elif test_mode == "comparison":
        print(f"\n📊 对比模式特性:")
        print(f"   ✅ SD 2.1 vs SDXL 性能对比")
        print(f"   ✅ 统一资源配置确保公平对比")
        print(f"   ✅ 所有测试使用相同的CPU内存配置")
        print(f"   ✅ 尊重用户指定的精度和批次大小")
        print(f"   ⚠️  用户需要自行确保配置合理性（避免OOM等问题）")
        print(f"   📋 统一配置: {comparison_memory_request}/{comparison_memory_limit} CPU内存, 用户指定精度, timeout={comparison_timeout}s")
    elif test_mode == "instance_optimized":
        print(f"\n🔧 实例优化模式特性:")
        print(f"   ✅ 针对每个实例类型的最佳配置")
        print(f"   ✅ 自动跳过不兼容配置")
        print(f"   ✅ 包含备用分辨率")
    
    print(f"\n🚀 运行测试:")
    print(f"   chmod +x run_universal_tests.sh")
    print(f"   ./run_universal_tests.sh {output_file}")
    
    return config

def validate_and_adjust_config(model, instance, batch, steps, width, height, precision, is_sdxl, test_mode, 
                              comparison_memory_request="12Gi", comparison_memory_limit="15Gi", comparison_timeout=2400):
    """验证和调整配置，确保兼容性"""
    
    # 对比模式下使用统一配置，确保公平对比
    if test_mode == "comparison":
        # 对比模式：所有测试使用相同的资源配置
        # 使用用户指定的内存配置
        memory_request = comparison_memory_request
        memory_limit = comparison_memory_limit
        timeout = comparison_timeout
        
        # 保持用户指定的精度和批次大小，不强制修改
        # 用户需要自己确保配置的合理性
        return model, instance, batch, steps, width, height, precision, memory_request, memory_limit, timeout
    
    # 非对比模式下的智能配置调整
    # SDXL在g4dn.xlarge上不推荐，除非是instance_optimized模式
    if is_sdxl and 'g4dn.xlarge' in instance and test_mode != "instance_optimized":
        return None  # 跳过不兼容配置
    
    # 在非对比模式下，仍然进行一些智能调整以避免明显的问题
    # 但用户可以通过选择合适的模式来避免这些调整
    
    # SDXL必须使用float16在小GPU内存实例上（仅在非对比模式下调整）
    if is_sdxl and instance in ['g5.xlarge', 'g6.xlarge', 'g4dn.xlarge'] and precision != 'float16':
        precision = 'float16'  # 自动调整
    
    # SDXL在小GPU内存实例上批次大小限制为1（仅在非对比模式下调整）
    if is_sdxl and instance in ['g5.xlarge', 'g6.xlarge', 'g4dn.xlarge'] and batch > 1:
        batch = 1  # 自动调整
    
    # g4dn.xlarge上的SDXL降低分辨率（仅在非对比模式下调整）
    if is_sdxl and 'g4dn.xlarge' in instance and width >= 1024:
        width, height = 896, 896  # 降低到更安全的分辨率
    
    # 根据实例类型和模型设置CPU内存
    if 'g6e.xlarge' in instance:
        # g6e.xlarge: 32GB CPU内存, 48GB GPU内存
        if is_sdxl:
            memory_request = "16Gi"
            memory_limit = "24Gi"
            timeout = 2400
        else:
            memory_request = "12Gi"
            memory_limit = "20Gi"
            timeout = 1800
    elif 'g5.xlarge' in instance or 'g6.xlarge' in instance:
        # g5/g6.xlarge: 16GB CPU内存, 24GB GPU内存
        if is_sdxl:
            memory_request = "8Gi"
            memory_limit = "12Gi"
            timeout = 2400
        else:
            memory_request = "6Gi"
            memory_limit = "10Gi"
            timeout = 1800
    elif 'g4dn.xlarge' in instance:
        # g4dn.xlarge: 16GB CPU内存, 16GB GPU内存
        if is_sdxl:
            memory_request = "8Gi"
            memory_limit = "12Gi"
            timeout = 3000  # 更长超时，因为可能很慢
        else:
            memory_request = "6Gi"
            memory_limit = "10Gi"
            timeout = 1800
    else:
        # 默认配置
        if is_sdxl:
            memory_request = "8Gi"
            memory_limit = "12Gi"
            timeout = 2400
        else:
            memory_request = "6Gi"
            memory_limit = "10Gi"
            timeout = 1800
    
    return model, instance, batch, steps, width, height, precision, memory_request, memory_limit, timeout

def main():
    parser = argparse.ArgumentParser(description='生成通用Stable Diffusion测试配置')
    
    # 测试模式选择
    parser.add_argument('--mode', choices=['mixed', 'sdxl_only', 'sd_only', 'comparison', 'instance_optimized'],
                       default='mixed',
                       help='测试模式: mixed(混合), sdxl_only(仅SDXL), sd_only(仅标准SD), comparison(对比), instance_optimized(实例优化)')
    
    parser.add_argument('--models', nargs='+', 
                       help='模型列表 (默认根据模式自动选择)')
    
    parser.add_argument('--instance-types', nargs='+',
                       help='实例类型列表 (默认根据模式自动选择)')
    
    parser.add_argument('--batch-sizes', nargs='+', type=int,
                       help='批次大小列表 (默认根据模式自动选择)')
    
    parser.add_argument('--inference-steps', nargs='+', type=int,
                       help='推理步数列表 (默认根据模式自动选择)')
    
    parser.add_argument('--resolutions', nargs='+',
                       help='分辨率列表 (默认根据模式自动选择)')
    
    parser.add_argument('--precisions', nargs='+',
                       help='精度列表 (默认根据模式自动选择)')
    
    parser.add_argument('--prompt',
                       default="a photo of an astronaut riding a horse on mars",
                       help='提示词 (默认: a photo of an astronaut riding a horse on mars)')
    
    parser.add_argument('--output', '-o',
                       default="universal_test_config.json",
                       help='输出文件名 (默认: universal_test_config.json)')
    
    # 对比模式专用参数
    parser.add_argument('--comparison-memory-request',
                       default="12Gi",
                       help='对比模式CPU内存请求 (默认: 12Gi，适合16GB内存机器)')
    
    parser.add_argument('--comparison-memory-limit',
                       default="15Gi", 
                       help='对比模式CPU内存限制 (默认: 15Gi，更好利用硬件资源)')
    
    parser.add_argument('--comparison-timeout', type=int,
                       default=2400,
                       help='对比模式超时时间秒数 (默认: 2400秒/40分钟)')
    
    args = parser.parse_args()
    
    # 显示模式说明
    mode_descriptions = {
        'mixed': '混合模式 - 标准配置，适合一般测试',
        'sdxl_only': 'SDXL专用模式 - 针对SDXL优化的配置',
        'sd_only': '标准SD专用模式 - 仅测试标准SD模型',
        'comparison': '对比模式 - SD 2.1 vs SDXL 性能对比',
        'instance_optimized': '实例优化模式 - 针对不同实例类型的最佳配置'
    }
    
    print(f"🎯 选择的测试模式: {args.mode}")
    print(f"📝 模式说明: {mode_descriptions[args.mode]}")
    
    # 显示对比模式的内存配置建议
    if args.mode == "comparison":
        print(f"\n💡 对比模式内存配置建议:")
        print(f"   16GB CPU内存机器: --comparison-memory-request 12Gi --comparison-memory-limit 15Gi (默认)")
        print(f"   32GB CPU内存机器: --comparison-memory-request 24Gi --comparison-memory-limit 30Gi")
        print(f"   保守配置 (如有其他工作负载): --comparison-memory-request 8Gi --comparison-memory-limit 12Gi")
        print(f"   当前配置: {args.comparison_memory_request}/{args.comparison_memory_limit}, 超时{args.comparison_timeout}秒")
    
    print()
    
    generate_test_config(
        models=args.models,
        instance_types=args.instance_types,
        batch_sizes=args.batch_sizes,
        inference_steps=args.inference_steps,
        resolutions=args.resolutions,
        precisions=args.precisions,
        prompt=args.prompt,
        output_file=args.output,
        test_mode=args.mode,
        comparison_memory_request=args.comparison_memory_request,
        comparison_memory_limit=args.comparison_memory_limit,
        comparison_timeout=args.comparison_timeout
    )

if __name__ == "__main__":
    main()
