package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
	"lazarus/internal/entities"
	"lazarus/internal/knowledge"
	"lazarus/internal/repository"
	labsvc "lazarus/internal/service/lab"
	risksvc "lazarus/internal/service/risk"
)

type AssembledContext struct {
	PatientModel        *entities.PatientModel
	RecentLabs          []entities.AnnotatedLab
	Trends              []entities.TrendSummary
	ActiveMeds          []entities.Medication
	PastMeds            []entities.Medication
	FlaggedInteractions []knowledge.DrugInteraction
	Visit               *entities.Visit
	Phase               string
	SystemPromptContext string
}

type Assembler struct {
	patients  *PatientModelStore
	labSvc    *labsvc.Service
	riskSvc   *risksvc.Service
	kbRepo    *knowledge.Repository
	medRepo   *repository.MedicationRepo
	visitRepo *repository.VisitRepo
}

func NewAssembler(
	db *sqlx.DB,
	patients *PatientModelStore,
	labSvc *labsvc.Service,
	riskSvc *risksvc.Service,
	kbRepo *knowledge.Repository,
) *Assembler {
	return &Assembler{
		patients:  patients,
		labSvc:    labSvc,
		riskSvc:   riskSvc,
		kbRepo:    kbRepo,
		medRepo:   repository.NewMedicationRepo(db),
		visitRepo: repository.NewVisitRepo(db),
	}
}

func (a *Assembler) Build(ctx context.Context, session *Session) (*AssembledContext, error) {
	ac := &AssembledContext{Phase: session.Phase}

	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		model, err := a.patients.Load(gctx, session.UserID)
		if err == nil {
			ac.PatientModel = model
		}
		return nil
	})
	g.Go(func() error {
		labs, err := a.labSvc.GetAnnotatedLabs(gctx, session.UserID, 90)
		if err == nil {
			ac.RecentLabs = labs
		}
		return nil
	})
	g.Go(func() error {
		trends, err := a.labSvc.GetTrendsForUser(gctx, session.UserID, 24)
		if err == nil {
			ac.Trends = trends
		}
		return nil
	})
	g.Go(func() error {
		allMeds, err := a.medRepo.ListAll(gctx, session.UserID)
		if err == nil {
			for _, m := range allMeds {
				if m.IsActive {
					ac.ActiveMeds = append(ac.ActiveMeds, m)
				} else {
					ac.PastMeds = append(ac.PastMeds, m)
				}
			}
		}
		return nil
	})
	g.Go(func() error {
		if session.VisitID != uuid.Nil {
			visit, err := a.visitRepo.Get(gctx, session.VisitID.String())
			if err == nil {
				ac.Visit = visit
			}
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// After concurrent fetch, check drug interactions
	if len(ac.ActiveMeds) > 1 {
		rxcuis := extractRxCUIs(ac.ActiveMeds)
		ac.FlaggedInteractions, _ = a.kbRepo.GetDrugInteractions(ctx, rxcuis)
	}

	ac.SystemPromptContext = renderContext(ac)
	return ac, nil
}

func extractRxCUIs(meds []entities.Medication) []string {
	rxcuis := make([]string, 0, len(meds))
	for _, m := range meds {
		if m.RxCUI != nil && *m.RxCUI != "" {
			rxcuis = append(rxcuis, *m.RxCUI)
		}
	}
	return rxcuis
}

func renderContext(ac *AssembledContext) string {
	var b strings.Builder

	if ac.PatientModel != nil {
		demo := ac.PatientModel.Demographics
		if !demo.DateOfBirth.IsZero() {
			b.WriteString(fmt.Sprintf("## Patient\nSex: %s | Smoker: %v\n", demo.Sex, demo.Smoker))
		}
		if len(ac.PatientModel.ActiveConditions) > 0 {
			b.WriteString("## Active Conditions\n")
			for _, c := range ac.PatientModel.ActiveConditions {
				b.WriteString(fmt.Sprintf("- %s (%s)\n", c.Name, c.ICD10Code))
			}
		}
		if len(ac.PatientModel.KeyConcerns) > 0 {
			b.WriteString("## Key Concerns\n")
			for _, c := range ac.PatientModel.KeyConcerns {
				b.WriteString(fmt.Sprintf("- [%s] %s\n", c.Severity, c.Description))
			}
		}
	}

	if len(ac.ActiveMeds) > 0 {
		b.WriteString("## Active Medications\n")
		for _, m := range ac.ActiveMeds {
			since := ""
			if m.StartedAt != nil {
				since = fmt.Sprintf(" (since %s)", m.StartedAt.Format("2006-01-02"))
			}
			b.WriteString(fmt.Sprintf("- %s %s %s%s\n", m.Name, m.Dose, m.Frequency, since))
		}
	}

	if len(ac.PastMeds) > 0 {
		b.WriteString("## Past Medications\n")
		for _, m := range ac.PastMeds {
			period := ""
			if m.StartedAt != nil && m.EndedAt != nil {
				period = fmt.Sprintf(" (%s → %s)", m.StartedAt.Format("2006-01-02"), m.EndedAt.Format("2006-01-02"))
			} else if m.EndedAt != nil {
				period = fmt.Sprintf(" (stopped %s)", m.EndedAt.Format("2006-01-02"))
			}
			b.WriteString(fmt.Sprintf("- %s %s %s%s\n", m.Name, m.Dose, m.Frequency, period))
		}
	}

	if len(ac.FlaggedInteractions) > 0 {
		b.WriteString("## Drug Interactions\n")
		for _, i := range ac.FlaggedInteractions {
			b.WriteString(fmt.Sprintf("- [%s] %s + %s: %s\n", i.Severity, i.DrugAName, i.DrugBName, i.Description))
		}
	}

	if len(ac.RecentLabs) > 0 {
		// Separate abnormal/critical from normal — give the agent the actual values
		var abnormals, normals []entities.AnnotatedLab
		for _, l := range ac.RecentLabs {
			if l.Flag != entities.FlagNormal && l.Flag != "" {
				abnormals = append(abnormals, l)
			} else {
				normals = append(normals, l)
			}
		}

		b.WriteString(fmt.Sprintf("## Lab Results (%d total)\n", len(ac.RecentLabs)))

		if len(abnormals) > 0 {
			b.WriteString(fmt.Sprintf("\n### Abnormal Results (%d)\n", len(abnormals)))
			for _, l := range abnormals {
				name := labDisplayName(l)
				unit := ""
				if l.Unit != nil {
					unit = *l.Unit
				}
				b.WriteString(fmt.Sprintf("- **%s**: %.2f %s [%s] (%s)\n",
					name, l.Value, unit, strings.ToUpper(l.Flag), l.CollectedAt.Format("2006-01-02")))
			}
		}

		if len(normals) > 0 {
			b.WriteString(fmt.Sprintf("\n### Normal Results (%d)\n", len(normals)))
			// Group by name, show latest only to keep context manageable
			seen := map[string]bool{}
			for _, l := range normals {
				name := labDisplayName(l)
				if seen[name] {
					continue
				}
				seen[name] = true
				unit := ""
				if l.Unit != nil {
					unit = *l.Unit
				}
				b.WriteString(fmt.Sprintf("- %s: %.2f %s (%s)\n",
					name, l.Value, unit, l.CollectedAt.Format("2006-01-02")))
			}
		}
	}

	if len(ac.Trends) > 0 {
		significant := false
		for _, t := range ac.Trends {
			if t.Significance == "significant" || t.Significance == "borderline" {
				if !significant {
					b.WriteString("\n## Notable Trends\n")
					significant = true
				}
				b.WriteString(fmt.Sprintf("- %s: %s (%.1f%% change, %s)\n",
					t.Name, t.Direction, t.PercentChange, t.Interpretation))
			}
		}
	}

	if ac.Visit != nil {
		reason, doctor := "", ""
		if ac.Visit.Reason != nil {
			reason = *ac.Visit.Reason
		}
		if ac.Visit.DoctorName != nil {
			doctor = *ac.Visit.DoctorName
		}
		b.WriteString(fmt.Sprintf("## Current Visit\nReason: %s | Doctor: %s | Phase: %s\n",
			reason, doctor, ac.Visit.Status))
	}

	return b.String()
}

func labDisplayName(l entities.AnnotatedLab) string {
	if l.LoincName != "" {
		return l.LoincName
	}
	if l.LabName != nil && *l.LabName != "" {
		return *l.LabName
	}
	if l.LoincCode != nil {
		return *l.LoincCode
	}
	return "Unknown"
}
