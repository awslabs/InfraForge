# Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Install AWS CLI silently
Invoke-Command -ScriptBlock {Start-Process "msiexec.exe" "/I https://awscli.amazonaws.com/AWSCLIV2.msi /quiet /norestart" -Wait}

# Download NVM setup executable
Invoke-WebRequest -Uri 'https://github.com/coreybutler/nvm-windows/releases/latest/download/nvm-setup.zip' -OutFile 'C:\Windows\Temp\nvm-setup.zip'
# Expand the downloaded zip file
Expand-Archive -Path 'C:\Windows\Temp\nvm-setup.zip' -DestinationPath 'C:\Windows\Temp'
# Run the NVM setup executable
Start-Process -FilePath 'C:\Windows\Temp\nvm-setup.exe' -ArgumentList '/DIR=c:\NVM /verysilent /norestart' -Wait
# Set NVM environment variables
[Environment]::SetEnvironmentVariable("NVM_HOME", "c:\NVM", "Machine")
[Environment]::SetEnvironmentVariable("NVM_SYMLINK", "c:\nvm\nodejs", "Machine")
$newPath = [Environment]::GetEnvironmentVariable("Path", "Machine") + ";$env:NVM_HOME;$env:NVM_SYMLINK"
[Environment]::SetEnvironmentVariable("Path", $newPath, "Machine")
$env:NVM_HOME = "c:\NVM"
$env:NVM_SYMLINK = "c:\nvm\nodejs"
$env:PATH += ";$env:NVM_HOME;$env:NVM_SYMLINK"
# Install and use the latest LTS version of Node.js
nvm install lts
nvm use lts

# Get the IMDSv2 token and current AWS Region
$IMDSToken = Invoke-RestMethod -Headers @{"X-aws-ec2-metadata-token-ttl-seconds" = "21600"} -Method PUT -Uri http://169.254.169.254/latest/api/token
$currentREGION = Invoke-RestMethod -Headers @{"X-aws-ec2-metadata-token" = $IMDSToken} -Method GET -Uri http://169.254.169.254/latest/meta-data/placement/region

# Check if the current Region is in China and configure npm registry accordingly
if ($currentRegion.StartsWith("cn-")) {
    # For China Regions, configure npm to use the mirror registry
    npm config set registry https://registry.npmmirror.com
}

# Install aws-cdk globally
npm install -g aws-cdk


