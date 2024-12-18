#!/bin/bash
set -euo pipefail
trap 'echo "Error occurred in script: ${BASH_SOURCE[0]}, line: ${LINENO}, command: ${BASH_COMMAND}" >&2' ERR

source_required_file() {
    local filepath="$1"
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo "$(date '+%Y.%m.%d:%H:%M:%S') - DEBUG: Sourcing ${filepath}" >&2
    fi
    if [[ -f "$filepath" ]]; then
        source "$filepath"
    else
        echo "$(date '+%Y.%m.%d:%H:%M:%S') - ERROR: Required file ${filepath} not found." >&2
        exit 1
    fi
}

project_root="$(dirname "$(dirname "$(realpath "${BASH_SOURCE[0]}")")")"
project_env_sh="${project_root}/common/repo_env.sh"
source_required_file "${project_env_sh}"

script_dir="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
script_name="$(basename "$(realpath "${BASH_SOURCE[0]}")")"
log_file="${script_dir}/${script_name}.log"

run_test_case() {
    local test_name="$1"
    local command="$2"
    local should_fail="$3"
    local ignore_exit_status="$4"
    local expected_result="$5"
    local saved_opts=""
    local prefix=" "
    local debug_mode="${DEBUG:-false}"

    log_message "${prefix}Running test command: ${command} with debug_mode=\"${debug_mode}\", should_fail=\"${should_fail}\", ignore_exit_status=\"${ignore_exit_status}\" prefix=\"${prefix}\"" "${LEVEL_DEBUG_1}"

    if run_command "${command}" "${saved_opts}" "${should_fail}" "${ignore_exit_status}"; then
        if [[ "${expected_result}" -eq 0 ]]; then
            echo "${test_name} Passed"
        else
            echo "${test_name} Failed: Expected failure but succeeded"
            exit 1
        fi
    else
        if [[ "${expected_result}" -ne 0 ]]; then
            echo "${test_name} Passed"
        else
            echo "${test_name} Failed: Expected success but failed"
            exit 1
        fi
    fi
}

echo "Running tests for run_command..."

test_name="Test 1: Successful command, should succeed"
command="bash -c \"exit 0\""
should_fail=false
ignore_exit_status=false
expected_result=0
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 2: Failing command, should fail"
command="bash -c \"exit 1\""
should_fail=true
ignore_exit_status=false
expected_result=1
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 3: Failing command, ignore_exit_status"
command="bash -c \"exit 1\""
should_fail=false
ignore_exit_status=true
expected_result=0
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 4: Successful command, but should_fail is true"
command="bash -c \"exit 0\""
should_fail=true
ignore_exit_status=false
expected_result=1
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 5: Failing command with both should_fail and ignore_exit_status"
command="bash -c \"exit 1\""
should_fail=true
ignore_exit_status=true
expected_result=0
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 6: Empty command, should not fail"
command=""
should_fail=false
ignore_exit_status=false
expected_result=0
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 7: Empty command, should fail"
command=""
should_fail=true
ignore_exit_status=false
expected_result=1
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 8: Command with syntax error, should fail"
command="not-a-command"
should_fail=true
ignore_exit_status=false
expected_result=2
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 9: Command with syntax error, ignore_exit_status"
command="not-a-command"
should_fail=false
ignore_exit_status=true
expected_result=0
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

DEBUG=true
test_name="Test 10: Debug mode enabled, successful command"
command="bash -c \"exit 0\""
should_fail=false
ignore_exit_status=false
expected_result=0
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"
DEBUG=false

test_name="Test 11: Stress test with multiple chained commands"
command="echo foo && echo bar && echo baz && echo qux"
should_fail=false
ignore_exit_status=false
expected_result=0
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

test_name="Test 12: Failing command, but should_fail is true"
command="bash -c \"exit 0\""
should_fail=true
ignore_exit_status=false
expected_result=0
run_test_case "${test_name}" "${command}" "${should_fail}" "${ignore_exit_status}" "${expected_result}"

echo "All tests completed successfully!"
