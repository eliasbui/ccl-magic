package department

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// AutoScaler handles dynamic scaling of department members
type AutoScaler struct {
	config    AutoScalingConfig
	manager   *Manager
	isRunning bool
	mu        sync.RWMutex

	// Scaling state
	lastScaleTime map[string]time.Time
	scaleCooldown map[string]time.Time

	// Control
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAutoScaler creates a new auto-scaler
func NewAutoScaler(config AutoScalingConfig, manager *Manager) *AutoScaler {
	ctx, cancel := context.WithCancel(context.Background())

	return &AutoScaler{
		config:        config,
		manager:       manager,
		lastScaleTime: make(map[string]time.Time),
		scaleCooldown: make(map[string]time.Time),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start begins the auto-scaling process
func (as *AutoScaler) Start(ctx context.Context) {
	slog.Info("Starting auto-scaler", "interval", as.config.CheckInterval)

	ticker := time.NewTicker(as.config.CheckInterval)
	defer ticker.Stop()

	as.isRunning = true

	for {
		select {
		case <-ctx.Done():
			slog.Info("Auto-scaler stopped")
			return
		case <-as.ctx.Done():
			return
		case <-ticker.C:
			as.checkAndScale()
		}
	}
}

// Stop stops the auto-scaler
func (as *AutoScaler) Stop() {
	as.mu.Lock()
	defer as.mu.Unlock()

	as.isRunning = false
	as.cancel()
}

// checkAndScale evaluates all departments and scales them if needed
func (as *AutoScaler) checkAndScale() {
	as.mu.Lock()
	defer as.mu.Unlock()

	if !as.isRunning {
		return
	}

	departments := as.manager.ListDepartments()
	now := time.Now()

	for _, dept := range departments {
		if !dept.AutoScale {
			continue
		}

		// Check cooldown period
		if cooldown, exists := as.scaleCooldown[dept.ID]; exists {
			if now.Sub(cooldown) < as.config.CooldownPeriod {
				continue
			}
		}

		// Evaluate scaling needs
		action := as.evaluateScalingNeeds(dept)
		if action != "none" {
			as.executeScalingAction(dept, action)
			as.scaleCooldown[dept.ID] = now
		}
	}
}

// evaluateScalingNeeds determines if a department needs to scale up or down
func (as *AutoScaler) evaluateScalingNeeds(dept *Department) string {
	stats, err := as.manager.GetDepartmentStats(dept.ID)
	if err != nil {
		slog.Warn("Failed to get department stats for scaling evaluation",
			"department", dept.ID,
			"error", err)
		return "none"
	}

	// Calculate utilization metrics
	totalCapacity := stats.ActiveMembers * 5 // Assume 5 tasks per member average
	activeTasks := as.countActiveTasks(dept.ID)
	utilization := float64(activeTasks) / float64(totalCapacity)

	slog.Debug("Department utilization",
		"department", dept.ID,
		"active_members", stats.ActiveMembers,
		"total_capacity", totalCapacity,
		"active_tasks", activeTasks,
		"utilization", utilization)

	// Scale up if utilization is high
	if utilization > as.config.ScaleUpThreshold {
		if stats.ActiveMembers < as.config.MaxMembersPerDept {
			if len(as.membersByRole(dept.ID)) < dept.MaxMembers {
				return "scale_up"
			}
		}
	}

	// Scale down if utilization is low
	if utilization < as.config.ScaleDownThreshold {
		if stats.ActiveMembers > dept.MinMembers {
			return "scale_down"
		}
	}

	return "none"
}

// executeScalingAction performs the actual scaling
func (as *AutoScaler) executeScalingAction(dept *Department, action string) {
	switch action {
	case "scale_up":
		as.scaleUp(dept)
	case "scale_down":
		as.scaleDown(dept)
	}

	as.lastScaleTime[dept.ID] = time.Now()
}

// scaleUp adds a new member to the department
func (as *AutoScaler) scaleUp(dept *Department) {
	// Determine which role to add based on current needs
	role := as.determineRoleToAdd(dept)
	if role == "" {
		slog.Info("Cannot determine role to add", "department", dept.ID)
		return
	}

	// Create a new member configuration
	member := &Member{
		ID:              fmt.Sprintf("member-%s-%d", dept.ID, time.Now().Unix()),
		Name:            fmt.Sprintf("Auto-Scaled %s", role),
		Role:            MemberRole(role),
		DepartmentID:    dept.ID,
		DepartmentType:  dept.Type,
		Status:          MemberStatusOnline,
		Specializations: as.getRoleSpecializations(role),
		CurrentTasks:    []string{},
		MaxConcurrent:   as.getRoleMaxConcurrent(role),
		Endpoint:        fmt.Sprintf("http://localhost:8080/members/%s", dept.ID),
		AuthMethod:      "api-key",
		HealthScore:     1.0,
		Performance:     make(map[string]float64),
		Capabilities:    as.getRoleCapabilities(role),
		IsLead:          isLeadRole(MemberRole(role)),
		Metadata: map[string]string{
			"auto_scaled":    "true",
			"created_at":     time.Now().Format(time.RFC3339),
			"scaling_reason": "high_utilization",
		},
	}

	// Register the new member
	if err := as.manager.RegisterMember(context.Background(), member); err != nil {
		slog.Error("Failed to register auto-scaled member",
			"department", dept.ID,
			"role", role,
			"error", err)
		return
	}

	slog.Info("Auto-scaled up department",
		"department", dept.ID,
		"member_id", member.ID,
		"role", role)
}

// scaleDown removes a member from the department
func (as *AutoScaler) scaleDown(dept *Department) {
	// Find a member that can be safely removed
	candidate := as.findScaleDownCandidate(dept)
	if candidate == nil {
		slog.Info("No suitable candidate for scale down", "department", dept.ID)
		return
	}

	// Ensure member has no active tasks
	if len(candidate.CurrentTasks) > 0 {
		slog.Info("Cannot scale down: candidate has active tasks",
			"department", dept.ID,
			"member_id", candidate.ID,
			"active_tasks", len(candidate.CurrentTasks))
		return
	}

	// Unregister the member
	if err := as.manager.UnregisterMember(context.Background(), candidate.ID); err != nil {
		slog.Error("Failed to unregister member during scale down",
			"department", dept.ID,
			"member_id", candidate.ID,
			"error", err)
		return
	}

	slog.Info("Auto-scaled down department",
		"department", dept.ID,
		"member_id", candidate.ID,
		"role", string(candidate.Role))
}

// determineRoleToAdd decides which role should be added to a department
func (as *AutoScaler) determineRoleToAdd(dept *Department) string {
	// Check role-specific scaling rules
	if as.config.RoleScaling != nil {
		currentRoles := as.membersByRole(dept.ID)

		// Find roles that need more members
		for role, desiredCount := range as.config.RoleScaling {
			currentCount := 0
			for _, memberRole := range currentRoles {
				if memberRole == role {
					currentCount++
				}
			}

			if currentCount < desiredCount {
				return role
			}
		}
	}

	// Default role mapping by department
	roleMap := map[DepartmentType][]string{
		DepartmentDevelopment: {"developer", "lead_dev", "developer"},
		DepartmentDevOps:       {"devops", "devops"},
		DepartmentSecurity:     {"security", "security"},
		DepartmentQA:          {"qa", "lead_test", "qa"},
	}

	if roles, exists := roleMap[dept.Type]; exists {
		// Return the role with the fewest members
		roleCounts := as.membersByRole(dept.ID)
		minCount := 999
		selectedRole := ""

		for _, role := range roles {
			count := 0
			for _, memberRole := range roleCounts {
				if memberRole == role {
					count++
				}
			}
			if count < minCount {
				minCount = count
				selectedRole = role
			}
		}

		return selectedRole
	}

	return ""
}

// findScaleDownCandidate finds a member that can be safely removed
func (as *AutoScaler) findScaleDownCandidate(dept *Department) *Member {
	members := as.manager.ListMembers(dept.ID)

	// Prefer non-lead, auto-scaled members with no active tasks
	var candidates []*Member

	for _, member := range members {
		// Skip lead members if there are other members
		if member.IsLead && len(members) > dept.MinMembers {
			continue
		}

		// Prefer auto-scaled members
		if member.Metadata["auto_scaled"] != "true" {
			continue
		}

		// Must have no active tasks
		if len(member.CurrentTasks) == 0 {
			candidates = append(candidates, member)
		}
	}

	// If no auto-scaled candidates, consider any non-lead with no tasks
	if len(candidates) == 0 {
		for _, member := range members {
			if !member.IsLead && len(member.CurrentTasks) == 0 {
				candidates = append(candidates, member)
			}
		}
	}

	// Select the newest member (most likely to be auto-scaled)
	if len(candidates) > 0 {
		var newest *Member
		for _, candidate := range candidates {
			if newest == nil || candidate.JoinedAt.After(newest.JoinedAt) {
				newest = candidate
			}
		}
		return newest
	}

	return nil
}

// Helper functions

func (as *AutoScaler) countActiveTasks(departmentID string) int {
	tasks := as.manager.ListTasks(departmentID, TaskStatusInProgress)
	return len(tasks)
}

func (as *AutoScaler) membersByRole(departmentID string) []string {
	members := as.manager.ListMembers(departmentID)
	var roles []string

	for _, member := range members {
		roles = append(roles, string(member.Role))
	}

	return roles
}

func (as *AutoScaler) getRoleSpecializations(role string) []string {
	specializations := map[string][]string{
		"ba":           {"requirements", "analysis", "user-stories", "business-process"},
		"pm":           {"planning", "coordination", "risk-management", "stakeholder-management"},
		"po":           {"product-vision", "prioritization", "backlog-management", "user-needs"},
		"lead_technical": {"architecture", "technical-leadership", "code-review", "mentoring"},
		"lead_ba":      {"business-analysis", "requirements-elicitation", "stakeholder-communication"},
		"lead_dev":     {"development", "code-quality", "technical-mentoring", "team-leadership"},
		"lead_test":    {"testing-strategy", "quality-assurance", "test-automation", "team-mentoring"},
		"developer":    {"coding", "debugging", "unit-testing", "code-review"},
		"devops":       {"ci-cd", "deployment", "infrastructure", "monitoring"},
		"qa":           {"testing", "test-automation", "quality-assurance", "bug-reporting"},
		"security":     {"security-analysis", "vulnerability-assessment", "compliance", "penetration-testing"},
	}

	if specs, exists := specializations[role]; exists {
		return specs
	}

	return []string{"general"}
}

func (as *AutoScaler) getRoleMaxConcurrent(role string) int {
	concurrency := map[string]int{
		"ba":            3,
		"pm":            5,
		"po":            4,
		"lead_technical": 2,
		"lead_ba":       2,
		"lead_dev":      2,
		"lead_test":     2,
		"developer":     3,
		"devops":        4,
		"qa":            4,
		"security":      3,
	}

	if max, exists := concurrency[role]; exists {
		return max
	}

	return 3
}

func (as *AutoScaler) getRoleCapabilities(role string) map[string]interface{} {
	capabilities := map[string]map[string]interface{}{
		"ba": {
			"requirements_analysis": true,
			"user_stories":        true,
			"process_modeling":     true,
		},
		"pm": {
			"project_planning":   true,
			"risk_assessment":    true,
			"resource_management": true,
		},
		"po": {
			"product_vision":     true,
			"backlog_management": true,
			"stakeholder_management": true,
		},
		"lead_technical": {
			"architecture_design": true,
			"code_review":       true,
			"technical_mentoring": true,
		},
		"lead_dev": {
			"development":        true,
			"code_review":        true,
			"team_coordination":  true,
		},
		"lead_test": {
			"test_strategy":      true,
			"quality_assurance":  true,
			"team_mentoring":     true,
		},
		"developer": {
			"coding":             true,
			"debugging":          true,
			"unit_testing":       true,
		},
		"devops": {
			"ci_cd":              true,
			"deployment":         true,
			"infrastructure":     true,
		},
		"qa": {
			"testing":            true,
			"test_automation":    true,
			"quality_assurance":  true,
		},
		"security": {
			"security_analysis":  true,
			"vulnerability_assessment": true,
			"compliance":         true,
		},
	}

	if caps, exists := capabilities[role]; exists {
		return caps
	}

	return map[string]interface{}{"general": true}
}

// GetScalingStatus returns the current scaling status
func (as *AutoScaler) GetScalingStatus() map[string]interface{} {
	as.mu.RLock()
	defer as.mu.RUnlock()

	status := make(map[string]interface{})
	status["is_running"] = as.isRunning
	status["last_scale_times"] = as.lastScaleTime
	status["scale_cooldowns"] = as.scaleCooldown
	status["config"] = as.config

	return status
}