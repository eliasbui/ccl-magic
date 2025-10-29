package department

import (
	"time"
)

// DepartmentType represents different types of departments in the IT organization
type DepartmentType string

const (
	DepartmentProductManager DepartmentType = "productManager"
	DepartmentDevelopment DepartmentType = "development"
	DepartmentDevOps       DepartmentType = "devops"
	DepartmentSecurity     DepartmentType = "security"
	DepartmentQA          DepartmentType = "qa"
)

// MemberRole represents specific roles within departments
type MemberRole string

const (
	RoleBA          MemberRole = "ba"           // Business Analyst
	RolePM          MemberRole = "pm"           // Project Manager
	RolePO          MemberRole = "po"           // Product Owner
	RoleLeadTechnical MemberRole = "lead_technical" // Technical Lead
	RoleLeadBA      MemberRole = "lead_ba"      // Business Analyst Lead
	RoleLeadDev     MemberRole = "lead_dev"     // Development Lead
	RoleLeadTest    MemberRole = "lead_test"    // QA/Test Lead
	RoleDeveloper   MemberRole = "developer"    // Software Developer
	RoleDevOps      MemberRole = "devops"       // DevOps Engineer
	RoleQA          MemberRole = "qa"           // QA Engineer
	RoleSecurity    MemberRole = "security"     // Security Engineer
)

// MemberStatus represents the current status of a department member
type MemberStatus string

const (
	MemberStatusOnline     MemberStatus = "online"
	MemberStatusBusy       MemberStatus = "busy"
	MemberStatusOffline    MemberStatus = "offline"
	MemberStatusUnhealthy  MemberStatus = "unhealthy"
)

// TaskStatus represents the status of a task in the workflow
type TaskStatus string

const (
	TaskStatusQueued     TaskStatus = "queued"
	TaskStatusAssigned   TaskStatus = "assigned"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusBlocked    TaskStatus = "blocked"
)

// Priority represents task priority levels
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// Department represents an IT department with specialized capabilities
type Department struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Type        DepartmentType    `json:"type"`
	Description string            `json:"description"`
	Capabilities []string         `json:"capabilities"`
	MaxMembers  int               `json:"max_members"`
	MinMembers  int               `json:"min_members"`
	AutoScale   bool              `json:"auto_scale"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// Member represents a Claude Code CLI instance with a specific role in a department
type Member struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Role            MemberRole             `json:"role"`
	DepartmentID    string                 `json:"department_id"`
	DepartmentType  DepartmentType         `json:"department_type"`
	Status          MemberStatus           `json:"status"`
	Specializations []string               `json:"specializations"`
	CurrentTasks    []string               `json:"current_tasks"`
	MaxConcurrent   int                    `json:"max_concurrent"`
	LastSeen        time.Time              `json:"last_seen"`
	JoinedAt        time.Time              `json:"joined_at"`
	Endpoint        string                 `json:"endpoint"`
	AuthMethod      string                 `json:"auth_method"`
	HealthScore     float64                `json:"health_score"`
	Performance     map[string]float64     `json:"performance"`
	Capabilities    map[string]interface{} `json:"capabilities"`
	IsLead          bool                   `json:"is_lead"`
	ReportsTo       string                 `json:"reports_to,omitempty"`
	TeamMembers     []string               `json:"team_members,omitempty"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
}

// Task represents a work item in the department workflow
type Task struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Type            string                 `json:"type"`
	Priority        Priority               `json:"priority"`
	Status          TaskStatus             `json:"status"`
	DepartmentID    string                 `json:"department_id"`
	AssignedMember  string                 `json:"assigned_member,omitempty"`
	RequestedBy     string                 `json:"requested_by"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	StartedAt       *time.Time             `json:"started_at,omitempty"`
	CompletedAt     *time.Time             `json:"completed_at,omitempty"`
	DueDate         *time.Time             `json:"due_date,omitempty"`
	EstimatedHours  *float64               `json:"estimated_hours,omitempty"`
	ActualHours     *float64               `json:"actual_hours,omitempty"`
	Tags            []string               `json:"tags"`
	Dependencies    []string               `json:"dependencies"`
	Attachments     []TaskAttachment       `json:"attachments,omitempty"`
	Results         map[string]interface{} `json:"results,omitempty"`
	AssignedRole    MemberRole             `json:"assigned_role,omitempty"`
	RequiredSkills  []string               `json:"required_skills,omitempty"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
}

// TaskAttachment represents files or data attached to tasks
type TaskAttachment struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Size        int64     `json:"size"`
	URL         string    `json:"url,omitempty"`
	Content     []byte    `json:"content,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// DepartmentConfig represents configuration for department management
type DepartmentConfig struct {
	Enabled        bool                    `json:"enabled"`
	Departments    map[string]Department    `json:"departments,omitempty"`
	AutoScaling    AutoScalingConfig       `json:"auto_scaling,omitempty"`
	HealthCheck    HealthCheckConfig       `json:"health_check,omitempty"`
	TaskRouting    TaskRoutingConfig       `json:"task_routing,omitempty"`
	Notifications  NotificationConfig      `json:"notifications,omitempty"`
	Reporting      ReportingConfig         `json:"reporting,omitempty"`
	Roles          RoleConfig              `json:"roles,omitempty"`
}

// RoleConfig defines role-specific configurations and permissions
type RoleConfig struct {
	RoleDefinitions map[string]RoleDefinition `json:"role_definitions,omitempty"`
	Permissions     map[string][]string       `json:"permissions,omitempty"`
	Capabilities    map[string][]string       `json:"capabilities,omitempty"`
}

// RoleDefinition defines the properties and responsibilities of each role
type RoleDefinition struct {
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	LeadRole        bool     `json:"lead_role"`
	DepartmentTypes []string `json:"department_types"`
	Responsibilities []string `json:"responsibilities"`
	RequiredSkills  []string `json:"required_skills"`
	CanAssignTo     []string `json:"can_assign_to,omitempty"`
	MaxConcurrent   int      `json:"max_concurrent"`
	DefaultTools    []string `json:"default_tools"`
}

// AutoScalingConfig defines how departments can automatically scale members
type AutoScalingConfig struct {
	Enabled           bool          `json:"enabled"`
	CheckInterval     time.Duration `json:"check_interval"`
	ScaleUpThreshold  float64       `json:"scale_up_threshold"`
	ScaleDownThreshold float64      `json:"scale_down_threshold"`
	MaxMembersPerDept int           `json:"max_members_per_department"`
	CooldownPeriod    time.Duration `json:"cooldown_period"`
	RoleScaling       map[string]int `json:"role_scaling,omitempty"`
}

// HealthCheckConfig defines health monitoring for members
type HealthCheckConfig struct {
	Enabled           bool          `json:"enabled"`
	CheckInterval     time.Duration `json:"check_interval"`
	Timeout           time.Duration `json:"timeout"`
	UnhealthyThreshold int          `json:"unhealthy_threshold"`
	RetryCount        int           `json:"retry_count"`
	RoleSpecificChecks map[string]HealthCheck `json:"role_specific_checks,omitempty"`
}

// HealthCheck defines role-specific health check parameters
type HealthCheck struct {
	ResponseTime time.Duration `json:"response_time"`
	TaskSuccess  float64       `json:"task_success"`
	Uptime       float64       `json:"uptime"`
}

// TaskRoutingConfig defines how tasks are routed to departments and members
type TaskRoutingConfig struct {
	Strategy           string                 `json:"strategy"` // round-robin, load-based, skill-based, role-based
	DepartmentRules    map[string][]string    `json:"department_rules,omitempty"`
	RoleRules          map[string][]string    `json:"role_rules,omitempty"`
	MemberRules        map[string][]string    `json:"member_rules,omitempty"`
	DefaultDepartment  string                 `json:"default_department"`
	DefaultRole        string                 `json:"default_role"`
	FallbackEnabled    bool                   `json:"fallback_enabled"`
	RoutingMetadata    map[string]interface{} `json:"routing_metadata,omitempty"`
}

// NotificationConfig defines event-driven notifications
type NotificationConfig struct {
	Enabled     bool     `json:"enabled"`
	Events      []string `json:"events"`
	Channels    []string `json:"channels"`
	Webhooks    []string `json:"webhooks,omitempty"`
	Emails      []string `json:"emails,omitempty"`
	RateLimit   int      `json:"rate_limit,omitempty"`
	RoleNotifications map[string][]string `json:"role_notifications,omitempty"`
}

// ReportingConfig defines progress tracking and analytics
type ReportingConfig struct {
	Enabled        bool          `json:"enabled"`
	ReportInterval time.Duration `json:"report_interval"`
	Metrics        []string      `json:"metrics"`
	Dashboards     []string      `json:"dashboards,omitempty"`
	ExportFormats  []string      `json:"export_formats,omitempty"`
	RoleReports    []string      `json:"role_reports,omitempty"`
}

// DepartmentStats represents statistics for a department
type DepartmentStats struct {
	DepartmentID    string            `json:"department_id"`
	TotalMembers    int               `json:"total_members"`
	ActiveMembers   int               `json:"active_members"`
	RoleDistribution map[string]int    `json:"role_distribution"`
	TotalTasks      int               `json:"total_tasks"`
	CompletedTasks  int               `json:"completed_tasks"`
	FailedTasks     int               `json:"failed_tasks"`
	AverageResponse float64           `json:"average_response"`
	LastUpdated     time.Time         `json:"last_updated"`
}

// MemberStats represents performance statistics for a member
type MemberStats struct {
	MemberID        string    `json:"member_id"`
	MemberRole      MemberRole `json:"member_role"`
	TotalTasks      int       `json:"total_tasks"`
	CompletedTasks  int       `json:"completed_tasks"`
	FailedTasks     int       `json:"failed_tasks"`
	AverageTime     float64   `json:"average_time"`
	SuccessRate     float64   `json:"success_rate"`
	CurrentLoad     int       `json:"current_load"`
	TeamTasks       int       `json:"team_tasks,omitempty"`
	LeadershipTasks int       `json:"leadership_tasks,omitempty"`
	LastUpdated     time.Time `json:"last_updated"`
}

// Team represents a team within a department led by a lead role
type Team struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DepartmentID string  `json:"department_id"`
	LeadID      string   `json:"lead_id"`
	LeadRole    MemberRole `json:"lead_role"`
	MemberIDs   []string `json:"member_ids"`
	Roles       []MemberRole `json:"roles"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Workflow represents a defined workflow for different task types and roles
type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	TaskType    string                 `json:"task_type"`
	Steps       []WorkflowStep         `json:"steps"`
	RequiredRoles []MemberRole         `json:"required_roles"`
	OptionalRoles []MemberRole         `json:"optional_roles"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// WorkflowStep represents a step in a workflow
type WorkflowStep struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	AssignedRole MemberRole `json:"assigned_role"`
	Required    bool        `json:"required"`
	Dependencies []string   `json:"dependencies,omitempty"`
	EstimatedTime float64   `json:"estimated_time,omitempty"`
	Tools       []string    `json:"tools,omitempty"`
}