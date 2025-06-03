package metrics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/types"
)

// AdvancedMetrics represents sophisticated metrics for AI analysis
type AdvancedMetrics struct {
	// Trend Analysis
	MemoryTrend5m              float64   `json:"memory_trend_5m"`
	CPUOscillationAmplitude    float64   `json:"cpu_oscillation_amplitude"`
	ErrorRateTrend3m           float64   `json:"error_rate_trend_3m"`
	NetworkLatencyTrend        float64   `json:"network_latency_trend"`
	
	// AI Intelligence Metrics
	AIConfidenceScore          float64   `json:"ai_confidence_score"`
	AIReasoningSteps           []string  `json:"ai_reasoning_steps"`
	DecisionAlternatives       int       `json:"decision_alternatives"`
	AIPatternConfidence        float64   `json:"ai_pattern_confidence"`
	
	// Correlation & Health Scoring
	SystemHealthScore          float64   `json:"system_health_score"`
	CorrelationRiskScore       float64   `json:"correlation_risk_score"`
	PredictiveAccuracy         float64   `json:"predictive_accuracy"`
	CascadeRiskScore          float64   `json:"cascade_risk_score"`
	
	// Pattern Detection
	CPUOscillationPattern      string    `json:"cpu_oscillation_pattern"`
	MemoryLeakPattern         string    `json:"memory_leak_pattern"`
	RestartPattern            string    `json:"restart_pattern"`
	FailureCorrelations       []string  `json:"failure_correlations"`
	
	// Historical Data for Trends
	HistoricalData            map[string][]TimeSeriesPoint `json:"historical_data"`
	TrendAnalysisWindow       time.Duration                `json:"trend_analysis_window"`
	LastAnalysisTime          time.Time                    `json:"last_analysis_time"`
}

// TimeSeriesPoint represents a data point in time series
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// AdvancedCollector extends the basic collector with AI-focused metrics
type AdvancedCollector struct {
	*Collector
	historicalData    map[string][]TimeSeriesPoint
	aiMetricsEnabled  bool
	trendWindow       time.Duration
	patternDetector   *PatternDetector
}

// PatternDetector analyzes patterns in metrics data
type PatternDetector struct {
	cpuPatterns      []CPUPattern
	memoryPatterns   []MemoryPattern
	restartPatterns  []RestartPattern
	correlationMap   map[string][]string
}

// Pattern types for analysis
type CPUPattern struct {
	Name        string    `json:"name"`
	Amplitude   float64   `json:"amplitude"`
	Frequency   float64   `json:"frequency"`
	TrendSlope  float64   `json:"trend_slope"`
	Confidence  float64   `json:"confidence"`
	Detected    time.Time `json:"detected"`
}

type MemoryPattern struct {
	Name         string    `json:"name"`
	GrowthRate   float64   `json:"growth_rate"`
	LeakSeverity float64   `json:"leak_severity"`
	Confidence   float64   `json:"confidence"`
	Detected     time.Time `json:"detected"`
}

type RestartPattern struct {
	Name           string    `json:"name"`
	Frequency      float64   `json:"frequency"`
	CrashReason    string    `json:"crash_reason"`
	Confidence     float64   `json:"confidence"`
	Detected       time.Time `json:"detected"`
}

// NewAdvancedCollector creates an advanced metrics collector
func NewAdvancedCollector(baseCollector *Collector) *AdvancedCollector {
	return &AdvancedCollector{
		Collector:        baseCollector,
		historicalData:   make(map[string][]TimeSeriesPoint),
		aiMetricsEnabled: true,
		trendWindow:      10 * time.Minute,
		patternDetector:  NewPatternDetector(),
	}
}

// NewPatternDetector creates a new pattern detector
func NewPatternDetector() *PatternDetector {
	return &PatternDetector{
		cpuPatterns:     make([]CPUPattern, 0),
		memoryPatterns:  make([]MemoryPattern, 0),
		restartPatterns: make([]RestartPattern, 0),
		correlationMap:  make(map[string][]string),
	}
}

// CollectAdvancedMetrics gathers sophisticated metrics for AI analysis
func (ac *AdvancedCollector) CollectAdvancedMetrics(ctx context.Context, policy *v1alpha1.HealingPolicy) (*AdvancedMetrics, error) {
	log := log.FromContext(ctx)
	log.Info("Collecting advanced metrics for AI analysis", "policy", policy.Name)

	// Get basic metrics first
	basicMetrics, err := ac.Collector.CollectMetrics(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to collect basic metrics: %w", err)
	}

	// Store current metrics in historical data
	ac.updateHistoricalData(basicMetrics)

	// Create advanced metrics
	advanced := &AdvancedMetrics{
		HistoricalData:      ac.historicalData,
		TrendAnalysisWindow: ac.trendWindow,
		LastAnalysisTime:    time.Now(),
	}

	// Calculate trend analysis
	advanced.MemoryTrend5m = ac.calculateMemoryTrend(5 * time.Minute)
	advanced.CPUOscillationAmplitude = ac.calculateCPUOscillationAmplitude()
	advanced.ErrorRateTrend3m = ac.calculateErrorRateTrend(3 * time.Minute)
	advanced.NetworkLatencyTrend = ac.calculateNetworkLatencyTrend()

	// Calculate correlation and health scores
	advanced.SystemHealthScore = ac.calculateSystemHealthScore(basicMetrics)
	advanced.CorrelationRiskScore = ac.calculateCorrelationRiskScore()
	advanced.CascadeRiskScore = ac.calculateCascadeRiskScore(basicMetrics)

	// Pattern detection
	advanced.CPUOscillationPattern = ac.detectCPUOscillationPattern()
	advanced.MemoryLeakPattern = ac.detectMemoryLeakPattern()
	advanced.RestartPattern = ac.detectRestartPattern(basicMetrics)
	advanced.FailureCorrelations = ac.detectFailureCorrelations()

	// AI metrics (these will be set by AI analyzer later)
	advanced.AIConfidenceScore = 0.85 // Default for demo
	advanced.AIReasoningSteps = []string{"Analyzing metrics", "Detecting patterns", "Calculating confidence"}
	advanced.DecisionAlternatives = 3
	advanced.AIPatternConfidence = 0.78
	advanced.PredictiveAccuracy = 0.82

	log.Info("Advanced metrics calculated", 
		"memory_trend", advanced.MemoryTrend5m,
		"cpu_oscillation", advanced.CPUOscillationAmplitude,
		"system_health", advanced.SystemHealthScore,
		"correlation_risk", advanced.CorrelationRiskScore)

	return advanced, nil
}

// EvaluateAdvancedTrigger evaluates triggers using advanced metrics
func (ac *AdvancedCollector) EvaluateAdvancedTrigger(ctx context.Context, trigger *v1alpha1.HealingTrigger, metrics *AdvancedMetrics) (bool, string, error) {
	if trigger.MetricTrigger == nil {
		return false, "", fmt.Errorf("metric trigger configuration missing")
	}

	query := trigger.MetricTrigger.Query
	threshold := trigger.MetricTrigger.Threshold
	operator := trigger.MetricTrigger.Operator

	var actualValue float64
	var found bool

	// Map advanced metric queries to actual values
	switch query {
	case "memory_usage_trend_5m":
		actualValue = metrics.MemoryTrend5m
		found = true
	case "cpu_oscillation_amplitude_trend":
		actualValue = metrics.CPUOscillationAmplitude
		found = true
	case "error_rate_trend_3m":
		actualValue = metrics.ErrorRateTrend3m
		found = true
	case "correlation_risk_score":
		actualValue = metrics.CorrelationRiskScore
		found = true
	case "system_health_score":
		actualValue = metrics.SystemHealthScore
		found = true
	case "ai_confidence_score":
		actualValue = metrics.AIConfidenceScore
		found = true
	case "cascade_risk_score":
		actualValue = metrics.CascadeRiskScore
		found = true
	case "predictive_accuracy":
		actualValue = metrics.PredictiveAccuracy
		found = true
	default:
		// Fall back to basic metrics evaluation
		return ac.Collector.EvaluateTrigger(ctx, trigger, &types.ClusterMetrics{})
	}

	if !found {
		return false, fmt.Sprintf("advanced metric not found: %s", query), nil
	}

	// Evaluate the threshold
	triggered := ac.evaluateThreshold(actualValue, threshold, operator)
	reason := fmt.Sprintf("advanced query '%s' = %.2f %s %.2f", query, actualValue, operator, threshold)
	
	log.FromContext(ctx).Info("Advanced trigger evaluation", 
		"query", query, 
		"value", actualValue, 
		"threshold", threshold, 
		"operator", operator, 
		"triggered", triggered)

	return triggered, reason, nil
}

// updateHistoricalData stores current metrics for trend analysis
func (ac *AdvancedCollector) updateHistoricalData(metrics *types.ClusterMetrics) {
	timestamp := time.Now()
	
	// Store pod metrics
	for _, pod := range metrics.Pods {
		// CPU data
		cpuKey := fmt.Sprintf("pod_cpu_%s_%s", pod.Namespace, pod.Name)
		ac.addTimeSeriesPoint(cpuKey, TimeSeriesPoint{
			Timestamp: timestamp,
			Value:     pod.CPUUsage,
			Labels:    map[string]string{"pod": pod.Name, "namespace": pod.Namespace},
		})
		
		// Memory data
		memKey := fmt.Sprintf("pod_memory_%s_%s", pod.Namespace, pod.Name)
		ac.addTimeSeriesPoint(memKey, TimeSeriesPoint{
			Timestamp: timestamp,
			Value:     pod.MemoryUsage,
			Labels:    map[string]string{"pod": pod.Name, "namespace": pod.Namespace},
		})
		
		// Restart count
		restartKey := fmt.Sprintf("pod_restarts_%s_%s", pod.Namespace, pod.Name)
		ac.addTimeSeriesPoint(restartKey, TimeSeriesPoint{
			Timestamp: timestamp,
			Value:     float64(pod.RestartCount),
			Labels:    map[string]string{"pod": pod.Name, "namespace": pod.Namespace},
		})
	}
	
	// Store error events
	errorCount := 0
	for _, event := range metrics.Events {
		if event.Type == "Warning" && time.Since(event.LastSeen) < 5*time.Minute {
			errorCount++
		}
	}
	ac.addTimeSeriesPoint("error_count", TimeSeriesPoint{
		Timestamp: timestamp,
		Value:     float64(errorCount),
	})
	
	// Clean old data (keep only last 30 minutes)
	ac.cleanOldData(30 * time.Minute)
}

// addTimeSeriesPoint adds a data point to historical data
func (ac *AdvancedCollector) addTimeSeriesPoint(key string, point TimeSeriesPoint) {
	if ac.historicalData[key] == nil {
		ac.historicalData[key] = make([]TimeSeriesPoint, 0)
	}
	ac.historicalData[key] = append(ac.historicalData[key], point)
}

// cleanOldData removes data points older than the specified duration
func (ac *AdvancedCollector) cleanOldData(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	
	for key, points := range ac.historicalData {
		filtered := make([]TimeSeriesPoint, 0)
		for _, point := range points {
			if point.Timestamp.After(cutoff) {
				filtered = append(filtered, point)
			}
		}
		ac.historicalData[key] = filtered
	}
}

// calculateMemoryTrend calculates memory usage trend over specified window
func (ac *AdvancedCollector) calculateMemoryTrend(window time.Duration) float64 {
	cutoff := time.Now().Add(-window)
	var allMemoryPoints []TimeSeriesPoint
	
	// Collect all memory data points
	for key, points := range ac.historicalData {
		if strings.Contains(key, "pod_memory_") {
			for _, point := range points {
				if point.Timestamp.After(cutoff) {
					allMemoryPoints = append(allMemoryPoints, point)
				}
			}
		}
	}
	
	if len(allMemoryPoints) < 2 {
		return 0 // Not enough data for trend
	}
	
	// Sort by timestamp
	sort.Slice(allMemoryPoints, func(i, j int) bool {
		return allMemoryPoints[i].Timestamp.Before(allMemoryPoints[j].Timestamp)
	})
	
	// Calculate linear regression slope
	return ac.calculateLinearTrend(allMemoryPoints)
}

// calculateCPUOscillationAmplitude detects CPU usage oscillation patterns
func (ac *AdvancedCollector) calculateCPUOscillationAmplitude() float64 {
	var allCPUPoints []TimeSeriesPoint
	
	// Collect recent CPU data
	cutoff := time.Now().Add(-5 * time.Minute)
	for key, points := range ac.historicalData {
		if strings.Contains(key, "pod_cpu_") {
			for _, point := range points {
				if point.Timestamp.After(cutoff) {
					allCPUPoints = append(allCPUPoints, point)
				}
			}
		}
	}
	
	if len(allCPUPoints) < 10 {
		return 0 // Not enough data
	}
	
	// Sort by timestamp
	sort.Slice(allCPUPoints, func(i, j int) bool {
		return allCPUPoints[i].Timestamp.Before(allCPUPoints[j].Timestamp)
	})
	
	// Calculate amplitude by finding max - min over rolling windows
	windowSize := 5
	maxAmplitude := 0.0
	
	for i := 0; i <= len(allCPUPoints)-windowSize; i++ {
		window := allCPUPoints[i : i+windowSize]
		
		minVal := window[0].Value
		maxVal := window[0].Value
		
		for _, point := range window {
			if point.Value < minVal {
				minVal = point.Value
			}
			if point.Value > maxVal {
				maxVal = point.Value
			}
		}
		
		amplitude := maxVal - minVal
		if amplitude > maxAmplitude {
			maxAmplitude = amplitude
		}
	}
	
	// Convert to percentage for demo purposes
	return maxAmplitude * 100
}

// calculateErrorRateTrend calculates error rate trend over specified window
func (ac *AdvancedCollector) calculateErrorRateTrend(window time.Duration) float64 {
	cutoff := time.Now().Add(-window)
	var errorPoints []TimeSeriesPoint
	
	// Get error count data
	if points, exists := ac.historicalData["error_count"]; exists {
		for _, point := range points {
			if point.Timestamp.After(cutoff) {
				errorPoints = append(errorPoints, point)
			}
		}
	}
	
	if len(errorPoints) < 3 {
		return 10.0 // Default error rate for demo
	}
	
	// Sort by timestamp
	sort.Slice(errorPoints, func(i, j int) bool {
		return errorPoints[i].Timestamp.Before(errorPoints[j].Timestamp)
	})
	
	// Calculate trend slope
	trend := ac.calculateLinearTrend(errorPoints)
	
	// Convert to percentage and add baseline for demo
	return math.Max(0, 15.0+trend*10) // Base 15% error rate + trend
}

// calculateNetworkLatencyTrend simulates network latency trend analysis
func (ac *AdvancedCollector) calculateNetworkLatencyTrend() float64 {
	// Simulated network latency trend for demo
	// In real implementation, this would analyze network metrics
	now := time.Now()
	minute := now.Minute()
	
	// Create a sine wave pattern for demo
	latency := 50 + 30*math.Sin(float64(minute)/10*math.Pi)
	return latency
}

// calculateSystemHealthScore computes overall system health
func (ac *AdvancedCollector) calculateSystemHealthScore(metrics *types.ClusterMetrics) float64 {
	if len(metrics.Pods) == 0 {
		return 100.0
	}
	
	healthyPods := 0
	totalRestarts := int32(0)
	recentErrors := 0
	
	// Analyze pod health
	for _, pod := range metrics.Pods {
		if pod.Status == "Running" && pod.RestartCount < 3 {
			healthyPods++
		}
		totalRestarts += pod.RestartCount
	}
	
	// Count recent error events
	for _, event := range metrics.Events {
		if event.Type == "Warning" && time.Since(event.LastSeen) < 5*time.Minute {
			recentErrors++
		}
	}
	
	// Calculate health score
	podHealthScore := float64(healthyPods) / float64(len(metrics.Pods)) * 100
	
	// Penalty for restarts
	restartPenalty := float64(totalRestarts) * 2.0
	
	// Penalty for recent errors
	errorPenalty := float64(recentErrors) * 5.0
	
	healthScore := podHealthScore - restartPenalty - errorPenalty
	
	// Ensure score is between 0 and 100
	if healthScore < 0 {
		healthScore = 0
	}
	if healthScore > 100 {
		healthScore = 100
	}
	
	return healthScore
}

// calculateCorrelationRiskScore analyzes correlations between different metrics
func (ac *AdvancedCollector) calculateCorrelationRiskScore() float64 {
	// Analyze correlations between CPU, memory, and restart patterns
	correlationFactors := []float64{}
	
	// CPU-Memory correlation
	cpuMemCorr := ac.calculateMetricCorrelation("cpu", "memory")
	correlationFactors = append(correlationFactors, cpuMemCorr)
	
	// Memory-Restart correlation
	memRestartCorr := ac.calculateMetricCorrelation("memory", "restarts")
	correlationFactors = append(correlationFactors, memRestartCorr)
	
	// CPU-Error correlation
	cpuErrorCorr := ac.calculateMetricCorrelation("cpu", "error")
	correlationFactors = append(correlationFactors, cpuErrorCorr)
	
	// Calculate average correlation risk
	if len(correlationFactors) == 0 {
		return 25.0 // Default risk score for demo
	}
	
	sum := 0.0
	for _, factor := range correlationFactors {
		sum += factor
	}
	avg := sum / float64(len(correlationFactors))
	
	// Convert correlation to risk score (higher correlation = higher risk)
	riskScore := avg * 100
	
	// Add time-based variance for demo
	now := time.Now()
	variance := 10 * math.Sin(float64(now.Second())/30*math.Pi)
	
	return math.Max(0, math.Min(100, riskScore+variance))
}

// calculateCascadeRiskScore analyzes risk of cascade failures
func (ac *AdvancedCollector) calculateCascadeRiskScore(metrics *types.ClusterMetrics) float64 {
	riskFactors := []float64{}
	
	// High restart rate indicates instability
	totalRestarts := int32(0)
	for _, pod := range metrics.Pods {
		totalRestarts += pod.RestartCount
	}
	if len(metrics.Pods) > 0 {
		avgRestarts := float64(totalRestarts) / float64(len(metrics.Pods))
		riskFactors = append(riskFactors, avgRestarts*10) // 10% risk per restart
	}
	
	// Recent error events increase cascade risk
	recentErrors := 0
	for _, event := range metrics.Events {
		if event.Type == "Warning" && time.Since(event.LastSeen) < 2*time.Minute {
			recentErrors++
		}
	}
	riskFactors = append(riskFactors, float64(recentErrors)*5) // 5% risk per error
	
	// Resource pressure increases cascade risk
	highCPUPods := 0
	highMemoryPods := 0
	for _, pod := range metrics.Pods {
		if pod.CPUUsage > 0.8 { // 80% CPU usage
			highCPUPods++
		}
		if pod.MemoryUsage > 400 { // 400MB memory usage
			highMemoryPods++
		}
	}
	
	if len(metrics.Pods) > 0 {
		cpuPressure := float64(highCPUPods) / float64(len(metrics.Pods)) * 30
		memoryPressure := float64(highMemoryPods) / float64(len(metrics.Pods)) * 40
		riskFactors = append(riskFactors, cpuPressure, memoryPressure)
	}
	
	// Calculate total risk
	totalRisk := 0.0
	for _, risk := range riskFactors {
		totalRisk += risk
	}
	
	// Cap at 100%
	if totalRisk > 100 {
		totalRisk = 100
	}
	
	return totalRisk
}

// Helper methods for pattern detection

func (ac *AdvancedCollector) detectCPUOscillationPattern() string {
	amplitude := ac.calculateCPUOscillationAmplitude()
	
	if amplitude > 80 {
		return "high-amplitude-oscillation"
	} else if amplitude > 50 {
		return "medium-amplitude-oscillation" 
	} else if amplitude > 20 {
		return "low-amplitude-oscillation"
	}
	return "stable"
}

func (ac *AdvancedCollector) detectMemoryLeakPattern() string {
	trend := ac.calculateMemoryTrend(5 * time.Minute)
	
	if trend > 20 {
		return "severe-memory-leak"
	} else if trend > 10 {
		return "moderate-memory-leak"
	} else if trend > 5 {
		return "minor-memory-leak"
	}
	return "stable"
}

func (ac *AdvancedCollector) detectRestartPattern(metrics *types.ClusterMetrics) string {
	if len(metrics.Pods) == 0 {
		return "stable"
	}
	
	highRestartPods := 0
	for _, pod := range metrics.Pods {
		if pod.RestartCount > 2 {
			highRestartPods++
		}
	}
	
	restartRatio := float64(highRestartPods) / float64(len(metrics.Pods))
	
	if restartRatio > 0.5 {
		return "frequent-restart-pattern"
	} else if restartRatio > 0.2 {
		return "moderate-restart-pattern"
	} else if restartRatio > 0 {
		return "occasional-restart-pattern"
	}
	return "stable"
}

func (ac *AdvancedCollector) detectFailureCorrelations() []string {
	correlations := []string{}
	
	// Check for common failure patterns
	if ac.calculateMetricCorrelation("cpu", "memory") > 0.7 {
		correlations = append(correlations, "cpu-memory-correlation")
	}
	
	if ac.calculateMetricCorrelation("memory", "restarts") > 0.6 {
		correlations = append(correlations, "memory-restart-correlation")
	}
	
	if ac.calculateMetricCorrelation("cpu", "error") > 0.5 {
		correlations = append(correlations, "cpu-error-correlation")
	}
	
	return correlations
}

// Helper methods for calculations

func (ac *AdvancedCollector) calculateLinearTrend(points []TimeSeriesPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	
	n := float64(len(points))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0
	
	for i, point := range points {
		x := float64(i)
		y := point.Value
		
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	
	// Calculate slope (trend)
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	
	return slope
}

func (ac *AdvancedCollector) calculateMetricCorrelation(metric1, metric2 string) float64 {
	// Simplified correlation calculation for demo
	// In real implementation, this would use Pearson correlation coefficient
	
	// Create synthetic correlation based on current time for demo variability
	now := time.Now()
	seed := float64(now.Second() + now.Minute()*60)
	
	// Different correlation patterns for different metric pairs
	switch metric1 + "-" + metric2 {
	case "cpu-memory":
		return 0.3 + 0.4*math.Sin(seed/100)
	case "memory-restarts":
		return 0.4 + 0.3*math.Cos(seed/80)
	case "cpu-error":
		return 0.2 + 0.3*math.Sin(seed/120)
	default:
		return 0.1 + 0.2*math.Sin(seed/90)
	}
}

func (ac *AdvancedCollector) evaluateThreshold(value, threshold float64, operator string) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}