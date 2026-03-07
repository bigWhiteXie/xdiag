package targets

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// SQLiteRepo 是使用 SQLite 的目标资产管理存储实现
type SQLiteRepo struct {
	db *gorm.DB
}

// NewSQLiteRepo 创建一个新的 SQLite 存储实例
func NewSQLiteRepo(dbPath string) (*SQLiteRepo, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &SQLiteRepo{db: db}

	// 自动迁移表结构
	if err := db.AutoMigrate(&Target{}); err != nil {
		return nil, fmt.Errorf("failed to migrate table: %w", err)
	}

	return repo, nil
}

// Create 创建一个新的目标
func (r *SQLiteRepo) Create(ctx context.Context, target *Target) error {
	now := time.Now().Format(time.RFC3339)
	target.CreatedAt = now
	target.UpdatedAt = now

	if err := r.db.WithContext(ctx).Create(target).Error; err != nil {
		return fmt.Errorf("failed to create target: %w", err)
	}

	return nil
}

// GetByID 根据ID获取目标
func (r *SQLiteRepo) GetByID(ctx context.Context, id uint) (*Target, error) {
	var target Target
	if err := r.db.WithContext(ctx).First(&target, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("target with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get target by id: %w", err)
	}

	return &target, nil
}

// GetByName 根据名称获取目标
func (r *SQLiteRepo) GetByName(ctx context.Context, name string) (*Target, error) {
	var target Target
	if err := r.db.WithContext(ctx).Where("name = ?", name).First(&target).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("target with name %s not found", name)
		}
		return nil, fmt.Errorf("failed to get target by name: %w", err)
	}

	return &target, nil
}

// List 获取目标列表，支持过滤
func (r *SQLiteRepo) List(ctx context.Context, filters map[string]Op) ([]*Target, error) {
	query := r.db.WithContext(ctx)

	for key, op := range filters {
		switch key {
		case "kind":
			query = applyFilter(query, "kind", op)
		case "tag":
			query = applyFilter(query, "tags", op)
		case "name":
			query = applyFilter(query, "name", op)
		case "address":
			query = applyFilter(query, "address", op)
		}
	}

	var targets []*Target
	if err := query.Find(&targets).Error; err != nil {
		return nil, fmt.Errorf("failed to list targets: %w", err)
	}

	return targets, nil
}

// applyFilter 应用过滤条件
func applyFilter(query *gorm.DB, field string, op Op) *gorm.DB {
	switch op.Op {
	case "eq":
		return query.Where(field+" = ?", op.Val)
	case "ne":
		return query.Where(field+" != ?", op.Val)
	case "like":
		return query.Where(field+" LIKE ?", "%"+op.Val+"%")
	default:
		return query
	}
}

// Update 更新目标
func (r *SQLiteRepo) Update(ctx context.Context, target *Target) error {
	target.UpdatedAt = time.Now().Format(time.RFC3339)

	if err := r.db.WithContext(ctx).Save(target).Error; err != nil {
		return fmt.Errorf("failed to update target: %w", err)
	}

	return nil
}

// Delete 删除目标
func (r *SQLiteRepo) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&Target{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete target: %w", err)
	}

	return nil
}

// Close 关闭数据库连接
func (r *SQLiteRepo) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// GetAllKinds 返回所有去重的target类型
func (r *SQLiteRepo) GetAllKinds() ([]string, error) {
	var kinds []string
	if err := r.db.Model(&Target{}).Distinct("kind").Order("kind").Pluck("kind", &kinds).Error; err != nil {
		return nil, fmt.Errorf("failed to get all kinds: %w", err)
	}

	return kinds, nil
}
