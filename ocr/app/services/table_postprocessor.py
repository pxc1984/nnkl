"""Conservative cleanup for MinerU Markdown and table cells."""

from __future__ import annotations

import re

_UNIT_REPLACEMENTS = (
    (r"\b[Мм][пП][аА]\b", "МПа"),
    (r"\bкгс/мм(?:\^?2|²)\b", "кгс/мм²"),
    (r"\bкгс/см(?:\^?2|²)\b", "кгс/см²"),
)
_NUMERIC_THOUSANDS_RE = re.compile(r"(?<=\d)[ \u00a0](?=\d{3}(?:\D|$))")
_DECIMAL_COMMA_RE = re.compile(r"(?<=\d),(?=\d)")
_HTML_CELL_RE = re.compile(r"(<t[dh]\b[^>]*>)(.*?)(</t[dh]>)", re.I | re.S)
_MARKDOWN_SEPARATOR_RE = re.compile(r"^\s*\|?(?:\s*:?-{3,}:?\s*\|)+\s*$")
_EXCESS_NEWLINES_RE = re.compile(r"\n{3,}")


def normalize_unit_text(text: str) -> str:
    result = text
    for pattern, replacement in _UNIT_REPLACEMENTS:
        result = re.sub(pattern, replacement, result, flags=re.IGNORECASE)
    return result


def normalize_numeric_text(text: str) -> str:
    result = _NUMERIC_THOUSANDS_RE.sub("", text)
    return _DECIMAL_COMMA_RE.sub(".", result)


def postprocess_table_cell(text: str) -> str:
    cleaned = " ".join(text.split())
    cleaned = normalize_numeric_text(cleaned)
    return normalize_unit_text(cleaned)


def _process_markdown_table_line(line: str) -> str:
    if "|" not in line or _MARKDOWN_SEPARATOR_RE.match(line):
        return line
    leading = "|" if line.lstrip().startswith("|") else ""
    trailing = "|" if line.rstrip().endswith("|") else ""
    cells = line.strip().strip("|").split("|")
    processed = [postprocess_table_cell(cell) for cell in cells]
    return f"{leading} " + " | ".join(processed) + f" {trailing}".rstrip()


def clean_markdown(markdown: str) -> str:
    """Removes common OCR artifacts without changing normal punctuation."""
    cleaned = markdown.replace("\x00", "").replace("\u00ad", "")
    cleaned = cleaned.replace("\r\n", "\n").replace("\r", "\n")
    cleaned = re.sub(r"[ \t]+\n", "\n", cleaned)
    cleaned = _EXCESS_NEWLINES_RE.sub("\n\n", cleaned)
    return cleaned.strip()


def postprocess_markdown_tables(markdown: str) -> str:
    if not markdown:
        return markdown

    cleaned = clean_markdown(markdown)

    def replace_html_cell(match: re.Match[str]) -> str:
        return f"{match.group(1)}{postprocess_table_cell(match.group(2))}{match.group(3)}"

    cleaned = _HTML_CELL_RE.sub(replace_html_cell, cleaned)

    lines = cleaned.splitlines()
    table_line_indexes = {
        index
        for index, line in enumerate(lines)
        if _MARKDOWN_SEPARATOR_RE.match(line)
    }
    for separator_index in table_line_indexes:
        for index in (separator_index - 1, *range(separator_index + 1, len(lines))):
            if index < 0 or index >= len(lines) or "|" not in lines[index]:
                break
            lines[index] = _process_markdown_table_line(lines[index])

    return "\n".join(lines).strip()