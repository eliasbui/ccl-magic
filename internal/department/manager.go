package department

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/eliasbui/ccl-magic/internal/pubsub"
)

// Manager handles all department operations including member management,
// task distribution, and scaling operations
type Manager struct {
	config      *DepartmentConfig
	departments map[string]*Department
	members     map[string]*Member
	tasks       map[string]*Task
	teams       map[string]*Team
	workflows   map[string]*Workflow

	// Event brokers for different event types
	departmentEvents *pubsub.Broker[*Department]
	memberEvents     *pubsub.Broker[*Member]
	taskEvents       *pubsub.Broker[*Task]

	// Statistics tracking
	departmentStats map[string]*DepartmentStats
	memberStats     map[string]*MemberStats

	// Management state
	isRunning bool
	mu        sync.RWMutex

	// Health monitoring
	healthChecker *HealthChecker

	// Task routing
	taskRouter *TaskRouter

	// Auto-scaling
	scaler *AutoScaler
}

// ManagerOption represents a configuration option for the department manager
type ManagerOption func(*Manager)

// NewManager creates a new department manager with the given configuration
func NewManager(ctx context.Context, config *DepartmentConfig, opts ...ManagerOption) (*Manager, error) {
	m := &Manager{
		config:          config,
		departments:     make(map[string]*Department),
		members:         make(map[string]*Member),
		tasks:           make(map[string]*Task),
		teams:           make(map[string]*Team),
		workflows:       make(map[string]*Workflow),
		departmentEvents: pubsub.NewBroker[*Department](),
		memberEvents:     pubsub.NewBroker[*Member](),
		taskEvents:       pubsub.NewBroker[*Task](),
		departmentStats:  make(map[string]*DepartmentStats),
		memberStats:      make(map[string]*MemberStats),
	}

	// Apply options
	for _, opt := range opts {
		opt(m)
	}

	// Initialize components
	if err := m.initializeComponents(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize components: %w", err)
	}

	// Set up default departments if none exist
	if len(m.departments) == 0 {
		if err := m.setupDefaultDepartments(); err != nil {
			return nil, fmt.Errorf("failed to setup default departments: %w", err)
		}
	}

	return m, nil
}

// initializeComponents sets up all the department manager components
func (m *Manager) initializeComponents(ctx context.Context) error {
	// Initialize health checker
	if m.config.HealthCheck.Enabled {
		m.healthChecker = NewHealthChecker(m.config.HealthCheck, m)
		go m.healthChecker.Start(ctx)
	}

	// Initialize task router
	m.taskRouter = NewTaskRouter(m.config.TaskRouting, m)

	// Initialize auto-scaler
	if m.config.AutoScaling.Enabled {
		m.scaler = NewAutoScaler(m.config.AutoScaling, m)
		go m.scaler.Start(ctx)
	}

	return nil
}

// setupDefaultDepartments creates the default department structure
func (m *Manager) setupDefaultDepartments() error {
	defaultDepartments := []Department{
		{
			ID:          "dept-dev",
			Name:        "Development Services",
			Type:        DepartmentDevelopment,
			Description: "Software development and coding services",
			Capabilities: []string{"coding", "code-review", "architecture", "debugging"},
			MaxMembers:  10,
			MinMembers:  2,
			AutoScale:   true,
		},
		{
			ID:          "dept-devops",
			Name:        "Infrastructure & Operations",
			Type:        DepartmentDevOps,
			Description: "CI/CD, deployment, and infrastructure management",
			Capabilities: []string{"ci-cd", "deployment", "monitoring", "infrastructure"},
			MaxMembers:  6,
			MinMembers:  1,
			AutoScale:   true,
		},
		{
			ID:          "dept-security",
			Name:        "Security & Compliance",
			Type:        DepartmentSecurity,
			Description: "Security scanning, compliance, and vulnerability assessment",
			Capabilities: []string{"security-scan", "compliance", "audit", "penetration-testing"},
			MaxMembers:  4,
			MinMembers:  1,
			AutoScale:   true,
		},
		{
			ID:          "dept-qa",
			Name:        "Quality Assurance",
			Type:        DepartmentQA,
			Description: "Testing automation and quality assurance",
			Capabilities: []string{"testing", "automation", "performance-testing", "integration-testing"},
			MaxMembers:  8,
			MinMembers:  2,
			AutoScale:   true,
		},
	}

	now := time.Now()
	for _, dept := range defaultDepartments {
		dept.CreatedAt = now
		dept.UpdatedAt = now
		m.departments[dept.ID] = &dept
		m.departmentStats[dept.ID] = &DepartmentStats{
			DepartmentID:    dept.ID,
			TotalMembers:    0,
			ActiveMembers:   0,
			RoleDistribution: make(map[string]int),
			LastUpdated:     now,
		}
	}

	return nil
}

// Start starts the department manager
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return fmt.Errorf("department manager is already running")
	}

	m.isRunning = true
	slog.Info("Department manager started")

	// Start background processes
	go m.statisticsUpdater(ctx)

	return nil
}

// Stop stops the department manager
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return nil
	}

	m.isRunning = false

	// Shutdown event brokers
	m.departmentEvents.Shutdown()
	m.memberEvents.Shutdown()
	m.taskEvents.Shutdown()

	slog.Info("Department manager stopped")
	return nil
}

// RegisterMember registers a new member in a department
func (m *Manager) RegisterMember(ctx context.Context, member *Member) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate department exists
	dept, exists := m.departments[member.DepartmentID]
	if !exists {
		return fmt.Errorf("department %s does not exist", member.DepartmentID)
	}

	// Check if we can add more members
	if dept.MaxMembers > 0 {
		currentCount := m.countDepartmentMembers(member.DepartmentID)
		if currentCount >= dept.MaxMembers {
			return fmt.Errorf("department %s has reached maximum member capacity", member.DepartmentID)
		}
	}

	// Set member metadata
	now := time.Now()
	member.JoinedAt = now
	member.LastSeen = now
	member.Status = MemberStatusOnline

	// Determine if this is a lead role
	member.IsLead = isLeadRole(member.Role)

	// Add member
	m.members[member.ID] = member

	// Update statistics
	m.updateDepartmentStats(member.DepartmentID)
	m.memberStats[member.ID] = &MemberStats{
		MemberID:   member.ID,
		MemberRole: member.Role,
		LastUpdated: now,
	}

	// Publish events
	m.memberEvents.Publish(pubsub.CreatedEvent, member)

	slog.Info("Member registered",
		"member_id", member.ID,
		"member_name", member.Name,
		"role", string(member.Role),
		"department", member.DepartmentID)

	return nil
}

// UnregisterMember removes a member from the department
func (m *Manager) UnregisterMember(ctx context.Context, memberID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	member, exists := m.members[memberID]
	if !exists {
		return fmt.Errorf("member %s does not exist", memberID)
	}

	// Check if member has active tasks
	if len(member.CurrentTasks) > 0 {
		return fmt.Errorf("cannot remove member %s: has %d active tasks", memberID, len(member.CurrentTasks))
	}

	// Remove member
	delete(m.members, memberID)
	delete(m.memberStats, memberID)

	// Update statistics
	m.updateDepartmentStats(member.DepartmentID)

	// Publish events
	m.memberEvents.Publish(pubsub.DeletedEvent, member)

	slog.Info("Member unregistered", "member_id", memberID, "member_name", member.Name)

	return nil
}

// UpdateMemberStatus updates a member's status
func (m *Manager) UpdateMemberStatus(ctx context.Context, memberID string, status MemberStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	member, exists := m.members[memberID]
	if !exists {
		return fmt.Errorf("member %s does not exist", memberID)
	}

	oldStatus := member.Status
	member.Status = status
	member.LastSeen = time.Now()

	// Update statistics
	m.updateDepartmentStats(member.DepartmentID)

	// Publish events
	m.memberEvents.Publish(pubsub.UpdatedEvent, member)

	slog.Info("Member status updated",
		"member_id", memberID,
		"old_status", string(oldStatus),
		"new_status", string(status))

	return nil
}

// CreateTask creates a new task and routes it to appropriate member
func (m *Manager) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate ID if not provided
	if task.ID == "" {
		task.ID = generateTaskID()
	}

	// Set timestamps
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now
	task.Status = TaskStatusQueued

	// Validate department exists
	if _, exists := m.departments[task.DepartmentID]; !exists {
		return nil, fmt.Errorf("department %s does not exist", task.DepartmentID)
	}

	// Add task
	m.tasks[task.ID] = task

	// Route task to appropriate member
	if m.taskRouter != nil {
		if err := m.taskRouter.RouteTask(ctx, task); err != nil {
			slog.Warn("Failed to route task", "task_id", task.ID, "error", err)
		}
	}

	// Publish events
	m.taskEvents.Publish(pubsub.CreatedEvent, task)

	slog.Info("Task created",
		"task_id", task.ID,
		"title", task.Title,
		"department", task.DepartmentID,
		"priority", string(task.Priority))

	return task, nil
}

// UpdateTaskStatus updates the status of a task
func (m *Manager) UpdateTaskStatus(ctx context.Context, taskID string, status TaskStatus, result map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s does not exist", taskID)
	}

	oldStatus := task.Status
	task.Status = status
	task.UpdatedAt = time.Now()

	// Handle status-specific logic
	switch status {
	case TaskStatusInProgress:
		if task.StartedAt == nil {
			start := time.Now()
			task.StartedAt = &start
		}
	case TaskStatusCompleted, TaskStatusFailed:
		if task.CompletedAt == nil {
			completed := time.Now()
			task.CompletedAt = &completed
		}
		// Update member stats and free up capacity
		if task.AssignedMember != "" {
			m.updateMemberTaskCompletion(task.AssignedMember, taskID, status == TaskStatusCompleted)
		}
	}

	// Store results if provided
	if result != nil {
		if task.Results == nil {
			task.Results = make(map[string]interface{})
		}
		for k, v := range result {
			task.Results[k] = v
		}
	}

	// Publish events
	m.taskEvents.Publish(pubsub.UpdatedEvent, task)

	slog.Info("Task status updated",
		"task_id", taskID,
		"old_status", string(oldStatus),
		"new_status", string(status))

	return nil
}

// GetDepartment returns a department by ID
func (m *Manager) GetDepartment(departmentID string) (*Department, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	dept, exists := m.departments[departmentID]
	if !exists {
		return nil, fmt.Errorf("department %s does not exist", departmentID)
	}
	return dept, nil
}

// GetMember returns a member by ID
func (m *Manager) GetMember(memberID string) (*Member, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	member, exists := m.members[memberID]
	if !exists {
		return nil, fmt.Errorf("member %s does not exist", memberID)
	}
	return member, nil
}

// GetTask returns a task by ID
func (m *Manager) GetTask(taskID string) (*Task, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s does not exist", taskID)
	}
	return task, nil
}

// ListDepartments returns all departments
func (m *Manager) ListDepartments() []*Department {
	m.mu.RLock()
	defer m.mu.RUnlock()

	departments := make([]*Department, 0, len(m.departments))
	for _, dept := range m.departments {
		departments = append(departments, dept)
	}
	return departments
}

// ListMembers returns all members, optionally filtered by department
func (m *Manager) ListMembers(departmentID string) []*Member {
	m.mu.RLock()
	defer m.mu.RUnlock()

	members := make([]*Member, 0)
	for _, member := range m.members {
		if departmentID == "" || member.DepartmentID == departmentID {
			members = append(members, member)
		}
	}
	return members
}

// ListTasks returns all tasks, optionally filtered by department and status
func (m *Manager) ListTasks(departmentID string, status TaskStatus) []*Task {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tasks := make([]*Task, 0)
	for _, task := range m.tasks {
		if (departmentID == "" || task.DepartmentID == departmentID) &&
			(status == "" || task.Status == status) {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetDepartmentStats returns statistics for a department
func (m *Manager) GetDepartmentStats(departmentID string) (*DepartmentStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats, exists := m.departmentStats[departmentID]
	if !exists {
		return nil, fmt.Errorf("department %s does not exist", departmentID)
	}
	return stats, nil
}

// GetMemberStats returns statistics for a member
func (m *Manager) GetMemberStats(memberID string) (*MemberStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats, exists := m.memberStats[memberID]
	if !exists {
		return nil, fmt.Errorf("member %s does not exist", memberID)
	}
	return stats, nil
}

// SubscribeToDepartmentEvents returns a channel for department events
func (m *Manager) SubscribeToDepartmentEvents(ctx context.Context) <-chan pubsub.Event[*Department] {
	return m.departmentEvents.Subscribe(ctx)
}

// SubscribeToMemberEvents returns a channel for member events
func (m *Manager) SubscribeToMemberEvents(ctx context.Context) <-chan pubsub.Event[*Member] {
	return m.memberEvents.Subscribe(ctx)
}

// SubscribeToTaskEvents returns a channel for task events
func (m *Manager) SubscribeToTaskEvents(ctx context.Context) <-chan pubsub.Event[*Task] {
	return m.taskEvents.Subscribe(ctx)
}

// Helper functions

func (m *Manager) countDepartmentMembers(departmentID string) int {
	count := 0
	for _, member := range m.members {
		if member.DepartmentID == departmentID {
			count++
		}
	}
	return count
}

func (m *Manager) updateDepartmentStats(departmentID string) {
	dept := m.departments[departmentID]
	stats := m.departmentStats[departmentID]

	// Count members and roles
	roleDistribution := make(map[string]int)
	activeMembers := 0

	for _, member := range m.members {
		if member.DepartmentID == departmentID {
			roleDistribution[string(member.Role)]++
			if member.Status == MemberStatusOnline || member.Status == MemberStatusBusy {
				activeMembers++
			}
		}
	}

	stats.TotalMembers = len(m.members)
	stats.ActiveMembers = activeMembers
	stats.RoleDistribution = roleDistribution
	stats.LastUpdated = time.Now()
}

func (m *Manager) updateMemberTaskCompletion(memberID, taskID string, success bool) {
	member, exists := m.members[memberID]
	if !exists {
		return
	}

	// Remove task from current tasks
	for i, task := range member.CurrentTasks {
		if task == taskID {
			member.CurrentTasks = append(member.CurrentTasks[:i], member.CurrentTasks[i+1:]...)
			break
		}
	}

	// Update member stats
	stats := m.memberStats[memberID]
	stats.TotalTasks++
	if success {
		stats.CompletedTasks++
	} else {
		stats.FailedTasks++
	}
	stats.CurrentLoad = len(member.CurrentTasks)
	stats.SuccessRate = float64(stats.CompletedTasks) / float64(stats.TotalTasks)
	stats.LastUpdated = time.Now()
}

func (m *Manager) statisticsUpdater(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second) // Update every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.updateAllStatistics()
		}
	}
}

func (m *Manager) updateAllStatistics() {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update department statistics
	for deptID := range m.departments {
		m.updateDepartmentStats(deptID)
	}

	// Update task statistics
	for _, stats := range m.memberStats {
		// Additional statistics calculations can be added here
		stats.LastUpdated = time.Now()
	}
}

func isLeadRole(role MemberRole) bool {
	return role == RoleLeadTechnical || role == RoleLeadBA || role == RoleLeadDev || role == RoleLeadTest
}

func generateTaskID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}