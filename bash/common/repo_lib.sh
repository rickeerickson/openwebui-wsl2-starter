#!/bin/bash
set -euo pipefail
trap 'echo "Error occurred in script: ${BASH_SOURCE[0]}, line: ${LINENO}, command: ${BASH_COMMAND}" >&2' ERR

source_required_file() {
    local filepath="$1"

    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo "$(date '+%Y.%m.%d:%H:%M:%S') - DEBUG: ${log_file:-${0##*/}}: ${0}: ${BASH_SOURCE[1]}::${FUNCNAME[1]}::${BASH_LINENO[1]} - ${BASH_SOURCE[0]}::${FUNCNAME[0]}::${BASH_LINENO[0]} -> sourcing ${filepath}" >&2
    fi

    if [[ -f "$filepath" ]]; then
        source "$filepath"
    else
        echo "$(date '+%Y.%m.%d:%H:%M:%S') - ERROR: ${log_file:-${0##*/}}: ${0}: ${BASH_SOURCE[1]}::${FUNCNAME[1]}::${BASH_LINENO[1]} - ${BASH_SOURCE[0]}::${FUNCNAME[0]}::${BASH_LINENO[0]} -> required file ${filepath} not found." >&2
        exit 1
    fi
}

get_git_root() {
    local repo_root
    if ! repo_root=$(git rev-parse --is-inside-work-tree &>/dev/null); then
        echo "Error: Not inside a Git repository." >&2
        exit 1
    fi
    git rev-parse --show-toplevel
}

log_message() {
    local message="${1:-}"
    local level="${2:-${LEVEL_DEFAULT}}"
    local verbosity="${VERBOSITY:-${VERBOSITY_DEFAULT}}"

    if (( level <= verbosity )); then
        local level_prefix=""
        case $level in
            "${LEVEL_ERROR}") level_prefix="${LEVEL_ERROR_PREFIX}" ;;
            "${LEVEL_WARNING}") level_prefix="${LEVEL_WARNING_PREFIX}" ;;
            "${LEVEL_INFO}") level_prefix="${LEVEL_INFO_PREFIX}" ;;
            "${LEVEL_DEBUG_1}") level_prefix="${LEVEL_DEBUG_1_PREFIX}" ;;
            "${LEVEL_DEBUG_2}") level_prefix="${LEVEL_DEBUG_2_PREFIX}" ;;
            *) level_prefix="LOG:" ;;
        esac

        local prefix="$(date '+%Y.%m.%d:%H:%M:%S') - ${level_prefix} "

        if [[ "${DEBUG:-false}" == "true" ]]; then
            prefix+="${log_file}: ${0}: ${BASH_SOURCE[1]}::${FUNCNAME[1]}::${BASH_LINENO[1]} - ${BASH_SOURCE[0]}::${FUNCNAME[0]}::${BASH_LINENO[0]} -> "
        fi

        echo "${prefix}${message}" | tee -a "${log_file}" >&2
    fi
}

get_shell_options() {
    local get_opts=$(set +o)
    echo "${get_opts}"
}

set_shell_options() {
    log_message "Setting shell options" "${LEVEL_DEBUG_1}"
    local set_opts="${1}"

    log_message "Setting shell options to ${set_opts}" "${LEVEL_DEBUG_2}"
    eval "${set_opts}"
}

disable_exit_on_failure_and_pipefail() {
    log_message "Setting shell options to disable exit on failure and pipefail" "${LEVEL_DEBUG_1}"
    set +e +o pipefail
}

pad_right() {
  local str="$1"
  local target_len="$2"
  local pad_char="${3:- }"  # Default to a space if no character is provided
  local str_len=${#str}
  local padding=$((target_len - str_len))

  printf '%s%*s' "$str" "$padding" "" | sed "s/ /${pad_char}/g"
}

run_command() {
    local command="$1"
    local ignore_exit_status="${2:-false}"
    local should_fail="${3:-false}"
    local saved_opts="${4:-}"
    local debug_mode="${DEBUG:-false}"
    local prefix=" "

    if [[ "${debug_mode}" == "true" ]]; then
        prefix="${command} :: "
    fi

    log_message "${prefix}Executing command: ${command}" "${LEVEL_INFO}"
    log_message "${prefix}Executing command: ${command} with ignore_exit_status=\"${ignore_exit_status}\", should_fail=\"${should_fail}\", debug_mode=\"${debug_mode}\"" "${LEVEL_DEBUG_1}"

    disable_exit_on_failure_and_pipefail

    local output
    output=$(bash -c "${command}" 2>&1)
    local command_exit_status=$?

    if [[ -n "${saved_opts}" ]]; then
        set_shell_options "${saved_opts}"
    fi

    echo "${output}" | while IFS= read -r line; do
        log_message "${prefix}${line}"
    done

    if [[ "${ignore_exit_status}" == "true" ]]; then
        log_message "${prefix}${command} exitted with code ${command_exit_status}, ignoring exit status." "${LEVEL_WARNING}"
        return 0
    fi

    if [[ "${command_exit_status}" -ne 0 ]]; then
        if [[ "${should_fail}" == "true" ]]; then
            log_message "${prefix}${command} failed as expected with exit code ${command_exit_status}." "${LEVEL_INFO}"
            return "${command_exit_status}"
        fi

        log_message "${prefix}${command} failed unexpectedly with exit code ${command_exit_status}." "${LEVEL_ERROR}"
        return "${command_exit_status}"
    fi

    if [[ "${should_fail}" == "true" ]]; then
        log_message "${prefix}${command} succeeded unexpectedly when failure was expected." "${LEVEL_ERROR}"
        return 1
    fi

    return 0
}

run_command_with_retry() {
    local command="$1"
    local should_fail="${2:-false}"
    local ignore_exit_status="${3:-false}"
    local retry_count=0
    local max_retries=5
    local fib1=10
    local fib2=10

    log_message "Running command with retries: ${command}" "${LEVEL_DEBUG_1}"

    while true; do
        log_message "Executing: ${command} in $(pwd)" "${LEVEL_INFO}"
        local saved_opts
        saved_opts=$(get_shell_options)

        local command_exit_status
        run_command "${command}" "${ignore_exit_status}" "${should_fail}" "${saved_opts}"
        command_exit_status=$?

        if [[ "${command_exit_status}" -eq 0 ]]; then
            return 0
        fi

        read retry_count fib1 fib2 <<<"$(retry_logic "$retry_count" "$max_retries" "$fib1" "$fib2" "$command")" || return "${command_exit_status}"
    done
}

update_system_packages() {
    log_message "Updating system packages..." "${LEVEL_INFO}"
    run_command_with_retry "sudo apt-get update"
    run_command_with_retry "sudo apt-get upgrade -y"
    run_command_with_retry "sudo apt-get dist-upgrade -y"
    run_command_with_retry "sudo apt-get autoremove -y"
    run_command_with_retry "sudo apt-get autoclean"
    log_message "System packages updated successfully." "${LEVEL_INFO}"
}

setup_docker_keyring() {
    log_message "Setting up Docker GPG keyring and repository..." "${LEVEL_INFO}"
    run_command_with_retry "sudo apt-get install ca-certificates curl"
    run_command_with_retry "sudo install -m 0755 -d /etc/apt/keyrings"
    run_command_with_retry "sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc"
    run_command_with_retry "sudo chmod a+r /etc/apt/keyrings/docker.asc"

    echo \
      "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
      $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
      sudo tee /etc/apt/sources.list.d/docker.list > /dev/null

    log_message "Docker GPG keyring and repository setup completed successfully." "${LEVEL_INFO}"
}

install_and_configure_docker() {
    log_message "Installing and configuring Docker..." "${LEVEL_INFO}"

    run_command_with_retry "sudo apt-get update"
    run_command_with_retry "sudo apt-get install docker-ce docker-ce-cli containerd.io docker-compose-plugin"
    run_command_with_retry "sudo nvidia-ctk runtime configure --runtime=docker"

    if ! getent group docker > /dev/null 2>&1; then
        run_command_with_retry "sudo groupadd docker"
    else
        log_message "Group 'docker' already exists. Skipping creation." "${LEVEL_INFO}"
    fi

    run_command_with_retry "sudo usermod -aG docker $USER"
    run_command_with_retry "sudo systemctl restart docker"
    run_command_with_retry "sudo docker run hello-world"

    log_message "Docker installation and configuration completed successfully." "${LEVEL_INFO}"
}

check_and_prompt_docker() {
    log_message "Checking Docker status..." "${LEVEL_INFO}"

    if ! systemctl is-active --quiet docker; then
        log_message "Docker is not running. Please start Docker." "${LEVEL_ERROR}"
        exit 1
    fi
}

ensure_port_available() {
    local port="$1"
    local pid

    log_message "Checking if port ${port} is available..." "${LEVEL_INFO}"

    if pid=$(sudo lsof -ti:"${port}"); then
        log_message "Port ${port} is already in use by PID ${pid}. Stopping process..." "${LEVEL_WARNING}"
        run_command_with_retry "sudo kill -9 ${pid}"
        log_message "Freed up port ${port}." "${LEVEL_INFO}"
    else
        log_message "Port ${port} is available." "${LEVEL_INFO}"
    fi
}

verify_ollama_setup() {
    local host="${1:-localhost}"
    local port="${2:-11434}"
    local url="http://${host}:${port}"

    log_message "Verifying Ollama setup at ${url}..." "${LEVEL_INFO}"

    run_command_with_retry "curl -s -o /dev/null --write-out \"%{response_code}\n\" ${url}"
    run_command_with_retry "ollama list"
    run_command_with_retry "ollama ps"
    run_command_with_retry "ss -tuln | grep ${port}"
    run_command_with_retry "docker logs ollama"

    log_message "Ollama setup verification completed successfully." "${LEVEL_INFO}"
}

verify_docker_environment() {
    log_message "Verifying Docker environment..." "${LEVEL_INFO}"
    run_command_with_retry "docker --version"
    run_command_with_retry "docker context ls"
    run_command_with_retry "sudo lsof -i -P -n | grep LISTEN"
}

verify_nvidia_environment() {
    log_message "Verifying NVIDIA environment..." "${LEVEL_INFO}"
    run_command_with_retry "nvidia-smi"
}

container_exists() {
    local container_name="$1"
    docker ps -a --filter "name=^${container_name}$" --format "{{.Names}}" | grep -q "^${container_name}$"
}

container_is_running() {
    local container_name="$1"
    docker ps --filter "name=^${container_name}$" --format "{{.Names}}" | grep -q "^${container_name}$"
}

wait_for_container_status_up() {
    local container_name="$1"
    local retry_count=0
    local max_retries=5
    local fib1=10
    local fib2=10

    log_message "Waiting for container '${container_name}' to start..." "${LEVEL_INFO}"

    while true; do
        local status

        local should_fail=false
        local ignore_exit_status=true
        run_command "docker ps --filter \"name=${container_name}\" --format \"{{.Status}}\" | head -n 1" "${ignore_exit_status}" "${should_fail}"
        status=$(docker ps --filter "name=${container_name}" --format "{{.Status}}" | head -n 1)

        if [[ $status == Up* ]]; then
            log_message "Container '${container_name}' is running with status '${status}'." "${LEVEL_INFO}"
            return 0
        fi

        if (( retry_count >= max_retries )); then
            log_message "Failed to confirm container '${container_name}' is running with status 'Up' after ${max_retries} retries. Current status: '${status}'." "${LEVEL_ERROR}"
            return 1
        fi

        log_message "Waiting for container '${container_name}' to start. Current status: '${status}'. Retry $((retry_count + 1))/${max_retries}" "${LEVEL_WARNING}"
        sleep "${fib1}"
        local new_delay=$((fib1 + fib2))
        fib1=$fib2
        fib2=$new_delay
        ((retry_count++))
    done
}

try_stop_container() {
    local container_name="$1"

    log_message "Stopping container '${container_name}'..." "${LEVEL_INFO}"
    run_command_with_retry "docker stop ${container_name}"
}

container_is_stopped() {
    local container_name="$1"
    local running_check
    local exited_check

    log_message "Checking if container '${container_name}' is stopped and exited..." "${LEVEL_INFO}"

    local should_fail=false
    local ignore_exit_status=true

    run_command "docker ps --filter \"name=${container_name}\" --format \"{{.Names}}\" | grep -q \"^${container_name}$\"" "${ignore_exit_status}" "${should_fail}"
    running_check=$(docker ps --filter "name=${container_name}" --format "{{.Names}}" | grep -q "^${container_name}$")

    run_command "docker ps -a --filter \"name=${container_name}\" --filter \"status=exited\" --format \"{{.Names}}\" | grep -q \"^${container_name}$\"" "${ignore_exit_status}" "${should_fail}"
    exited_check=$(docker ps -a --filter "name=${container_name}" --filter "status=exited" --format "{{.Names}}" | grep -q "^${container_name}$")

    if $running_check && $exited_check; then
        return 0
    else
        return 1
    fi
}

wait_for_container_stop() {
    local container_name="$1"
    local retry_count=0
    local max_retries=5
    local fib1=10
    local fib2=10

    log_message "Waiting for container '${container_name}' to stop and exit..." "${LEVEL_INFO}"

    while true; do
        if container_is_stopped "${container_name}"; then
            log_message "Container '${container_name}' has stopped successfully and is in the exited state." "${LEVEL_INFO}"
            return 0
        fi

        if (( retry_count >= max_retries )); then
            log_message "Failed to confirm container '${container_name}' is stopped and exited after ${max_retries} retries. Giving up." "${LEVEL_ERROR}"
            return 1
        fi

        log_message "Waiting for container '${container_name}' to stop and exit... Retry ${retry_count}/${max_retries}" "${LEVEL_WARNING}"
        sleep "${fib1}"
        local new_delay=$((fib1 + fib2))
        fib1=$fib2
        fib2=$new_delay
        ((retry_count++))
    done
}

list_running_containers()
{
    log_message "Listing running containers..." "${LEVEL_INFO}"
    run_command_with_retry "docker ps"
}

stop_container() {
    local container_name="$1"

    log_message "Stopping container '${container_name}'..." "${LEVEL_INFO}"

    list_running_containers

    if ! container_exists "${container_name}"; then
        log_message "Container '${container_name}' does not exist. Skipping stop step." "${LEVEL_WARNING}"
        return 0
    fi

    if container_is_running "${container_name}"; then
        log_message "Stopping container '${container_name}'..." "${LEVEL_INFO}"
        if try_stop_container "${container_name}"; then
            wait_for_container_stop "${container_name}"
        else
            log_message "Failed to issue stop command for container '${container_name}'." "${LEVEL_ERROR}"
            return 1
        fi
    else
        log_message "Container '${container_name}' is not running. Skipping stop step." "${LEVEL_WARNING}"
        return 0
    fi
}

remove_container() {
    local container_name="$1"
    local retry_count=0
    local max_retries=5
    local fib1=10
    local fib2=10

    log_message "Removing container '${container_name}'..." "${LEVEL_INFO}"

    stop_container "${container_name}" || return  # Return if the container stop fails

    if ! docker ps -a | grep -q "${container_name}"; then
        log_message "Container '${container_name}' not found. Skipping remove step." "${LEVEL_WARNING}"
        return 0
    fi

    log_message "Removing container: ${container_name}..." "${LEVEL_INFO}"

    while true; do
        if run_command_with_retry "docker rm -f ${container_name}"; then
            log_message "Container '${container_name}' removed successfully." "${LEVEL_INFO}"
            return 0
        fi

        if (( retry_count >= max_retries )); then
            log_message "Failed to remove container '${container_name}' after ${max_retries} retries. Giving up." "${LEVEL_ERROR}"
            return 1
        fi

        log_message "Retrying to remove container '${container_name}'... Retry ${retry_count}/${max_retries}" "${LEVEL_WARNING}"
        sleep "${fib1}"
        local new_delay=$((fib1 + fib2))
        fib1=$fib2
        fib2=$new_delay
        ((retry_count++))
    done
}

stop_and_remove_container() {
    local container_name="$1"

    log_message "Stopping and removing container '${container_name}'..." "${LEVEL_INFO}"

    list_running_containers

    if container_exists "${container_name}"; then
        if container_is_running "${container_name}"; then
            log_message "Stopping container '${container_name}'..." "${LEVEL_INFO}"
            if try_stop_container "${container_name}"; then
                wait_for_container_stop "${container_name}"
            else
                log_message "Failed to stop container '${container_name}'." "${LEVEL_ERROR}"
                return 1
            fi
        fi

        log_message "Removing container '${container_name}'..." "${LEVEL_INFO}"
        if ! remove_container "${container_name}"; then
            log_message "Failed to remove container '${container_name}'." "${LEVEL_ERROR}"
            return 1
        fi
    else
        log_message "Container '${container_name}' does not exist. Skipping stop and remove steps." "${LEVEL_WARNING}"
    fi
}

pull_docker_image() {
    local image_name="$1"
    local tag="${2:-latest}"

    log_message "Pulling Docker image: ${image_name}:${tag}..." "${LEVEL_INFO}"
    run_command_with_retry "docker pull ${image_name}:${tag}"
    log_message "Docker image ${image_name}:${tag} pulled successfully." "${LEVEL_INFO}"
}

stop_remove_run_ollama_container() {
    local host="$1"
    local port="$2"
    local container_tag="${3:-latest}"

    log_message "Stopping and removing Ollama container..." "${LEVEL_INFO}"

    stop_and_remove_container "ollama" || return 1

    log_message "Running Ollama container..." "${LEVEL_INFO}"
    if ! docker run -d \
        --gpus all \
        --network=host \
        --volume "${OLLAMA_VOLUME_NAME}:/root/.ollama" \
        --env OLLAMA_HOST="${host}" \
        --restart always \
        --name "${OLLAMA_CONTAINER_NAME}" \
        "ollama/ollama:${container_tag}"; then
        log_message "Failed to start Ollama container." "${LEVEL_ERROR}"
        return 1
    fi

    wait_for_container_status_up "${OLLAMA_CONTAINER_NAME}" || return 1
    log_message "Ollama container started successfully." "${LEVEL_INFO}"
}

stop_remove_run_open_webui_container() {
    local ollama_host="$1"
    local ollama_port="$2"
    local open_webui_port="$3"
    local container_tag="${4:-latest}"
    local ollama_url="http://${ollama_host}:${ollama_port}"

    log_message "Stopping and removing Open-WebUI container..." "${LEVEL_INFO}"

    stop_and_remove_container "${OPEN_WEBUI_CONTAINER_NAME}" || return 1

    log_message "Running Open-WebUI container..." "${LEVEL_INFO}"
    if ! docker run -d \
        --gpus all \
        --network=host \
        --volume "${OPEN_WEBUI_VOLUME_NAME}:/app/backend/data" \
        --env OLLAMA_BASE_URL=${ollama_url} \
        --env PORT=${open_webui_port} \
        --name "${OPEN_WEBUI_CONTAINER_NAME}" \
        --restart always \
        "ghcr.io/open-webui/open-webui:${container_tag}"; then
        log_message "Failed to start Open-WebUI container." "${LEVEL_ERROR}"
        return 1
    fi

    wait_for_container_status_up "${OPEN_WEBUI_CONTAINER_NAME}" || return 1
    log_message "Open-WebUI container started successfully." "${LEVEL_INFO}"
}

verify_open_webui_setup() {
    local host="${1:-localhost}"
    local port="${2:-3000}"
    local url="http://${host}:${port}"

    log_message "Verifying Open-WebUI setup on ${url}..." "${LEVEL_INFO}"

    retry_count=0
    max_retries=5
    while ! ss -tuln | grep -q "${port}" && [[ $retry_count -lt $max_retries ]]; do
        log_message "Waiting for Open-WebUI to start on port ${port}... Retry $((retry_count + 1))/${max_retries}" "${LEVEL_INFO}"
        sleep $((2 ** retry_count))
        ((retry_count++))
    done

    if ! ss -tuln | grep -q "${port}"; then
        log_message "Open-WebUI is not listening on port ${port} after ${max_retries} attempts." "${LEVEL_ERROR}"
        exit 1
    fi

    run_command_with_retry "curl -s -o /dev/null --write-out \"%{response_code}\n\" ${url}"
    run_command_with_retry "docker logs \"${OPEN_WEBUI_CONTAINER_NAME}\""

    log_message "Open-WebUI setup verified successfully." "${LEVEL_INFO}"
}

install_nvidia_container_toolkit() {
    log_message "Installing NVIDIA Container Toolkit..." "${LEVEL_INFO}"

    # Download the NVIDIA GPG key and add it to the apt keyring
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | \
        sudo gpg --dearmor --yes -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg

    # Add the NVIDIA repository with the signed-by option
    curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
        sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
        sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list > /dev/null

    # Explicitly import the GPG key into the trusted keyring to avoid NO_PUBKEY errors
    sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys DDCAE044F796ECB0

    # Update the package lists and install the toolkit
    run_command_with_retry "sudo apt-get update"
    run_command_with_retry "sudo DEBIAN_FRONTEND=noninteractive apt-get install -y nvidia-container-toolkit"
    verify_nvidia_environment
}

pull_ollama_models() {
    log_message "Pulling Ollama models..." "${LEVEL_INFO}"

    source "$(dirname "${BASH_SOURCE[0]}")/default_ollama_models.sh"

    local models=("${DEFAULT_OLLAMA_MODELS[@]}")  # Copy default models

    log_message "Fetching installed models..." "${LEVEL_INFO}"
    local installed_models
    installed_models=$(ollama list | awk '{if(NR>1) print $1}')

    log_message "Adding installed models to predefined list..." "${LEVEL_INFO}"
    for model in $installed_models; do
        if [[ ! " ${models[@]} " =~ " ${model} " ]]; then
            models+=("$model")
        fi
    done

    log_message "Pulling models for Ollama..." "${LEVEL_INFO}"
    for model in "${models[@]}"; do
        log_message "Pulling model: $model" "${LEVEL_INFO}"
        run_command_with_retry "ollama pull $model"
    done

    log_message "Model pulling completed." "${LEVEL_INFO}"
}