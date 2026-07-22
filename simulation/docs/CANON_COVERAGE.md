# Canon coverage

> **Автогенерация.** Не редактируйте — обновляйте через `python -m scripts.gen_canon_coverage simulation/docs/CANON_COVERAGE.md`.

Скан показывает, какие секции `docs/volnix_protocol.md` встречаются
в коде ядра, аудите и тестах симуляции.

| §X.Y | engine | audit | probes | scenarios | tests | status |
| ---- | :----: | :---: | :----: | :-------: | :---: | ------ |
| §3.1 | 6 | 1 | 1 | 0 | 1 | ✅ covered + tested |
| §3.2 | 0 | 0 | 0 | 0 | 0 | ❌ missing |
| §3.3 | 0 | 0 | 0 | 0 | 0 | ❌ missing |
| §4.1 | 13 | 5 | 3 | 0 | 4 | ✅ covered + tested |
| §4.2 | 15 | 3 | 5 | 0 | 6 | ✅ covered + tested |
| §5.1 | 4 | 2 | 0 | 1 | 1 | ✅ covered + tested |
| §5.2 | 16 | 3 | 7 | 1 | 1 | ✅ covered + tested |
| §5.3 | 0 | 0 | 0 | 0 | 0 | ❌ missing |
| §5.4 | 26 | 2 | 3 | 4 | 3 | ✅ covered + tested |
| §5.5 | 15 | 1 | 5 | 2 | 4 | ✅ covered + tested |
| §6.1 | 9 | 0 | 0 | 2 | 0 | ✅ covered + tested |
| §6.2 | 0 | 0 | 0 | 0 | 0 | ❌ missing |
| §6.3 | 16 | 0 | 0 | 0 | 5 | ✅ covered + tested |

### Дополнительные секции (не в default списке)

- §0: tests=1
- §1: engine=1
- §2: engine=1
- §6: engine=1
- §7.2: engine=3
