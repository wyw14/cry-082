FROM postgres:17-alpine
HEALTHCHECK --interval=5s --timeout=3s --start-period=5s --retries=12 \
    CMD pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" || exit 1
