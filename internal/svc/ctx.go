package svc

import (
	"xdiag/internal/app/playbook"
	"xdiag/internal/app/targets"
	"xdiag/internal/config"

	"github.com/cloudwego/eino/components/model"
)

var (
	svcCtx *ServiceContext = &ServiceContext{}
)

type ServiceContext struct {
	Model       model.ToolCallingChatModel
	TargetsRepo targets.Repo
	BookRepo    playbook.Repo
	Config      config.Config
}

func SetModel(model model.ToolCallingChatModel) {
	svcCtx.Model = model
}

func SetTargetsRepo(repo targets.Repo) {
	svcCtx.TargetsRepo = repo
}

func SetBookRepo(repo playbook.Repo) {
	svcCtx.BookRepo = repo
}

func SetConfig(cfg config.Config) {
	svcCtx.Config = cfg
}

func GetServiceContext() *ServiceContext {
	return svcCtx
}
