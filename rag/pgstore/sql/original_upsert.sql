INSERT INTO rag_original (document_id, data) VALUES ($1, $2)
ON CONFLICT (document_id) DO UPDATE SET data = EXCLUDED.data