SELECT o.data FROM rag_original o
JOIN rag_document d ON d.id = o.document_id
WHERE d.uuid = $1