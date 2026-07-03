#!/usr/bin/env python3
"""CLI-интерфейс парсера PDF-документов."""

import argparse
import logging
import sys
from pathlib import Path

from rich.console import Console
from rich.logging import RichHandler
from rich.progress import Progress, SpinnerColumn, TextColumn

from config import get_settings
from llm_client import LLMError
from parser import PDFParser

console = Console()


def setup_logging(verbose: bool = False):
    """Настройка логирования."""
    level = logging.DEBUG if verbose else logging.INFO
    logging.basicConfig(
        level=level,
        format="%(message)s",
        datefmt="[%X]",
        handlers=[RichHandler(console=console, rich_tracebacks=True)],
    )


def create_parser() -> argparse.ArgumentParser:
    """Создаёт парсер аргументов командной строки."""
    parser = argparse.ArgumentParser(
        description="Парсер PDF-документов с неявной структурой в Markdown",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Примеры:
  %(prog)s document.pdf                    # Базовый режим
  %(prog)s document.pdf -o result.md       # С выходным файлом
  %(prog)s document.pdf --text-only        # Только текст, без изображений
  %(prog)s document.pdf --batch            # Batch-режим (весь документ разом)
  %(prog)s document.pdf -v                 # Подробный вывод
        """,
    )

    parser.add_argument("input", help="Путь к PDF-файлу")
    parser.add_argument("-o", "--output", help="Путь для сохранения markdown (по умолчанию stdout)")
    parser.add_argument("--text-only", action="store_true", help="Не отправлять изображения (быстрее)")
    parser.add_argument("--batch", action="store_true", help="Batch-режим: весь документ одним запросом")
    parser.add_argument("--context", type=int, default=3, help="Окно контекста страниц (по умолчанию: 3)")
    parser.add_argument("-v", "--verbose", action="store_true", help="Подробный вывод")

    return parser


def main():
    """Точка входа CLI."""
    args = create_parser().parse_args()
    setup_logging(args.verbose)

    input_path = Path(args.input)

    # Проверки
    if not input_path.exists():
        console.print(f"[red]Ошибка: файл не найден: {input_path}[/red]")
        sys.exit(1)

    if not input_path.suffix.lower() == ".pdf":
        console.print(f"[red]Ошибка: файл должен быть PDF[/red]")
        sys.exit(1)

    # Проверка конфигурации
    try:
        settings = get_settings()
        if not settings.yandex_api_key:
            console.print(
                "[red]Ошибка: YANDEX_API_KEY не задан.\n"
                "Скопируйте .env.example в .env и укажите ваш API-ключ.[/red]"
            )
            sys.exit(1)
    except Exception as e:
        console.print(f"[red]Ошибка конфигурации: {e}[/red]")
        sys.exit(1)

    # Парсинг
    try:
        with Progress(
            SpinnerColumn(),
            TextColumn("[progress.description]{task.description}"),
            console=console,
            transient=True,
        ) as progress:
            progress.add_task(description="Инициализация парсера...", total=None)
            parser = PDFParser(settings)

            progress.add_task(description="Обработка PDF...", total=None)

            if args.batch:
                result = parser.parse_batch(
                    pdf_path=input_path,
                    use_images=not args.text_only,
                )
            else:
                result = parser.parse(
                    pdf_path=input_path,
                    use_images=not args.text_only,
                    context_window=args.context,
                )

        # Вывод результата
        if args.output:
            output_path = Path(args.output)
            output_path.write_text(result, encoding="utf-8")
            console.print(f"[green]Результат сохранён: {output_path.absolute()}[/green]")
            console.print(f"[dim]{len(result)} символов[/dim]")
        else:
            console.print(result)

    except LLMError as e:
        console.print(f"[red]Ошибка LLM API: {e}[/red]")
        sys.exit(1)
    except KeyboardInterrupt:
        console.print("\n[yellow]Прервано пользователем[/yellow]")
        sys.exit(130)
    except Exception as e:
        console.print(f"[red]Неожиданная ошибка: {e}[/red]")
        if args.verbose:
            console.print_exception()
        sys.exit(1)


if __name__ == "__main__":
    main()
