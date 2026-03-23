SELECT v.uuid, v.content_type, v.mime_type, v.data, v.text, v.embedding, v.metadata,
       s.uuid, s.heading, s.idx,
       d.uuid, d.title, d.source_uri
FROM rag_variant v
JOIN rag_section s ON s.id = v.section_id
JOIN rag_document d ON d.id = s.document_id
WHERE v.uuid = $1