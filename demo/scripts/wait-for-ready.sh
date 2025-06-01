#!/bin/bash
# Wait for ready functions

wait_for_pods() {
    local namespace=$1
    local label=$2
    local timeout=${3:-300}
    
    echo -e "${YELLOW}⏳ Waiting for pods with label $label in namespace $namespace...${NC}"
    
    local count=0
    while [ $count -lt $timeout ]; do
        local ready_pods=$(kubectl get pods -n "$namespace" -l "$label" --no-headers 2>/dev/null | grep "1/1.*Running" | wc -l || echo 0)
        local total_pods=$(kubectl get pods -n "$namespace" -l "$label" --no-headers 2>/dev/null | wc -l || echo 0)
        
        if [ "$ready_pods" -gt 0 ] && [ "$ready_pods" -eq "$total_pods" ]; then
            echo -e "${GREEN}✅ Pods ready!${NC}"
            return 0
        fi
        
        if [ $((count % 30)) -eq 0 ]; then
            echo -e "${YELLOW}   Still waiting... ($count/${timeout}s) - $ready_pods/$total_pods pods ready${NC}"
        fi
        
        sleep 5
        count=$((count + 5))
    done
    
    echo -e "${YELLOW}⚠️ Timeout waiting for pods, continuing anyway${NC}"
    return 0
}

wait_for_deployment() {
    local namespace=$1
    local deployment=$2
    local timeout=${3:-300}
    
    echo -e "${YELLOW}⏳ Waiting for deployment $deployment in namespace $namespace...${NC}"
    
    # Check if deployment exists first, retry if not
    local retries=0
    while [ $retries -lt 10 ]; do
        if kubectl get deployment "$deployment" -n "$namespace" >/dev/null 2>&1; then
            break
        fi
        echo -e "${YELLOW}   Deployment not found, waiting... (retry $retries/10)${NC}"
        sleep 10
        retries=$((retries + 1))
    done
    
    local count=0
    while [ $count -lt $timeout ]; do
        local ready=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
        local desired=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo 1)
        
        if [ "${ready:-0}" -eq "${desired:-1}" ] && [ "$ready" -gt 0 ]; then
            echo -e "${GREEN}✅ Deployment $deployment ready!${NC}"
            return 0
        fi
        
        if [ $((count % 30)) -eq 0 ]; then
            echo -e "${YELLOW}   Still waiting... ($count/${timeout}s) - $ready/$desired replicas ready${NC}"
        fi
        
        sleep 5
        count=$((count + 5))
    done
    
    echo -e "${YELLOW}⚠️ Deployment $deployment not ready in time, continuing anyway${NC}"
    return 0
}