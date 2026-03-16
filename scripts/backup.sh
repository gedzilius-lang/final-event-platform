#!/usr/bin/env bash
# NiteOS Postgres backup script — VPS A
# Run via systemd timer (see infra/systemd/) or manually.
#
# Required env:
#   DATABASE_URL        — postgres connection string
#
# Optional env:
#   BACKUP_DIR          — local backup directory (default: /opt/niteos/backups)
#   BACKUP_RETENTION    — number of local backups to keep (default: 14)
#   BACKUP_UPLOAD       — upload backend: rclone | awscli | none (default: none)
#   BACKUP_REMOTE       — upload destination
#                         rclone:  "<remote>:<bucket>/<prefix>"  e.g. "r2:niteos-backups/pg"
#                         awscli:  "s3://<bucket>/<prefix>"      e.g. "s3://niteos-backups/pg"
#
# Exit codes: 0 = success, 1 = fatal error
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/opt/niteos/backups}"
BACKUP_RETENTION="${BACKUP_RETENTION:-14}"
BACKUP_UPLOAD="${BACKUP_UPLOAD:-none}"
BACKUP_REMOTE="${BACKUP_REMOTE:-}"
TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)
BACKUP_FILE="$BACKUP_DIR/niteos_${TIMESTAMP}.sql.gz"

# ── Validate ───────────────────────────────────────────────────────────────────
if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "ERROR: DATABASE_URL is not set" >&2
  exit 1
fi

if [[ "$BACKUP_UPLOAD" != "none" && -z "$BACKUP_REMOTE" ]]; then
  echo "ERROR: BACKUP_REMOTE must be set when BACKUP_UPLOAD=$BACKUP_UPLOAD" >&2
  exit 1
fi

# ── Dump ───────────────────────────────────────────────────────────────────────
mkdir -p "$BACKUP_DIR"
echo "[backup] dumping to $BACKUP_FILE"
pg_dump "${DATABASE_URL}" | gzip > "$BACKUP_FILE"
SIZE=$(du -sh "$BACKUP_FILE" | cut -f1)
echo "[backup] dump complete: $SIZE"

# ── Upload ─────────────────────────────────────────────────────────────────────
case "$BACKUP_UPLOAD" in
  rclone)
    echo "[backup] uploading via rclone to $BACKUP_REMOTE"
    rclone copy "$BACKUP_FILE" "$BACKUP_REMOTE" --checksum
    echo "[backup] upload complete"
    ;;
  awscli)
    echo "[backup] uploading via aws s3 cp to $BACKUP_REMOTE"
    aws s3 cp "$BACKUP_FILE" "${BACKUP_REMOTE}/$(basename "$BACKUP_FILE")" \
      --storage-class STANDARD_IA
    echo "[backup] upload complete"
    ;;
  none)
    echo "[backup] no upload configured (BACKUP_UPLOAD=none)"
    ;;
  *)
    echo "ERROR: unknown BACKUP_UPLOAD backend: $BACKUP_UPLOAD" >&2
    exit 1
    ;;
esac

# ── Prune local ────────────────────────────────────────────────────────────────
KEEP=$(( BACKUP_RETENTION ))
TO_DELETE=$(ls -t "$BACKUP_DIR"/niteos_*.sql.gz 2>/dev/null | tail -n +$(( KEEP + 1 )))
if [[ -n "$TO_DELETE" ]]; then
  echo "[backup] pruning old backups (keeping last $KEEP)"
  echo "$TO_DELETE" | xargs rm -f
fi

echo "[backup] done — $(ls "$BACKUP_DIR"/niteos_*.sql.gz 2>/dev/null | wc -l | tr -d ' ') backups retained locally"
