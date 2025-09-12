// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// 创建一个新的 MCP server
	s := server.NewMCPServer(
		"InfraForge",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// 添加 listTemplates 工具
	listTemplatesT := mcp.NewTool("listTemplates",
		mcp.WithDescription("列出可用的 InfraForge 配置模板"),
	)
	s.AddTool(listTemplatesT, listTemplatesHandler)
	
	// 添加 getOperationManual 工具
	getOperationManualT := mcp.NewTool("getOperationManual",
		mcp.WithDescription("获取 InfraForge 操作手册 - 强制要求"),
	)
	s.AddTool(getOperationManualT, getOperationManualHandler)

	// 添加 deployInfra 工具
	deployInfraT := mcp.NewTool("deployInfra",
		mcp.WithDescription("选择配置文件并部署基础设施"),
		mcp.WithString("sourceConfig",
			mcp.Description("要使用的源配置文件名称"),
		),
		mcp.WithString("description",
			mcp.Description("方案描述，用于 Amazon Q 理解需求"),
		),
		mcp.WithString("preserveOriginal",
			mcp.Description("是否保留原始配置文件（true/false）"),
		),
		mcp.WithString("modifications",
			mcp.Description("JSON格式的配置修改，用于指定实例类型等具体参数"),
		),
	)
	s.AddTool(deployInfraT, deployInfraHandler)

	// 添加 getDeploymentStatus 工具
	getDeploymentStatusT := mcp.NewTool("getDeploymentStatus",
		mcp.WithDescription("获取部署状态"),
		mcp.WithString("stackName",
			mcp.Required(),
			mcp.Description("要查询的堆栈名称"),
		),
	)
	s.AddTool(getDeploymentStatusT, getDeploymentStatusHandler)

	// 添加 getStackOutputs 工具
	getStackOutputsT := mcp.NewTool("getStackOutputs",
		mcp.WithDescription("获取堆栈的输出参数"),
		mcp.WithString("stackName",
			mcp.Required(),
			mcp.Description("要查询的堆栈名称"),
		),
	)
	s.AddTool(getStackOutputsT, getStackOutputsHandler)

	// 启动 stdio 服务器
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// listTemplatesHandler 列出当前目录和configs目录下的所有配置模板
func listTemplatesHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	templates := []string{}

	// 首先在当前目录查找
	matches, err := filepath.Glob("config_*.json")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("查找模板时出错: %v", err)), nil
	}
	templates = append(templates, matches...)

	// 然后在 configs 目录及其子目录中查找
	configsDir := "configs"
	if _, err := os.Stat(configsDir); err == nil {
		err := filepath.Walk(configsDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasPrefix(filepath.Base(path), "config_") && strings.HasSuffix(filepath.Base(path), ".json") {
				templates = append(templates, path)
			}
			return nil
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("查找 configs 目录中的模板时出错: %v", err)), nil
		}
	}

	// 添加默认配置文件
	defaultConfig := "config.json"
	if _, err := os.Stat(defaultConfig); err == nil {
		templates = append([]string{defaultConfig + " (默认)"}, templates...)
	}

	if len(templates) == 0 {
		return mcp.NewToolResultText("未找到配置模板"), nil
	}

	// 读取每个模板文件，提取描述信息
	var templateInfos []string
	for _, template := range templates {
		description := "无描述"
		
		// 读取文件内容
		data, err := os.ReadFile(template)
		if err == nil {
			var config map[string]interface{}
			if err := json.Unmarshal(data, &config); err == nil {
				if global, ok := config["global"].(map[string]interface{}); ok {
					if desc, ok := global["description"].(string); ok && desc != "" {
						description = desc
					}
				}
			}
		}
		
		templateInfos = append(templateInfos, fmt.Sprintf("%s - %s", template, description))
	}

	// 添加提示，告知用户必须使用 getOperationManual 工具获取详细操作手册
	result := "**⚠️ 严重警告 ⚠️**: 为避免配置错误和部署失败，请先执行 getOperationManual 获取完整操作手册。所有操作必须严格遵循手册要求，不得有任何偏差。\n"

	result += "\n\n可用的配置模板:\n" + strings.Join(templateInfos, "\n")

	return mcp.NewToolResultText(result), nil
}

// getOperationManualHandler 返回 InfraForge 的强制性操作手册
func getOperationManualHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 强制性操作手册
	result := "**InfraForge 操作手册 - 强制要求:**\n"
	result += "**⚠️ 严重警告 ⚠️**: 不遵循以下要求将导致部署失败或资源配置错误。所有操作必须严格按照规定执行，不得有任何偏差。\n\n"
	result += "1. **必须** 选择合适的模板文件并查看模板内容：每个模板都附有详细说明，描述其具体用途。**必须先查阅模板文件内容**，以充分了解其配置结构和语法规则，尽量使用默认配置, 除非用户要求更改。【不遵循将导致配置错误】\n"
	result += "2. **必须** 遵循配置修改原则：当需要自定义配置时，**只能**通过修改或添加 forges.*.instances 中的配置项来实现，**禁止**直接修改 forges.*.defaults 部分。因为 instance 配置会覆盖 defaults 中的配置，这样可以确保不同实例有各自的配置而不影响其他实例。保持 defaults 部分不变可以维护共享基础配置的一致性。【违反此规则将破坏配置继承机制】\n"
	result += "3. **必须** 严格遵循用户要求：只修改用户明确要求的配置项，**禁止**添加额外的配置或擅自更改默认值。【添加未经授权的配置将导致意外行为】\n"
	result += "4. **必须** 遵守模板命名规则：模板文件名格式为 config_xxx.json，生成临时配置文件时**禁止**使用相同的命名规则，以防冲突。【命名冲突将导致配置覆盖】\n"
	result += "5. **必须** 保持堆栈名称一致：部署新的方案本质是在合并新方案到现有 stack, 所以**禁止**修改模板中 stackName，因为修改了也没用，deployInfra 智能合并会使用 ./config.json 中 stackName。【修改堆栈名称将导致部署失败】\n"
	result += "6. 使用现有VPC时，**必须**在 vpc.defaults 中设置 vpcID。【否则将创建新VPC导致网络冲突】\n"
	result += "7. **必须** 遵循模板适用性原则：若未找到合适的模板，**必须**直接向用户反馈，明确说明当前没有满足要求的解决方案，**禁止**提供不确定的信息。【提供错误信息将导致用户决策失误】\n"
	result += "8. **必须** 理解配置合并机制：deployInfra 会合并所选模板配置和当前目录 config.json 中的配置，且 config.json 配置具有优先权，以保护现有环境。如需删除或修改资源，**必须**修改当前目录 config.json 内容，只有模板名为 config.json 时 deployInfra 才不会做合并, 修改后使用 config.json 作为模板部署以实现变更。【不遵循将导致配置合并错误】\n"
	result += "9. **必须** 使用正确的部署监控工具：使用 getDeploymentStatus 和 getStackOutputs 工具监控部署状态和获取输出信息。【否则无法获取准确的部署状态】\n"
	result += "10. **必须** 遵守远程操作限制：可以使用 aws cli ssm 连接到节点执行任务，但由于 Amazon Q 不支持交互操作，**禁止**使用 aws cli ssm 的 start-session 命令。【违反此规则将导致操作中断】\n"
	result += "11. **必须** 遵循性能测试模板选择规则：对于单机性能测试，**禁止**选择 ec2 模板，**必须**选择 sysbench 模板，不同 ec2 类型要注意 osArch 参数。【选择错误模板将导致测试结果不准确】\n"

	return mcp.NewToolResultText(result), nil
}

func formatJSON(jsonStr string) string {
	var obj interface{}
	err := json.Unmarshal([]byte(jsonStr), &obj)
	if err != nil {
		return jsonStr // 如果解析失败，返回原始字符串
	}
	
	prettyJSON, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return jsonStr // 如果格式化失败，返回原始字符串
	}
	
	return string(prettyJSON)
}

// safelyMergeConfigs 安全地合并配置，保持 dst 的优先级
// 只添加 dst 中不存在的配置项
func safelyMergeConfigs(dst, src map[string]interface{}) {
	for k, v := range src {
		if _, exists := dst[k]; !exists {
			// 只有当目标中不存在该键时才添加
			dst[k] = v
		} else if dstMap, isDstMap := dst[k].(map[string]interface{}); isDstMap {
			if srcMap, isSrcMap := v.(map[string]interface{}); isSrcMap {
				// 如果两者都是嵌套的映射，递归合并
				safelyMergeConfigs(dstMap, srcMap)
			}
			// 如果目标是映射但源不是，保留目标
		} else if dstArray, isDstArray := dst[k].([]interface{}); isDstArray {
			if srcArray, isSrcArray := v.([]interface{}); isSrcArray {
				// 如果两者都是数组，需要特殊处理
				// 对于 instances 数组，我们需要根据 id 字段合并
				if k == "instances" {
					// 创建一个映射，用于快速查找实例
					instanceMap := make(map[string]int)
					for i, item := range dstArray {
						if itemMap, isItemMap := item.(map[string]interface{}); isItemMap {
							if id, hasID := itemMap["id"].(string); hasID {
								instanceMap[id] = i
							}
						}
					}
					
					// 合并或添加源数组中的实例
					for _, srcItem := range srcArray {
						if srcItemMap, isSrcItemMap := srcItem.(map[string]interface{}); isSrcItemMap {
							if id, hasID := srcItemMap["id"].(string); hasID {
								if idx, exists := instanceMap[id]; exists {
									// 如果目标数组中已存在相同ID的实例，合并它们
									if dstItemMap, isDstItemMap := dstArray[idx].(map[string]interface{}); isDstItemMap {
										safelyMergeConfigs(dstItemMap, srcItemMap)
									}
								} else {
									// 如果目标数组中不存在该ID的实例，添加它
									dstArray = append(dstArray, srcItem)
									dst[k] = dstArray
								}
							} else {
								// 如果源实例没有ID，直接添加它
								dstArray = append(dstArray, srcItem)
								dst[k] = dstArray
							}
						}
					}
				} else {
					// 对于其他类型的数组，简单地追加不重复的元素
					for _, srcItem := range srcArray {
						found := false
						for _, dstItem := range dstArray {
							if fmt.Sprintf("%v", srcItem) == fmt.Sprintf("%v", dstItem) {
								found = true
								break
							}
						}
						if !found {
							dstArray = append(dstArray, srcItem)
						}
					}
					dst[k] = dstArray
				}
			}
		}
		// 在所有其他情况下，保留目标中的值
	}
}

// removeConfigPath 从配置中移除指定路径的项
func removeConfigPath(config map[string]interface{}, path string) {
	parts := strings.Split(path, ".")
	if len(parts) == 1 {
		// 直接删除顶层键
		delete(config, parts[0])
		return
	}
	
	// 处理嵌套路径
	current := config
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		
		// 处理数组索引，格式如 "forges.ec2.instances[2]"
		indexStart := strings.Index(part, "[")
		if indexStart > 0 && strings.HasSuffix(part, "]") {
			key := part[:indexStart]
			indexStr := part[indexStart+1 : len(part)-1]
			index, err := strconv.Atoi(indexStr)
			if err != nil {
				// 索引格式错误，无法继续
				return
			}
			
			if nextObj, ok := current[key]; ok {
				if nextArray, ok := nextObj.([]interface{}); ok && index >= 0 && index < len(nextArray) {
					// 如果是最后一个部分，从数组中删除该元素
					if i == len(parts)-2 && parts[i+1] == "" {
						newArray := append(nextArray[:index], nextArray[index+1:]...)
						current[key] = newArray
						return
					}
					
					// 否则继续处理下一级
					if nextMap, ok := nextArray[index].(map[string]interface{}); ok {
						current = nextMap
						continue
					}
				}
			}
			// 路径不存在或格式错误，无需继续
			return
		}
		
		// 处理普通嵌套对象
		if next, ok := current[part].(map[string]interface{}); ok {
			current = next
		} else {
			// 路径不存在，无需继续
			return
		}
	}
	
	// 删除最后一级的键
	delete(current, parts[len(parts)-1])
}

// deployInfraHandler 部署基础设施
func deployInfraHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 获取参数
	sourceConfig := request.GetString("sourceConfig", "config.json")
	description := request.GetString("description", "")
	preserveOriginal := request.GetString("preserveOriginal", "false")
	modifications := request.GetString("modifications", "")
	
	// 默认配置文件名
	defaultConfig := "config.json"
	
	// 如果源配置就是默认配置，并且没有修改，直接执行部署
	if sourceConfig == defaultConfig && (modifications == "" || modifications == "{}") {
		// 执行 CDK 部署命令
		cmd := exec.Command("cdk", "deploy", "--app", "./infraforge", "--require-approval", "never")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("部署失败: %v\n%s", err, string(output))), nil
		}

		result := fmt.Sprintf("部署已启动:\n%s", string(output))
		if description != "" {
			result = fmt.Sprintf("部署描述: %s\n\n%s", description, result)
		}

		return mcp.NewToolResultText(result), nil
	}
	
	// 首先读取源配置文件
	var sourceConfigMap map[string]interface{}
	
	if _, err := os.Stat(sourceConfig); os.IsNotExist(err) {
		return mcp.NewToolResultError(fmt.Sprintf("配置文件 %s 不存在", sourceConfig)), nil
	}

	// 读取源配置文件
	sourceConfigData, err := os.ReadFile(sourceConfig)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("读取配置文件 %s 失败: %v", sourceConfig, err)), nil
	}

	// 解析源配置文件
	if err := json.Unmarshal(sourceConfigData, &sourceConfigMap); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("解析配置文件 %s 失败: %v", sourceConfig, err)), nil
	}
	
	// 创建源配置的副本，以便在需要时应用修改
	var workingConfigMap map[string]interface{}
	workingConfigBytes, err := json.Marshal(sourceConfigMap)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("复制源配置失败: %v", err)), nil
	}
	if err := json.Unmarshal(workingConfigBytes, &workingConfigMap); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("解析复制的配置失败: %v", err)), nil
	}
	
	// 应用修改（如果有）
	if modifications != "" && modifications != "{}" {
		var modMap map[string]interface{}
		if err := json.Unmarshal([]byte(modifications), &modMap); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("解析修改参数失败: %v", err)), nil
		}
		
		// 合并修改到工作配置中
		mergeConfigs(workingConfigMap, modMap)
	}
	
	// 检查是否需要保留原始配置
	if preserveOriginal == "true" {
		// 保留源配置文件不变，将修改后的配置写入 config.json
		
		// 然后读取默认配置文件 config.json
		var configMap map[string]interface{}
		
		// 检查默认配置文件是否存在
		defaultConfigData, err := os.ReadFile(defaultConfig)
		if err == nil {
			// 创建带时间戳的备份
			timestamp := time.Now().Format("20060102_150405")
			backupFile := fmt.Sprintf("%s.%s.bak", defaultConfig, timestamp)
			if err := os.WriteFile(backupFile, defaultConfigData, 0644); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("创建配置文件备份失败: %v", err)), nil
			}
			
			// 解析默认配置文件
			if err := json.Unmarshal(defaultConfigData, &configMap); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("解析默认配置文件失败: %v", err)), nil
			}
			
			// 将工作配置合并到默认配置中，保持默认配置的优先级
			safelyMergeConfigs(configMap, workingConfigMap)
		} else if os.IsNotExist(err) {
			// 默认配置文件不存在，直接使用工作配置
			configMap = workingConfigMap
		} else {
			// 其他错误
			return mcp.NewToolResultError(fmt.Sprintf("读取默认配置文件失败: %v", err)), nil
		}
		
		// 将合并后的配置转换回 JSON
		configData, err := json.MarshalIndent(configMap, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("生成修改后的配置失败: %v", err)), nil
		}

		// 写入配置文件
		if err := os.WriteFile(defaultConfig, configData, 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("写入配置文件失败: %v", err)), nil
		}
	} else {
		// 不保留源配置文件，将修改后的配置写回源配置文件
		
		// 创建源配置文件的备份
		timestamp := time.Now().Format("20060102_150405")
		backupFile := fmt.Sprintf("%s.%s.bak", sourceConfig, timestamp)
		if err := os.WriteFile(backupFile, sourceConfigData, 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("创建源配置文件备份失败: %v", err)), nil
		}
		
		// 将工作配置写回源配置文件
		workingConfigData, err := json.MarshalIndent(workingConfigMap, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("生成修改后的源配置失败: %v", err)), nil
		}
		
		if err := os.WriteFile(sourceConfig, workingConfigData, 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("写入源配置文件失败: %v", err)), nil
		}
		
		// 然后读取默认配置文件 config.json
		var configMap map[string]interface{}
		
		// 检查默认配置文件是否存在
		defaultConfigData, err := os.ReadFile(defaultConfig)
		if err == nil {
			// 创建带时间戳的备份
			timestamp := time.Now().Format("20060102_150405")
			backupFile := fmt.Sprintf("%s.%s.bak", defaultConfig, timestamp)
			if err := os.WriteFile(backupFile, defaultConfigData, 0644); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("创建配置文件备份失败: %v", err)), nil
			}
			
			// 解析默认配置文件
			if err := json.Unmarshal(defaultConfigData, &configMap); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("解析默认配置文件失败: %v", err)), nil
			}
			
			// 将工作配置合并到默认配置中，保持默认配置的优先级
			safelyMergeConfigs(configMap, workingConfigMap)
		} else if os.IsNotExist(err) {
			// 默认配置文件不存在，直接使用工作配置
			configMap = workingConfigMap
		} else {
			// 其他错误
			return mcp.NewToolResultError(fmt.Sprintf("读取默认配置文件失败: %v", err)), nil
		}
		
		// 将合并后的配置转换回 JSON
		configData, err := json.MarshalIndent(configMap, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("生成修改后的配置失败: %v", err)), nil
		}

		// 写入配置文件
		if err := os.WriteFile(defaultConfig, configData, 0644); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("写入配置文件失败: %v", err)), nil
		}
	}

	// 执行 CDK 部署命令
	cmd := exec.Command("cdk", "deploy", "--app", "./infraforge", "--require-approval", "never")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("部署失败: %v\n%s", err, string(output))), nil
	}

	result := fmt.Sprintf("部署已启动:\n%s", string(output))
	if description != "" {
		result = fmt.Sprintf("部署描述: %s\n\n%s", description, result)
	}

	return mcp.NewToolResultText(result), nil
}

// getDeploymentStatusHandler 获取部署状态
func getDeploymentStatusHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stackName, err := request.RequireString("stackName")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 执行 AWS CLI 命令获取堆栈状态
	cmd := exec.Command("aws", "cloudformation", "describe-stacks", "--stack-name", stackName, "--query", "Stacks[0].StackStatus", "--output", "text")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取堆栈状态失败: %v\n%s", err, string(output))), nil
	}

	status := strings.TrimSpace(string(output))
	result := fmt.Sprintf("堆栈 %s 的当前状态: %s", stackName, status)
	result += "\n\n注意:\n可以用 aws cli ssm 连到节点执行任务, 但 Amazon Q 交不支持交互操作，所以不要使用 start-session\n"
	return mcp.NewToolResultText(result), nil
}

// getStackOutputsHandler 获取堆栈输出
func getStackOutputsHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stackName, err := request.RequireString("stackName")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// 执行 AWS CLI 命令获取堆栈输出
	cmd := exec.Command("aws", "cloudformation", "describe-stacks", "--stack-name", stackName, "--query", "Stacks[0].Outputs", "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("获取堆栈输出失败: %v\n%s", err, string(output))), nil
	}

	// 格式化输出
	var outputs []map[string]interface{}
	if err := json.Unmarshal(output, &outputs); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("解析堆栈输出失败: %v", err)), nil
	}

	if len(outputs) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("堆栈 %s 没有输出参数", stackName)), nil
	}

	// 构建格式化的输出
	var result strings.Builder
	result.WriteString(fmt.Sprintf("堆栈 %s 的输出参数:\n\n", stackName))
	
	for _, output := range outputs {
		key := output["OutputKey"].(string)
		value := output["OutputValue"].(string)
		description := ""
		if desc, ok := output["Description"]; ok {
			description = desc.(string)
		}
		
		result.WriteString(fmt.Sprintf("- %s: %s\n", key, value))
		if description != "" {
			result.WriteString(fmt.Sprintf("  描述: %s\n", description))
		}
		result.WriteString("\n")
	}

	result.WriteString("\n### 注意:\n可以用 aws cli ssm 连到节点执行任务, 但 Amazon Q 交不支持交互操作，所以不要使用 start-session\n")

	return mcp.NewToolResultText(result.String()), nil
}

// mergeConfigs 递归合并配置
func mergeConfigs(dst, src map[string]interface{}) {
	for k, v := range src {
		if srcMap, ok := v.(map[string]interface{}); ok {
			if dstMap, ok := dst[k].(map[string]interface{}); ok {
				mergeConfigs(dstMap, srcMap)
			} else {
				dst[k] = srcMap
			}
		} else if srcArray, ok := v.([]interface{}); ok && k == "instances" {
			// 特殊处理 instances 数组
			dstArray, ok := dst[k].([]interface{})
			if !ok {
				dst[k] = srcArray
				continue
			}
			
			// 创建一个映射，用于快速查找实例
			instanceMap := make(map[string]int)
			for i, item := range dstArray {
				if itemMap, isItemMap := item.(map[string]interface{}); isItemMap {
					if id, hasID := itemMap["id"].(string); hasID {
						instanceMap[id] = i
					}
				}
			}
			
			// 合并或添加源数组中的实例
			for _, srcItem := range srcArray {
				if srcItemMap, isSrcItemMap := srcItem.(map[string]interface{}); isSrcItemMap {
					if id, hasID := srcItemMap["id"].(string); hasID {
						if idx, exists := instanceMap[id]; exists {
							// 如果目标数组中已存在相同ID的实例，合并它们
							if dstItemMap, isDstItemMap := dstArray[idx].(map[string]interface{}); isDstItemMap {
								mergeConfigs(dstItemMap, srcItemMap)
							}
						} else {
							// 如果目标数组中不存在该ID的实例，添加它
							dstArray = append(dstArray, srcItem)
						}
					} else {
						// 如果源实例没有ID，直接添加它
						dstArray = append(dstArray, srcItem)
					}
				}
			}
			
			// 更新目标数组
			dst[k] = dstArray
		} else {
			dst[k] = v
		}
	}
}
