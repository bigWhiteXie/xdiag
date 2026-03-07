package playbook

import (
	"fmt"
	"io/ioutil"
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
				data, err := ioutil.ReadFile(filePath)
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