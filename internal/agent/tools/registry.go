package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"lazarus/internal/entities"
	"lazarus/internal/knowledge"
	"lazarus/internal/provider"
	labsvc "lazarus/internal/service/lab"
	risksvc "lazarus/internal/service/risk"
)

type UserContext struct {
	UserID       string
	PatientModel *entities.PatientModel
	VisitID      string
	Phase        string
}

type Tool struct {
	Name          string
	Description   string
	Schema        []byte   // JSON Schema for LLM
	Phases        []string // which phases this tool is available
	Execute       func(ctx context.Context, args json.RawMessage, uc *UserContext) (any, error)
	HumanLabel    func(args json.RawMessage) string
	ResultSummary func(result any) string
}

type Registry struct {
	tools map[string]*Tool
}

type Deps struct {
	DB          *sqlx.DB
	KBRepo      *knowledge.Repository
	LabSvc      *labsvc.Service
	RiskSvc     *risksvc.Service
	ProviderReg *provider.Registry
}

func NewRegistry(deps *Deps) *Registry {
	r := &Registry{tools: make(map[string]*Tool)}
	r.register(searchKBTool(deps))
	r.register(flagAbnormalsTool(deps))
	r.register(getTrendsTool(deps))
	r.register(calcRiskTool(deps))
	r.register(checkInteractionsTool(deps))
	r.register(lookupConditionTool(deps))
	r.register(recordOutcomeTool(deps))
	r.register(updateModelTool(deps))
	r.register(savePlanTool(deps))
	r.register(addDoctorQuestionTool(deps))
	r.register(checkContraindicationsTool(deps))
	return r
}

func (r *Registry) register(t *Tool) { r.tools[t.Name] = t }

func (r *Registry) ForPhase(phase string) []*Tool {
	var result []*Tool
	for _, t := range r.tools {
		for _, p := range t.Phases {
			// General phase gets tools from both preparing and during
			if p == phase || (phase == entities.PhaseGeneral && (p == entities.PhasePreparing || p == entities.PhaseDuring)) {
				result = append(result, t)
				break
			}
		}
	}
	return result
}

func (r *Registry) Execute(ctx context.Context, name string, args json.RawMessage, uc *UserContext) (any, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
	return t.Execute(ctx, args, uc)
}

func (r *Registry) HumanLabel(name string, args json.RawMessage) string {
	t, ok := r.tools[name]
	if !ok || t.HumanLabel == nil {
		return fmt.Sprintf("Running %s...", name)
	}
	return t.HumanLabel(args)
}

func (r *Registry) Summary(name string, result any) string {
	t, ok := r.tools[name]
	if !ok || t.ResultSummary == nil {
		return ""
	}
	return t.ResultSummary(result)
}

func mustSchema(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
