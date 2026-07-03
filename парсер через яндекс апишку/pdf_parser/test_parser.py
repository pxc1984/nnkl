#!/usr/bin/env python3
"""Тестовый скрипт для проверки пайплайна без реального API."""

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))

from pdf_extractor import PDFExtractor


def mock_llm_analyze(text: str, page_num: int) -> str:
    """Имитирует работу LLM для тестирования пайплайна."""
    lines = text.strip().split("\n")

    # Определяем тип страницы
    if page_num == 1:
        return "# Титульный слайд\n\nДоклад о курсе обучения"

    result_lines = []
    in_list = False

    for line in lines:
        stripped = line.strip()
        if not stripped:
            continue

        # Заголовки
        if "курс" in stripped.lower() and len(stripped) < 30:
            result_lines.append(f"## {stripped}")
        elif "модуль" in stripped.lower() and len(stripped) < 40:
            result_lines.append(f"### {stripped}")
        # Списки
        elif stripped.startswith("•") or stripped.startswith("o") or stripped.startswith(""):
            item = stripped[1:].strip()
            result_lines.append(f"- {item}")
            in_list = True
        elif stripped.startswith("-") and in_list:
            result_lines.append(f"  {stripped}")
        else:
            in_list = False
            result_lines.append(stripped)

    return "\n\n".join(result_lines)


def test_extraction_only():
    """Тестирует только извлечение текста."""
    pdf_path = Path("/mnt/agents/upload/Доклад_Вострикова Н.М.pdf")

    print("=" * 60)
    print("ТЕСТ 1: Извлечение текста (текстовый режим)")
    print("=" * 60)

    extractor = PDFExtractor(dpi=150)
    pages = extractor.extract_text_only(pdf_path)

    for page in pages[:3]:  # Первые 3 страницы
        print(f"\n--- Страница {page.page_number} ---")
        print(page.text[:400])
        print("...")

    print(f"\nВсего извлечено страниц: {len(pages)}")
    return pages


def test_with_images():
    """Тестирует извлечение с изображениями."""
    pdf_path = Path("/mnt/agents/upload/Доклад_Вострикова Н.М.pdf")

    print("\n" + "=" * 60)
    print("ТЕСТ 2: Извлечение с изображениями")
    print("=" * 60)

    extractor = PDFExtractor(dpi=100, max_image_size=512)
    pages = extractor.extract(pdf_path)

    for page in pages[:3]:
        img_kb = len(page.image_base64) // 1024 if page.image_base64 else 0
        print(f"Страница {page.page_number}: "
              f"{page.width}x{page.height}px, "
              f"изображение: {img_kb}KB, "
              f"текст: {len(page.text)} символов")

    return pages


def test_full_pipeline_mock():
    """Тестирует полный пайплайн с моком LLM."""
    pdf_path = Path("/mnt/agents/upload/Доклад_Вострикова Н.М.pdf")

    print("\n" + "=" * 60)
    print("ТЕСТ 3: Полный пайплайн (с моком LLM)")
    print("=" * 60)

    extractor = PDFExtractor(dpi=150)
    pages = extractor.extract_text_only(pdf_path)

    results = []
    for page in pages:
        md = mock_llm_analyze(page.text, page.page_number)
        results.append(md)

    full_md = "\n\n---\n\n".join(results)

    output_path = Path("test_output.md")
    output_path.write_text(full_md, encoding="utf-8")

    print(f"\nРезультат сохранён: {output_path.absolute()}")
    print(f"Общий размер: {len(full_md)} символов")
    print("\n--- ПРЕВЬЮ ---\n")
    print(full_md[:1500])
    print("\n...")


if __name__ == "__main__":
    test_extraction_only()
    test_with_images()
    test_full_pipeline_mock()
    print("\n" + "=" * 60)
    print("Все тесты пройдены успешно!")
    print("=" * 60)
