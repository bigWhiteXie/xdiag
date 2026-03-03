package targets

import "context"

// Target 表示一个目标资产
type Target struct {
	ID          uint   `json:"id" yaml:"id"`
	Name        string `json:"name" yaml:"name"`
	Kind        string `json:"kind" yaml:"kind"` // node, postgres, mysql, redis
	Address     string `json:"address" yaml:"address"`
	Port        int    `json:"port" yaml:"port"`
	Username    string `json:"username" yaml:"username"`
	Password    string `json:"password" yaml:"password"`
	SSHKey      string `json:"ssh_key" yaml:"ssh_key"`
	Tags        string `json:"tags" yaml:"tags"`         // 逗号分隔的标签
	CreatedAt   string `json:"created_at" yaml:"created_at"`
	UpdatedAt   string `json:"updated_at" yaml:"updated_at"`
}

// Repo 定义目标资产管理的数据存储接口
type Repo interface {
	Create(ctx context.Context, target *Target) error
	GetByID(ctx context.Context, id uint) (*Target, error)
	GetByName(ctx context.Context, name string) (*Target, error)
	List(ctx context.Context, filters map[string]string) ([]*Target, error)
	Update(ctx context.Context, target *Target) error
	Delete(ctx context.Context, id uint) error
	Close() error
}