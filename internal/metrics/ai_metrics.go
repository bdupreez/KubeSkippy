package metrics

import (
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/kubeskippy/kubeskippy/internal/controller"
)

var (
	// AI reasoning metrics - initialized by SetAIMetrics from main.go
	aiReasoningStepsTotal    *prometheus.CounterVec
	aiDecisionConfidence     *prometheus.HistogramVec
	aiAlternativesConsidered *prometheus.CounterVec
	aiConfidenceFactors      *prometheus.CounterVec
)

// SetAIMetrics sets the AI metrics references from main.go
func SetAIMetrics(reasoningSteps, alternatives, confidenceFactors *prometheus.CounterVec, decisionConfidence *prometheus.HistogramVec) {
	aiReasoningStepsTotal = reasoningSteps
	aiDecisionConfidence = decisionConfidence
	aiAlternativesConsidered = alternatives
	aiConfidenceFactors = confidenceFactors
}

// AIMetricsRecorder records AI reasoning metrics
type AIMetricsRecorder struct{}

// NewAIMetricsRecorder creates a new AI metrics recorder
func NewAIMetricsRecorder() *AIMetricsRecorder {
	return &AIMetricsRecorder{}
}

// RecordAIAnalysis records metrics for a complete AI analysis
func (r *AIMetricsRecorder) RecordAIAnalysis(analysis *controller.AIAnalysis) {
	if analysis == nil {
		return
	}

	model := analysis.ModelVersion
	if model == "" {
		model = "unknown"
	}

	// Record reasoning steps
	for _, step := range analysis.ReasoningSteps {
		stepType := "general"
		if strings.Contains(strings.ToLower(step.Description), "pattern") {
			stepType = "pattern_recognition"
		} else if strings.Contains(strings.ToLower(step.Description), "correlation") {
			stepType = "correlation_analysis"
		} else if strings.Contains(strings.ToLower(step.Description), "trend") {
			stepType = "trend_analysis"
		} else if strings.Contains(strings.ToLower(step.Description), "root") {
			stepType = "root_cause_analysis"
		}

		if aiReasoningStepsTotal != nil {
			aiReasoningStepsTotal.WithLabelValues(model, stepType).Inc()
		}
	}

	// Record recommendation metrics
	for _, rec := range analysis.Recommendations {
		actionType := r.normalizeActionType(rec.Action)

		// Record confidence level
		if aiDecisionConfidence != nil {
			aiDecisionConfidence.WithLabelValues(model, actionType).Observe(rec.Confidence)
		}

		// Record alternatives considered
		for _, alt := range rec.Reasoning.Alternatives {
			rejectedStr := strconv.FormatBool(alt.Rejected)
			altActionType := r.normalizeActionType(alt.Action)

			if aiAlternativesConsidered != nil {
				aiAlternativesConsidered.WithLabelValues(model, altActionType, rejectedStr).Inc()
			}
		}

		// Record confidence factors
		for _, factor := range rec.Reasoning.ConfidenceFactors {
			factorType := r.normalizeFactorType(factor.Factor)

			if aiConfidenceFactors != nil {
				aiConfidenceFactors.WithLabelValues(model, factorType, factor.Impact).Inc()
			}
		}
	}
}

// normalizeActionType converts action descriptions to normalized types
func (r *AIMetricsRecorder) normalizeActionType(action string) string {
	action = strings.ToLower(action)

	if strings.Contains(action, "restart") {
		return "restart"
	} else if strings.Contains(action, "scale") {
		return "scale"
	} else if strings.Contains(action, "delete") {
		return "delete"
	} else if strings.Contains(action, "patch") {
		return "patch"
	} else if strings.Contains(action, "rolling") {
		return "rolling_update"
	}

	return "other"
}

// normalizeFactorType converts confidence factor descriptions to normalized types
func (r *AIMetricsRecorder) normalizeFactorType(factor string) string {
	factor = strings.ToLower(factor)

	if strings.Contains(factor, "pattern") {
		return "pattern_recognition"
	} else if strings.Contains(factor, "history") || strings.Contains(factor, "historical") {
		return "historical_data"
	} else if strings.Contains(factor, "metric") || strings.Contains(factor, "measurement") {
		return "metric_quality"
	} else if strings.Contains(factor, "correlation") {
		return "correlation_strength"
	} else if strings.Contains(factor, "risk") {
		return "risk_assessment"
	} else if strings.Contains(factor, "evidence") {
		return "evidence_quality"
	}

	return "other"
}

// RecordReasoningStep records a single reasoning step metric
func (r *AIMetricsRecorder) RecordReasoningStep(model, stepType string) {
	if aiReasoningStepsTotal != nil {
		aiReasoningStepsTotal.WithLabelValues(model, stepType).Inc()
	}
}

// RecordDecisionConfidence records a confidence level metric
func (r *AIMetricsRecorder) RecordDecisionConfidence(model, actionType string, confidence float64) {
	if aiDecisionConfidence != nil {
		aiDecisionConfidence.WithLabelValues(model, actionType).Observe(confidence)
	}
}

// RecordAlternativeConsidered records an alternative consideration metric
func (r *AIMetricsRecorder) RecordAlternativeConsidered(model, actionType string, rejected bool) {
	rejectedStr := strconv.FormatBool(rejected)
	if aiAlternativesConsidered != nil {
		aiAlternativesConsidered.WithLabelValues(model, actionType, rejectedStr).Inc()
	}
}

// RecordConfidenceFactor records a confidence factor metric
func (r *AIMetricsRecorder) RecordConfidenceFactor(model, factorType, impact string) {
	if aiConfidenceFactors != nil {
		aiConfidenceFactors.WithLabelValues(model, factorType, impact).Inc()
	}
}
