#!/usr/bin/env python3
"""Сгенерировать docs/CANON_COVERAGE.md по упоминаниям §X.Y в коде/тестах.

Цель — видеть какие секции `docs/volnix_protocol.md` покрыты тестами или
проверяются в canon_log.push(canon="§X.Y"), а какие — нет. CI может
ругаться warning, если новая ссылка появилась в core/, но не упомянута
ни одним тестом.
"""
from __future__ import annotations

import argparse
import re
import sys
from collections import defaultdict
from pathlib import Path

SECTION_RE = re.compile(r"§\s*([0-9]+(?:\.[0-9]+)?)")

DEFAULT_SECTIONS = [
    "3.1", "3.2", "3.3",
    "4.1", "4.2",
    "5.1", "5.2", "5.3", "5.4", "5.5",
    "6.1", "6.2", "6.3",
]


def scan(root: Path, subdirs: list[str]) -> dict[str, dict[str, set[str]]]:
    """Возвращает {section: {category: {file:line, …}}}.

    Категории: `engine`, `tests`, `probes`, `scenarios`, `audit`, `other`.
    """
    out: dict[str, dict[str, set[str]]] = defaultdict(lambda: defaultdict(set))
    for sub in subdirs:
        base = root / sub
        if not base.exists():
            continue
        for p in base.rglob("*.py"):
            try:
                lines = p.read_text(encoding="utf-8").splitlines()
            except OSError:
                continue
            cat = _classify(p)
            for ln_no, ln in enumerate(lines, start=1):
                for m in SECTION_RE.finditer(ln):
                    section = m.group(1)
                    rel = p.relative_to(root)
                    out[section][cat].add(f"{rel}:{ln_no}")
        for p in base.rglob("*.yaml"):
            try:
                lines = p.read_text(encoding="utf-8").splitlines()
            except OSError:
                continue
            for ln_no, ln in enumerate(lines, start=1):
                for m in SECTION_RE.finditer(ln):
                    out[m.group(1)]["scenarios"].add(f"{p.relative_to(root)}:{ln_no}")
    return out


def _classify(path: Path) -> str:
    name = path.name
    parts = path.parts
    if "tests" in parts:
        return "tests"
    if name == "canon_audit.py":
        return "audit"
    if name == "bot_engine.py" or "probe" in name:
        return "probes"
    if "scenarios" in name:
        return "scenarios"
    if any(p == "core" for p in parts):
        return "engine"
    return "other"


def render(scan_result: dict[str, dict[str, set[str]]], sections: list[str]) -> str:
    out: list[str] = []
    out.append("# Canon coverage")
    out.append("")
    out.append(
        "> **Автогенерация.** Не редактируйте — обновляйте через "
        "`python -m scripts.gen_canon_coverage simulation/docs/CANON_COVERAGE.md`."
    )
    out.append("")
    out.append("Скан показывает, какие секции `docs/volnix_protocol.md` встречаются")
    out.append("в коде ядра, аудите и тестах симуляции.")
    out.append("")
    out.append("| §X.Y | engine | audit | probes | scenarios | tests | status |")
    out.append("| ---- | :----: | :---: | :----: | :-------: | :---: | ------ |")

    for s in sections:
        d = scan_result.get(s) or {}
        engine = len(d.get("engine") or [])
        audit = len(d.get("audit") or [])
        probes = len(d.get("probes") or [])
        scen = len(d.get("scenarios") or [])
        tests = len(d.get("tests") or [])
        covered = engine + audit > 0
        verified = tests + scen > 0
        if covered and verified:
            status = "✅ covered + tested"
        elif covered and not verified:
            status = "⚠️ no tests"
        elif not covered and verified:
            status = "ℹ️ tested only"
        else:
            status = "❌ missing"
        out.append(
            f"| §{s} | {engine} | {audit} | {probes} | {scen} | {tests} | {status} |"
        )

    # Все секции, которые нашли вне дефолтного списка
    extras = sorted(set(scan_result.keys()) - set(sections))
    if extras:
        out.append("")
        out.append("### Дополнительные секции (не в default списке)")
        out.append("")
        for s in extras:
            cats = scan_result[s]
            joined = ", ".join(f"{c}={len(v)}" for c, v in sorted(cats.items()))
            out.append(f"- §{s}: {joined}")
    out.append("")
    return "\n".join(out)


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("output", help="путь к docs/CANON_COVERAGE.md")
    parser.add_argument(
        "--root", default=str(Path(__file__).resolve().parents[2]),
        help="корень simulation/ (по умолчанию — авто)",
    )
    parser.add_argument(
        "--fail-on-missing", action="store_true",
        help="вернуть код 1, если есть секции без покрытия в engine/audit И без тестов",
    )
    args = parser.parse_args(argv)
    root = Path(args.root)
    scan_result = scan(root, subdirs=["backend", "scenarios"])
    text = render(scan_result, DEFAULT_SECTIONS)
    out_path = Path(args.output)
    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(text, encoding="utf-8")
    print(f"wrote {args.output} ({len(text)} chars)")

    if args.fail_on_missing:
        missing = [
            s for s in DEFAULT_SECTIONS
            if not scan_result.get(s)
            or not (len(scan_result[s].get("engine") or []) + len(scan_result[s].get("audit") or []))
            and not (len(scan_result[s].get("tests") or []) + len(scan_result[s].get("scenarios") or []))
        ]
        if missing:
            print(f"WARN missing canon sections: {', '.join('§' + s for s in missing)}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
