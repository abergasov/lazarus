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
		meds, err := a.medRepo.ListActive(gctx, session.UserID)
		if err == nil {
			ac.ActiveMeds = meds
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
		if m.RxCUI != "" {
			rxcuis = append(rxcuis, m.RxCUI)
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
			b.WriteString(fmt.Sprintf("- %s %s %s\n", m.Name, m.Dose, m.Frequency))
		}
	}

	if len(ac.FlaggedInteractions) > 0 {
		b.WriteString("## ⚠️ Drug Interactions\n")
		for _, i := range ac.FlaggedInteractions {
			b.WriteString(fmt.Sprintf("- [%s] %s + %s: %s\n", i.Severity, i.DrugAName, i.DrugBName, i.Description))
		}
	}

	if len(ac.RecentLabs) > 0 {
		abnormal := 0
		for _, l := range ac.RecentLabs {
			if l.Flag != entities.FlagNormal {
				abnormal++
			}
		}
		b.WriteString(fmt.Sprintf("## Recent Labs\n%d results (%d abnormal)\n", len(ac.RecentLabs), abnormal))
	}

	if ac.Visit != nil {
		b.WriteString(fmt.Sprintf("## Current Visit\nReason: %s | Doctor: %s | Phase: %s\n",
			ac.Visit.Reason, ac.Visit.DoctorName, ac.Visit.Status))
	}

	return b.String()
}
