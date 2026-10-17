#!/usr/bin/env bash
# Sign an evaluation record with the evaluator's SSH key (OpenSSH 8.0+).
#
# Usage: record/sign.sh <result.json> <identity> [key]
#   result.json  the record written by tests/run.sh
#   identity     the evaluator's identity string, e.g. jkim2@42gyeongsan
#                (it is the principal that verifiers look up in their
#                allowed_signers file)
#   key          private key to sign with (default: ~/.ssh/id_ed25519)
#
# Produces <result.json>.sig next to the record, in the OpenSSH signature
# format, under the namespace "community-subject". No server is involved.

NAMESPACE="community-subject"

if [ "$#" -lt 2 ] || [ "$#" -gt 3 ]; then
    echo "usage: $0 <result.json> <identity> [key]" >&2
    exit 2
fi
RESULT=$1
IDENTITY=$2
KEY=${3:-$HOME/.ssh/id_ed25519}

if [ ! -f "$RESULT" ]; then
    echo "sign.sh: record not found: $RESULT" >&2
    exit 1
fi
if [ ! -f "$KEY" ]; then
    echo "sign.sh: private key not found: $KEY" >&2
    echo "  create one with: ssh-keygen -t ed25519 -f $KEY" >&2
    exit 1
fi
if ! command -v ssh-keygen >/dev/null 2>&1; then
    echo "sign.sh: ssh-keygen not found (OpenSSH 8.0 or newer is required)" >&2
    exit 1
fi

# ssh-keygen -Y sign writes <file>.sig; it reports what it does on stderr.
if ! ssh-keygen -Y sign -f "$KEY" -n "$NAMESPACE" "$RESULT"; then
    echo "sign.sh: signing failed" >&2
    exit 1
fi

echo "signed:     $RESULT"
echo "signature:  $RESULT.sig"
echo "identity:   $IDENTITY"
echo "namespace:  $NAMESPACE"
echo "key:        $KEY"

# The line a verifier needs in their allowed_signers file.
PUB=""
if [ -f "$KEY.pub" ]; then
    PUB=$(cut -d' ' -f1,2 "$KEY.pub")
elif PUB=$(ssh-keygen -y -f "$KEY" 2>/dev/null); then
    PUB=$(printf '%s' "$PUB" | cut -d' ' -f1,2)
fi
if [ -n "$PUB" ]; then
    echo "allowed_signers line for verifiers:"
    echo "  $IDENTITY namespaces=\"$NAMESPACE\" $PUB"
fi
exit 0
