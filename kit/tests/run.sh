#!/usr/bin/env bash
# gauss_jordan community subject (v0.1): machine evaluation runner.
#
# Usage: tests/run.sh "<command>"
#   <command> is how to run the submission, for example "./gauss_jordan"
#   or "python3 tests/reference.py". The runner appends the mode flag and
#   the input path:  <command> [-r|-d|-i] <file>
#   Some cases pass "-" as the file name and feed the input on stdin.
#
# Environment:
#   RESULT       where to write the JSON record (default: tests/result.json
#                next to this script)
#   SUBJECT      path of the subject markdown whose frontmatter gives the
#                subject id and version (default: ../gauss_jordan_subject.md
#                relative to the kit directory, then ../SUBJECT.md)
#   TIMEOUT_SEC  per-run time limit when the "timeout" command exists
#                (default: 10)
#
# Exit status: 0 when every case passes, 1 otherwise, 2 on usage error.
# Needs only POSIX tools plus bash: cmp, grep, sed, awk, date, uname and
# sha256sum (or shasum). No Python is required.

if [ "$#" -ne 1 ] || [ -z "$1" ]; then
    echo "usage: $0 \"<command>\"" >&2
    echo "  example: $0 ./gauss_jordan" >&2
    exit 2
fi
CMD=$1

TESTS_DIR=$(cd "$(dirname "$0")" && pwd)
KIT_DIR=$(dirname "$TESTS_DIR")
CASES=$TESTS_DIR/cases
EXPECTED=$TESTS_DIR/expected
RESULT=${RESULT:-$TESTS_DIR/result.json}
TIMEOUT_SEC=${TIMEOUT_SEC:-10}

TMP=$(mktemp -d "${TMPDIR:-/tmp}/gjkit.XXXXXX" 2>/dev/null)
if [ -z "$TMP" ] || [ ! -d "$TMP" ]; then
    TMP="${TMPDIR:-/tmp}/gjkit.$$"
    mkdir -p "$TMP" 2>/dev/null
fi
if [ -z "$TMP" ] || [ ! -d "$TMP" ] || [ ! -w "$TMP" ]; then
    echo "run.sh: cannot create a temporary directory (set TMPDIR)" >&2
    exit 2
fi
trap 'rm -rf "$TMP"' EXIT

RUN_PREFIX=""
if command -v timeout >/dev/null 2>&1; then
    RUN_PREFIX="timeout $TIMEOUT_SEC"
fi

passed=0
failed=0
CASES_JSON=""

# ---------------------------------------------------------------- helpers

# run_sub <stdin file or ""> [args...]  : runs the submission, fills
# $TMP/out, $TMP/err and STATUS.
run_sub() {
    stdin_file=$1
    shift
    if [ -n "$stdin_file" ]; then
        eval "$RUN_PREFIX $CMD" '"$@"' <"$stdin_file" >"$TMP/out" 2>"$TMP/err"
    else
        eval "$RUN_PREFIX $CMD" '"$@"' </dev/null >"$TMP/out" 2>"$TMP/err"
    fi
    STATUS=$?
}

mode_flag() {
    case $1 in
        rref) printf '' ;;
        rank) printf '%s' "-r" ;;
        det)  printf '%s' "-d" ;;
        inv)  printf '%s' "-i" ;;
    esac
}

record() {   # record <name> <mode> <pass|fail> [detail lines...]
    name=$1
    mode=$2
    verdict=$3
    shift 3
    if [ "$verdict" = pass ]; then
        passed=$((passed + 1))
        printf 'PASS %s %s\n' "$name" "$mode"
        json_pass=true
    else
        failed=$((failed + 1))
        printf 'FAIL %s %s\n' "$name" "$mode"
        for line in "$@"; do
            printf '     %s\n' "$line"
        done
        json_pass=false
    fi
    entry="    {\"name\": \"$name\", \"mode\": \"$mode\", \"pass\": $json_pass}"
    if [ -z "$CASES_JSON" ]; then
        CASES_JSON=$entry
    else
        CASES_JSON="$CASES_JSON,
$entry"
    fi
}

# check_ok <name> <mode> <expected file>: exit status 0 and byte-exact stdout
check_ok() {
    exp=$EXPECTED/$3
    if [ "$STATUS" -eq 0 ] && cmp -s "$TMP/out" "$exp"; then
        record "$1" "$2" pass
    else
        detail=""
        if [ "$STATUS" -ne 0 ]; then
            detail="exit status $STATUS (expected 0)"
        fi
        if ! cmp -s "$TMP/out" "$exp"; then
            detail="${detail:+$detail, }stdout differs from expected/$3"
        fi
        record "$1" "$2" fail "$detail" \
            "expected: $(tr '\n' '|' <"$exp")" \
            "got:      $(head -c 400 "$TMP/out" | tr '\n' '|')" \
            "stderr:   $(head -c 200 "$TMP/err" | tr '\n' '|')"
    fi
}

# check_err <name> <mode>: exit status 1, empty stdout, "Error" on stderr
check_err() {
    ok=1
    reasons=""
    if [ "$STATUS" -ne 1 ]; then
        ok=0
        reasons="exit status $STATUS (expected 1)"
    fi
    if [ -s "$TMP/out" ]; then
        ok=0
        reasons="$reasons; stdout not empty: $(head -c 200 "$TMP/out" | tr '\n' '|')"
    fi
    if ! grep -q 'Error' "$TMP/err"; then
        ok=0
        reasons="$reasons; stderr lacks Error: $(head -c 200 "$TMP/err" | tr '\n' '|')"
    fi
    if [ "$ok" -eq 1 ]; then
        record "$1" "$2" pass
    else
        record "$1" "$2" fail "${reasons#; }"
    fi
}

# t_ok <case> <mode> [expected name]  : normal case read from a file
t_ok() {
    flag=$(mode_flag "$2")
    expname=${3:-$1.$2.out}
    run_sub "" $flag "$CASES/$1.txt"
    check_ok "$1" "$2" "$expname"
}

# t_ok_stdin <case> <mode> [expected name] : normal case fed on stdin as "-"
t_ok_stdin() {
    flag=$(mode_flag "$2")
    expname=${3:-$1.$2.out}
    run_sub "$CASES/$1.txt" $flag -
    check_ok "${1}_stdin" "$2" "$expname"
}

# t_err <case> <mode> : error case read from a file
t_err() {
    flag=$(mode_flag "$2")
    run_sub "" $flag "$CASES/$1.txt"
    check_err "$1" "$2"
}

# t_err_stdin <case> <mode> : error case fed on stdin as "-"
t_err_stdin() {
    flag=$(mode_flag "$2")
    run_sub "$CASES/$1.txt" $flag -
    check_err "${1}_stdin" "$2"
}

# ------------------------------------------------------------------ cases

echo "gauss_jordan kit: running \"$CMD\""

# Section 3.5 examples and evaluation sheet section 2 (rref, default mode)
t_ok system rref
t_ok singular rref
t_ok swap rref
t_ok zero rref
t_ok contradiction rref
t_ok one rref
t_ok row rref
t_ok col rref
t_ok inv rref
t_ok fractions rref
t_ok detsign rref
t_ok unreduced rref
t_ok whitespace rref

# Section 3 (rank)
t_ok system rank
t_ok singular rank
t_ok zero rank
t_ok swap rank
t_ok contradiction rank
t_ok one rank
t_ok row rank
t_ok col rank
t_ok inv rank
t_ok fractions rank
t_ok detsign rank
t_ok unreduced rank
t_ok whitespace rank

# Section 3 (determinant)
t_ok inv det
t_ok fractions det
t_ok singular det
t_ok one det
t_ok detsign det
t_ok unreduced det
t_err swap det
t_err zero det
t_err system det
t_err row det
t_err col det
t_err contradiction det

# Section 4 (inverse)
t_ok inv inv
t_ok fractions inv
t_ok one inv
t_ok singular inv
t_ok detsign inv
t_ok unreduced inv
t_err swap inv
t_err zero inv
t_err system inv
t_err row inv
t_err col inv
t_err contradiction inv

# Section 1: standard input as "-"
t_ok_stdin system rref
t_ok_stdin system rank
t_ok_stdin whitespace rref
t_err_stdin zero_denominator rref

# Section 1: overflow (any elimination order leaves long long)
t_err overflow rref
t_err overflow rank
t_err overflow det
t_err overflow inv

# Section 1: input format errors
t_err bad_token rref
t_err bad_suffix rref
t_err bad_dots rref
t_err zero_denominator rref
t_err negative_denominator rref
t_err empty_numerator rref
t_err bad_header rref
t_err bad_header_three rref
t_err zero_rows rref
t_err zero_cols rref
t_err too_many_rows rref
t_err dimension_mismatch rref
t_err dimension_mismatch_cols rref
t_err empty rref

# Section 1: arguments and files
run_sub ""
check_err no_args args
run_sub "" "$CASES/system.txt" "$CASES/inv.txt" "$CASES/swap.txt"
check_err three_args args
run_sub "" -x "$CASES/system.txt"
check_err unknown_flag args
run_sub "" -r -d "$CASES/system.txt"
check_err two_flags args
run_sub "" "$CASES/does_not_exist.txt"
check_err missing_file args
if [ "$(id -u)" = "0" ]; then
    echo "SKIP unreadable_file args (running as root, permission bits are not enforced)"
else
    cp "$CASES/system.txt" "$TMP/unreadable.txt"
    chmod 000 "$TMP/unreadable.txt"
    run_sub "" "$TMP/unreadable.txt"
    check_err unreadable_file args
    chmod 600 "$TMP/unreadable.txt"
fi

total=$((passed + failed))
echo "summary: passed=$passed failed=$failed total=$total"

# ------------------------------------------------------------ result.json

json_escape() {
    printf '%s' "$1" | tr -d '\000-\037' | sed -e 's/\\/\\\\/g' -e 's/"/\\"/g'
}

json_string_or_null() {   # empty -> null, else "escaped"
    if [ -z "$1" ]; then
        printf 'null'
    else
        printf '"%s"' "$(json_escape "$1")"
    fi
}

frontmatter_field() {     # <file> <key> : value of "key: value" in the frontmatter
    awk -v key="$2" '
        NR == 1 { if ($0 != "---") exit; next }
        $0 == "---" { exit }
        index($0, key ":") == 1 {
            v = substr($0, length(key) + 2)
            sub(/^[ \t]+/, "", v); sub(/[ \t\r]+$/, "", v)
            gsub(/^"|"$/, "", v)
            print v; exit
        }' "$1" 2>/dev/null
}

SUBJECT_FILE=${SUBJECT:-}
if [ -z "$SUBJECT_FILE" ]; then
    for candidate in "$KIT_DIR/../gauss_jordan_subject.md" "$KIT_DIR/../SUBJECT.md"; do
        if [ -f "$candidate" ]; then
            SUBJECT_FILE=$candidate
            break
        fi
    done
fi
subject_id=""
subject_version=""
if [ -n "$SUBJECT_FILE" ] && [ -f "$SUBJECT_FILE" ]; then
    subject_id=$(frontmatter_field "$SUBJECT_FILE" id)
    subject_version=$(frontmatter_field "$SUBJECT_FILE" version)
fi
subject_id=${subject_id:-gauss_jordan}
subject_version=${subject_version:-0.1}

commit=$(git rev-parse HEAD 2>/dev/null || true)
uname_s=$(uname -srm 2>/dev/null || uname 2>/dev/null || true)
compiler=""
if command -v cc >/dev/null 2>&1; then
    compiler=$(cc --version 2>/dev/null | head -n 1)
fi
runner_sha=""
if command -v sha256sum >/dev/null 2>&1; then
    runner_sha=$(sha256sum "$TESTS_DIR/run.sh" | cut -d' ' -f1)
elif command -v shasum >/dev/null 2>&1; then
    runner_sha=$(shasum -a 256 "$TESTS_DIR/run.sh" | cut -d' ' -f1)
fi
timestamp=$(date -u +%Y-%m-%dT%H:%M:%SZ)

{
    printf '{\n'
    printf '  "subject": {"id": %s, "version": %s},\n' \
        "$(json_string_or_null "$subject_id")" "$(json_string_or_null "$subject_version")"
    printf '  "submission": {"commit": %s},\n' "$(json_string_or_null "$commit")"
    printf '  "environment": {"uname": %s, "compiler": %s, "runner_sha256": %s},\n' \
        "$(json_string_or_null "$uname_s")" "$(json_string_or_null "$compiler")" \
        "$(json_string_or_null "$runner_sha")"
    printf '  "cases": [\n%s\n  ],\n' "$CASES_JSON"
    printf '  "summary": {"passed": %d, "failed": %d, "total": %d},\n' "$passed" "$failed" "$total"
    printf '  "timestamp": "%s"\n' "$timestamp"
    printf '}\n'
} >"$RESULT"
echo "result: $RESULT"

if [ "$failed" -eq 0 ]; then
    exit 0
fi
exit 1
