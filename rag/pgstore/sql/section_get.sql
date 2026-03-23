SELECT s.uuid, s.idx, s.heading,
       v.uuid, v.content_type, v.mime_type, v.data, v.text, v.embedding, v.metadata
FROM rag_section s
JOIN rag_document d ON d.id = s.document_id
LEFT JOIN rag_variant v ON v.section_id = s.id
WHERE d.uuid = $1
ORDER BY s.idx, v.id