package svc

import (
	"xdiag/internal/app/targets"

	"github.com/cloudwego/eino/components/model"
)

var (
	svcCtx *ServiceContext = &ServiceContext{}
)

type ServiceContext struct {
	Model       model.ToolCallingChatModel
	TargetsRepo targets.Repo
}

func SetModel(model model.ToolCallingChatModel) {
	svcCtx.Model = model
}

func SetTargetsRepo(repo targets.Repo) {
	svcCtx.TargetsRepo = repo
}

func GetServiceContext() *ServiceContext {
	return svcCtx
}
