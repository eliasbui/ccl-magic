package department

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sort"
	"strings"
	"time"
)

// TaskRouter handles intelligent task routing to appropriate members
type TaskRouter struct {
	config  TaskRoutingConfig
	manager *Manager
}

// NewTaskRouter creates a new task router
func NewTaskRouter(config TaskRoutingConfig, manager *Manager) *TaskRouter {
	return &TaskRouter{
		config:  config,
		manager: manager,
	}
}

// RouteTask assigns a task to the most appropriate member
func (tr *TaskRouter) RouteTask(ctx context.Context, task *Task) error {
	// Determine target department if not specified
	if task.DepartmentID == "" {
		deptID, err := tr.determineDepartment(task)
		if err != nil {
			return fmt.Errorf("failed to determine department: %w", err)
		}
		task.DepartmentID = deptID
	}

	// Find suitable members
	candidates, err := tr.findSuitableMembers(task)
	if err != nil {
		return fmt.Errorf("failed to find suitable members: %w", err)
	}

	if len(candidates) == 0 {
		if tr.config.FallbackEnabled {
			return tr.fallbackRouting(task)
		}
		return fmt.Errorf("no suitable members found for task %s", task.ID)
	}

	// Select member based on routing strategy
	selectedMember, err := tr.selectMember(task, candidates)
	if err != nil {
		return fmt.Errorf("failed to select member: %w", err)
	}

	// Assign task to member
	return tr.assignTaskToMember(task, selectedMember)
}

// determineDepartment determines the best department for a task
func (tr *TaskRouter) determineDepartment(task *Task) (string, error) {
	// Check department-specific rules
	for deptID, keywords := range tr.config.DepartmentRules {
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(task.Description), strings.ToLower(keyword)) ||
				strings.Contains(strings.ToLower(task.Title), strings.ToLower(keyword)) {
				return deptID, nil
			}
		}
	}

	// Check task type mappings
	taskTypeDept := map[string]string{
		"development":    "dept-dev",
		"coding":         "dept-dev",
		"code-review":    "dept-dev",
		"bug":            "dept-dev",
		"feature":        "dept-dev",
		"deployment":     "dept-devops",
		"ci-cd":          "dept-devops",
		"infrastructure": "dept-devops",
		"monitoring":     "dept-devops",
		"security":       "dept-security",
		"compliance":     "dept-security",
		"audit":          "dept-security",
		"testing":        "dept-qa",
		"qa":             "dept-qa",
		"test":           "dept-qa",
		"performance":    "dept-qa",
	}

	if deptID, exists := taskTypeDept[task.Type]; exists {
		return deptID, nil
	}

	// Use default department
	if tr.config.DefaultDepartment != "" {
		return tr.config.DefaultDepartment, nil
	}

	return "", fmt.Errorf("cannot determine department for task %s", task.ID)
}

// findSuitableMembers finds members capable of handling the task
func (tr *TaskRouter) findSuitableMembers(task *Task) ([]*Member, error) {
	// Get all members in the target department
	members := tr.manager.ListMembers(task.DepartmentID)
	if len(members) == 0 {
		return nil, fmt.Errorf("no members in department %s", task.DepartmentID)
	}

	var suitable []*Member

	for _, member := range members {
		if tr.isMemberSuitable(member, task) {
			suitable = append(suitable, member)
		}
	}

	return suitable, nil
}

// isMemberSuitable checks if a member is suitable for a task
func (tr *TaskRouter) isMemberSuitable(member *Member, task *Task) bool {
	// Check member status
	if member.Status != MemberStatusOnline && member.Status != MemberStatusBusy {
		return false
	}

	// Check if member has capacity
	if len(member.CurrentTasks) >= member.MaxConcurrent {
		return false
	}

	// Check role-specific rules
	if tr.config.RoleRules != nil {
		if rules, exists := tr.config.RoleRules[string(member.Role)]; exists {
			for _, keyword := range rules {
				if !strings.Contains(strings.ToLower(task.Description), strings.ToLower(keyword)) &&
					!strings.Contains(strings.ToLower(task.Title), strings.ToLower(keyword)) {
					return false
				}
			}
		}
	}

	// Check required skills
	if len(task.RequiredSkills) > 0 {
		hasRequiredSkills := true
		for _, skill := range task.RequiredSkills {
			found := false
			for _, memberSkill := range member.Specializations {
				if strings.EqualFold(memberSkill, skill) {
					found = true
					break
				}
			}
			if !found {
				hasRequiredSkills = false
				break
			}
		}
		if !hasRequiredSkills {
			return false
		}
	}

	// Check if role is assigned or if we need to assign one
	if task.AssignedRole != "" && member.Role != task.AssignedRole {
		return false
	}

	return true
}

// selectMember selects the best member based on the routing strategy
func (tr *TaskRouter) selectMember(task *Task, candidates []*Member) (*Member, error) {
	switch tr.config.Strategy {
	case "round-robin":
		return tr.selectRoundRobin(candidates)
	case "load-based":
		return tr.selectByLoad(candidates)
	case "skill-based":
		return tr.selectBySkill(task, candidates)
	case "role-based":
		return tr.selectByRole(task, candidates)
	default:
		return tr.selectByLoad(candidates)
	}
}

// selectRoundRobin selects members in a round-robin fashion
func (tr *TaskRouter) selectRoundRobin(candidates []*Member) (*Member, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates available")
	}

	// Simple round-robin based on current load
	sort.Slice(candidates, func(i, j int) bool {
		return len(candidates[i].CurrentTasks) < len(candidates[j].CurrentTasks)
	})

	return candidates[0], nil
}

// selectByLoad selects the member with the lowest current load
func (tr *TaskRouter) selectByLoad(candidates []*Member) (*Member, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates available")
	}

	var selected *Member
	minLoad := 999

	for _, member := range candidates {
		currentLoad := len(member.CurrentTasks)
		if currentLoad < minLoad {
			minLoad = currentLoad
			selected = member
		}
	}

	return selected, nil
}

// selectBySkill selects the member with the best matching skills
func (tr *TaskRouter) selectBySkill(task *Task, candidates []*Member) (*Member, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates available")
	}

	// Calculate skill match scores
	type memberScore struct {
		member *Member
		score  int
	}

	var scores []memberScore

	for _, member := range candidates {
		score := 0

		// Score based on required skills
		for _, skill := range task.RequiredSkills {
			for _, memberSkill := range member.Specializations {
				if strings.EqualFold(memberSkill, skill) {
					score += 10
					break
				}
			}
		}

		// Score based on current load (lower load = higher score)
		score += (member.MaxConcurrent - len(member.CurrentTasks)) * 2

		// Score based on performance
		if stats, err := tr.manager.GetMemberStats(member.ID); err == nil {
			score += int(stats.SuccessRate * 5)
		}

		scores = append(scores, memberScore{member: member, score: score})
	}

	// Sort by score (highest first)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	return scores[0].member, nil
}

// selectByRole selects a member based on role requirements
func (tr *TaskRouter) selectByRole(task *Task, candidates []*Member) (*Member, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidates available")
	}

	// If task requires a specific role, filter by that role first
	if task.AssignedRole != "" {
		var roleCandidates []*Member
		for _, member := range candidates {
			if member.Role == task.AssignedRole {
				roleCandidates = append(roleCandidates, member)
			}
		}
		if len(roleCandidates) > 0 {
			candidates = roleCandidates
		}
	}

	// Prioritize leads for complex tasks
	if strings.Contains(strings.ToLower(task.Description), "lead") ||
		task.Priority == PriorityCritical {
		var leads []*Member
		var others []*Member

		for _, member := range candidates {
			if member.IsLead {
				leads = append(leads, member)
			} else {
				others = append(others, member)
			}
		}

		if len(leads) > 0 {
			candidates = leads
		} else if len(others) > 0 {
			candidates = others
		}
	}

	// Select by load from the filtered candidates
	return tr.selectByLoad(candidates)
}

// assignTaskToMember assigns a task to a member
func (tr *TaskRouter) assignTaskToMember(task *Task, member *Member) error {
	// Update task
	task.AssignedMember = member.ID
	task.AssignedRole = member.Role
	task.Status = TaskStatusAssigned
	task.UpdatedAt = time.Now()

	// Update member
	member.CurrentTasks = append(member.CurrentTasks, task.ID)
	if len(member.CurrentTasks) >= member.MaxConcurrent {
		member.Status = MemberStatusBusy
	}

	// Update member statistics
	if stats, err := tr.manager.GetMemberStats(member.ID); err == nil {
		stats.CurrentLoad = len(member.CurrentTasks)
		stats.LastUpdated = time.Now()
	}

	slog.Info("Task assigned to member",
		"task_id", task.ID,
		"task_title", task.Title,
		"member_id", member.ID,
		"member_name", member.Name,
		"member_role", string(member.Role),
		"department", member.DepartmentID)

	return nil
}

// fallbackRouting provides fallback routing when no suitable members are found
func (tr *TaskRouter) fallbackRouting(task *Task) error {
	// Try to find any available member in any department
	allMembers := tr.manager.ListMembers("")

	var available []*Member
	for _, member := range allMembers {
		if member.Status == MemberStatusOnline && len(member.CurrentTasks) < member.MaxConcurrent {
			available = append(available, member)
		}
	}

	if len(available) == 0 {
		return fmt.Errorf("no available members for fallback routing")
	}

	// Select a member randomly from available ones
	selected := available[rand.Intn(len(available))]

	// Update task department
	task.DepartmentID = selected.DepartmentID

	slog.Warn("Task routed using fallback",
		"task_id", task.ID,
		"task_title", task.Title,
		"fallback_member", selected.ID,
		"fallback_department", selected.DepartmentID)

	return tr.assignTaskToMember(task, selected)
}

// ReassignTask reassigns a task to a different member
func (tr *TaskRouter) ReassignTask(ctx context.Context, taskID string, reason string) error {
	task, err := tr.manager.GetTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Remove from current member
	if task.AssignedMember != "" {
		member, err := tr.manager.GetMember(task.AssignedMember)
		if err == nil {
			// Remove task from member's current tasks
			for i, currentTask := range member.CurrentTasks {
				if currentTask == taskID {
					member.CurrentTasks = append(member.CurrentTasks[:i], member.CurrentTasks[i+1:]...)
					break
				}
			}

			// Update member status if no longer busy
			if len(member.CurrentTasks) < member.MaxConcurrent {
				member.Status = MemberStatusOnline
			}
		}
	}

	// Reset task assignment
	task.AssignedMember = ""
	task.AssignedRole = ""
	task.Status = TaskStatusQueued
	task.UpdatedAt = time.Now()

	// Route to new member
	if err := tr.RouteTask(ctx, task); err != nil {
		return fmt.Errorf("failed to reassign task: %w", err)
	}

	slog.Info("Task reassigned",
		"task_id", taskID,
		"task_title", task.Title,
		"reason", reason)

	return nil
}