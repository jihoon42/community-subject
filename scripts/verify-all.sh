#!/usr/bin/env bash
# Verify every record under records/<evaluatee>/*.json against the
# repository's allowed_signers, using kit/record/verify.sh.
# The signer identity is read from the signature key with
# ssh-keygen -Y find-principals, so records do not need to name it.
# Exit status: 0 when every record verifies, 1 otherwise.

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SIGNERS=$ROOT/allowed_signers
VERIFY=$ROOT/kit/record/verify.sh
NAMESPACE=community-subject

if [ ! -f "$SIGNERS" ]; then
    echo "verify-all: $SIGNERS not found" >&2
    exit 1
fi

total=0
failed=0
for rec in "$ROOT"/records/*/*.json; do
    [ -f "$rec" ] || continue
    total=$((total + 1))
    rel=${rec#"$ROOT"/}
    sig=$rec.sig
    if [ ! -f "$sig" ]; then
        echo "FAIL $rel (no .sig file)"
        failed=$((failed + 1))
        continue
    fi
    id=$(ssh-keygen -Y find-principals -s "$sig" -f "$SIGNERS" 2>/dev/null | head -n 1)
    if [ -z "$id" ]; then
        echo "FAIL $rel (signing key is not in allowed_signers)"
        failed=$((failed + 1))
        continue
    fi
    if "$VERIFY" "$rec" "$SIGNERS" "$id" >/dev/null 2>&1; then
        echo "OK   $rel ($id)"
    else
        echo "FAIL $rel ($id, signature does not match the file)"
        failed=$((failed + 1))
    fi
done
echo "records: $total, failed: $failed, namespace: $NAMESPACE"
[ "$failed" -eq 0 ]
