#!/bin/bash
# Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

#####################################################################
# Enhanced userdata script for InfraForge
# 
# This script serves as a generic userdata launcher that downloads and
# executes specific userdata modules based on parameters.
# It supports all major Linux distributions and provides robust error
# handling and logging.
#####################################################################

set -o pipefail

# Configuration variables (will be replaced by template engine)
export S3_LOCATION='{{s3Location}}'
export USER_DATA_LOCATION="https://aws-hpc-builder.s3.amazonaws.com/project/apps/aws-auto-launch/userdata"
export CUSTOM_USER_DATA_LOCATION='{{customUserDataLocation}}'

# Use custom location if specified (and placeholder was replaced)
if [ "${CUSTOM_USER_DATA_LOCATION}" != "{{customUserDataLocation}}" ]; then
    export USER_DATA_LOCATION="${CUSTOM_USER_DATA_LOCATION}"
fi

# export USER_DATA_TOKEN='{{userDataToken}}'
export USER_DATA_MODULES='{{userDataToken}}'
export MAGIC_TOKEN='{{magicToken}}'
export AWS_DEFAULT_OUTPUT=json

# Log file setup
LOGFILE="/var/log/userdata-execution.log"
LOGLEVEL="INFO"  # Possible values: DEBUG, INFO, WARN, ERROR

# Create log directory if it doesn't exist
mkdir -p "$(dirname "$LOGFILE")" 2>/dev/null

#####################################################################
# Logging functions
#####################################################################

log() {
    local level="$1"
    local message="$2"
    local timestamp=$(date +"%Y-%m-%d %H:%M:%S")
    
    # Log levels: DEBUG=0, INFO=1, WARN=2, ERROR=3
    local log_priority=1
    case "$LOGLEVEL" in
        DEBUG) log_priority=0 ;;
        INFO)  log_priority=1 ;;
        WARN)  log_priority=2 ;;
        ERROR) log_priority=3 ;;
    esac
    
    local msg_priority=1
    case "$level" in
        DEBUG) msg_priority=0 ;;
        INFO)  msg_priority=1 ;;
        WARN)  msg_priority=2 ;;
        ERROR) msg_priority=3 ;;
    esac
    
    # Only log if message priority is >= log level priority
    if [ $msg_priority -ge $log_priority ]; then
        echo "[$timestamp] [$level] $message" | tee -a "$LOGFILE"
    fi
}

log_debug() { log "DEBUG" "$1"; }
log_info() { log "INFO" "$1"; }
log_warn() { log "WARN" "$1"; }
log_error() { log "ERROR" "$1"; }

#####################################################################
# Metadata retrieval functions
#####################################################################

get_instance_metadata() {
    local metadata_path="$1"
    local token=""
    local max_attempts=5
    local attempt=1
    
    while [ $attempt -le $max_attempts ]; do
        token=$(curl -s -f -X PUT "http://169.254.169.254/latest/api/token" \
                -H "X-aws-ec2-metadata-token-ttl-seconds: 21600" 2>/dev/null)
        
        if [ -n "$token" ]; then
            local result=$(curl -s -f -H "X-aws-ec2-metadata-token: ${token}" \
                          "http://169.254.169.254/latest/meta-data/${metadata_path}" 2>/dev/null)
            if [ -n "$result" ]; then
                echo "$result"
                return 0
            fi
        fi
        
        log_warn "Failed to retrieve metadata (attempt $attempt/$max_attempts). Retrying..."
        sleep $((attempt * 2))
        attempt=$((attempt + 1))
    done
    
    log_error "Failed to retrieve metadata after $max_attempts attempts"
    return 1
}

#####################################################################
# OS detection and package management
#####################################################################

detect_os() {
    log_info "Detecting operating system..."
    
    if [ ! -f /etc/os-release ]; then
        log_error "Cannot detect OS: /etc/os-release not found"
        return 1
    fi
    
    # Source the OS release information
    . /etc/os-release
    
    # Store original version ID
    ORIGINAL_VERSION_ID="${VERSION_ID}"
    # Extract major version number
    VERSION_ID=$(echo "${VERSION_ID}" | cut -f1 -d.)
    
    log_info "Detected OS: ${NAME} ${ORIGINAL_VERSION_ID}"
    
    # Determine package manager type and standardized version
    case "${NAME}" in
        "Amazon Linux"|"Rocky Linux"|"Oracle Linux Server"|"Red Hat Enterprise Linux Server"|"Red Hat Enterprise Linux"|"CentOS Linux"|"CentOS Stream"|"Alibaba Cloud Linux"|"Alibaba Cloud Linux (Aliyun Linux)")
            export PACKAGE_TYPE="rpm"
            case "${VERSION_ID}" in
                2|7)
                    export STD_VERSION_ID=7
                    export PKG_INSTALL="yum -y install"
                    export PKG_UPDATE="yum -y update"
                    ;;
                3|8)
                    export STD_VERSION_ID=8
                    export PKG_INSTALL="dnf -y install --allowerasing"
                    export PKG_UPDATE="dnf -y update"
                    ;;
                9|10|2022|2023)
                    export STD_VERSION_ID=9
                    export PKG_INSTALL="dnf -y install --allowerasing"
                    export PKG_UPDATE="dnf -y update"
                    ;;
                *)
                    log_error "Unsupported Linux system: ${NAME} ${VERSION_ID}"
                    return 1
                    ;;
            esac
            ;;
        "Ubuntu"|"Debian GNU/Linux")
            export PACKAGE_TYPE="deb"
            export PKG_INSTALL="apt-get -y install"
            export PKG_UPDATE="apt-get -y update"
            case "${VERSION_ID}" in
                10|18)
                    export STD_VERSION_ID=18
                    ;;
                11|12|20|22|24)
                    export STD_VERSION_ID=20
                    ;;
                *)
                    log_error "Unsupported Linux system: ${NAME} ${VERSION_ID}"
                    return 1
                    ;;
            esac
            ;;
        *)
            log_error "Unsupported Linux system: ${NAME} ${VERSION_ID}"
            return 1
            ;;
    esac
    
    log_info "OS detection complete: ${NAME} ${ORIGINAL_VERSION_ID} (Standard version: ${STD_VERSION_ID}, Package type: ${PACKAGE_TYPE})"
    return 0
}

install_dependencies() {
    log_info "Installing system dependencies..."
    
    # Update package lists
    #log_debug "Updating package lists"
    #sudo $PKG_UPDATE
    
    # Install required packages
    log_debug "Installing required packages"
    sudo $PKG_INSTALL unzip jq curl wget
    
    log_info "System dependencies installed successfully"
}

#####################################################################
# AWS CLI installation
#####################################################################

install_awscli() {
    if command -v aws >/dev/null 2>&1; then
        log_info "AWS CLI already installed"
        return 0
    fi
    
    log_info "Installing AWS CLI..."
    
    local tmpdir="${WORK_DIR}/awscli"
    mkdir -p "${tmpdir}"
    cd "${tmpdir}"
    
    # Download and install AWS CLI
    log_debug "Downloading AWS CLI installer"
    if ! curl -s -f "https://awscli.amazonaws.com/awscli-exe-linux-$(arch).zip" -o "awscliv2.zip"; then
        log_error "Failed to download AWS CLI"
        return 1
    fi
    
    log_debug "Extracting AWS CLI installer"
    if ! unzip -q awscliv2.zip; then
        log_error "Failed to extract AWS CLI"
        return 1
    fi
    
    log_debug "Installing AWS CLI"
    if ! sudo ./aws/install; then
        log_error "Failed to install AWS CLI"
        return 1
    fi
    
    cd - >/dev/null
    log_info "AWS CLI installed successfully"
    return 0
}

#####################################################################
# Userdata module management
#####################################################################

download_and_prepare_modules() {
    log_info "Downloading and preparing userdata modules..."

    cd "${WORK_DIR}"
    local module_count=0

    # Split different tasks/modules
    read -ra ENTRIES <<< "${USER_DATA_MODULES}"

    for entry in "${ENTRIES[@]}"; do
        # Extract module name and parameters
        local module params
        if [[ "$entry" == *":"* ]]; then
            # Module with parameters
            module=${entry%%:*}
            params=${entry#*:}
            log_debug "Found module with params: ${module}, params: ${params}"
        else
            # Module without parameters
            module=$entry
            params=""
            log_debug "Found module without params: ${module}"
        fi

        # Download module template
        log_debug "Downloading template for module: ${module}"
        if ! curl --retry 5 --retry-delay 2 -s -f -JLOk "${USER_DATA_LOCATION}/${module}_template.sh"; then
            log_error "Failed to download template for module: ${module}"
            continue
        fi

        module_count=$((module_count + 1))
        local output_file="$(printf "%.3d" ${module_count})-${module}.sh"

        # Replace basic placeholders in template
	# Magic token is JSON format, does not contain #, use # separator for magic token processing
        log_debug "Configuring module: ${module}"
        sed -e "s|XXX_AWS_DEFAULT_REGION_XXX|${AWS_DEFAULT_REGION}|g" \
            -e "s|XXX_AWS_PEER_SERVER_XXX|${AWS_PEER_SERVER_MAGIC}|g" \
            -e "s#XXX_MAGIC_TOKEN_XXX#${MAGIC_TOKEN}#g" \
            -e "s|XXX_MODULE_PARAMS_XXX|${params}|g" \
            -e "s|XXX_PKG_SRC_URL_XXX|${URL_MAGIC}|g" \
            -e "s|XXX_S3_LOCATION_XXX|${S3_LOCATION}/${module}|g" \
            "${module}_template.sh" > "${output_file}"

        # Make script executable
        chmod +x "${output_file}"

        # Clean up template file
        rm -f "${module}_template.sh"

        log_info "Module prepared: ${module}"
    done

    if [ ${module_count} -eq 0 ]; then
        log_warning "No modules were prepared"
    else
        log_info "Total modules prepared: ${module_count}"
    fi
}

execute_modules() {
    log_info "Executing userdata modules..."
    
    cd "${WORK_DIR}"
    local executed=0
    local failed=0
    
    # Execute each module in order (sorted by filename)
    for module_script in $(ls -1 [0-9]*.sh 2>/dev/null); do
        log_info "Executing module: ${module_script}"
        
        # Check if this is a non-root module
        if echo "${module_script}" | grep -q "\-nonroot"; then
            log_debug "Module requires non-root execution"
            
            # Find the default user (UID 1000)
            local default_user=$(id -nu 1000 2>/dev/null)
            local default_group=$(id -ng 1000 2>/dev/null)
            
            if [ -z "${default_user}" ]; then
                log_error "Cannot execute non-root module: No user with UID 1000 found"
                failed=$((failed + 1))
                continue
            fi
            
            # Copy the script to the user's home directory
            local user_home="/home/${default_user}"
            cp "${module_script}" "${user_home}/"
            chown "${default_user}:${default_group}" "${user_home}/${module_script}"
            
            # Execute as the non-root user
            log_debug "Executing as user: ${default_user}"
            if sudo -u "${default_user}" bash "${user_home}/${module_script}"; then
                log_info "Module executed successfully: ${module_script}"
                executed=$((executed + 1))
            else
                log_error "Module execution failed: ${module_script}"
                failed=$((failed + 1))
            fi
            
            # Clean up
            rm -f "${user_home}/${module_script}"
        else
            # Execute as current user (typically root in userdata)
            if bash "${module_script}"; then
                log_info "Module executed successfully: ${module_script}"
                executed=$((executed + 1))
            else
                log_error "Module execution failed: ${module_script}"
                failed=$((failed + 1))
            fi
        fi
    done
    
    log_info "Module execution complete: ${executed} succeeded, ${failed} failed"
    
    if [ ${failed} -gt 0 ]; then
        return 1
    fi
    
    return 0
}

#####################################################################
# Main execution
#####################################################################

main() {
    log_info "Starting userdata execution"
    
    # Create working directory
    export WORK_DIR=$(mktemp -d /tmp/userdata.XXXXXX)
    log_debug "Working directory: ${WORK_DIR}"
    
    # Get AWS region from instance metadata
    export AWS_DEFAULT_REGION=$(get_instance_metadata "placement/region")
    if [ -z "${AWS_DEFAULT_REGION}" ]; then
        log_error "Failed to determine AWS region"
        exit 1
    fi
    log_info "AWS Region: ${AWS_DEFAULT_REGION}"
    
    # Detect OS and set up package management
    if ! detect_os; then
        log_error "OS detection failed"
        exit 1
    fi
    
    # Install system dependencies
    if ! install_dependencies; then
        log_error "Failed to install system dependencies"
        exit 1
    fi
    
    # Install AWS CLI if needed
    if ! install_awscli; then
        log_warn "AWS CLI installation failed, but continuing execution"
    fi
    
    # Download and prepare userdata modules
    if ! download_and_prepare_modules; then
        log_error "Failed to prepare userdata modules"
        exit 1
    fi
    
    # Execute the modules
    if ! execute_modules; then
        log_warn "Some modules failed to execute"
        # Continue execution even if some modules failed
    fi
    
    # Clean up
    cd /
    rm -rf "${WORK_DIR}"
    log_debug "Cleaned up working directory"
    
    log_info "Userdata execution completed"
    
    # ECS may add commands after this point
    # exit 0
}

# Start execution
main
