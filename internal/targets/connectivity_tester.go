package targets

import "context"

// ConnectivityTester 定义连通性测试接口
type ConnectivityTester interface {
	Test(ctx context.Context, target *Target) (*ConnectivityResult, error)
}

// ConnectivityResult 连通性测试结果
type ConnectivityResult struct {
	Status       string            `json:"status"`       // success 或 failed
	PingStatus   string            `json:"ping_status"`  // ping 结果
	AuthStatus   string            `json:"auth_status"`  // 认证结果
	Message      string            `json:"message"`      // 详细信息
	ExtraDetails map[string]string `json:"extra_details,omitempty"` // 额外的详细信息
}

// NewConnectivityTester 创建指定类型的连通性测试器
func NewConnectivityTester(targetKind string) (ConnectivityTester, error) {
	switch targetKind {
	case "node":
		return &NodeConnectivityTester{}, nil
	case "postgres":
		return &PostgresConnectivityTester{}, nil
	case "mysql":
		return &MySQLConnectivityTester{}, nil
	case "redis":
		return &RedisConnectivityTester{}, nil
	default:
		return nil, &UnsupportedTargetTypeError{Kind: targetKind}
	}
}

// UnsupportedTargetTypeError 不支持的目标类型错误
type UnsupportedTargetTypeError struct {
	Kind string
}

func (e *UnsupportedTargetTypeError) Error() string {
	return "unsupported target kind: " + e.Kind
}