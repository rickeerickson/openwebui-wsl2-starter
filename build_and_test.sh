#!/bin/bash
#
# Lint, validate, and test the openwebui-wsl2-starter repo.
#
# Runs bash syntax checks, shellcheck, markdownlint, PowerShell
# parse/lint, and unit tests. Missing tools are skipped gracefully.
#
# Usage:
#   ./build_and_test.sh              # Run all checks
#   ./build_and_test.sh --install    # Install missing tools first
#   ./build_and_test.sh --fix        # Auto-fix (markdownlint)
#   ./build_and_test.sh --shell      # Shell checks only
#   ./build_and_test.sh --markdown   # Markdown checks only
#   ./build_and_test.sh --powershell # PowerShell checks only
#   ./build_and_test.sh --test       # Tests only
#   ./build_and_test.sh --help       # Usage info

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VENV_DIR="${SCRIPT_DIR}/.venv"

# Tools that can be installed via pip (run inside venv)
PIP_TOOLS=(shellcheck-py)

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Result tracking (parallel arrays for macOS bash 3.2 compat)
RESULT_NAMES=()
RESULT_STATUSES=()
RESULT_DETAILS=()
EXIT_CODE=0

# Flags
FIX_MODE=false
INSTALL_MODE=false
RUN_ALL=true
RUN_SHELL=false
RUN_MARKDOWN=false
RUN_POWERSHELL=false
RUN_TEST=false

# -------------------------------------------------------------------
# Utility functions
# -------------------------------------------------------------------

check_tool_installed() {
    local tool="$1"
    if ! command -v "${tool}" &>/dev/null; then
        echo -e "  ${YELLOW}[SKIP]${NC} ${tool} not installed (rerun with --install)"
        return 1
    fi
    return 0
}

record_result() {
    local name="$1"
    local status="$2"
    local detail="$3"
    RESULT_NAMES+=("${name}")
    RESULT_STATUSES+=("${status}")
    RESULT_DETAILS+=("${detail}")
    if [[ "${status}" == "FAIL" ]]; then
        EXIT_CODE=1
    fi
}

should_run() {
    local flag="$1"
    [[ "${RUN_ALL}" == "true" ]] || [[ "${flag}" == "true" ]]
}

print_section_header() {
    local title="$1"
    echo ""
    echo -e "${BLUE}=== ${title} ===${NC}"
}

# -------------------------------------------------------------------
# Python venv setup
# -------------------------------------------------------------------

setup_venv() {
    if ! command -v python3 &>/dev/null; then
        echo -e "${YELLOW}python3 not found, skipping venv${NC}"
        return 1
    fi
    if [[ ! -d "${VENV_DIR}" ]]; then
        echo -e "${BLUE}Creating venv at ${VENV_DIR}...${NC}"
        python3 -m venv "${VENV_DIR}"
    fi
    # shellcheck source=/dev/null
    source "${VENV_DIR}/bin/activate"
    pip install --quiet --upgrade pip
    return 0
}

# -------------------------------------------------------------------
# Install missing tools
# -------------------------------------------------------------------

detect_platform() {
    case "$(uname -s)" in
        Darwin) echo "macos" ;;
        Linux)  echo "linux" ;;
        *)      echo "unknown" ;;
    esac
}

install_brew_or_apt() {
    local tool="$1"
    local platform
    platform=$(detect_platform)

    if [[ "${platform}" == "macos" ]]; then
        if command -v brew &>/dev/null; then
            echo -e "  ${BLUE}brew install ${tool}${NC}"
            brew install "${tool}"
            return $?
        fi
        echo -e "  ${RED}brew not found${NC}"
        return 1
    elif [[ "${platform}" == "linux" ]]; then
        if command -v apt-get &>/dev/null; then
            echo -e "  ${BLUE}sudo apt-get install -y ${tool}${NC}"
            sudo apt-get install -y "${tool}"
            return $?
        fi
        echo -e "  ${RED}apt-get not found${NC}"
        return 1
    fi
    echo -e "  ${RED}Unsupported platform: ${platform}${NC}"
    return 1
}

install_pip_tool() {
    local tool="$1"
    if [[ -z "${VIRTUAL_ENV:-}" ]]; then
        if ! setup_venv; then
            echo -e "  ${RED}Cannot install ${tool}: no venv${NC}"
            return 1
        fi
    fi
    echo -e "  ${BLUE}pip install ${tool}${NC}"
    pip install --quiet "${tool}"
}

install_tools() {
    print_section_header "Installing missing tools"

    # Prefer pip (shellcheck-py) for venv isolation,
    # fall back to brew/apt
    if ! command -v shellcheck &>/dev/null; then
        echo -e "  Installing shellcheck..."
        install_pip_tool "shellcheck-py" \
            || install_brew_or_apt "shellcheck" \
            || echo -e "  ${RED}Could not install shellcheck${NC}"
    else
        echo -e "  ${GREEN}[OK]${NC}   shellcheck already installed"
    fi

    # markdownlint-cli: brew on macOS, npm elsewhere
    if ! command -v markdownlint &>/dev/null; then
        echo -e "  Installing markdownlint-cli..."
        local platform
        platform=$(detect_platform)
        if [[ "${platform}" == "macos" ]] \
            && command -v brew &>/dev/null; then
            echo -e "  ${BLUE}brew install markdownlint-cli${NC}"
            brew install markdownlint-cli
        elif command -v npm &>/dev/null; then
            echo -e "  ${BLUE}npm install -g markdownlint-cli${NC}"
            npm install -g markdownlint-cli
        else
            echo -e "  ${RED}No package manager found for markdownlint-cli (need brew or npm)${NC}"
        fi
    else
        echo -e "  ${GREEN}[OK]${NC}   markdownlint already installed"
    fi

    # pwsh: brew cask on macOS, apt on Linux
    if ! command -v pwsh &>/dev/null; then
        echo -e "  Installing PowerShell..."
        local platform
        platform=$(detect_platform)
        if [[ "${platform}" == "macos" ]] \
            && command -v brew &>/dev/null; then
            echo -e "  ${BLUE}brew install --cask powershell${NC}"
            brew install --cask powershell
        elif [[ "${platform}" == "linux" ]] \
            && command -v apt-get &>/dev/null; then
            echo -e "  ${BLUE}Installing pwsh via apt${NC}"
            sudo apt-get update \
                && sudo apt-get install -y powershell
        else
            echo -e "  ${RED}Could not install pwsh (need brew or apt)${NC}"
        fi
    else
        echo -e "  ${GREEN}[OK]${NC}   pwsh already installed"
    fi

    # PSScriptAnalyzer: install module if pwsh is available
    if command -v pwsh &>/dev/null; then
        if ! pwsh -NoProfile -Command \
            'if (Get-Module -ListAvailable PSScriptAnalyzer) { exit 0 } else { exit 1 }' \
            2>/dev/null; then
            echo -e "  Installing PSScriptAnalyzer..."
            echo -e "  ${BLUE}pwsh Install-Module PSScriptAnalyzer${NC}"
            pwsh -NoProfile -Command \
                'Install-Module PSScriptAnalyzer -Force -Scope CurrentUser'
        else
            echo -e "  ${GREEN}[OK]${NC}   PSScriptAnalyzer already installed"
        fi
    fi

    echo ""
}

# -------------------------------------------------------------------
# Check: bash -n syntax
# -------------------------------------------------------------------

check_bash_syntax() {
    print_section_header "Syntax: bash -n"

    local fail_count=0
    local pass_count=0

    while IFS= read -r -d '' sh_file; do
        local rel_path="${sh_file#"${SCRIPT_DIR}"/}"
        if bash -n "${sh_file}" 2>&1; then
            echo -e "  ${GREEN}[OK]${NC}   ${rel_path}"
            ((pass_count++))
        else
            echo -e "  ${RED}[FAIL]${NC} ${rel_path}"
            ((fail_count++))
        fi
    done < <(find "${SCRIPT_DIR}" -name "*.sh" \
        -not -path "*/.venv/*" -not -path "*/.git/*" \
        -type f -print0)

    if [[ ${fail_count} -eq 0 ]]; then
        record_result "Bash syntax" "PASS" \
            "${pass_count} file(s) passed"
    else
        record_result "Bash syntax" "FAIL" \
            "${fail_count} failed, ${pass_count} passed"
    fi
}

# -------------------------------------------------------------------
# Check: shellcheck
# -------------------------------------------------------------------

check_shellcheck() {
    print_section_header "Shell Lint (shellcheck)"

    if ! check_tool_installed "shellcheck"; then
        record_result "Shell lint" "SKIP" \
            "not installed (rerun with --install)"
        return
    fi

    local fail_count=0
    local pass_count=0

    while IFS= read -r -d '' sh_file; do
        local rel_path="${sh_file#"${SCRIPT_DIR}"/}"
        if shellcheck -x -S warning "${sh_file}" \
            >/dev/null 2>&1; then
            echo -e "  ${GREEN}[OK]${NC}   ${rel_path}"
            ((pass_count++))
        else
            echo -e "  ${RED}[FAIL]${NC} ${rel_path}"
            shellcheck -x -S warning "${sh_file}" 2>&1 \
                | head -20
            ((fail_count++))
        fi
    done < <(find "${SCRIPT_DIR}" -name "*.sh" \
        -not -path "*/.venv/*" -not -path "*/.git/*" \
        -type f -print0)

    if [[ ${fail_count} -eq 0 ]]; then
        record_result "Shell lint" "PASS" \
            "${pass_count} file(s) passed"
    else
        record_result "Shell lint" "FAIL" \
            "${fail_count} failed, ${pass_count} passed"
    fi
}

# -------------------------------------------------------------------
# Check: markdownlint
# -------------------------------------------------------------------

check_markdown() {
    print_section_header "Markdown Lint (markdownlint)"

    if ! check_tool_installed "markdownlint"; then
        record_result "Markdown lint" "SKIP" \
            "not installed (rerun with --install)"
        return
    fi

    local mdl_args=()
    if [[ "${FIX_MODE}" == "true" ]]; then
        mdl_args+=("--fix")
    fi

    local fail_count=0
    local pass_count=0

    while IFS= read -r -d '' md_file; do
        local rel_path="${md_file#"${SCRIPT_DIR}"/}"
        if markdownlint ${mdl_args[@]+"${mdl_args[@]}"} \
            "${md_file}" >/dev/null 2>&1; then
            echo -e "  ${GREEN}[OK]${NC}   ${rel_path}"
            ((pass_count++))
        else
            echo -e "  ${RED}[FAIL]${NC} ${rel_path}"
            markdownlint "${md_file}" 2>&1 | head -20
            ((fail_count++))
        fi
    done < <(find "${SCRIPT_DIR}" -name "*.md" \
        -not -path "*/.venv/*" -not -path "*/.git/*" \
        -type f -print0)

    if [[ ${fail_count} -eq 0 ]]; then
        record_result "Markdown lint" "PASS" \
            "${pass_count} file(s) passed"
    else
        record_result "Markdown lint" "FAIL" \
            "${fail_count} failed, ${pass_count} passed"
    fi
}

# -------------------------------------------------------------------
# Check: PowerShell syntax parse
# -------------------------------------------------------------------

check_powershell_syntax() {
    print_section_header "Syntax: PowerShell parse"

    if ! check_tool_installed "pwsh"; then
        record_result "PowerShell syntax" "SKIP" \
            "pwsh not installed"
        return
    fi

    local fail_count=0
    local pass_count=0

    while IFS= read -r -d '' ps_file; do
        local rel_path="${ps_file#"${SCRIPT_DIR}"/}"
        local errors
        errors=$(pwsh -NoProfile -Command "
            \$errors = \$null
            [System.Management.Automation.Language.Parser]::ParseFile('${ps_file}', [ref]\$null, [ref]\$errors) | Out-Null
            if (\$errors.Count -gt 0) { \$errors | ForEach-Object { Write-Output \$_.ToString() }; exit 1 }
        " 2>&1) || true
        if [[ -z "${errors}" ]]; then
            echo -e "  ${GREEN}[OK]${NC}   ${rel_path}"
            ((pass_count++))
        else
            echo -e "  ${RED}[FAIL]${NC} ${rel_path}"
            echo "${errors}"
            ((fail_count++))
        fi
    done < <(find "${SCRIPT_DIR}" \
        \( -name "*.ps1" -o -name "*.psm1" \) \
        -not -path "*/.venv/*" -not -path "*/.git/*" \
        -type f -print0)

    if [[ ${fail_count} -eq 0 ]]; then
        record_result "PowerShell syntax" "PASS" \
            "${pass_count} file(s) passed"
    else
        record_result "PowerShell syntax" "FAIL" \
            "${fail_count} failed, ${pass_count} passed"
    fi
}

# -------------------------------------------------------------------
# Check: PSScriptAnalyzer
# -------------------------------------------------------------------

check_psscriptanalyzer() {
    print_section_header "PowerShell Lint (PSScriptAnalyzer)"

    if ! check_tool_installed "pwsh"; then
        record_result "PowerShell lint" "SKIP" \
            "pwsh not installed"
        return
    fi

    if ! pwsh -NoProfile -Command \
        'if (Get-Module -ListAvailable PSScriptAnalyzer) { exit 0 } else { exit 1 }' \
        2>/dev/null; then
        echo -e "  ${YELLOW}[SKIP]${NC} PSScriptAnalyzer module not installed (rerun with --install)"
        record_result "PowerShell lint" "SKIP" \
            "module not installed (rerun with --install)"
        return
    fi

    local fail_count=0
    local pass_count=0

    while IFS= read -r -d '' ps_file; do
        local rel_path="${ps_file#"${SCRIPT_DIR}"/}"
        local output
        output=$(pwsh -NoProfile -Command \
            "Invoke-ScriptAnalyzer -Path '${ps_file}' -Settings '${SCRIPT_DIR}/.PSScriptAnalyzerSettings.psd1' -Severity Warning,Error 2>&1" \
            2>&1)
        if [[ -z "${output}" ]]; then
            echo -e "  ${GREEN}[OK]${NC}   ${rel_path}"
            ((pass_count++))
        else
            echo -e "  ${RED}[FAIL]${NC} ${rel_path}"
            echo "${output}"
            ((fail_count++))
        fi
    done < <(find "${SCRIPT_DIR}" \
        \( -name "*.ps1" -o -name "*.psm1" \) \
        -not -path "*/.venv/*" -not -path "*/.git/*" \
        -type f -print0)

    if [[ ${fail_count} -eq 0 ]]; then
        record_result "PowerShell lint" "PASS" \
            "${pass_count} file(s) passed"
    else
        record_result "PowerShell lint" "FAIL" \
            "${fail_count} failed, ${pass_count} passed"
    fi
}

# -------------------------------------------------------------------
# Check: Tests
# -------------------------------------------------------------------

check_tests() {
    print_section_header "Tests"

    local test_dir="${SCRIPT_DIR}/bash/tests"
    if [[ ! -d "${test_dir}" ]]; then
        echo -e "  ${YELLOW}[SKIP]${NC} test directory not found"
        record_result "Tests" "SKIP" "test directory not found"
        return
    fi

    local fail_count=0
    local pass_count=0

    while IFS= read -r -d '' test_file; do
        local rel_path="${test_file#"${SCRIPT_DIR}"/}"
        if bash "${test_file}" >/dev/null 2>&1; then
            echo -e "  ${GREEN}[OK]${NC}   ${rel_path}"
            ((pass_count++))
        else
            echo -e "  ${RED}[FAIL]${NC} ${rel_path}"
            bash "${test_file}" 2>&1 | tail -10
            ((fail_count++))
        fi
    done < <(find "${test_dir}" -name "test_*.sh" -print0)

    if [[ ${fail_count} -eq 0 ]]; then
        record_result "Tests" "PASS" \
            "${pass_count} file(s) passed"
    else
        record_result "Tests" "FAIL" \
            "${fail_count} failed, ${pass_count} passed"
    fi
}

# -------------------------------------------------------------------
# Argument parsing
# -------------------------------------------------------------------

while [[ $# -gt 0 ]]; do
    case $1 in
        --install) INSTALL_MODE=true; shift ;;
        --fix) FIX_MODE=true; shift ;;
        --shell)
            RUN_ALL=false; RUN_SHELL=true; shift ;;
        --markdown)
            RUN_ALL=false; RUN_MARKDOWN=true; shift ;;
        --powershell)
            RUN_ALL=false; RUN_POWERSHELL=true; shift ;;
        --test)
            RUN_ALL=false; RUN_TEST=true; shift ;;
        --help|-h)
            cat << 'EOF'
Usage: ./build_and_test.sh [OPTIONS]

Checks:
  Bash syntax       bash -n on all .sh files
  Shell lint        shellcheck on all .sh files
  Markdown lint     markdownlint on all .md files (--fix applies)
  PowerShell syntax pwsh parse on all .ps1/.psm1 files
  PowerShell lint   PSScriptAnalyzer on all .ps1/.psm1 files
  Tests             bash test suites in bash/tests/

Options:
  --install        Install missing linters and tools
  --fix            Auto-fix where possible (markdownlint)
  --shell          Run only shell checks (syntax + lint)
  --markdown       Run only markdown lint
  --powershell     Run only PowerShell checks
  --test           Run only tests
  --help, -h       Show this help

Multiple flags can be combined (e.g., --shell --markdown).
Missing tools are skipped gracefully.
EOF
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# -------------------------------------------------------------------
# Setup
# -------------------------------------------------------------------

setup_venv || true

if [[ "${INSTALL_MODE}" == "true" ]]; then
    install_tools
fi

echo -e "${BLUE}=== openwebui-wsl2-starter: Build & Test ===${NC}"
echo ""
echo "Directory: ${SCRIPT_DIR}"
[[ "${FIX_MODE}" == "true" ]] \
    && echo -e "Fix mode:  ${GREEN}enabled${NC}"
echo ""

# -------------------------------------------------------------------
# Main dispatch
# -------------------------------------------------------------------

should_run "${RUN_SHELL}" && check_bash_syntax
should_run "${RUN_SHELL}" && check_shellcheck
should_run "${RUN_MARKDOWN}" && check_markdown
should_run "${RUN_POWERSHELL}" && check_powershell_syntax
should_run "${RUN_POWERSHELL}" && check_psscriptanalyzer
should_run "${RUN_TEST}" && check_tests

# -------------------------------------------------------------------
# Summary
# -------------------------------------------------------------------

echo ""
echo -e "${BLUE}=== Summary ===${NC}"
echo ""

pass_count=0
fail_count=0
skip_count=0

for i in "${!RESULT_NAMES[@]}"; do
    local_name="${RESULT_NAMES[$i]}"
    local_status="${RESULT_STATUSES[$i]}"
    local_detail="${RESULT_DETAILS[$i]}"

    case "${local_status}" in
        PASS)
            echo -e "  ${GREEN}[PASS]${NC} ${local_name} - ${local_detail}"
            ((pass_count++))
            ;;
        FAIL)
            echo -e "  ${RED}[FAIL]${NC} ${local_name} - ${local_detail}"
            ((fail_count++))
            ;;
        SKIP)
            echo -e "  ${YELLOW}[SKIP]${NC} ${local_name} - ${local_detail}"
            ((skip_count++))
            ;;
    esac
done

echo ""
echo -e "Passed: ${GREEN}${pass_count}${NC}  Failed: ${RED}${fail_count}${NC}  Skipped: ${YELLOW}${skip_count}${NC}"

if [[ ${fail_count} -gt 0 ]]; then
    echo ""
    echo -e "${RED}Build failed.${NC}"
fi

exit ${EXIT_CODE}
