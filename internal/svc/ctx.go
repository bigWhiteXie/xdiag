package svc

import (
	"github.com/bigWhiteXie/xdiag/internal/app/playbook"
	"github.com/bigWhiteXie/xdiag/internal/app/targets"
	"github.com/bigWhiteXie/xdiag/internal/config"
	"github.com/bigWhiteXie/xdiag/pkg/logger"

	"github.com/cloudwego/eino/components/model"
	"go.uber.org/zap"
)

var (
	svcCtx *ServiceContext = &ServiceContext{}
)

type ServiceContext struct {
	Model       model.ToolCallingChatModel
	TargetsRepo targets.Repo
	BookRepo    playbook.Repo
	Config      config.Config
	Logger      *zap.Logger
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

func InitLogger(level string, development bool) error {
	if err := logger.Init(level, development); err != nil {
		return err
	}
	svcCtx.Logger = logger.GetLogger()
	return nil
}

func GetServiceContext() *ServiceContext {
	return svcCtx
}
