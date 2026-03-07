package targets

import "context"

const (
	// TargetKind constants
	TargetKindNode     = "node"
	TargetKindPostgres = "postgres"
	TargetKindMySQL    = "mysql"
	TargetKindRedis    = "redis"

	// Connectivity status constants
	StatusSuccess = "success"
	StatusFailed  = "failed"

	// Message constants
	MessagePingFailed         = "Ping test failed: %v"
	MessageAuthFailed         = "%s authentication failed: %v"
	MessageConnectivityPassed = "%s connectivity test passed"

	// Extra details keys
	ExtraDetailPingTime      = "ping_time"
	ExtraDetailAuthMethod    = "auth_method"
	ExtraDetailSSHAuthMethod = "ssh_auth_method"

	// Port constants
	DefaultSSHPort        = 22
	DefaultPostgreSQLPort = 5432
	DefaultMySQLPort      = 3306
	DefaultRedisPort      = 6379
)

// Op represents an operation for filtering
type Op struct {
	Op  string // operation type: eq, like, ne, gt, lt, etc.
	Val string // value to compare against
}

// Target 表示一个目标资产
type Target struct {
	ID        uint   `json:"id" yaml:"id" gorm:"primaryKey;autoIncrement"`
	Name      string `json:"name" yaml:"name" gorm:"uniqueIndex;not null"`
	Kind      string `json:"kind" yaml:"kind" gorm:"index;not null"` // node, postgres, mysql, redis
	Address   string `json:"address" yaml:"address" gorm:"index;not null"`
	Port      int    `json:"port" yaml:"port" gorm:"not null"`
	Username  string `json:"username" yaml:"username"`
	Password  string `json:"password" yaml:"password"`
	Node      string `json:"ssh_key" yaml:"ssh_key" gorm:"column:ssh_key"`
	Tags      string `json:"tags" yaml:"tags"` // 逗号分隔的标签
	CreatedAt string `json:"created_at" yaml:"created_at"`
	UpdatedAt string `json:"updated_at" yaml:"updated_at"`
}

// TableName 指定表名
func (Target) TableName() string {
	return "targets"
}

// Repo 定义目标资产管理的数据存储接口
type Repo interface {
	Create(ctx context.Context, target *Target) error
	GetByID(ctx context.Context, id uint) (*Target, error)
	GetByName(ctx context.Context, name string) (*Target, error)
	List(ctx context.Context, filters map[string]Op) ([]*Target, error)
	Update(ctx context.Context, target *Target) error
	Delete(ctx context.Context, id uint) error
	GetAllKinds() ([]string, error)
	Close() error
}
