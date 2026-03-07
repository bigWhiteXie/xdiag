package targets

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// SQLiteRepo 是使用 SQLite 的目标资产管理存储实现
type SQLiteRepo struct {
	db *sql.DB
}

// NewSQLiteRepo 创建一个新的 SQLite 存储实例
func NewSQLiteRepo(dbPath string) (*SQLiteRepo, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	repo := &SQLiteRepo{db: db}

	// 创建目标表
	err = repo.createTable()
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return repo, nil
}

// createTable 创建目标表
func (r *SQLiteRepo) createTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS targets (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		kind TEXT NOT NULL,
		address TEXT NOT NULL,
		port INTEGER NOT NULL,
		username TEXT,
		password TEXT,
		ssh_key TEXT,
		tags TEXT,
		created_at TEXT DEFAULT CURRENT_TIMESTAMP,
		updated_at TEXT DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_targets_name ON targets(name);
	CREATE INDEX IF NOT EXISTS idx_targets_kind ON targets(kind);
	CREATE INDEX IF NOT EXISTS idx_targets_address ON targets(address);
}
	`

	_, err := r.db.Exec(query)
	return err
}

// Create 创建一个新的目标
func (r *SQLiteRepo) Create(ctx context.Context, target *Target) error {
	now := time.Now().Format(time.RFC3339)
	target.CreatedAt = now
	target.UpdatedAt = now

	query := `
	INSERT INTO targets (name, kind, address, port, username, password, ssh_key, tags, created_at, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		target.Name,
		target.Kind,
		target.Address,
		target.Port,
		target.Username,
		target.Password,
		target.SSHKey,
		target.Tags,
		target.CreatedAt,
		target.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to insert target: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	target.ID = uint(id)
	return nil
}

// GetByID 根据ID获取目标
func (r *SQLiteRepo) GetByID(ctx context.Context, id uint) (*Target, error) {
	query := `SELECT id, name, kind, address, port, username, password, ssh_key, tags, created_at, updated_at FROM targets WHERE id = ?`
	row := r.db.QueryRowContext(ctx, query, id)

	var target Target
	err := row.Scan(
		&target.ID,
		&target.Name,
		&target.Kind,
		&target.Address,
		&target.Port,
		&target.Username,
		&target.Password,
		&target.SSHKey,
		&target.Tags,
		&target.CreatedAt,
		&target.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("target with id %d not found", id)
		}
		return nil, fmt.Errorf("failed to get target by id: %w", err)
	}

	return &target, nil
}

// GetByName 根据名称获取目标
func (r *SQLiteRepo) GetByName(ctx context.Context, name string) (*Target, error) {
	query := `SELECT id, name, kind, address, port, username, password, ssh_key, tags, created_at, updated_at FROM targets WHERE name = ?`
	row := r.db.QueryRowContext(ctx, query, name)

	var target Target
	err := row.Scan(
		&target.ID,
		&target.Name,
		&target.Kind,
		&target.Address,
		&target.Port,
		&target.Username,
		&target.Password,
		&target.SSHKey,
		&target.Tags,
		&target.CreatedAt,
		&target.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("target with name %s not found", name)
		}
		return nil, fmt.Errorf("failed to get target by name: %w", err)
	}

	return &target, nil
}

// List 获取目标列表，支持过滤
func (r *SQLiteRepo) List(ctx context.Context, filters map[string]Op) ([]*Target, error) {
	query := `SELECT id, name, kind, address, port, username, password, ssh_key, tags, created_at, updated_at FROM targets WHERE 1=1`
	args := []interface{}{}

	for key, op := range filters {
		switch key {
		case "kind":
			switch op.Op {
			case "eq":
				query += " AND kind = ?"
				args = append(args, op.Val)
			case "ne":
				query += " AND kind != ?"
				args = append(args, op.Val)
			case "like":
				query += " AND kind LIKE ?"
				args = append(args, "%"+op.Val+"%")
			}
		case "tag":
			switch op.Op {
			case "eq":
				query += " AND tags = ?"
				args = append(args, op.Val)
			case "ne":
				query += " AND tags != ?"
				args = append(args, op.Val)
			case "like":
				query += " AND tags LIKE ?"
				args = append(args, "%"+op.Val+"%")
			}
		case "name":
			switch op.Op {
			case "eq":
				query += " AND name = ?"
				args = append(args, op.Val)
			case "ne":
				query += " AND name != ?"
				args = append(args, op.Val)
			case "like":
				query += " AND name LIKE ?"
				args = append(args, "%"+op.Val+"%")
			}
		case "address":
			switch op.Op {
			case "eq":
				query += " AND address = ?"
				args = append(args, op.Val)
			case "ne":
				query += " AND address != ?"
				args = append(args, op.Val)
			case "like":
				query += " AND address LIKE ?"
				args = append(args, "%"+op.Val+"%")
			}
		}
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list targets: %w", err)
	}
	defer rows.Close()

	var targets []*Target
	for rows.Next() {
		var target Target
		err := rows.Scan(
			&target.ID,
			&target.Name,
			&target.Kind,
			&target.Address,
			&target.Port,
			&target.Username,
			&target.Password,
			&target.SSHKey,
			&target.Tags,
			&target.CreatedAt,
			&target.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan target: %w", err)
		}

		targets = append(targets, &target)
	}

	return targets, nil
}

// Update 更新目标
func (r *SQLiteRepo) Update(ctx context.Context, target *Target) error {
	query := `
	UPDATE targets SET 
		name=?, kind=?, address=?, port=?, username=?, password=?, ssh_key=?, tags=?, updated_at=?
	WHERE id=?
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		target.Name,
		target.Kind,
		target.Address,
		target.Port,
		target.Username,
		target.Password,
		target.SSHKey,
		target.Tags,
		time.Now().Format(time.RFC3339),
		target.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update target: %w", err)
	}

	return nil
}

// Delete 删除目标
func (r *SQLiteRepo) Delete(ctx context.Context, id uint) error {
	query := `DELETE FROM targets WHERE id = ?`
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete target: %w", err)
	}

	return nil
}

// Close 关闭数据库连接
func (r *SQLiteRepo) Close() error {
	return r.db.Close()
}

// GetAllKinds 返回所有去重的target类型
func (r *SQLiteRepo) GetAllKinds() ([]string, error) {
	query := "SELECT DISTINCT kind FROM targets ORDER BY kind"
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all kinds: %w", err)
	}
	defer rows.Close()

	var kinds []string
	for rows.Next() {
		var kind string
		err := rows.Scan(&kind)
		if err != nil {
			return nil, fmt.Errorf("failed to scan kind: %w", err)
		}
		kinds = append(kinds, kind)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error occurred while iterating rows: %w", err)
	}

	return kinds, nil
}
