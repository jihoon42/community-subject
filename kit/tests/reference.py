#!/usr/bin/env python3
"""Reference implementation of the gauss_jordan community subject (v0.1).

This file exists only to validate the evaluation kit itself (expected
outputs, runner behaviour). It is not a model answer for the subject,
which must be written in C.

Usage: reference.py [-r|-d|-i] file      ("-" reads standard input)

Behaviour follows sections 3.1 to 3.4 of the subject:
  - exact rational arithmetic (fractions.Fraction), always reduced
  - rref (default), -r rank, -d determinant, -i inverse ("singular")
  - every error: "Error" on stderr, exit status 1, nothing on stdout
  - long long emulation: after every arithmetic operation the reduced
    result must satisfy |numerator| <= 2**63-1 and denominator <= 2**63-1,
    otherwise the computation stops with Error. Input numerators and
    denominators are checked the same way before reduction.
Standard library only.
"""
import sys
from fractions import Fraction

LL_MAX = 2 ** 63 - 1
MAX_DIM = 100
FLAGS = {"-r": "rank", "-d": "det", "-i": "inv"}


class GJError(Exception):
    """Any condition that must end with Error / exit status 1."""


# ---------------------------------------------------------------- arithmetic

def check(x):
    """Emulate long long storage: the reduced fraction must fit."""
    if abs(x.numerator) > LL_MAX or x.denominator > LL_MAX:
        raise GJError("overflow")
    return x


def add(a, b):
    return check(a + b)


def sub(a, b):
    return check(a - b)


def mul(a, b):
    return check(a * b)


def div(a, b):
    if b == 0:
        raise GJError("division by zero")
    return check(a / b)


# ------------------------------------------------------------------- parsing

def is_digits(s):
    return s != "" and all(c in "0123456789" for c in s)


def parse_entry(tok):
    """Signed decimal integer, or integer/positive-integer. Reduced on read."""
    sign = 1
    body = tok
    if body[:1] in ("+", "-"):
        sign = -1 if body[0] == "-" else 1
        body = body[1:]
    if "/" in body:
        p, q = body.split("/", 1)
    else:
        p, q = body, "1"
    if not is_digits(p) or not is_digits(q):
        raise GJError("bad token: " + tok)
    num, den = int(p), int(q)
    if den == 0:
        raise GJError("zero denominator")
    if num > LL_MAX or den > LL_MAX:
        raise GJError("overflow in input")
    return Fraction(sign * num, den)


def parse_header(line):
    toks = line.split()
    if len(toks) != 2 or not is_digits(toks[0]) or not is_digits(toks[1]):
        raise GJError("bad first line")
    m, n = int(toks[0]), int(toks[1])
    if not (1 <= m <= MAX_DIM and 1 <= n <= MAX_DIM):
        raise GJError("dimension out of range")
    return m, n


def parse_matrix(data):
    """data: str. Returns (m, n, rows). Strict on line and entry counts."""
    lines = data.split("\n")
    if lines and lines[-1] == "":
        lines.pop()          # the final newline is optional
    if not lines:
        raise GJError("empty input")
    m, n = parse_header(lines[0])
    if len(lines) != m + 1:
        raise GJError("line count mismatch")
    rows = []
    for line in lines[1:]:
        toks = line.split()  # one or more blanks separate entries
        if len(toks) != n:
            raise GJError("entry count mismatch")
        rows.append([parse_entry(t) for t in toks])
    return m, n, rows


def read_input(path):
    try:
        if path == "-":
            raw = sys.stdin.buffer.read()
        else:
            with open(path, "rb") as f:
                raw = f.read()
    except OSError as exc:
        raise GJError("cannot read file") from exc
    # Any non-ASCII byte is a bad token anyway; decode without failing.
    return raw.decode("latin-1")


# ------------------------------------------------------------------ algebra

def rref(M, ncols):
    """In-place Gauss-Jordan over the first ncols columns of M.

    Columns are scanned left to right. The pivot is the first nonzero
    entry at or below the current pivot row (rows are swapped if needed);
    a column with no such entry is skipped. Returns (pivot_cols, swaps).
    """
    m = len(M)
    width = len(M[0])
    r = 0
    pivots = []
    swaps = 0
    for c in range(ncols):
        if r >= m:
            break
        p = None
        for i in range(r, m):
            if M[i][c] != 0:
                p = i
                break
        if p is None:
            continue
        if p != r:
            M[r], M[p] = M[p], M[r]
            swaps += 1
        piv = M[r][c]
        M[r] = [div(x, piv) if x != 0 else x for x in M[r]]
        for i in range(m):
            if i == r or M[i][c] == 0:
                continue
            f = M[i][c]
            row = M[i]
            prow = M[r]
            M[i] = [sub(row[j], mul(f, prow[j])) if prow[j] != 0 else row[j]
                    for j in range(width)]
        pivots.append(c)
        r += 1
    return pivots, swaps


def determinant(M):
    """Forward elimination: product of pivots, sign from row swaps."""
    n = len(M)
    sign = 1
    d = Fraction(1)
    for c in range(n):
        p = None
        for i in range(c, n):
            if M[i][c] != 0:
                p = i
                break
        if p is None:
            return Fraction(0)
        if p != c:
            M[c], M[p] = M[p], M[c]
            sign = -sign
        piv = M[c][c]
        for i in range(c + 1, n):
            if M[i][c] == 0:
                continue
            f = div(M[i][c], piv)
            row = M[i]
            prow = M[c]
            M[i] = [sub(row[j], mul(f, prow[j])) if prow[j] != 0 else row[j]
                    for j in range(n)]
        d = mul(d, piv)
    return d if sign == 1 else -d


def inverse(M):
    """[A | I] -> [I | A^-1] with the same elimination; None if singular."""
    n = len(M)
    aug = [list(M[i]) + [Fraction(1 if i == j else 0) for j in range(n)]
           for i in range(n)]
    pivots, _ = rref(aug, n)
    if len(pivots) != n:
        return None
    return [row[n:] for row in aug]


# ------------------------------------------------------------------- output

def fmt(x):
    if x.denominator == 1:
        return str(x.numerator)
    return "%d/%d" % (x.numerator, x.denominator)


def fmt_matrix(M):
    return "".join(" ".join(fmt(x) for x in row) + "\n" for row in M)


# --------------------------------------------------------------------- main

def run(argv):
    if len(argv) == 1:
        mode, path = "rref", argv[0]
    elif len(argv) == 2 and argv[0] in FLAGS:
        mode, path = FLAGS[argv[0]], argv[1]
    else:
        raise GJError("usage")
    m, n, rows = parse_matrix(read_input(path))
    if mode in ("det", "inv") and m != n:
        raise GJError("not square")
    if mode == "rref":
        rref(rows, n)
        return fmt_matrix(rows)
    if mode == "rank":
        pivots, _ = rref(rows, n)
        return "%d\n" % len(pivots)
    if mode == "det":
        return fmt(determinant(rows)) + "\n"
    inv = inverse(rows)
    if inv is None:
        return "singular\n"
    return fmt_matrix(inv)


def main():
    try:
        out = run(sys.argv[1:])
    except GJError:
        sys.stderr.write("Error\n")
        return 1
    except Exception:  # never crash: the subject forbids unexpected exits
        sys.stderr.write("Error\n")
        return 1
    sys.stdout.write(out)
    sys.stdout.flush()
    return 0


if __name__ == "__main__":
    sys.exit(main())
