# InfraForge - AWS Infrastructure as Configuration Framework

[English](README.md) | [中文](README.zh-CN.md)

InfraForge is an innovative Infrastructure as Configuration (IaC) framework that revolutionizes how organizations deploy and manage AWS resources. Built with AWS CDK and Go, this enterprise-grade solution transforms complex cloud architectures into simple JSON configurations through its modular "forge" component system.

## Project Overview

InfraForge allows you to define, deploy, and manage complex AWS infrastructure using a configuration-driven approach. The framework abstracts away the complexity of AWS CloudFormation and CDK, providing a higher-level interface for common infrastructure patterns.

The system is designed as a comprehensive platform that supports any AWS service and solution, allowing continuous optimization and enhancement through its modular architecture. InfraForge delivers significant business value by enabling rapid deployment, cost reduction, and operational simplification.

## Architecture

![InfraForge Architecture](docs/architecture.svg)

The InfraForge architecture consists of four key components:

- **🤖 Amazon Q CLI:** Natural language understanding and intent processing
- **🧠 MCP Server:** Self-learning solution discovery and intelligent guidance generation
- **⚙️ InfraForge Engine:** Modular deployment orchestration and dependency management
- **📋 Solution Templates:** Configuration-driven infrastructure patterns and best practices

### Key Features

- **Modular Architecture:** Infrastructure components are organized as "forges" that can be composed together
- **Configuration-Driven:** Infrastructure defined through JSON configuration files
- **Multi-Resource Support:** Supports various AWS services including DS, EC2, ECS, EKS, EFS, FSx Lustre, and more
- **Cross-Stack References:** Resources can reference and depend on each other
- **Flexible Deployment Options:** Deploy entire stacks or individual components
- **Amazon Q Integration:** Optional MCP server for conversational infrastructure management

## Project Structure

```
InfraForge/
├── cmd/                  # Command-line interface code
│   └── infraforge/       # Main CLI application and executable
├── configs/              # Solution-specific configuration files
│   ├── batch/            # AWS Batch solutions
│   ├── bench/            # Benchmarking solutions
│   ├── directoryservice/ # Directory Service solutions
│   ├── ec2/              # EC2 solutions
│   ├── ecs/              # ECS solutions
│   ├── eks/              # EKS solutions
│   ├── enclave/          # Enclave solutions
│   ├── hyperpod/         # SageMaker HyperPod solutions
│   ├── kafka/            # Kafka solutions
│   ├── kudu/             # Kudu solutions
│   ├── kubernetes/       # Kubernetes solutions
│   ├── netbench/         # Network benchmarking solutions
│   ├── parallelcluster/  # AWS ParallelCluster solutions
│   ├── rds/              # RDS solutions
│   ├── redroid/          # Redroid solutions
│   └── web3/             # Web3 solutions
├── core/                 # Core framework functionality
├── forges/               # Infrastructure component implementations
│   ├── aws/              # AWS-specific forge implementations
│   │   ├── batch/        # AWS Batch forges
│   │   ├── ds/           # Directory Service forges
│   │   ├── ec2/          # EC2 instance forges
│   │   ├── ecs/          # ECS cluster and service forges
│   │   ├── eks/          # EKS cluster forges
│   │   ├── hyperpod/     # SageMaker HyperPod forges
│   │   ├── iam/          # IAM role and policy forges
│   │   ├── lambda/       # Lambda function forges
│   │   ├── parallelcluster/ # AWS ParallelCluster forges
│   │   ├── rds/          # RDS database forges
│   │   ├── storage/      # Storage-related forges (EFS, FSx)
│   │   └── vpc/          # VPC and networking forges
│   ├── desktop/          # Desktop environment forges
│   ├── kubernetes/       # Kubernetes-related forges
│   └── monitoring/       # Monitoring and observability forges
├── registry/             # Forge registry and management
├── scripts/              # User data scripts and templates
├── tools/                # Utility tools and scripts
├── docs/                 # Documentation
├── examples/             # Example configurations and usage
└── tests/                # Test cases and test utilities
```

## Configuration

InfraForge uses JSON configuration files to define infrastructure. The main configuration file is `config.json`, which defines:

- Global settings like stack name
- Enabled forges to deploy
- Resource configurations for different forge types

Solution-specific configurations are stored in the `configs/` directory with naming convention `config_<solution>.json`. To use a specific solution configuration, copy it to `cmd/infraforge/config.json` before deployment.

Example configuration structure:

```json
{
    "global": {
        "stackName": "aws-infra-forge",
        "dualStack": true
    },
    "enabledForges": [
        "efs1",
        "ecs1",
        "ds1",
        "windows2022",
        "ubuntu2204",
        "al2023"
    ],
    "forges": {
        "vpc": { ... },
        "ds":  { ... },
        "efs": { ... },
        "lustre": { ... },
        "ecs": { ... },
        "eks": { ... },
        "ec2": { ... }
    }
}
```

## Supported Forge Types

InfraForge currently supports the following forge types:

- **VPC:** Network infrastructure with public, private, and isolated subnets
- **EC2:** Virtual machines with various OS options (Amazon Linux, Ubuntu, Windows, CentOS)
- **ECS:** Container orchestration with Fargate and EC2 launch types
- **EKS:** Managed Kubernetes clusters with Karpenter support
- **AWS Batch:** Managed batch computing service
- **AWS ParallelCluster:** HPC cluster management
- **SageMaker HyperPod:** Distributed machine learning training
- **RDS:** Managed relational database service
- **EFS:** Elastic File System for shared storage
- **FSx Lustre:** High-performance file systems for compute workloads
- **Lambda:** Serverless functions
- **IAM:** Identity and access management resources
- **Directory Service:** Managed Microsoft Active Directory

## Getting Started

### Prerequisites

- Go 1.23 or later
- AWS CDK CLI
- AWS CLI configured with appropriate credentials
- Node.js (required by CDK)

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/awslabs/InfraForge.git
   cd InfraForge
   ```

2. Install dependencies:
   ```
   go mod download
   ```

3. Build the application:
   ```
   cd cmd/infraforge
   go build
   ```

### Usage

1. Choose a solution configuration from the `configs/` directory:
   ```
   # For example, to use the ParallelCluster solution:
   cp configs/parallelcluster/config_parallelcluster.json cmd/infraforge/config.json
   
   # Or to use the benchmarking solution:
   cp configs/bench/config_sysbench.json cmd/infraforge/config.json
   ```

2. Run the bootstrap script:
   ```
   ./bootstrap.sh
   ```

3. Deploy your infrastructure:
   ```
   cd cmd/infraforge
   ./deploy.sh
   ```

4. To destroy the infrastructure:
   ```
   ./destroy.sh
   ```

## Amazon Q Integration (Optional)

For conversational infrastructure management with Amazon Q:

1. Build the MCP server:
   ```bash
   cd tools/mcp/
   go build
   sudo cp infraforge_mcp_server /usr/local/bin/
   sudo chmod +x /usr/local/bin/infraforge_mcp_server
   ```

2. Prepare the working directory:
   ```bash
   cd cmd/infraforge
   cp -r ../../configs .
   ```

3. Start Amazon Q Chat with InfraForge tools:
   ```bash
   q chat --trust-tools=fs_read,report_issue,infraforge___getDeploymentStatus,infraforge___getStackOutputs,infraforge___getOperationManual,infraforge___listTemplates
   ```

4. Use conversational commands:
   ```
   > List available templates
   > Deploy a ParallelCluster cluster
   > Check deployment status
   ```

For detailed usage, see [User Guide](docs/user-guide.md).

## Managing Solution Configurations

InfraForge supports multiple solution configurations organized by category in the `configs/` directory:

- `batch/`: AWS Batch solutions
- `bench/`: Benchmarking solutions
- `directoryservice/`: Directory Service solutions
- `ec2/`: EC2 solutions
- `ecs/`: ECS solutions
- `eks/`: EKS solutions
- `enclave/`: Enclave solutions
- `hyperpod/`: SageMaker HyperPod solutions
- `kafka/`: Kafka solutions
- `kudu/`: Kudu solutions
- `kubernetes/`: Kubernetes solutions
- `netbench/`: Network benchmarking solutions
- `parallelcluster/`: AWS ParallelCluster solutions
- `rds/`: RDS solutions
- `redroid/`: Redroid solutions
- `web3/`: Web3 solutions

To create a new solution:
1. Identify the appropriate category directory in `configs/` or create a new one
2. Create a new configuration file named `config_<solution>.json`
3. Copy and modify an existing configuration or start from scratch
4. To deploy, copy your solution config to `cmd/infraforge/config.json`

## Useful Commands

- `cdk deploy`: Deploy the stack to your default AWS account/region
- `cdk diff`: Compare deployed stack with current state
- `cdk synth`: Emit the synthesized CloudFormation template
- `go test`: Run unit tests
- `cdk --app ./infraforge deploy`: Deploy using InfraForge CDK application

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the [Apache License 2.0](LICENSE) - see the LICENSE file for details.
