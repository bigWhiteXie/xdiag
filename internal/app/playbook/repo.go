package playbook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Repo 定义了playbook存储库的操作接口
type Repo interface {
	// ListPlaybooks 查询所有一级playbook信息，支持根据tags筛选
	ListPlaybooks(tags []string) ([]Playbook, error)

	// GetBook 根据一级playbook的名称和ref的名称获得对应的book结构体
	GetBook(playbookName, refName string) (*Book, error)

	// LoadPlaybook 加载指定的playbook
	LoadPlaybook(playbookName string) (*Playbook, error)

	// PlaybookExists 检查playbook是否存在
	PlaybookExists(playbookName string) bool

	// SavePlaybook 保存playbook的introduction.yaml
	SavePlaybook(playbook *Playbook) error

	// SaveBook 保存book文件到指定playbook的refs目录
	SaveBook(playbookName string, book *Book) error

	// UpdatePlaybookRef 更新playbook的refs列表
	UpdatePlaybookRef(playbookName string, ref Ref) error
}

// repo 实现Repo接口
type repo struct {
	playbooksDir string
}

// NewRepo 创建一个新的playbook存储库实例
func NewRepo(playbooksDir string) Repo {
	return &repo{
		playbooksDir: playbooksDir,
	}
}

// ListPlaybooks 查询所有一级playbook信息，支持根据tags筛选
func (r *repo) ListPlaybooks(tags []string) ([]Playbook, error) {
	playbooks, err := LoadPlaybooks(r.playbooksDir)
	if err != nil {
		return nil, fmt.Errorf("加载playbooks失败: %w", err)
	}

	// 如果没有指定tags，则返回所有playbooks
	if len(tags) == 0 {
		return playbooks, nil
	}

	// 根据tags筛选playbooks
	var filtered []Playbook
	for _, pb := range playbooks {
		if containsAllTags(pb.Tags, tags) {
			filtered = append(filtered, pb)
		}
	}

	return filtered, nil
}

// GetBook 根据一级playbook的名称和ref的名称获得对应的book结构体
func (r *repo) GetBook(playbookName, refName string) (*Book, error) {
	playbookPath := filepath.Join(r.playbooksDir, playbookName)

	pb, err := LoadSinglePlaybook(playbookPath)
	if err != nil {
		return nil, fmt.Errorf("加载playbook '%s' 失败: %w", playbookName, err)
	}

	if pb == nil {
		return nil, fmt.Errorf("playbook '%s' 不存在", playbookName)
	}

	// 在refs目录中查找对应的book文件
	refsDir := filepath.Join(playbookPath, "refs")
	if _, err := os.Stat(refsDir); err == nil {
		// 遍历refs目录中的所有yaml文件
		entries, err := os.ReadDir(refsDir)
		if err != nil {
			return nil, fmt.Errorf("读取refs目录失败: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
				filePath := filepath.Join(refsDir, entry.Name())

				// 读取并解析文件
				data, err := os.ReadFile(filePath)
				if err != nil {
					continue
				}

				var book Book
				if err := yaml.Unmarshal(data, &book); err != nil {
					continue
				}

				// 检查book名称是否匹配refName
				if book.Name == refName {
					return &book, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("在playbook '%s' 中未找到ref '%s' 对应的book文件", playbookName, refName)
}

// LoadPlaybook 加载指定的playbook
func (r *repo) LoadPlaybook(playbookName string) (*Playbook, error) {
	playbookPath := filepath.Join(r.playbooksDir, playbookName)
	return LoadSinglePlaybook(playbookPath)
}

// PlaybookExists 检查playbook是否存在
func (r *repo) PlaybookExists(playbookName string) bool {
	playbookPath := filepath.Join(r.playbooksDir, playbookName)
	introPath := filepath.Join(playbookPath, "introduction.yaml")
	_, err := os.Stat(introPath)
	return err == nil
}

// SavePlaybook 保存playbook的introduction.yaml
func (r *repo) SavePlaybook(playbook *Playbook) error {
	playbookPath := filepath.Join(r.playbooksDir, playbook.Name)

	// 创建playbook目录
	if err := os.MkdirAll(playbookPath, 0755); err != nil {
		return fmt.Errorf("创建playbook目录失败: %w", err)
	}

	// 保存introduction.yaml
	introPath := filepath.Join(playbookPath, "introduction.yaml")
	data, err := yaml.Marshal(playbook)
	if err != nil {
		return fmt.Errorf("序列化playbook失败: %w", err)
	}

	if err := os.WriteFile(introPath, data, 0644); err != nil {
		return fmt.Errorf("写入introduction.yaml失败: %w", err)
	}

	return nil
}

// SaveBook 保存book文件到指定playbook的refs目录
func (r *repo) SaveBook(playbookName string, book *Book) error {
	playbookPath := filepath.Join(r.playbooksDir, playbookName)
	refsDir := filepath.Join(playbookPath, "refs")

	// 创建refs目录
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		return fmt.Errorf("创建refs目录失败: %w", err)
	}

	// 保存book文件
	bookPath := filepath.Join(refsDir, fmt.Sprintf("%s.yaml", book.Name))
	data, err := yaml.Marshal(book)
	if err != nil {
		return fmt.Errorf("序列化book失败: %w", err)
	}

	if err := os.WriteFile(bookPath, data, 0644); err != nil {
		return fmt.Errorf("写入book文件失败: %w", err)
	}

	return nil
}

// UpdatePlaybookRef 更新playbook的refs列表
func (r *repo) UpdatePlaybookRef(playbookName string, ref Ref) error {
	playbookPath := filepath.Join(r.playbooksDir, playbookName)
	introPath := filepath.Join(playbookPath, "introduction.yaml")

	// 读取现有的playbook
	data, err := os.ReadFile(introPath)
	if err != nil {
		return fmt.Errorf("读取introduction.yaml失败: %w", err)
	}

	var playbook Playbook
	if err := yaml.Unmarshal(data, &playbook); err != nil {
		return fmt.Errorf("解析introduction.yaml失败: %w", err)
	}

	// 检查ref是否已存在
	refExists := false
	for i, r := range playbook.Refs {
		if r.Name == ref.Name {
			// 更新现有ref
			playbook.Refs[i] = ref
			refExists = true
			break
		}
	}

	// 如果不存在，添加新ref
	if !refExists {
		playbook.Refs = append(playbook.Refs, ref)
	}

	// 写回文件
	updatedData, err := yaml.Marshal(&playbook)
	if err != nil {
		return fmt.Errorf("序列化playbook失败: %w", err)
	}

	if err := os.WriteFile(introPath, updatedData, 0644); err != nil {
		return fmt.Errorf("写入introduction.yaml失败: %w", err)
	}

	return nil
}

// containsAllTags 检查playbook的tags是否包含所有指定的tags
func containsAllTags(playbookTags, requiredTags []string) bool {
	if len(requiredTags) == 0 {
		return true
	}

	for _, requiredTag := range requiredTags {
		found := false
		for _, tag := range playbookTags {
			if tag == requiredTag {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}
