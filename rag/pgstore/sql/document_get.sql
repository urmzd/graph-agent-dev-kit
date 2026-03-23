SELECT uuid, source_uri, fingerprint, title, metadata, created_at, updated_at
FROM rag_document WHERE uuid = $1