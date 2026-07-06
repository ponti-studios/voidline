"""Calendar import pipeline — ICS ingestion and occurrence expansion."""

from __future__ import annotations

import json
import sqlite3
from datetime import date, datetime, timezone
from pathlib import Path
from typing import Any

from warehouse.calendar import RawEvent, _infer_source_system, parse_ics_file


def import_ics_files(
    db_path: str,
    import_dir: Path,
    *,
    source_system: str | None = None,
    future_years: int = 2,
    past_years: int = 1,
) -> dict:
    """Import ICS files from a directory into the warehouse database.

    Returns a summary dict with keys:
        batch_id, file_count, event_count, warning_count, error_count,
        errors (list of (path, message) tuples)
    """
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row

    import_batch_id = f"import-{datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S')}"

    # Find all ICS files
    ics_files = sorted(import_dir.rglob("*.ics"))
    if not ics_files:
        conn.close()
        return {
            "batch_id": import_batch_id,
            "file_count": 0,
            "event_count": 0,
            "warning_count": 0,
            "error_count": 0,
            "errors": [],
        }

    total_events = 0
    total_warnings = 0
    errors: list[tuple[str, str]] = []

    for file_path in ics_files:
        inferred = source_system or _infer_source_system(None, file_path)

        try:
            events = parse_ics_file(file_path, import_batch_id, inferred)
        except Exception as exc:
            errors.append((str(file_path), str(exc)))
            continue

        event_count = len(events)
        for event in events:
            _insert_raw_event(conn, event)

        file_warnings = sum(1 for e in events if e.parse_warnings_json is not None)
        total_events += event_count
        total_warnings += file_warnings

    # Record the batch
    conn.execute(
        """
        INSERT INTO cal_import_batches
            (id, imported_at, source_paths_json,
             file_count, event_count, warning_count, error_count)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        """,
        (
            import_batch_id,
            datetime.now(timezone.utc).isoformat(),
            json.dumps([str(p) for p in ics_files]),
            len(ics_files),
            total_events,
            total_warnings,
            len(errors),
        ),
    )

    # Expand occurrences
    today = date.today()
    expand_from = today.replace(year=today.year - past_years)
    expand_to = today.replace(year=today.year + future_years)
    occ_count = _expand_occurrences(conn, expand_from, expand_to)

    conn.commit()
    conn.close()

    return {
        "batch_id": import_batch_id,
        "file_count": len(ics_files),
        "event_count": total_events,
        "warning_count": total_warnings,
        "error_count": len(errors),
        "occurrence_count": occ_count,
        "expand_window": f"{expand_from} → {expand_to}",
        "errors": errors,
    }


def _insert_raw_event(conn: sqlite3.Connection, event: RawEvent) -> int:
    """Insert a parsed RawEvent into calendar_events_raw."""
    conn.execute(
        """
        INSERT OR REPLACE INTO calendar_events_raw (
            import_batch_id, source_system, source_file, source_path, source_hash,
            uid, recurrence_id_raw, recurrence_id_utc, sequence,
            dtstamp_utc, created_utc, last_modified_utc,
            calendar_name, prodid, method, event_type, classification, status, transp,
            summary, description, location,
            dtstart_raw, dtstart_tzid, dtstart_utc, dtstart_kind,
            dtend_raw, dtend_tzid, dtend_utc, dtend_kind,
            duration_raw, all_day,
            rrule_raw, exdate_raw, rdate_raw, exrule_raw, tzid,
            organizer, attendees_json, categories_json, url,
            raw, parse_warnings_json, parse_error, ingested_at
        ) VALUES (
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
            ?, ?, ?, ?, ?, ?, ?
        )
        """,
        (
            event.import_batch_id,
            event.source_system,
            event.source_file,
            event.source_path,
            event.source_hash,
            event.uid,
            event.recurrence_id_raw,
            event.recurrence_id_utc,
            event.sequence,
            event.dtstamp_utc,
            event.created_utc,
            event.last_modified_utc,
            event.calendar_name,
            event.prodid,
            event.method,
            event.event_type,
            event.classification,
            event.status,
            event.transp,
            event.summary,
            event.description,
            event.location,
            event.dtstart_raw,
            event.dtstart_tzid,
            event.dtstart_utc,
            event.dtstart_kind,
            event.dtend_raw,
            event.dtend_tzid,
            event.dtend_utc,
            event.dtend_kind,
            event.duration_raw,
            1 if event.all_day else 0,
            event.rrule_raw,
            event.exdate_raw,
            event.rdate_raw,
            event.exrule_raw,
            event.tzid,
            event.organizer,
            event.attendees_json,
            event.categories_json,
            event.url,
            event.raw,
            event.parse_warnings_json,
            event.parse_error,
            event.ingested_at,
        ),
    )
    return conn.execute("SELECT last_insert_rowid()").fetchone()[0]


def _expand_occurrences(
    conn: sqlite3.Connection,
    window_start: date,
    window_end: date,
) -> int:
    """Expand all raw events into occurrences within the given window."""
    from dateutil import parser as dateutil_parser

    rows = conn.execute(
        """
        SELECT id, uid, dtstart_utc, dtstart_raw, dtstart_tzid, dtend_utc,
               all_day, status, summary, description, location,
               recurrence_id_raw, recurrence_id_utc, rrule_raw
        FROM calendar_events_raw
        """
    ).fetchall()

    version = datetime.now(timezone.utc).strftime("%Y%m%d%H%M%S")
    expanded_at = datetime.now(timezone.utc).isoformat()
    occ_count = 0

    for row in rows:
        raw_id = row["id"]
        dtstart_utc = row["dtstart_utc"]
        if not dtstart_utc:
            continue

        try:
            start_dt = dateutil_parser.isoparse(dtstart_utc)
        except (ValueError, OverflowError):
            continue

        # Skip events outside the window
        start_date = start_dt.date()
        if start_date < window_start or start_date > window_end:
            continue

        dtend_utc_val = row["dtend_utc"]
        end_dt = None
        if dtend_utc_val:
            try:
                end_dt = dateutil_parser.isoparse(dtend_utc_val)
            except (ValueError, OverflowError):
                pass

        all_day = bool(row["all_day"])
        occ_date = start_dt.strftime("%Y-%m-%d") if all_day else None

        occurrence_key = start_dt.isoformat()
        if end_dt:
            occurrence_key += ":" + end_dt.isoformat()

        conn.execute(
            """
            INSERT OR REPLACE INTO calendar_event_occurrences (
                raw_event_id, uid, occurrence_key, recurrence_id_utc,
                occurrence_start_utc, occurrence_end_utc, occurrence_date,
                is_all_day, is_generated, is_override, is_cancelled, is_excluded,
                status, summary, description, location,
                expansion_window_start, expansion_window_end, expanded_at, expansion_version
            ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
            """,
            (
                raw_id,
                row["uid"],
                occurrence_key,
                row["recurrence_id_utc"],
                dtstart_utc,
                end_dt.isoformat() if end_dt else None,
                occ_date,
                1 if all_day else 0,
                1 if row["rrule_raw"] else 0,
                1 if row["recurrence_id_raw"] else 0,
                1 if row["status"] == "CANCELLED" else 0,
                0,
                row["status"],
                row["summary"],
                row["description"],
                row["location"],
                window_start.isoformat(),
                window_end.isoformat(),
                expanded_at,
                version,
            ),
        )
        occ_count += 1

    return occ_count


def query_occurrences(
    db_path: str,
    text: str,
    *,
    from_date: str | None = None,
    to_date: str | None = None,
    limit: int = 50,
) -> list[dict]:
    """Search calendar occurrences by text."""
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row

    search = f"%{text.lower()}%"
    sql = """
        SELECT o.uid, o.occurrence_start_utc, o.occurrence_date,
               o.summary, o.is_cancelled,
               r.source_file, r.source_system
        FROM calendar_event_occurrences o
        JOIN calendar_events_raw r ON r.id = o.raw_event_id
        WHERE (LOWER(o.summary) LIKE ? OR LOWER(o.description) LIKE ? OR LOWER(o.location) LIKE ?)
    """
    params: list[Any] = [search, search, search]

    if from_date:
        sql += " AND o.occurrence_start_utc >= ?"
        params.append(from_date)
    if to_date:
        sql += " AND o.occurrence_start_utc <= ?"
        params.append(to_date)

    sql += " ORDER BY o.occurrence_start_utc LIMIT ?"
    params.append(limit)

    rows = conn.execute(sql, params).fetchall()
    conn.close()

    return [
        {
            "uid": r["uid"],
            "start_utc": r["occurrence_start_utc"],
            "date": r["occurrence_date"],
            "summary": r["summary"],
            "is_cancelled": bool(r["is_cancelled"]),
            "source_file": r["source_file"],
            "source_system": r["source_system"],
        }
        for r in rows
    ]


def get_calendar_stats(db_path: str) -> dict:
    """Get calendar import statistics."""
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row

    stats: dict[str, Any] = {}

    stats["raw_events"] = conn.execute("SELECT COUNT(*) FROM calendar_events_raw").fetchone()[0]

    stats["occurrences"] = conn.execute(
        "SELECT COUNT(*) FROM calendar_event_occurrences"
    ).fetchone()[0]

    stats["batches"] = conn.execute("SELECT COUNT(*) FROM cal_import_batches").fetchone()[0]

    # By source system
    rows = conn.execute(
        "SELECT source_system, COUNT(*) as cnt"
        " FROM calendar_events_raw"
        " GROUP BY source_system ORDER BY cnt DESC"
    ).fetchall()
    stats["by_source_system"] = {r["source_system"]: r["cnt"] for r in rows}

    # Recurring
    stats["recurring"] = conn.execute(
        "SELECT COUNT(*) FROM calendar_events_raw WHERE rrule_raw IS NOT NULL"
    ).fetchone()[0]

    # Cancelled
    stats["cancelled"] = conn.execute(
        "SELECT COUNT(*) FROM calendar_event_occurrences WHERE is_cancelled = 1"
    ).fetchone()[0]

    conn.close()
    return stats


def run_calendar_doctor(db_path: str) -> list[dict]:
    """Run data-quality checks on calendar data."""
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row

    findings: list[dict] = []

    total_raw = conn.execute("SELECT COUNT(*) FROM calendar_events_raw").fetchone()[0]
    if total_raw == 0:
        findings.append(
            {"severity": "info", "check": "no_data", "detail": "No calendar events imported yet"}
        )

    missing_uid = conn.execute(
        "SELECT COUNT(*) FROM calendar_events_raw WHERE uid LIKE 'MISSING-UID-%'"
    ).fetchone()[0]
    if missing_uid:
        findings.append(
            {
                "severity": "warn",
                "check": "missing_uid",
                "count": missing_uid,
                "detail": "Events with auto-generated UIDs",
            }
        )

    no_dtstart = conn.execute(
        "SELECT COUNT(*) FROM calendar_events_raw WHERE dtstart_utc IS NULL AND dtstart_raw IS NULL"
    ).fetchone()[0]
    if no_dtstart:
        findings.append(
            {
                "severity": "warn",
                "check": "no_dtstart",
                "count": no_dtstart,
                "detail": "Events with no start time",
            }
        )

    orphaned = conn.execute(
        """SELECT COUNT(*) FROM calendar_event_occurrences o
           WHERE NOT EXISTS (SELECT 1 FROM calendar_events_raw r WHERE r.id = o.raw_event_id)"""
    ).fetchone()[0]
    if orphaned:
        findings.append(
            {
                "severity": "error",
                "check": "orphaned_occurrences",
                "count": orphaned,
                "detail": "Occurrences with no parent raw event",
            }
        )

    duplicates = conn.execute(
        """SELECT COUNT(*) FROM (
            SELECT uid, occurrence_key, COUNT(*) as cnt
            FROM calendar_event_occurrences
            GROUP BY uid, occurrence_key HAVING cnt > 1
        )"""
    ).fetchone()[0]
    if duplicates:
        findings.append(
            {
                "severity": "warn",
                "check": "duplicate_keys",
                "count": duplicates,
                "detail": "Duplicate occurrence keys",
            }
        )

    conn.close()
    return findings
