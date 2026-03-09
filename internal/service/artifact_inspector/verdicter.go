package artifact_inspector

import (
	"context"
	"fmt"
	"lazarus/internal/entities"
)

func (s *Service) markArtifactClean(ctx context.Context, artifact *entities.Artifact) error {
	return s.repo.UpdateArtifactStatus(ctx, artifact.ID, entities.ArtifactStatusClean)
}

func (s *Service) purgeArtifact(ctx context.Context, artifact *entities.Artifact) error {
	if err := s.repo.UpdateArtifactStatus(ctx, artifact.ID, entities.ArtifactStatusRejected); err != nil {
		return fmt.Errorf("error purging artifact %s: %w", artifact.ID.String(), err)
	}
	if err := s.bucketClient.Delete(ctx, artifact.ObjectKey); err != nil {
		return fmt.Errorf("error deleting artifact %s from bucket: %w", artifact.ID.String(), err)
	}
	return nil
}
