INSERT INTO kg_episode (uuid, name, body, source, group_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING id