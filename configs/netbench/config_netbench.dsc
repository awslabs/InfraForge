  "description": "网络性能测试环境配置模板。支持的测试工具包括 redis、ethr、iperf2、iperf3、sockperf 和 netperf。使用方法：1) 通过修改 netbenchworker 实例的 userData    Token 中的 modules 参数可以选择特定的测试工具，多个工具用分号分隔；2) 可以调整实例类型以满足不同的测试需求；3) 部署后可通过 SSM 连接到实例执行测试命令。注意事项：确保 server 和 worker 实
    例在同一个 VPC 中以获得准确的测试结果。"
