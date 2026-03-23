package pgstore

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/urmzd/saige/knowledge/types"
)

// CreateEpisode creates an episode and links it to entities via mentions.
func (s *Store) CreateEpisode(ctx context.Context, input *types.EpisodeInput, entityUUIDs []string) (string, error) {
	episodeUUID := uuid.New().String()

	var episodeID int64
	err := s.pool.QueryRow(ctx, episodeCreateSQL,
		episodeUUID, input.Name, input.Body, input.Source, input.GroupID,
	).Scan(&episodeID)
	if err != nil {
		return "", fmt.Errorf("create episode %s: %w", input.Name, err)
	}

	for _, entUUID := range entityUUIDs {
		entID, err := s.entityID(ctx, entUUID)
		if err != nil {
			s.logger.Warn("create mention: entity not found", "uuid", entUUID, "error", err)
			continue
		}
		if _, err := s.pool.Exec(ctx, episodeMentionSQL, episodeID, entID); err != nil {
			s.logger.Warn("create mention failed", "episode", input.Name, "error", err)
		}
	}

	return episodeUUID, nil
}
