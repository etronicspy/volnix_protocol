#!/usr/bin/env python3
"""Сгенерировать simulation/docs/API.md из FastAPI OpenAPI-схемы.

Используется в CI, чтобы документация не отставала от кода.
"""
from __future__ import annotations

import argparse
import sys
from pathlib import Path


def render(spec: dict) -> str:
    out: list[str] = []
    info = spec.get("info") or {}
    title = info.get("title") or "Simulation API"
    version = info.get("version") or "?"
    out.append(f"# {title} (v{version})")
    out.append("")
    out.append("> **Автогенерация.** Не редактируйте файл руками — обновляйте через")
    out.append("> `python -m scripts.gen_api_md simulation/docs/API.md` (см. CI).")
    out.append("")

    paths = spec.get("paths") or {}
    sections: dict[str, list[tuple[str, str, dict]]] = {}
    for path, methods in paths.items():
        for method, op in (methods or {}).items():
            if method.upper() not in {"GET", "POST", "PUT", "DELETE", "PATCH"}:
                continue
            tag = (op.get("tags") or ["misc"])[0]
            sections.setdefault(tag, []).append((path, method.upper(), op))

    if not sections:
        # Группируем по префиксу пути, чтобы оглавление было полезным
        for path, methods in paths.items():
            head = "/" + (path.strip("/").split("/")[0] if path.strip("/") else "root")
            for method, op in (methods or {}).items():
                if method.upper() not in {"GET", "POST", "PUT", "DELETE", "PATCH"}:
                    continue
                sections.setdefault(head, []).append((path, method.upper(), op))

    for tag in sorted(sections):
        out.append(f"## `{tag}`")
        out.append("")
        for path, method, op in sorted(sections[tag], key=lambda x: (x[0], x[1])):
            summary = op.get("summary") or op.get("operationId") or ""
            description = (op.get("description") or "").strip()
            out.append(f"### `{method} {path}` — {summary}")
            if description:
                out.append("")
                out.append(description)
            params = op.get("parameters") or []
            if params:
                out.append("")
                out.append("**Параметры:**")
                out.append("")
                out.append("| name | in | type | required | default | description |")
                out.append("| ---- | -- | ---- | -------- | ------- | ----------- |")
                for p in params:
                    name = p.get("name", "")
                    pin = p.get("in", "")
                    sch = p.get("schema") or {}
                    ptype = sch.get("type") or (
                        " | ".join(t.get("type", "?") for t in (sch.get("anyOf") or []))
                        or "?"
                    )
                    req = "✓" if p.get("required") else ""
                    default = sch.get("default", "")
                    desc = p.get("description", "")
                    out.append(f"| `{name}` | {pin} | {ptype} | {req} | {default} | {desc} |")
            body = (op.get("requestBody") or {}).get("content") or {}
            if body:
                out.append("")
                out.append("**Тело запроса:**")
                for media, body_spec in body.items():
                    schema_ref = (body_spec.get("schema") or {}).get("$ref") or ""
                    inline = body_spec.get("schema") or {}
                    out.append(f"- `{media}` — {schema_ref or inline.get('type', 'object')}")
            out.append("")
    return "\n".join(out) + "\n"


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("output", help="путь к docs/API.md")
    args = parser.parse_args(argv)
    # Импорт после парсинга — чтобы --help работал без подгрузки FastAPI
    sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
    import main as backend_main  # noqa: WPS433
    spec = backend_main.app.openapi()
    text = render(spec)
    Path(args.output).parent.mkdir(parents=True, exist_ok=True)
    Path(args.output).write_text(text, encoding="utf-8")
    print(f"wrote {args.output} ({len(text)} chars)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1:]))
