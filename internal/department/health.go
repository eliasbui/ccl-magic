package department

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// HealthChecker monitors the health of department members
type HealthChecker struct {
	config  HealthCheckConfig
	manager *Manager
	client  *http.Client

	// Health tracking
	healthStatus map[string]*MemberHealth
	mu           sync.RWMutex

	// Control
	ctx    context.Context
	cancel context.CancelFunc
}

// MemberHealth tracks the health status of a member
type MemberHealth struct {
	MemberID        string    `json:"member_id"`
	Status          string    `json:"status"`
	LastCheck       time.Time `json:"last_check"`
	ResponseTime    float64   `json:"response_time"`
	SuccessRate     float64   `json:"success_rate"`
	FailedChecks    int       `json:"failed_checks"`
	ConsecutiveFails int      `json:"consecutive_fails"`
	IsHealthy       bool      `json:"is_healthy"`
	LastError       string    `json:"last_error,omitempty"`
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(config HealthCheckConfig, manager *Manager) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())

	return &HealthChecker{
		config:       config,
		manager:      manager,
		client:       &http.Client{Timeout: config.Timeout},
		healthStatus: make(map[string]*MemberHealth),
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Start begins the health checking process
func (h *HealthChecker) Start(ctx context.Context) {
	slog.Info("Starting health checker", "interval", h.config.CheckInterval)

	ticker := time.NewTicker(h.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("Health checker stopped")
			return
		case <-h.ctx.Done():
			return
		case <-ticker.C:
			h.performHealthCheck()
		}
	}
}

// Stop stops the health checker
func (h *HealthChecker) Stop() {
	h.cancel()
}

// performHealthCheck checks the health of all registered members
func (h *HealthChecker) performHealthCheck() {
	members := h.manager.ListMembers("")

	var wg sync.WaitGroup
	for _, member := range members {
		if member.Status == MemberStatusOffline {
			continue
		}

		wg.Add(1)
		go func(m *Member) {
			defer wg.Done()
			h.checkMemberHealth(m)
		}(member)
	}

	wg.Wait()
}

// checkMemberHealth performs a health check on a single member
func (h *HealthChecker) checkMemberHealth(member *Member) {
	start := time.Now()

	// Perform the actual health check
	healthy, responseTime, err := h.pingMember(member)

	checkTime := time.Now()

	h.mu.Lock()
	defer h.mu.Unlock()

	health, exists := h.healthStatus[member.ID]
	if !exists {
		health = &MemberHealth{
			MemberID: member.ID,
		}
		h.healthStatus[member.ID] = health
	}

	// Update health status
	health.LastCheck = checkTime
	health.ResponseTime = responseTime

	if healthy {
		health.FailedChecks = 0
		health.ConsecutiveFails = 0
		health.IsHealthy = true
		health.Status = "healthy"
		health.LastError = ""

		// Update member status if it was unhealthy
		if member.Status == MemberStatusUnhealthy {
			h.manager.UpdateMemberStatus(context.Background(), member.ID, MemberStatusOnline)
		}
	} else {
		health.FailedChecks++
		health.ConsecutiveFails++
		health.IsHealthy = false
		health.Status = "unhealthy"

		if err != nil {
			health.LastError = err.Error()
		}

		// Mark member as unhealthy if threshold is reached
		if health.ConsecutiveFails >= h.config.UnhealthyThreshold {
			h.manager.UpdateMemberStatus(context.Background(), member.ID, MemberStatusUnhealthy)
			slog.Warn("Member marked as unhealthy",
				"member_id", member.ID,
				"consecutive_failures", health.ConsecutiveFails,
				"last_error", health.LastError)
		}
	}

	// Calculate success rate based on recent checks
	h.calculateSuccessRate(member.ID)
}

// pingMember sends a health check request to a member
func (h *HealthChecker) pingMember(member *Member) (bool, float64, error) {
	start := time.Now()

	// Create health check URL
	healthURL := fmt.Sprintf("%s/health", member.Endpoint)

	// Create request
	req, err := http.NewRequestWithContext(context.Background(), "GET", healthURL, nil)
	if err != nil {
		return false, 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication headers if needed
	if member.AuthMethod != "" {
		switch member.AuthMethod {
		case "bearer":
			// In a real implementation, you'd get the token from a secure source
			req.Header.Set("Authorization", "Bearer "+member.ID)
		case "api-key":
			req.Header.Set("X-API-Key", member.ID)
		}
	}

	// Perform the request
	resp, err := h.client.Do(req)
	if err != nil {
		return false, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseTime := time.Since(start).Seconds()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return false, responseTime, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse response body
	var healthResp struct {
		Status  string                 `json:"status"`
		Metrics map[string]interface{} `json:"metrics,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&healthResp); err != nil {
		return false, responseTime, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check if member reports as healthy
	if healthResp.Status != "healthy" && healthResp.Status != "ok" {
		return false, responseTime, fmt.Errorf("member reports status: %s", healthResp.Status)
	}

	// Apply role-specific health checks
	if !h.checkRoleSpecificHealth(member, healthResp.Metrics) {
		return false, responseTime, fmt.Errorf("role-specific health check failed")
	}

	return true, responseTime, nil
}

// checkRoleSpecificHealth applies role-specific health criteria
func (h *HealthChecker) checkRoleSpecificHealth(member *Member, metrics map[string]interface{}) bool {
	roleChecks, exists := h.config.RoleSpecificChecks[string(member.Role)]
	if !exists {
		return true // No specific checks for this role
	}

	// Check response time
	if roleChecks.ResponseTime > 0 {
		if responseTime, ok := metrics["response_time"].(float64); ok {
			if responseTime > roleChecks.ResponseTime.Seconds() {
				return false
			}
		}
	}

	// Check task success rate
	if roleChecks.TaskSuccess > 0 {
		if successRate, ok := metrics["task_success_rate"].(float64); ok {
			if successRate < roleChecks.TaskSuccess {
				return false
			}
		}
	}

	// Check uptime
	if roleChecks.Uptime > 0 {
		if uptime, ok := metrics["uptime"].(float64); ok {
			if uptime < roleChecks.Uptime {
				return false
			}
		}
	}

	return true
}

// calculateSuccessRate calculates the success rate for a member
func (h *HealthChecker) calculateSuccessRate(memberID string) {
	health := h.healthStatus[memberID]

	// Get member statistics
	memberStats, err := h.manager.GetMemberStats(memberID)
	if err != nil {
		return
	}

	if memberStats.TotalTasks > 0 {
		health.SuccessRate = float64(memberStats.CompletedTasks) / float64(memberStats.TotalTasks)
	}
}

// GetMemberHealth returns the health status of a member
func (h *HealthChecker) GetMemberHealth(memberID string) (*MemberHealth, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	health, exists := h.healthStatus[memberID]
	if !exists {
		return nil, fmt.Errorf("no health data for member %s", memberID)
	}

	return health, nil
}

// GetAllHealthStatus returns the health status of all members
func (h *HealthChecker) GetAllHealthStatus() map[string]*MemberHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*MemberHealth)
	for id, health := range h.healthStatus {
		result[id] = health
	}
	return result
}

// GetHealthyMembers returns a list of healthy members
func (h *HealthChecker) GetHealthyMembers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var healthy []string
	for id, health := range h.healthStatus {
		if health.IsHealthy {
			healthy = append(healthy, id)
		}
	}
	return healthy
}

// GetUnhealthyMembers returns a list of unhealthy members
func (h *HealthChecker) GetUnhealthyMembers() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var unhealthy []string
	for id, health := range h.healthStatus {
		if !health.IsHealthy {
			unhealthy = append(unhealthy, id)
		}
	}
	return unhealthy
}