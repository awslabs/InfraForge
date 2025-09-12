<powershell>

######################################################################################################################
#  Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.                                                #
#                                                                                                                    #
#  Licensed under the Apache License, Version 2.0 (the "License"). You may not use this file except in compliance    #
#  with the License. A copy of the License is located at                                                             #
#                                                                                                                    #
#      http://www.apache.org/licenses/LICENSE-2.0                                                                    #
#                                                                                                                    #
#  or in the 'license' file accompanying this file. This file is distributed on an 'AS IS' BASIS, WITHOUT WARRANTIES #
#  OR CONDITIONS OF ANY KIND, express or implied. See the License for the specific language governing permissions    #
#  and limitations under the License.                                                                                #
######################################################################################################################

# DCV parameters: https://docs.aws.amazon.com/dcv/latest/adminguide/config-param-ref.html
# Stop DCV service
# Stop-Service -Name dcvserver

# LOG: Default User Data: Get-Content C:\ProgramData\Amazon\EC2-Windows\Launch\Log\UserdataExecutionInfraForge.log

function Write-ToLog {
    Param (
        [ValidateNotNullOrEmpty()]
        [Parameter(Mandatory=$true)]
        [String] $Message,
        [String] $LogFile = ('{0}\ProgramData\Amazon\EC2-Windows\Launch\Log\UserdataExecutionInfraForge.log' -f $env:SystemDrive),
        [ValidateSet('Error','Warn','Info')]
        [string] $Level = 'Info'
    )

    if (-not(Test-Path -Path $LogFile)) {
        $null = New-Item -Path $LogFile -ItemType File -Force
    }


    $FormattedDate = Get-Date -Format 'yyyy-MM-dd HH:mm:ss'
    switch ($Level) {
        'Error' {
            $LevelText = 'ERROR:'
        }
        'Warn' {
            $LevelText = 'WARNING:'
        }
        'Info' {
            $LevelText = 'INFO:'
        }
    }
    # If Level == Error send ses message ?
    "$FormattedDate $LevelText $Message" | Out-File -FilePath $LogFile -Append
}

$IMDSToken = Invoke-RestMethod -Headers @{"X-aws-ec2-metadata-token-ttl-seconds" = "21600"} -Method PUT -Uri http://169.254.169.254/latest/api/token
$Hostname = Invoke-RestMethod -Headers @{"X-aws-ec2-metadata-token" = $IMDSToken} -Method GET -Uri http://169.254.169.254/latest/meta-data/hostname
$InstanceId = Invoke-RestMethod -Headers @{"X-aws-ec2-metadata-token" = $IMDSToken} -Method GET -Uri http://169.254.169.254/latest/meta-data/instance-id
$DCVHostAltname = $Hostname.split(".")[0]

$SSMService = Get-Service -Name AmazonSSMAgent -ErrorAction SilentlyContinue

if ($null -eq $SSMService) {
    Write-ToLog -Message "Install Session Manager plugin"
    Invoke-WebRequest -uri https://s3.amazonaws.com/session-manager-downloads/plugin/latest/windows/SessionManagerPluginSetup.exe -OutFile C:\Windows\Temp\SessionManagerPluginSetup.exe
    Invoke-Command -ScriptBlock {Start-Process "C:\Windows\Temp\SessionManagerPluginSetup.exe" -ArgumentList "/quiet" -Wait}
}

Write-ToLog -Message "Configure Nice DCV"
$IMDSToken = Invoke-RestMethod -Headers @{"X-aws-ec2-metadata-token-ttl-seconds" = "21600"} -Method PUT -Uri http://169.254.169.254/latest/api/token
$InstanceType = Invoke-RestMethod -Headers @{'X-aws-ec2-metadata-token' = $IMDSToken} -Method GET -Uri http://169.254.169.254/latest/meta-data/instance-type
$OSVersion = ((Get-ItemProperty -Path "Microsoft.PowerShell.Core\Registry::\HKEY_LOCAL_MACHINE\SOFTWARE\Microsoft\Windows NT\CurrentVersion" -Name ProductName).ProductName) -replace  "[^0-9]" , ''

$DCVService = Get-Service -Name dcvserver -ErrorAction SilentlyContinue

if ($null -eq $DCVService) {
    if((("$OSVersion" -ne "2019") -and ("$OSVersion" -ne "2022") -and ("$OSVersion" -ne "10") -and ("$OSVersion" -ne "11")) -and (($InstanceType[0] -ne 'g') -or ($InstanceType[0] -ne 'p'))) {
        $VirtualDisplayDriverRequired = $true
    }

    if($VirtualDisplayDriverRequired){
        Start-Job -Name WebReq -ScriptBlock { Invoke-WebRequest -uri https://d1uj6qtbmh3dt5.cloudfront.net/nice-dcv-virtual-display-x64-Release.msi -OutFile C:\Windows\Temp\DCVDisplayDriver.msi ; Invoke-WebRequest -uri https://d1uj6qtbmh3dt5.cloudfront.net/nice-dcv-server-x64-Release.msi -OutFile C:\Windows\Temp\DCVServer.msi }
    } else {
        Start-Job -Name WebReq -ScriptBlock { Invoke-WebRequest -uri https://d1uj6qtbmh3dt5.cloudfront.net/nice-dcv-server-x64-Release.msi -OutFile C:\Windows\Temp\DCVServer.msi }
    }

    Wait-Job -Name WebReq
    if($VirtualDisplayDriverRequired){
        Invoke-Command -ScriptBlock {Start-Process "msiexec.exe" -ArgumentList "/I C:\Windows\Temp\DCVDisplayDriver.msi /quiet /norestart" -Wait}
    }

    Invoke-Command -ScriptBlock {Start-Process "msiexec.exe" -ArgumentList "/I C:\Windows\Temp\DCVServer.msi ADDLOCAL=ALL /quiet /norestart /l*v dcv_install_msi.log " -Wait}
}

while (-not(Get-Service dcvserver -ErrorAction SilentlyContinue)) { Start-Sleep -Milliseconds 250 }
Write-ToLog -Message "Edit dcv.conf"
$WindowsHostname = $env:COMPUTERNAME
New-Item -Path "Microsoft.PowerShell.Core\Registry::\HKEY_USERS\S-1-5-18\Software\GSettings\com\nicesoftware\dcv\" -Name connectivity -Force

$dcvPath = "Microsoft.PowerShell.Core\Registry::\HKEY_USERS\S-1-5-18\Software\GSettings\com\nicesoftware\dcv"
Set-ItemProperty -Path "$dcvPath\session-management" -Name create-session -Value 1 -force
New-ItemProperty -Path "$dcvPath\connectivity" -Name enable-quic-frontend -PropertyType DWORD -Value 1 -force
New-ItemProperty -Path "$dcvPath\security" -Name no-tls-strict -PropertyType DWORD -Value 1 -force
New-ItemProperty -Path "Microsoft.PowerShell.Core\Registry::\HKEY_USERS\S-1-5-18\Software\GSettings\com\nicesoftware\dcv\security" -Name "authentication" -PropertyType "String" -Value "none" -Force
Stop-Service dcvserver
Start-Sleep -Milliseconds 3000
Start-Service dcvserver

Write-ToLog -Message "OS auto-lock"
New-ItemProperty -Path "Microsoft.PowerShell.Core\Registry::\HKEY_USERS\S-1-5-18\Software\GSettings\com\nicesoftware\dcv\security" -Name "os-auto-lock" -PropertyType "DWord" -Value 0 -Force

Write-ToLog -Message "Disable sleep"
New-Item -Path "Microsoft.PowerShell.Core\Registry::\HKEY_USERS\S-1-5-18\Software\GSettings\com\nicesoftware\dcv\" -Name "windows" -Force
New-ItemProperty -Path "Microsoft.PowerShell.Core\Registry::\HKEY_USERS\S-1-5-18\Software\GSettings\com\nicesoftware\dcv\security" -Name "disable-display-sleep" -PropertyType "DWord" -Value 1 -Force

Write-ToLog -Message "Restart Computer to validate all Windows changes. Use -Force to force reboot even if users are logged in (in case of custom AMI)"
Restart-Computer -Force

</powershell>
