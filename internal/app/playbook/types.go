package playbook

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

// Playbook 定义
type Playbook struct {
	Name string
	Desc string
	Tags []string
	Refs []Ref
	Path string // 添加路径字段
}

// IntroductionYaml 对应 introduction.yaml 文件结构
type IntroductionYaml struct {
	Metadata Metadata `yaml:"metadata"`
	Refs     []Ref    `yaml:"refs"`
}

type Metadata struct {
	Name         string   `yaml:"name"`
	Desc         string   `yaml:"desc"`
	RequiredTags []string `yaml:"required_tags"` // 如果metadata中也包含tags
}

type Ref struct {
	Name string `yaml:"name"`
	Desc string `yaml:"desc"`
	Log  string `yaml:"log"`
}

type Book struct {
	Name  string `yaml:"name"`
	Steps []Step `yaml:"steps"`
}

type Step struct {
	Kind  string      `yaml:"kind"`
	Desc  string      `yaml:"desc"`
	Cases []CaseBlock `yaml:"cases,omitempty"`
}

type CaseBlock struct {
	Case  string `yaml:"case"`
	Steps []Step `yaml:"steps"`
}

// LoadPlaybooks 加载指定目录下的所有Playbook
func LoadPlaybooks(dir string) ([]Playbook, error) {
	var playbooks []Playbook

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			playbookPath := filepath.Join(dir, entry.Name())
			playbook, err := LoadSinglePlaybook(playbookPath)
			if err != nil {
				continue
			}
			if playbook != nil {
				playbooks = append(playbooks, *playbook)
			}
		}
	}

	return playbooks, nil
}

// LoadSinglePlaybook 加载单个playbook
func LoadSinglePlaybook(path string) (*Playbook, error) {
	// 检查目录是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, nil
	}

	introPath := filepath.Join(path, "introduction.yaml")
	if _, err := os.Stat(introPath); os.IsNotExist(err) {
		introPath = filepath.Join(path, "metadata.yaml") // 兼容开发计划中提到的metadata.yaml
		if _, err := os.Stat(introPath); os.IsNotExist(err) {
			return nil, nil
		}
	}

	introData, err := ioutil.ReadFile(introPath)
	if err != nil {
		return nil, err
	}

	var introYaml IntroductionYaml
	if err := yaml.Unmarshal(introData, &introYaml); err != nil {
		return nil, err
	}

	playbook := &Playbook{
		Name: introYaml.Metadata.Name,
		Desc: introYaml.Metadata.Desc,
		Tags: introYaml.Metadata.RequiredTags,
		Refs: introYaml.Refs,
		Path: path, // 保存路径
	}

	// 获取refs目录下的诊断方案
	refsDir := filepath.Join(path, "refs")
	if _, err := os.Stat(refsDir); err == nil {
		refFiles, err := os.ReadDir(refsDir)
		if err != nil {
			return nil, err
		}

		for _, refFile := range refFiles {
			if !refFile.IsDir() && strings.HasSuffix(refFile.Name(), ".yaml") {
				refPath := filepath.Join(refsDir, refFile.Name())
				refData, err := ioutil.ReadFile(refPath)
				if err != nil {
					continue
				}

				var book Book
				if err := yaml.Unmarshal(refData, &book); err != nil {
					continue
				}

				// 检查是否已经存在同名的ref
				exists := false
				for i, ref := range playbook.Refs {
					if ref.Name == book.Name {
						// 如果存在，则更新描述
						playbook.Refs[i].Desc = book.Name
						exists = true
						break
					}
				}
				if !exists {
					playbook.Refs = append(playbook.Refs, Ref{
						Name: book.Name,
						Desc: book.Name,
						Log:  "",
					})
				}
			}
		}
	}

	return playbook, nil
}
