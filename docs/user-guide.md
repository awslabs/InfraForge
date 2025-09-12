# InfraForge User Guide

## 🚀 Installation

### Prerequisites
- Go 1.23 or later
- AWS CLI configured with appropriate credentials
- AWS CDK CLI installed

### Installation Steps

1. **Extract and Build**
   ```bash
   tar xvf InfraForge.tar.gz
   cd InfraForge/cmd/infraforge
   go build
   chmod +x infraforge
   cp -a ../../configs .
   ```

2. **Optional: Build MCP Server for Amazon Q Integration**
   ```bash
   cd ../../tools/mcp/
   go build
   sudo cp infraforge_mcp_server /usr/local/bin/
   sudo chmod +x /usr/local/bin/infraforge_mcp_server
   ```

## 🛠️ Basic Usage

### 1. Initialize CDK Environment
```bash
# Copy desired configuration
cp configs/parallelcluster/config_parallelcluster.json config.json

# Bootstrap CDK (one-time setup per region)
cdk bootstrap --app ./infraforge --force --require-approval never
```

### 2. Deploy Infrastructure
```bash
# Deploy with current configuration
cdk deploy --app ./infraforge
```

### 3. Update Infrastructure
```bash
# Modify config.json as needed, then redeploy
cdk deploy --app ./infraforge
```

### 4. Destroy Infrastructure
```bash
cdk destroy --app ./infraforge
```

## 💬 Optional: Amazon Q Chat Integration

If you want to use InfraForge with Amazon Q Chat for conversational infrastructure management:

### 1. Initialize Environment
```bash
./bootstrap.sh
```

### 2. Start Amazon Q Chat with InfraForge Tools
```bash
q chat --trust-tools=fs_read,report_issue,infraforge___getDeploymentStatus,infraforge___getStackOutputs,infraforge___getOperationManual,infraforge___listTemplates
```

### 3. Available Tools
Once connected, you'll have access to these InfraForge tools:

| Tool | Permission | Description |
|------|------------|-------------|
| `infraforge___deployInfra` | not trusted | Deploy infrastructure from templates |
| `infraforge___getDeploymentStatus` | trusted | Check deployment status |
| `infraforge___getOperationManual` | trusted | Get operation instructions |
| `infraforge___getStackOutputs` | trusted | Retrieve stack outputs |
| `infraforge___listTemplates` | trusted | List available configuration templates |

### 4. Example Usage with Amazon Q

**Performance Testing Setup:**
```
Help me spin up a c6i.xlarge and c7g.xlarge instances to run performance tests, 
with test modules including c2clat, pts, where pts includes sub-modules stream, 
mbw, byte, and stress-ng. After the tests, save the results to s3://aws-infra-forge.
```

**Amazon Q will:**
1. List available templates using `listTemplates`
2. Get operation manual using `getOperationManual`
3. Read template configuration using `fs_read`
4. Deploy customized infrastructure using `deployInfra`

## 📋 Available Templates

InfraForge includes pre-configured templates for various use cases:

### Benchmarking
- `configs/bench/config_bench.json` - General benchmarking setup
- `configs/bench/config_sysbench.json` - System benchmarking with c2clat, pts modules

### High Performance Computing
- `configs/parallelcluster/config_parallelcluster.json` - AWS ParallelCluster setup

### Container Computing
- `configs/batch/config_batch.json` - AWS Batch container workloads

### Specialized Workloads
- `configs/kudu/config_kudu.json` - Apache Kudu deployment
- `configs/enclave/config_enclave.json` - AWS Nitro Enclaves
- `configs/web3/config_agave.json` - Web3 blockchain nodes

### Networking
- `configs/netbench/config_netbench.json` - Network performance testing
- `configs/netbench/config_locust_redis.json` - Load testing with Redis

## 🔧 Configuration Customization

### Basic Structure
```json
{
    "global": {
        "stackName": "my-infrastructure",
        "description": "Custom infrastructure deployment"
    },
    "enabledForges": ["ec2instance1", "efs1", "batch1"],
    "forges": {
        "ec2": {
            "instances": [
                {
                    "id": "ec2instance1",
                    "instanceType": "c6i.xlarge",
                    "userDataToken": "sysbench:modules=c2clat"
                }
            ]
        },
        "efs": {
            "instances": [
                {
                    "id": "efs1"
                }
            ]
        }
    }
}
```

### Key Configuration Points

- **enabledForges:**  List of specific instance IDs to deploy (e.g., `["ec2instance1", "efs1", "batch1"]`)
- **VPC:**  Always created automatically as the foundation layer - no need to specify in enabledForges
- **Instance IDs:**  Must match between `enabledForges` and the actual instance definitions in `forges`

### Common Parameters
- **instanceType:**  EC2 instance type (e.g., `c6i.xlarge`, `c7g.xlarge`)
- **userDataToken:**  Automated software installation and configuration
- **dependsOn:**  Resource dependencies (e.g., `"EFS:efs1,LUSTRE:lustre1"`)

## 📊 Monitoring and Outputs

### Check Deployment Status
```bash
# Manual check
aws cloudformation describe-stacks --stack-name aws-infra-forge

# Or via Amazon Q Chat (if using integration)
> Check deployment status
```

### Get Stack Outputs
```bash
# Manual check
aws cloudformation describe-stacks --stack-name aws-infra-forge --query 'Stacks[0].Outputs'

# Or via Amazon Q Chat (if using integration)
> Show me the stack outputs
```

## 🔍 Troubleshooting

### Common Issues

1. **CDK Bootstrap Required**
   ```bash
   cdk bootstrap --app ./infraforge --force --require-approval never
   ```

2. **Permission Denied**
   ```bash
   chmod +x infraforge
   ```

3. **AWS Credentials**
   ```bash
   aws configure
   # or
   export AWS_PROFILE=your-profile
   ```

### Getting Help
- Review available templates in `configs/` directory
- Check AWS CloudFormation console for deployment details
- Use Amazon Q Chat integration for conversational assistance (optional)

## 🎯 Best Practices

1. **Start with Templates:**  Use existing templates as starting points
2. **Test Small:**  Deploy simple configurations first
3. **Monitor Resources:**  Check AWS costs and resource usage
4. **Clean Up:**  Destroy resources when testing is complete
5. **Version Control:**  Keep configuration files in version control

## 📚 Next Steps

- Explore available templates in the `configs/` directory
- Customize configurations for your specific use cases
- Integrate with CI/CD pipelines for automated deployments
- Try the optional Amazon Q Chat integration for conversational infrastructure management
- Contribute new templates and forge types to the project
