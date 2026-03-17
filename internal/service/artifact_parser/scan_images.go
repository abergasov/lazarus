package artifact_parser

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"lazarus/internal/entities"
	"lazarus/internal/entities/prompts"
	"lazarus/internal/utils"
	"strings"
	"time"
)

// scanImages load images from bucket and send them to llm vision provider
// after that try to decode object results and save to db
func (s *Service) scanImages(ctx context.Context, artifact *entities.Artifact) error {
	imgScanProvider, model, err := s.registry.GetForRole(entities.AgentRoleVision)
	if err != nil {
		return fmt.Errorf("get image scan provider: %w", err)
	}
	artifactDerivatives, err := s.repo.GetAllDerivativesForArtifact(ctx, artifact.ID)
	if err != nil {
		return fmt.Errorf("get all derivatives for artifact: %w", err)
	}

	imagesContent := make([][]byte, 0, len(artifactDerivatives))
	for _, derivative := range artifactDerivatives {
		imageContent, errC := s.bucketClient.DownloadBytes(ctx, derivative.ObjectKey)
		if errC != nil {
			return fmt.Errorf("download image content: %w", errC)
		}
		imagesContent = append(imagesContent, imageContent)
	}
	provImages := make([]*entities.AgentRequestImage, 0, len(artifactDerivatives))
	for i := range imagesContent {
		provImages = append(provImages, &entities.AgentRequestImage{
			Data:     imagesContent[i],
			MimeType: artifactDerivatives[i].DetectedMIME,
		})
	}
	req := &entities.AgentRequest{
		Model:        model,
		SystemPrompt: prompts.ParseDocumentSystemPrompt,
		Messages: []*entities.AgentRequestMessage{
			{
				Role:    entities.RoleUser,
				Content: "Extract structured data from this document. JSON only, no commentary.",
			},
		},
		Images:      provImages,
		MaxTokens:   4096,
		Temperature: 0,
	}
	ch, err := imgScanProvider.Stream(ctx, req)
	if err != nil {
		return fmt.Errorf("error stream images: %w", err)
	}
	var sb strings.Builder
	for ev := range ch {
		if ev.Type == entities.EventTypeText {
			sb.WriteString(ev.Text)
		}
		if ev.Type == entities.EventTypeError && ev.Error != nil {
			return fmt.Errorf("error streaming images: %w", ev.Error)
		}
	}
	raw := utils.SanitizeResponseJSON(sb.String())
	var data entities.ArtifactParsedData
	if err = json.Unmarshal([]byte(raw), &data); err != nil {
		// todo resend to rebuild response?
		// often tiny mismatch because llm not follow response rules strictly, so try to fix it and decode again
		return fmt.Errorf("error unmarshalling llm response json: %w", err)
	}

	// Ensure that we actually extracted some structured data; otherwise, signal failure
	if len(data.LabResults) == 0 && len(data.Medications) == 0 { //nolint:staticcheck // it needs to be resolved
		// this check is nice to have,
		// but it can trigger drain situation when wrong or malicious document will not have any result
		// todo detect is it bad document or parse error return fmt.Errorf("no structured data extracted from images")
	}
	s.saveLabResults(ctx, artifact, &data)
	s.saveMedications(ctx, artifact, &data)
	return nil
}

func (s *Service) saveLabResults(ctx context.Context, artifact *entities.Artifact, data *entities.ArtifactParsedData) {
	docDate := time.Now()
	if data.Date != "" {
		if t, errD := utils.ExtractDateFromString(data.Date); errD == nil {
			docDate = t
		}
	}

	for _, l := range data.LabResults {
		collectedAt := docDate
		if l.Date != "" {
			if t, errD := utils.ExtractDateFromString(l.Date); errD == nil {
				collectedAt = t
			}
		}
		flag := strings.ToLower(l.Flag)
		if flag == "" {
			flag = "normal"
		}
		lab := &entities.LabResult{
			UserID:     artifact.OwnerID,
			DocumentID: artifact.ID,
			Value:      l.Value,
			Unit: sql.NullString{
				String: l.Unit,
				Valid:  l.Unit != "",
			},
			Flag: flag,
			NormalizedName: sql.NullString{
				String: utils.NormalizeLabName(l.Name),
				Valid:  l.Name != "",
			},
			LabName: sql.NullString{
				String: l.Name,
				Valid:  l.Name != "",
			},
			CollectedAt: collectedAt,
		}
		if _, errI := s.repo.InsertLabResult(ctx, lab); errI != nil {
			s.log.Error("error inserting lab result", errI)
		}
	}
}

func (s *Service) saveMedications(ctx context.Context, artifact *entities.Artifact, data *entities.ArtifactParsedData) {
	for _, m := range data.Medications {
		if m.Name == "" {
			continue
		}
		med := &entities.Medication{
			UserID:    artifact.OwnerID,
			Name:      m.Name,
			Dose:      m.Dose,
			Frequency: m.Frequency,
		}
		if _, err := s.repo.InsertMedication(ctx, med); err != nil {
			s.log.Error("error inserting medication", err)
		}
	}
}
