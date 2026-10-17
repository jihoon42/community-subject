#!/usr/bin/env bash
# Verify a signed evaluation record without any server (OpenSSH 8.0+).
#
# Usage: record/verify.sh <result.json> <allowed_signers> <identity>
#   result.json      the record; <result.json>.sig must sit next to it
#   allowed_signers  file in the OpenSSH allowed_signers format, one line
#                    per trusted evaluator (see allowed_signers.example)
#   identity         the evaluator identity to check, e.g. jkim2@42gyeongsan
#
# The exit status of ssh-keygen -Y verify is propagated: 0 means the
# record is byte for byte what that identity signed, anything else means
# it is not.

NAMESPACE="community-subject"

if [ "$#" -ne 3 ]; then
    echo "usage: $0 <result.json> <allowed_signers> <identity>" >&2
    exit 2
fi
RESULT=$1
SIGNERS=$2
IDENTITY=$3
SIG=$RESULT.sig

if [ ! -f "$RESULT" ]; then
    echo "verify.sh: record not found: $RESULT" >&2
    exit 1
fi
if [ ! -f "$SIG" ]; then
    echo "verify.sh: signature not found: $SIG" >&2
    exit 1
fi
if [ ! -f "$SIGNERS" ]; then
    echo "verify.sh: allowed_signers file not found: $SIGNERS" >&2
    exit 1
fi
if ! command -v ssh-keygen >/dev/null 2>&1; then
    echo "verify.sh: ssh-keygen not found (OpenSSH 8.0 or newer is required)" >&2
    exit 1
fi

ssh-keygen -Y verify -f "$SIGNERS" -I "$IDENTITY" -n "$NAMESPACE" -s "$SIG" <"$RESULT"
rc=$?
if [ "$rc" -eq 0 ]; then
    echo "VERIFIED: $RESULT is signed by $IDENTITY (namespace $NAMESPACE)"
else
    echo "NOT VERIFIED: $RESULT does not carry a valid signature by $IDENTITY (ssh-keygen exit $rc)"
fi
exit "$rc"
