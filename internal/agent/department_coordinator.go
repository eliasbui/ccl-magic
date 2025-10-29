package agent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"charm.land/fantasy"
	"github.com/eliasbui/ccl-magic/internal/agent/prompt"
	"github.com/eliasbui/ccl-magic/internal/config"
	"github.com/eliasbui/ccl-magic/internal/csync"
	"github.com/eliasbui/ccl-magic/internal/department"
	"github.com/eliasbui/ccl-magic/internal/history"
	"github.com/eliasbui/ccl-magic/internal/lsp"
	"github.com/eliasbui/ccl-magic/internal/message"
	"github.com/eliasbui/ccl-magic/internal/permission"
	"github.com/eliasbui/ccl-magic/internal/pubsub"
	"github.com/eliasbui/ccl-magic/internal/session"
)

// DepartmentCoordinator extends the base coordinator to support department management
type DepartmentCoordinator struct {
	*coordinator // Embedded base coordinator

	departmentManager *department.Manager
	config           *config.Config
}

// NewDepartmentCoordinator creates a new coordinator with department management capabilities
func NewDepartmentCoordinator(
	ctx context.Context,
	cfg *config.Config,
	sessions session.Service,
	messages message.Service,
	permissions permission.Service,
	history history.Service,
	lspClients *csync.Map[string, *lsp.Client],
) (Coordinator, error) {
	// Create base coordinator
	baseCoord := &coordinator{
		cfg:         cfg,
		sessions:    sessions,
		messages:    messages,
		permissions: permissions,
		history:     history,
		lspClients:  lspClients,
		agents:      make(map[string]SessionAgent),
	}

	deptCoord := &DepartmentCoordinator{
		coordinator: baseCoord,
		config:      cfg,
	}

	// Initialize department manager if enabled
	if cfg.Department != nil && cfg.Department.Enabled {
		if err := deptCoord.initializeDepartmentManager(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize department manager: %w", err)
		}
	} else {
		// Fall back to regular agent setup
		if err := deptCoord.setupDefaultAgent(ctx); err != nil {
			return nil, fmt.Errorf("failed to setup default agent: %w", err)
		}
	}

	return deptCoord, nil
}

// initializeDepartmentManager sets up the department management system
func (dc *DepartmentCoordinator) initializeDepartmentManager(ctx context.Context) error {
	// Create department manager
	deptManager, err := department.NewManager(ctx, dc.config.Department)
	if err != nil {
		return fmt.Errorf("failed to create department manager: %w", err)
	}
	dc.departmentManager = deptManager

	// Start department manager
	if err := deptManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start department manager: %w", err)
	}

	// Set up event subscriptions
	go dc.handleDepartmentEvents(ctx)
	go dc.handleMemberEvents(ctx)
	go dc.handleTaskEvents(ctx)

	slog.Info("Department coordinator initialized", "departments_enabled", true)

	return nil
}

// setupDefaultAgent sets up the default agent when department management is disabled
func (dc *DepartmentCoordinator) setupDefaultAgent(ctx context.Context) error {
	agentCfg, ok := dc.config.Agents[config.AgentCoder]
	if !ok {
		return fmt.Errorf("coder agent not configured")
	}

	prompt, err := coderPrompt(prompt.WithWorkingDir(dc.config.WorkingDir()))
	if err != nil {
		return fmt.Errorf("failed to create coder prompt: %w", err)
	}

	agent, err := dc.buildAgent(ctx, prompt, agentCfg)
	if err != nil {
		return fmt.Errorf("failed to build agent: %w", err)
	}

	dc.currentAgent = agent
	dc.agents[config.AgentCoder] = agent

	slog.Info("Default agent coordinator initialized", "departments_enabled", false)

	return nil
}

// Run implements Coordinator interface with department routing
func (dc *DepartmentCoordinator) Run(ctx context.Context, sessionID, prompt string, attachments ...message.Attachment) (*fantasy.AgentResult, error) {
	// If department management is enabled, try to route through department system
	if dc.departmentManager != nil {
		return dc.runWithDepartmentRouting(ctx, sessionID, prompt, attachments...)
	}

	// Fall back to base coordinator behavior
	return dc.coordinator.Run(ctx, sessionID, prompt, attachments...)
}

// runWithDepartmentRouting routes the request through the department system
func (dc *DepartmentCoordinator) runWithDepartmentRouting(ctx context.Context, sessionID, prompt string, attachments ...message.Attachment) (*fantasy.AgentResult, error) {
	// Create a task from the user request
	task := &department.Task{
		Title:          extractTaskTitle(prompt),
		Description:    prompt,
		Type:          determineTaskType(prompt),
		Priority:       determineTaskPriority(prompt),
		RequestedBy:    "user",
		DepartmentID:   "", // Will be determined by task router
		Attachments:    convertAttachments(attachments),
		RequiredSkills: extractRequiredSkills(prompt),
	}

	// Create task through department manager
	createdTask, err := dc.departmentManager.CreateTask(ctx, task)
	if err != nil {
		slog.Warn("Failed to create department task, falling back to base coordinator", "error", err)
		return dc.coordinator.Run(ctx, sessionID, prompt, attachments...)
	}

	// Wait for task assignment and execution
	return dc.waitForTaskCompletion(ctx, sessionID, createdTask.ID, prompt, attachments...)
}

// waitForTaskCompletion waits for a department task to be completed and returns the result
func (dc *DepartmentCoordinator) waitForTaskCompletion(ctx context.Context, sessionID, taskID, prompt string, attachments ...message.Attachment) (*fantasy.AgentResult, error) {
	// Subscribe to task events
	taskEvents := dc.departmentManager.SubscribeToTaskEvents(ctx)
	defer close(taskEvents)

	// Poll for task completion
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(30 * time.Minute) // 30 minute timeout
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-timeout.C:
			return nil, fmt.Errorf("task %s timed out", taskID)

		case <-ticker.C:
			task, err := dc.departmentManager.GetTask(taskID)
			if err != nil {
				continue
			}

			switch task.Status {
			case department.TaskStatusCompleted:
				return dc.createResultFromTask(task), nil

			case department.TaskStatusFailed:
				return nil, fmt.Errorf("task %s failed: %s", taskID, task.Results["error"])

			case department.TaskStatusAssigned:
				// Task is assigned, execute it through the appropriate member
				if task.AssignedMember != "" {
					return dc.executeTaskForMember(ctx, sessionID, task, prompt, attachments...)
				}

			default:
				// Continue waiting
			}

		case event := <-taskEvents:
			if event.Payload.ID == taskID {
				switch event.Type {
				case pubsub.UpdatedEvent:
					if event.Payload.Status == department.TaskStatusCompleted {
						return dc.createResultFromTask(event.Payload), nil
					}
					if event.Payload.Status == department.TaskStatusFailed {
						return nil, fmt.Errorf("task failed: %s", event.Payload.Results["error"])
					}
				}
			}
		}
	}
}

// executeTaskForMember executes a task using a specific department member
func (dc *DepartmentCoordinator) executeTaskForMember(ctx context.Context, sessionID string, task *department.Task, prompt string, attachments ...message.Attachment) (*fantasy.AgentResult, error) {
	// Get the member assigned to the task
	member, err := dc.departmentManager.GetMember(task.AssignedMember)
	if err != nil {
		return nil, fmt.Errorf("failed to get assigned member: %w", err)
	}

	// Update task status to in progress
	if err := dc.departmentManager.UpdateTaskStatus(ctx, task.ID, department.TaskStatusInProgress, nil); err != nil {
		slog.Warn("Failed to update task status", "error", err)
	}

	// Execute the task using the base coordinator
	result, err := dc.coordinator.Run(ctx, sessionID, prompt, attachments...)
	if err != nil {
		// Mark task as failed
		updateErr := dc.departmentManager.UpdateTaskStatus(ctx, task.ID, department.TaskStatusFailed, map[string]interface{}{
			"error": err.Error(),
		})
		if updateErr != nil {
			slog.Warn("Failed to update task status to failed", "error", updateErr)
		}
		return nil, err
	}

	// Mark task as completed with results
	taskResults := map[string]interface{}{
		"response":    result.Content,
		"tool_calls":  result.ToolCalls,
		"member_id":   member.ID,
		"member_role": string(member.Role),
		"execution_time": time.Now().Format(time.RFC3339),
	}

	if err := dc.departmentManager.UpdateTaskStatus(ctx, task.ID, department.TaskStatusCompleted, taskResults); err != nil {
		slog.Warn("Failed to update task status to completed", "error", err)
	}

	return result, nil
}

// createResultFromTask creates a fantasy.AgentResult from a completed task
func (dc *DepartmentCoordinator) createResultFromTask(task *department.Task) *fantasy.AgentResult {
	content, _ := task.Results["response"].(string)

	var toolCalls []fantasy.ToolCall
	if calls, ok := task.Results["tool_calls"].([]fantasy.ToolCall); ok {
		toolCalls = calls
	}

	return &fantasy.AgentResult{
		Content:   content,
		ToolCalls: toolCalls,
	}
}

// handleDepartmentEvents handles department-related events
func (dc *DepartmentCoordinator) handleDepartmentEvents(ctx context.Context) {
	if dc.departmentManager == nil {
		return
	}

	events := dc.departmentManager.SubscribeToDepartmentEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			dc.processDepartmentEvent(event)
		}
	}
}

// handleMemberEvents handles member-related events
func (dc *DepartmentCoordinator) handleMemberEvents(ctx context.Context) {
	if dc.departmentManager == nil {
		return
	}

	events := dc.departmentManager.SubscribeToMemberEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			dc.processMemberEvent(event)
		}
	}
}

// handleTaskEvents handles task-related events
func (dc *DepartmentCoordinator) handleTaskEvents(ctx context.Context) {
	if dc.departmentManager == nil {
		return
	}

	events := dc.departmentManager.SubscribeToTaskEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case event := <-events:
			dc.processTaskEvent(event)
		}
	}
}

// processDepartmentEvent processes department events
func (dc *DepartmentCoordinator) processDepartmentEvent(event pubsub.Event[*department.Department]) {
	dept := event.Payload

	switch event.Type {
	case pubsub.CreatedEvent:
		slog.Info("Department created", "department_id", dept.ID, "name", dept.Name)
	case pubsub.UpdatedEvent:
		slog.Info("Department updated", "department_id", dept.ID, "name", dept.Name)
	case pubsub.DeletedEvent:
		slog.Info("Department deleted", "department_id", dept.ID, "name", dept.Name)
	}
}

// processMemberEvent processes member events
func (dc *DepartmentCoordinator) processMemberEvent(event pubsub.Event[*department.Member]) {
	member := event.Payload

	switch event.Type {
	case pubsub.CreatedEvent:
		slog.Info("Member joined",
			"member_id", member.ID,
			"name", member.Name,
			"role", string(member.Role),
			"department", member.DepartmentID)
	case pubsub.UpdatedEvent:
		slog.Info("Member updated",
			"member_id", member.ID,
			"status", string(member.Status),
			"current_tasks", len(member.CurrentTasks))
	case pubsub.DeletedEvent:
		slog.Info("Member left", "member_id", member.ID, "name", member.Name)
	}
}

// processTaskEvent processes task events
func (dc *DepartmentCoordinator) processTaskEvent(event pubsub.Event[*department.Task]) {
	task := event.Payload

	switch event.Type {
	case pubsub.CreatedEvent:
		slog.Info("Task created",
			"task_id", task.ID,
			"title", task.Title,
			"department", task.DepartmentID,
			"priority", string(task.Priority))
	case pubsub.UpdatedEvent:
		slog.Info("Task updated",
			"task_id", task.ID,
			"status", string(task.Status),
			"assigned_member", task.AssignedMember)
	}
}

// GetDepartmentManager returns the department manager instance
func (dc *DepartmentCoordinator) GetDepartmentManager() *department.Manager {
	return dc.departmentManager
}

// CreateCustomerRequest creates a task from a customer request
func (dc *DepartmentCoordinator) CreateCustomerRequest(ctx context.Context, title, description, requestedBy string, priority department.Priority, attachments []message.Attachment) (*department.Task, error) {
	if dc.departmentManager == nil {
		return nil, fmt.Errorf("department management is not enabled")
	}

	task := &department.Task{
		Title:       title,
		Description: description,
		Type:        "customer_request",
		Priority:    priority,
		RequestedBy: requestedBy,
		Attachments: convertAttachments(attachments),
	}

	return dc.departmentManager.CreateTask(ctx, task)
}

// GetDepartmentStatus returns the status of all departments
func (dc *DepartmentCoordinator) GetDepartmentStatus() (map[string]interface{}, error) {
	if dc.departmentManager == nil {
		return nil, fmt.Errorf("department management is not enabled")
	}

	status := make(map[string]interface{})

	// Get department statistics
	departments := dc.departmentManager.ListDepartments()
	for _, dept := range departments {
		stats, err := dc.departmentManager.GetDepartmentStats(dept.ID)
		if err != nil {
			continue
		}
		status[dept.ID] = map[string]interface{}{
			"name":       dept.Name,
			"type":       string(dept.Type),
			"stats":      stats,
			"auto_scale": dept.AutoScale,
		}
	}

	// Get member information
	members := dc.departmentManager.ListMembers("")
	status["members"] = map[string]interface{}{
		"total":   len(members),
		"online":  countMembersByStatus(members, department.MemberStatusOnline),
		"busy":    countMembersByStatus(members, department.MemberStatusBusy),
		"offline": countMembersByStatus(members, department.MemberStatusOffline),
	}

	// Get task information
	tasks := dc.departmentManager.ListTasks("", "")
	status["tasks"] = map[string]interface{}{
		"total":     len(tasks),
		"queued":    countTasksByStatus(tasks, department.TaskStatusQueued),
		"active":    countTasksByStatus(tasks, department.TaskStatusInProgress),
		"completed": countTasksByStatus(tasks, department.TaskStatusCompleted),
		"failed":    countTasksByStatus(tasks, department.TaskStatusFailed),
	}

	return status, nil
}

// Helper functions

func extractTaskTitle(prompt string) string {
	// Simple title extraction - in a real implementation, this would be more sophisticated
	lines := strings.Split(prompt, "\n")
	if len(lines) > 0 && len(strings.TrimSpace(lines[0])) < 100 {
		return strings.TrimSpace(lines[0])
	}

	// Truncate prompt if too long
	if len(prompt) > 50 {
		return prompt[:47] + "..."
	}
	return prompt
}

func determineTaskType(prompt string) string {
	prompt = strings.ToLower(prompt)

	if strings.Contains(prompt, "bug") || strings.Contains(prompt, "fix") {
		return "bug_fix"
	}
	if strings.Contains(prompt, "feature") || strings.Contains(prompt, "implement") {
		return "feature_development"
	}
	if strings.Contains(prompt, "test") || strings.Contains(prompt, "qa") {
		return "testing"
	}
	if strings.Contains(prompt, "deploy") || strings.Contains(prompt, "release") {
		return "deployment"
	}
	if strings.Contains(prompt, "security") || strings.Contains(prompt, "vulnerability") {
		return "security"
	}

	return "general"
}

func determineTaskPriority(prompt string) department.Priority {
	prompt = strings.ToLower(prompt)

	if strings.Contains(prompt, "urgent") || strings.Contains(prompt, "critical") || strings.Contains(prompt, "asap") {
		return department.PriorityCritical
	}
	if strings.Contains(prompt, "high") || strings.Contains(prompt, "important") {
		return department.PriorityHigh
	}
	if strings.Contains(prompt, "low") || strings.Contains(prompt, "minor") {
		return department.PriorityLow
	}

	return department.PriorityMedium
}

func extractRequiredSkills(prompt string) []string {
	prompt = strings.ToLower(prompt)
	var skills []string

	skillKeywords := map[string][]string{
		"go":         {"golang", "go "},
		"javascript": {"javascript", "js ", "node"},
		"python":     {"python", "py "},
		"docker":     {"docker", "container"},
		"kubernetes": {"kubernetes", "k8s"},
		"security":   {"security", "vulnerability", "penetration"},
		"testing":    {"test", "testing", "qa"},
	}

	for skill, keywords := range skillKeywords {
		for _, keyword := range keywords {
			if strings.Contains(prompt, keyword) {
				skills = append(skills, skill)
				break
			}
		}
	}

	return skills
}

func convertAttachments(attachments []message.Attachment) []department.TaskAttachment {
	var taskAttachments []department.TaskAttachment

	for i, att := range attachments {
		taskAtt := department.TaskAttachment{
			ID:        fmt.Sprintf("att-%d", i),
			Name:      att.Name,
			Type:      att.ContentType,
			Size:      int64(len(att.Content)),
			Content:   att.Content,
			CreatedAt: time.Now(),
		}
		taskAttachments = append(taskAttachments, taskAtt)
	}

	return taskAttachments
}

func countMembersByStatus(members []*department.Member, status department.MemberStatus) int {
	count := 0
	for _, member := range members {
		if member.Status == status {
			count++
		}
	}
	return count
}

func countTasksByStatus(tasks []*department.Task, status department.TaskStatus) int {
	count := 0
	for _, task := range tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}