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

script_dir="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
repo_lib_sh="${script_dir}/repo_lib.sh"
source_required_file "${repo_lib_sh}"

script_dir="$(dirname "$(realpath "${BASH_SOURCE[0]}")")"
script_name="$(basename "$(realpath "${BASH_SOURCE[0]}")")"
log_file="${script_dir}/${script_name}.log"

readonly REPO_ROOT=$(get_git_root)
readonly SEPARATOR_LONG=$(printf '=%.0s' {1..80})

readonly LEVEL_ERROR=0
readonly LEVEL_WARNING=1
readonly LEVEL_INFO=2
readonly LEVEL_DEBUG=3
readonly LEVEL_DEFAULT=${LEVEL_INFO}
readonly VERBOSITY_DEFAULT=${LEVEL_INFO}

readonly LEVEL_PREFIX_PAD_STRING="WARNING:"
readonly LEVEL_PREFIX_PAD_LENGTH="${#LEVEL_PREFIX_PAD_STRING}"

readonly LEVEL_ERROR_PREFIX=$(pad_right "ERROR:" $LEVEL_PREFIX_PAD_LENGTH)
readonly LEVEL_WARNING_PREFIX=$(pad_right "WARNING:" $LEVEL_PREFIX_PAD_LENGTH)
readonly LEVEL_INFO_PREFIX=$(pad_right "INFO:" $LEVEL_PREFIX_PAD_LENGTH)
readonly LEVEL_DEBUG_PREFIX=$(pad_right "DEBUG:" $LEVEL_PREFIX_PAD_LENGTH)
