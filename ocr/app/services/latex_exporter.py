"""Экспорт DoclingDocument в LaTeX с tabularx и поддержкой объединённых ячеек."""

from __future__ import annotations

from pathlib import Path
from typing import Any

import structlog
from docling_core.transforms.serializer.base import (
    BaseTableSerializer,
    SerializationResult,
)
from docling_core.transforms.serializer.common import create_ser_result
from docling_core.transforms.serializer.latex import (
    LaTeXDocSerializer,
    LaTeXParams,
    _escape_latex,
)
from docling_core.types.doc.base import ImageRefMode
from docling_core.types.doc.document import DoclingDocument, RichTableCell, TableItem
from typing_extensions import override

from app.services.table_postprocessor import estimate_column_widths

logger = structlog.get_logger(__name__)

LATEX_PACKAGES = [
    r"\usepackage[utf8]{inputenc}",
    r"\usepackage[T1]{fontenc}",
    r"\usepackage[russian,english]{babel}",
    r"\usepackage{unicode-math}",
    r"\usepackage{amsmath,amssymb}",
    r"\usepackage{graphicx}",
    r"\usepackage{tabularx}",
    r"\usepackage{multirow}",
    r"\usepackage{booktabs}",
    r"\usepackage{array}",
    r"\usepackage{hyperref}",
    r"\usepackage{xcolor}",
    r"\usepackage{microtype}",
    r"\usepackage{float}",
]


class TabularxTableSerializer(BaseTableSerializer):
    """Сериализатор таблиц с tabularx, multicolumn и multirow."""

    @override
    def serialize(
        self,
        *,
        item: TableItem,
        doc_serializer: Any,
        doc: DoclingDocument,
        **kwargs: Any,
    ) -> SerializationResult:
        params = LaTeXParams(**kwargs)
        res_parts: list[SerializationResult] = []

        if item.self_ref in doc_serializer.get_excluded_refs(**kwargs):
            return create_ser_result(text="", span_source=item)

        if params.include_annotations:
            ann_res = doc_serializer.serialize_annotations(item=item, **kwargs)
            if ann_res.text:
                res_parts.append(ann_res)

        table_text = _build_tabularx_table(item, doc_serializer, doc, params)
        cap_res = doc_serializer.serialize_captions(item=item, **kwargs)
        cap_text = cap_res.text

        if table_text or cap_text:
            content: list[str] = ["\\begin{table}[H]"]
            if cap_text:
                content.append(f"\\caption{{{cap_text}}}")
            if table_text:
                content.append(table_text)
            content.append("\\end{table}")
            res_parts.append(
                create_ser_result(text="\n".join(content), span_source=item)
            )

        return create_ser_result(
            text="\n\n".join([r.text for r in res_parts if r.text]),
            span_source=res_parts,
        )


def _cell_text(
    cell: Any, doc_serializer: Any, doc: DoclingDocument, params: LaTeXParams
) -> str:
    if isinstance(cell, RichTableCell):
        text = doc_serializer.serialize(item=cell.ref.resolve(doc=doc), **{}).text
    else:
        raw = cell.text or ""
        text = _escape_latex(raw) if params.escape_latex else raw
    return text.replace("\n", " ").strip()


def _build_tabularx_table(
    item: TableItem,
    doc_serializer: Any,
    doc: DoclingDocument,
    params: LaTeXParams,
) -> str:
    if not item.data or not item.data.grid:
        return ""

    grid = item.data.grid
    num_cols = item.data.num_cols or max(len(row) for row in grid)
    estimate_column_widths(item)
    colspec = (
        "|"
        + "|".join(">{\\raggedright\\arraybackslash}X" for _ in range(num_cols))
        + "|"
    )

    lines = [
        f"\\begin{{tabularx}}{{\\textwidth}}{{{colspec}}}",
        "\\hline",
    ]

    # Отслеживаем занятые ячейки из-за rowspan
    skip_cells: set[tuple[int, int]] = set()

    for row_idx, row in enumerate(grid):
        col_idx = 0
        cells_in_row: list[str] = []

        while col_idx < num_cols:
            if (row_idx, col_idx) in skip_cells:
                col_idx += 1
                continue

            if col_idx >= len(row) or row[col_idx] is None:
                cells_in_row.append("")
                col_idx += 1
                continue

            cell = row[col_idx]
            text = _cell_text(cell, doc_serializer, doc, params)
            col_span = getattr(cell, "col_span", 1) or 1
            row_span = getattr(cell, "row_span", 1) or 1

            # Помечаем ячейки, покрытые rowspan/colspan
            for r in range(row_idx, row_idx + row_span):
                for c in range(col_idx, col_idx + col_span):
                    if r != row_idx or c != col_idx:
                        skip_cells.add((r, c))

            if row_span > 1 and col_span > 1:
                cell_latex = (
                    f"\\multirow{{{row_span}}}{{*}}"
                    f"{{\\multicolumn{{{col_span}}}{{|c|}}{{{text}}}}}"
                )
            elif row_span > 1:
                cell_latex = f"\\multirow{{{row_span}}}{{*}}{{{text}}}"
            elif col_span > 1:
                align = "c"
                cell_latex = f"\\multicolumn{{{col_span}}}{{|{align}|}}{{{text}}}"
            else:
                cell_latex = text

            cells_in_row.append(cell_latex)
            col_idx += col_span

        # Дополняем пустыми ячейками до num_cols
        while len(cells_in_row) < num_cols:
            cells_in_row.append("")

        lines.append(" & ".join(cells_in_row) + r" \\ \hline")

    lines.append("\\end{tabularx}")
    return "\n".join(lines)


def export_to_latex(
    document: DoclingDocument,
    *,
    image_output_dir: Path | None = None,
) -> str:
    """
    Экспортирует DoclingDocument в полноценный LaTeX-документ.

    Использует кастомный TabularxTableSerializer для сложных таблиц
    материаловедческих справочников.
    """
    image_mode = (
        ImageRefMode.REFERENCED if image_output_dir else ImageRefMode.PLACEHOLDER
    )

    params = LaTeXParams(
        image_mode=image_mode,
        page_break_command=None,
        escape_latex=True,
        document_class=r"\documentclass[11pt,a4paper]{article}",
        packages=LATEX_PACKAGES,
    )

    serializer = LaTeXDocSerializer(
        doc=document,
        params=params,
        table_serializer=TabularxTableSerializer(),
    )

    if image_output_dir is not None:
        image_output_dir.mkdir(parents=True, exist_ok=True)
        kwargs: dict[str, Any] = {"image_dir": str(image_output_dir)}
    else:
        kwargs = {}

    result = serializer.serialize(**kwargs)
    latex_content = result.text

    logger.info(
        "latex_exporter.completed",
        length=len(latex_content),
        tables=len(document.tables),
    )
    return latex_content


def export_to_markdown(document: DoclingDocument) -> str:
    """Экспорт в Markdown через встроенный метод Docling."""
    return document.export_to_markdown()
