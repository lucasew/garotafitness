#!/usr/bin/env python3
"""Translate stream/mpz/guest/decoder.c into Go."""
import re
from pathlib import Path

root = Path(__file__).resolve().parents[1]
src = (root / "guest" / "decoder.c").read_text()
tables = (root / "guest" / "mpz-tables.inc").read_text()
out_dir = root

body = tables.split("{", 1)[1].split("}", 1)[0]
nums = [int(n) for n in re.findall(r"\d+", body)]
if len(nums) != 0x15000:
    raise SystemExit(f"tables {len(nums)}")
with (out_dir / "tables.go").open("w") as f:
    f.write("package mpz\n\n// tables is the MPZ constant window from guest/mpz-tables.inc.\nvar tables = [0x15000]byte{\n")
    for i in range(0, len(nums), 16):
        chunk = ",".join(str(n) for n in nums[i : i + 16])
        f.write(chunk + ",\n")
    f.write("}\n")

regs = r"(?<!\.)\b(eax|ebx|ecx|edx|esi|edi|ebp|esp)\b"
imul_re = re.compile(
    r"\((uint32_t|uint64_t)\)\(\(int64_t\)\(int32_t\)([a-z]+)\*\(int32_t\)"
    r"([a-z]+|\d+u|rd32\(s,\([^)]*\)\))\)"
)

def go_ty(name: str) -> str:
    return name[:-2]

def imull_operand(op: str) -> str:
    if re.fullmatch(r"[a-z]+", op):
        return f"int64(int32(s.{op}))"
    if re.fullmatch(r"\d+u", op):
        return f"int64(int32({op[:-1]}))"
    return f"int64(int32({op}))"

def rewrite_imull(m: re.Match) -> str:
    prod = f"{imull_operand(m.group(2))}*{imull_operand(m.group(3))}"
    if m.group(1) == "uint64_t":
        return f"uint64({prod})"
    return f"uint32({prod})"

TYPE_GO = (
    ("uint32_t", "uint32"),
    ("int32_t", "int32"),
    ("uint64_t", "uint64"),
    ("int64_t", "int64"),
    ("int16_t", "int16"),
)

def parse_cast(s: str, i: int):
    if i >= len(s) or s[i] != "(":
        return None
    for cty, gty in TYPE_GO:
        token = "(" + cty + ")"
        if s.startswith(token, i):
            operand, j = parse_cast_operand(s, i + len(token))
            return f"{gty}({operand})", j
    return None

def parse_cast_operand(s: str, i: int):
    cast = parse_cast(s, i)
    if cast:
        return cast
    if i < len(s) and s[i] == "(":
        inner, j = take_balanced(s, i)
        return rewrite_casts(inner), j
    m = re.match(r"0x[0-9A-Fa-f]+u?|\d+u?|[A-Za-z_][A-Za-z0-9_]*", s[i:])
    if not m:
        raise SystemExit("cast operand: " + s[max(0, i - 20) : i + 60])
    return m.group(0), i + len(m.group(0))

def take_balanced(s: str, i: int):
    depth = 0
    for j in range(i, len(s)):
        if s[j] == "(":
            depth += 1
        elif s[j] == ")":
            depth -= 1
            if depth == 0:
                return s[i + 1 : j], j + 1
    raise SystemExit("unbalanced: " + s[i : i + 80])

def rewrite_casts(code: str) -> str:
    out = []
    i = 0
    while i < len(code):
        if code[i] == "(":
            cast = parse_cast(code, i)
            if cast:
                expr, i = cast
                out.append(expr)
                continue
        out.append(code[i])
        i += 1
    rewritten = "".join(out)
    if re.search(r"\b(?:uint32_t|int32_t|uint64_t|int64_t|int16_t)\b", rewritten):
        raise SystemExit("remaining cast: " + rewritten)
    return rewritten

def go_cond(cond: str) -> str:
    cond = cond.replace("!s.zf", "(s.zf==0)")
    cond = cond.replace("!s.cf", "(s.cf==0)")
    cond = re.sub(r"s\.zf(?!\s*[=!])", "(s.zf!=0)", cond)
    cond = re.sub(r"s\.cf(?!\s*[=!])", "(s.cf!=0)", cond)
    return cond

def transform(code: str) -> str:
    code = re.sub(r"^TRACE\([^;]*\);\s*", "", code)
    if "?" in code:
        raise SystemExit("unexpected token in " + code)
    code = code.replace("~", "^")
    code = imul_re.sub(rewrite_imull, code)
    code = re.sub(
        r"q=\(uint64_t\)([a-z]+)\*([a-z]+)",
        lambda m: f"q=uint64(s.{m.group(1)})*uint64(s.{m.group(2)})",
        code,
    )
    code = re.sub(r"\(void\)\s*alu\(s,", "_ = s.alu(", code)
    code = re.sub(r"\(void\)\s*", "", code)
    for fn in (
        "rd8",
        "rd16",
        "rd32",
        "wr8",
        "wr16",
        "wr32",
        "push",
        "alu",
        "shift",
        "shrd",
        "divide",
        "callback",
    ):
        code = code.replace(f"{fn}(s,", f"s.{fn}(")
    code = code.replace("pop(s)", "s.pop()")
    code = rewrite_casts(code)
    code = code.replace("s->", "s.")
    code = re.sub(r"f_([0-9a-fA-F]+)\(", r"f\1(", code)
    code = re.sub(regs, lambda m: "s." + m.group(1), code)
    code = re.sub(r"\b(0x[0-9a-fA-F]+)u\b", r"\1", code)
    code = re.sub(r"\b(\d+)u\b", r"\1", code)
    code = re.sub(r"\bs\.(eax|edx)=q>>32", r"s.\1=uint32(q>>32)", code)
    m = re.match(r"if\((.*)\)goto (L[0-9a-fA-F]+);$", code)
    if m:
        return f"if {go_cond(m.group(1))} {{ goto {m.group(2)} }}"
    # setcc stores a 0/1 condition into a byte. C uses the comparison as an int.
    code = code.replace("(!s.zf&&s.sf==s.of)", "flag(s.zf==0&&s.sf==s.of)")
    code = code.replace("(!s.zf)", "flag(s.zf==0)")
    code = code.replace("(s.sf!=s.of)", "flag(s.sf!=s.of)")
    code = code.replace("(s.sf==s.of)", "flag(s.sf==s.of)")
    if not code.endswith(";") and code != "":
        raise SystemExit("no semicolon: " + code)
    return code

func_start = re.compile(
    r"^static void (f_[0-9a-fA-F]+)\(State \*s\)\{uint32_t t; uint64_t q; goto (L[0-9a-fA-F]+);$"
)
label_line = re.compile(r"^(L[0-9a-fA-F]+):\s*/\*.*?\*/\s*(.*)$")
proto = re.compile(r"^static void f_[0-9a-fA-F]+\(State \*s\);$")

functions = []
cur = None
for raw in src.splitlines():
    line = raw.strip()
    if not line or line.startswith("/*") or line.startswith("#include") or line.startswith("static void callback"):
        continue
    if proto.match(line):
        continue
    if line in {"case 0x10018e24: f_10018e24(s); return;", "case 0x10018e10: f_10018e10(s); return;", "default: hostcall(s,target); }}"}:
        continue
    m = func_start.match(line)
    if m:
        if cur:
            raise SystemExit("unclosed " + cur[0])
        cur = [m.group(1), m.group(2), []]
        continue
    if line == "}":
        if not cur:
            raise SystemExit("extra brace")
        functions.append(cur)
        cur = None
        continue
    m = label_line.match(line)
    if not m or cur is None:
        raise SystemExit("odd line: " + line)
    cur[2].append((m.group(1), transform(m.group(2))))

if cur:
    raise SystemExit("eof in " + cur[0])
print(f"functions {len(functions)}")

lift = out_dir / "lift.go"
with lift.open("w") as f:
    f.write("package mpz\n\n// Code generated by guest/c2go.py from guest/decoder.c. DO NOT EDIT.\n\n")
    for name, entry, stmts in functions:
        goname = "f" + name[2:]
        body_txt = "\n".join(code for _, code in stmts)
        uses_t = re.search(r"\bt\b", body_txt) is not None
        uses_q = re.search(r"\bq\b", body_txt) is not None
        f.write(f"func {goname}(s *state) {{\n")
        if uses_t:
            f.write("var t uint32\n")
        if uses_q:
            f.write("var q uint64\n")
        used = set(re.findall(r"\bgoto (L[0-9a-fA-F]+)", body_txt))
        used.add(entry)
        # Go rejects an unused label. A goto hidden under `if false` keeps the
        # label but is unreachable, and `go vet` fails the build on it. Labels
        # that nothing jumps to are only reached by fallthrough, so omit them.
        f.write(f"goto {entry}\n")
        for lab, code in stmts:
            if lab in used:
                f.write(f"{lab}:\n")
            f.write(f"{code}\n")
        f.write("}\n\n")
print("wrote", lift, lift.stat().st_size)
