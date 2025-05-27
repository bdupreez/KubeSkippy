# AI-Driven Healing in KubeSkippy: How It Works with Ollama

## Overview

AI-driven healing enhances KubeSkippy's decision-making by using Large Language Models (LLMs) to analyze complex cluster states and provide intelligent remediation recommendations. It integrates with Ollama (local) or OpenAI (cloud) to add reasoning capabilities beyond simple threshold-based rules.

## Architecture

```
┌─────────────────────┐     ┌─────────────────────┐     ┌─────────────────────┐
│   Metrics/Events    │────▶│   AI Analyzer       │────▶│    Ollama/LLM       │
│   - CPU/Memory      │     │                     │     │                     │
│   - Restarts        │     │ 1. Build Prompt     │     │ - Local inference   │
│   - Error rates     │     │ 2. Query AI         │     │ - Privacy-focused   │
│   - Pod status      │     │ 3. Parse Response   │     │ - No data leaves    │
└─────────────────────┘     │ 4. Validate Safety  │     │   your cluster      │
                            └─────────────────────┘     └─────────────────────┘
                                       │
                                       ▼
                            ┌─────────────────────┐
                            │  Healing Actions    │
                            │  - Prioritized      │
                            │  - Confidence-based │
                            │  - Safety-validated │
                            └─────────────────────┘
```

## How Ollama Integration Works

### 1. **Local LLM Deployment**

Ollama runs as a service in your cluster:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ollama
  namespace: kubeskippy-system
spec:
  template:
    spec:
      containers:
      - name: ollama
        image: ollama/ollama:latest
        ports:
        - containerPort: 11434
```

### 2. **Model Selection**

Default configuration uses Llama2, but you can use any Ollama model:

```yaml
ai:
  provider: ollama
  endpoint: http://ollama.kubeskippy-system:11434
  model: llama2  # or codellama, mistral, mixtral, etc.
```

### 3. **The AI Analysis Flow**

#### Step 1: Trigger Evaluation
When normal triggers fire (CPU > 80%, crashes, etc.), the system collects comprehensive metrics:

```go
metrics := &ClusterMetrics{
    Pods: []PodMetrics{
        {
            Name: "app-xyz",
            CPUUsage: 890,      // millicores
            MemoryUsage: 1500,  // MB
            RestartCount: 5,
            Status: "CrashLoopBackOff",
        },
    },
    Events: []EventMetrics{
        {
            Type: "Warning",
            Reason: "OOMKilled",
            Count: 3,
        },
    },
}
```

#### Step 2: Prompt Construction
The AI Analyzer builds a structured prompt:

```
You are a Kubernetes cluster healing expert. Analyze the following cluster state:

CLUSTER METRICS:
{
  "pods": [{
    "name": "app-xyz",
    "cpu_usage": 890,
    "memory_usage": 1500,
    "restart_count": 5,
    "status": "CrashLoopBackOff"
  }],
  "events": [{
    "type": "Warning",
    "reason": "OOMKilled",
    "count": 3
  }]
}

DETECTED ISSUES:
- Pod app-xyz has restarted 5 times
- Memory usage at 1500MB (limit 512MB)
- OOMKilled events detected

Please provide recommendations...
```

#### Step 3: Ollama Query
The prompt is sent to Ollama via HTTP API:

```go
request := OllamaRequest{
    Model:       "llama2",
    Prompt:      prompt,
    Temperature: 0.7,  // Balance between creativity and consistency
    Stream:      false,
}

// POST to http://ollama:11434/api/generate
response := ollamaClient.Query(ctx, prompt, temperature)
```

#### Step 4: AI Response
Ollama analyzes the situation and returns structured recommendations:

```
SUMMARY:
The application is experiencing memory pressure leading to OOM kills and crash loops. 
The pod is consuming 3x its memory limit.

ISSUES:
- Memory leak in application
  Severity: High
  Impact: Service unavailability, repeated crashes
  Root Cause: Likely memory leak or insufficient memory allocation

RECOMMENDATIONS:
1. Increase memory limit to 2GB
   Target: Deployment/app
   Reason: Current usage exceeds limit by 3x
   Risk: May mask underlying memory leak
   Confidence: 0.85

2. Restart pod with heap dump enabled
   Target: Pod/app-xyz
   Reason: Collect diagnostic data
   Risk: Minimal
   Confidence: 0.90

3. Scale horizontally to distribute load
   Target: Deployment/app
   Reason: Reduce memory pressure per pod
   Risk: May increase total resource usage
   Confidence: 0.75
```

#### Step 5: Action Creation
Based on AI recommendations with high confidence (>0.7), healing actions are created:

```yaml
apiVersion: kubeskippy.io/v1alpha1
kind: HealingAction
metadata:
  name: ai-memory-increase-abc123
  annotations:
    ai.confidence: "0.85"
    ai.reasoning: "Current usage exceeds limit by 3x"
spec:
  action:
    type: patch
    patchAction:
      patch: |
        spec:
          containers:
          - name: app
            resources:
              limits:
                memory: "2Gi"
```

## What Makes It "AI-Driven"?

### 1. **Contextual Understanding**
Unlike simple threshold rules, AI understands relationships:
- "High CPU + increased error rate = likely performance issue"
- "Memory growth + restart pattern = probable memory leak"
- "Multiple pods failing similarly = systemic issue"

### 2. **Root Cause Analysis**
AI can infer causes from symptoms:
```
Symptoms: OOMKilled, LinearMemoryGrowth, NoTrafficSpikes
AI Inference: "Memory leak in application code, not load-related"
```

### 3. **Intelligent Prioritization**
AI ranks actions by likely effectiveness:
```
1. Quick fix: Restart (90% confidence)
2. Medium fix: Increase resources (85% confidence)  
3. Long fix: Refactor application (60% confidence)
```

### 4. **Safety Reasoning**
AI considers side effects:
```
"Scaling up will help but may increase costs by $X/month"
"Restarting during peak hours risks 5% traffic loss"
```

## Real-World Example

### Scenario: Complex Failure Pattern

**Traditional Rule**: 
```yaml
if cpu > 80% then scale_up
```

**AI-Driven Analysis**:
```
OBSERVATION: CPU spikes to 95% every 30 minutes, coinciding with:
- Cron job execution
- Database connection pool exhaustion  
- Memory allocation spike

INFERENCE: The cron job is creating too many DB connections

RECOMMENDATION: 
1. Patch cron job with connection limit (confidence: 0.92)
2. Temporary scale during cron window (confidence: 0.88)
3. Long-term: Implement connection pooling (confidence: 0.95)
```

## Configuration Examples

### Basic Ollama Setup
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: kubeskippy-config
data:
  config.yaml: |
    ai:
      provider: ollama
      endpoint: http://ollama:11434
      model: llama2
      temperature: 0.7
      timeout: 30s
      minConfidence: 0.75
```

### Advanced Configuration
```yaml
ai:
  provider: ollama
  endpoint: http://ollama:11434
  model: codellama:13b  # Better for analyzing logs/code
  temperature: 0.3      # Lower = more deterministic
  timeout: 60s
  minConfidence: 0.8    # Higher threshold for production
  validateResponses: true
  maxRecommendations: 3
  
  # Custom prompts for specific scenarios
  customPrompts:
    memoryAnalysis: |
      Focus on memory leak patterns and JVM heap analysis...
    
    performanceAnalysis: |
      Analyze response time degradation and bottlenecks...
```

## Benefits of AI-Driven Healing

### 1. **Handles Unknown Patterns**
- Learns from new failure modes
- Adapts to your specific environment
- No need to pre-program every scenario

### 2. **Reduces False Positives**
- Understands context (maintenance windows, deployments)
- Correlates multiple signals
- Avoids unnecessary actions

### 3. **Provides Explanations**
```
"Restarting pod app-xyz because:
- Memory usage pattern indicates leak (87% confidence)
- Similar issue resolved by restart 3 times this week
- No recent code changes that would fix root cause"
```

### 4. **Continuous Learning**
- Each action's outcome feeds back
- Confidence scores improve over time
- Recommendations get more specific

## Privacy & Security

### Local Processing with Ollama
- **No data leaves your cluster**
- Models run entirely on your infrastructure
- Complete control over model selection
- Air-gapped environment compatible

### Data Sent to AI
Only anonymized metrics:
- Resource usage numbers
- Event types and counts
- Status conditions
- **No**: Secrets, environment variables, or sensitive data

## Limitations

1. **Resource Requirements**
   - Ollama needs 4-8GB RAM for smaller models
   - GPU recommended for faster inference

2. **Response Time**
   - AI analysis adds 5-30 seconds
   - Not suitable for sub-second responses

3. **Model Accuracy**
   - Depends on model quality
   - May need fine-tuning for specialized environments

## Future Enhancements

1. **Custom Model Training**
   - Train on your cluster's historical data
   - Learn organization-specific patterns

2. **Multi-Model Ensemble**
   - Different models for different problem types
   - Consensus-based recommendations

3. **Feedback Loop**
   - Track action effectiveness
   - Automatically adjust confidence thresholds
   - Improve prompts based on outcomes

## Summary

AI-driven healing transforms KubeSkippy from a rule-based system to an intelligent assistant that:
- **Understands** complex failure patterns
- **Reasons** about root causes
- **Recommends** contextual solutions
- **Learns** from outcomes

With Ollama integration, you get enterprise-grade AI capabilities while maintaining complete data privacy and control.