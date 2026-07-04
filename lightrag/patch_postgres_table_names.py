from pathlib import Path

path = Path("/usr/local/lib/python3.12/site-packages/lightrag/kg/postgres_impl.py")
text = path.read_text(encoding="utf-8")
old = """        # Get base table name
        base_table = namespace_to_table_name(self.namespace)
        if not base_table:
            raise ValueError(f\"Unknown namespace: {self.namespace}\")

        # New table name (with suffix)
"""
new = """        # Get base table name
        base_table = namespace_to_table_name(self.namespace)
        if not base_table:
            raise ValueError(f\"Unknown namespace: {self.namespace}\")

        # Keep provider URIs usable as model names without exceeding PostgreSQL's
        # 63-character identifier limit. Preserve a hash tail to avoid collisions.
        max_suffix_len = PG_MAX_IDENTIFIER_LENGTH - len(base_table) - 1 - len("_PK")
        if len(self.model_suffix) > max_suffix_len:
            suffix_hash = hashlib.sha1(self.model_suffix.encode()).hexdigest()[:8]
            self.model_suffix = f\"{self.model_suffix[:max_suffix_len - 9]}_{suffix_hash}\"

        # New table name (with suffix)
"""
if old not in text:
    raise SystemExit("LightRAG PostgreSQL patch target was not found")
path.write_text(text.replace(old, new, 1), encoding="utf-8")