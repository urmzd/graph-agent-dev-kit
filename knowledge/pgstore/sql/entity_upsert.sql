INSERT INTO kg_entity (uuid, name, type, summary, embedding)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (name, type) DO UPDATE
  SET summary = EXCLUDED.summary,
      embedding = COALESCE(EXCLUDED.embedding, kg_entity.embedding)
RETURNING uuid