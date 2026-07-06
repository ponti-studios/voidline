"""Finance CLI — CSV import pipeline."""

from __future__ import annotations

import csv

import typer

from warehouse.core import AppSettings
from warehouse.core.errors import ConfigError
from warehouse.finance.pipeline import import_file

app = typer.Typer(help="Personal finance tools.")

# Common CSV header aliases for auto-detection
_HEADER_ALIASES = {
    "date": {"date", "posted_on", "transaction_date", "posted date"},
    "name": {"name", "description", "payee", "merchant", "memo"},
    "amount": {"amount", "amount_usd", "value"},
    "account": {"account", "account_name", "account name"},
    "category": {"category", "subcategory"},
    "parent_category": {"parent_category", "parent category", "main category"},
    "type": {"type", "transaction_type", "kind"},
    "note": {"note", "notes", "memo", "description 2"},
    "account_mask": {"account_mask", "account mask", "mask"},
    "tags": {"tags", "labels"},
    "status": {"status", "state"},
    "excluded": {"excluded", "hidden"},
    "recurring": {"recurring", "is_recurring"},
}


def _auto_map(headers: list[str]) -> dict[str, str] | None:
    """Try to auto-detect a column mapping from common CSV headers."""

    lower = [h.strip().lower() for h in headers]
    mapping: dict[str, str] = {}
    for canonical, aliases in _HEADER_ALIASES.items():
        for alias in aliases:
            if alias in lower:
                idx = lower.index(alias)
                mapping[canonical] = headers[idx]
                break

    required = {"date", "name", "amount"}
    if required & mapping.keys() != required:
        return None
    return mapping


def _get_settings() -> AppSettings:
    try:
        s = AppSettings.from_config()
        s.ensure_database()
    except ConfigError as exc:
        typer.echo(f"Error: {exc}", err=True)
        raise typer.Exit(1) from exc
    return s


@app.command("import")
def import_csv(
    csv_path: str = typer.Argument(..., help="Path to the CSV export to import."),
    column_map: str | None = typer.Option(
        None,
        "--map",
        help="Column mapping: date=Date,name=Description,amount=Amt (auto-detected if omitted).",
    ),
    dry_run: bool = typer.Option(False, "--dry-run", "-n", help="Report without writing."),
    since: str | None = typer.Option(
        None,
        "--since",
        help="ISO date (YYYY-MM-DD, inclusive). Only rows on/after this date are merged.",
    ),
) -> None:
    """Import transactions from a CSV file."""

    settings = _get_settings()

    mapping: dict[str, str] | None = None
    if column_map:
        mapping = {}
        for pair in column_map.split(","):
            if "=" not in pair:
                raise typer.BadParameter(f"--map entries must be field=Header, got: {pair!r}")
            field, header = pair.split("=", 1)
            mapping[field.strip()] = header.strip()
    else:
        with open(csv_path, encoding="utf-8-sig", newline="") as f:
            headers = next(csv.reader(f))
        mapping = _auto_map(headers)
        if mapping is None:
            raise typer.BadParameter(
                "Could not auto-detect CSV headers. "
                "Pass --map date=Date,name=Description,amount=Amt to specify column mapping."
            )

    report = import_file(
        settings.database_path,
        csv_path,
        connector_name="generic",
        column_map=mapping,
        dry_run=dry_run,
        since=since,
    )
    typer.echo(report.render())
