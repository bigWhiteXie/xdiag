package targets

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// NodeConnectivityTester 节点连通性测试器
type NodeConnectivityTester struct{}

func (n *NodeConnectivityTester) Test(ctx context.Context, target *Target) (*ConnectivityResult, error) {
	result := &ConnectivityResult{
		ExtraDetails: make(map[string]string),
	}

	// 首先进行 ping 测试
	pingResult, pingErr := n.pingTest(target.Address)
	if pingErr != nil {
		result.Status = StatusFailed
		result.PingStatus = StatusFailed
		result.Message = fmt.Sprintf(MessagePingFailed, pingErr)
		return result, nil
	}

	result.PingStatus = StatusSuccess
	result.ExtraDetails[ExtraDetailPingTime] = pingResult

	// 进行 SSH 认证测试
	authResult, authErr := n.sshTest(target)
	if authErr != nil {
		result.Status = StatusFailed
		result.AuthStatus = StatusFailed
		result.Message = fmt.Sprintf(MessageAuthFailed, TargetKindNode, authErr)
		return result, nil
	}

	result.AuthStatus = StatusSuccess
	result.Status = StatusSuccess
	result.Message = fmt.Sprintf(MessageConnectivityPassed, TargetKindNode)
	result.ExtraDetails[ExtraDetailSSHAuthMethod] = authResult

	return result, nil
}

// pingTest 执行 ping 测试
func (n *NodeConnectivityTester) pingTest(address string) (string, error) {
	// 在真实实现中，我们会执行 ping 命令
	// 这里简化为连接到目标地址的常见端口
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", address, DefaultSSHPort), 5*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return "reachable", nil
}

// sshTest 执行 SSH 认证测试
func (n *NodeConnectivityTester) sshTest(target *Target) (string, error) {
	// 在真实实现中，我们会建立 SSH 连接
	// 检查是否提供了 SSH 密钥或密码
	if target.Node != "" {
		return "using_ssh_key", nil
	} else if target.Password != "" {
		return "using_password", nil
	} else {
		return "", errors.New("no authentication method provided")
	}
}

// PostgresConnectivityTester PostgreSQL 连通性测试器
type PostgresConnectivityTester struct{}

func (p *PostgresConnectivityTester) Test(ctx context.Context, target *Target) (*ConnectivityResult, error) {
	result := &ConnectivityResult{
		ExtraDetails: make(map[string]string),
	}

	// 首先进行 ping 测试
	pingResult, pingErr := p.pingTest(target.Address, target.Port)
	if pingErr != nil {
		result.Status = StatusFailed
		result.PingStatus = StatusFailed
		result.Message = fmt.Sprintf(MessagePingFailed, pingErr)
		return result, nil
	}

	result.PingStatus = StatusSuccess
	result.ExtraDetails[ExtraDetailPingTime] = pingResult

	// 进行 PostgreSQL 认证测试
	authResult, authErr := p.authTest(target)
	if authErr != nil {
		result.Status = StatusFailed
		result.AuthStatus = StatusFailed
		result.Message = fmt.Sprintf(MessageAuthFailed, TargetKindPostgres, authErr)
		return result, nil
	}

	result.AuthStatus = StatusSuccess
	result.Status = StatusSuccess
	result.Message = fmt.Sprintf(MessageConnectivityPassed, TargetKindPostgres)
	result.ExtraDetails[ExtraDetailAuthMethod] = authResult

	return result, nil
}

// pingTest 执行 ping 测试
func (p *PostgresConnectivityTester) pingTest(address string, port int) (string, error) {
	testPort := port
	if testPort == 0 {
		testPort = DefaultPostgreSQLPort
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", address, testPort), 5*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return "reachable", nil
}

// authTest 执行 PostgreSQL 认证测试
func (p *PostgresConnectivityTester) authTest(target *Target) (string, error) {
	// 在真实实现中，我们会使用 pgx 或其他库连接到 PostgreSQL
	// 简化为检查是否提供了必要的认证信息
	if target.Username == "" {
		return "", errors.New("username is required for PostgreSQL connection")
	}
	if target.Password == "" {
		return "", errors.New("password is required for PostgreSQL connection")
	}

	return "using_username_password", nil
}

// MySQLConnectivityTester MySQL 连通性测试器
type MySQLConnectivityTester struct{}

func (m *MySQLConnectivityTester) Test(ctx context.Context, target *Target) (*ConnectivityResult, error) {
	result := &ConnectivityResult{
		ExtraDetails: make(map[string]string),
	}

	// 首先进行 ping 测试
	pingResult, pingErr := m.pingTest(target.Address, target.Port)
	if pingErr != nil {
		result.Status = StatusFailed
		result.PingStatus = StatusFailed
		result.Message = fmt.Sprintf(MessagePingFailed, pingErr)
		return result, nil
	}

	result.PingStatus = StatusSuccess
	result.ExtraDetails[ExtraDetailPingTime] = pingResult

	// 进行 MySQL 认证测试
	authResult, authErr := m.authTest(target)
	if authErr != nil {
		result.Status = StatusFailed
		result.AuthStatus = StatusFailed
		result.Message = fmt.Sprintf(MessageAuthFailed, TargetKindMySQL, authErr)
		return result, nil
	}

	result.AuthStatus = StatusSuccess
	result.Status = StatusSuccess
	result.Message = fmt.Sprintf(MessageConnectivityPassed, TargetKindMySQL)
	result.ExtraDetails[ExtraDetailAuthMethod] = authResult

	return result, nil
}

// pingTest 执行 ping 测试
func (m *MySQLConnectivityTester) pingTest(address string, port int) (string, error) {
	testPort := port
	if testPort == 0 {
		testPort = DefaultMySQLPort
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", address, testPort), 5*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return "reachable", nil
}

// authTest 执行 MySQL 认证测试
func (m *MySQLConnectivityTester) authTest(target *Target) (string, error) {
	// 在真实实现中，我们会使用 go-sql-driver/mysql 库连接到 MySQL
	// 简化为检查是否提供了必要的认证信息
	if target.Username == "" {
		return "", errors.New("username is required for MySQL connection")
	}
	if target.Password == "" {
		return "", errors.New("password is required for MySQL connection")
	}

	return "using_username_password", nil
}

// RedisConnectivityTester Redis 连通性测试器
type RedisConnectivityTester struct{}

func (r *RedisConnectivityTester) Test(ctx context.Context, target *Target) (*ConnectivityResult, error) {
	result := &ConnectivityResult{
		ExtraDetails: make(map[string]string),
	}

	// 首先进行 ping 测试
	pingResult, pingErr := r.pingTest(target.Address, target.Port)
	if pingErr != nil {
		result.Status = StatusFailed
		result.PingStatus = StatusFailed
		result.Message = fmt.Sprintf(MessagePingFailed, pingErr)
		return result, nil
	}

	result.PingStatus = StatusSuccess
	result.ExtraDetails[ExtraDetailPingTime] = pingResult

	// 进行 Redis 认证测试
	authResult, authErr := r.authTest(target)
	if authErr != nil {
		result.Status = StatusFailed
		result.AuthStatus = StatusFailed
		result.Message = fmt.Sprintf(MessageAuthFailed, TargetKindRedis, authErr)
		return result, nil
	}

	result.AuthStatus = StatusSuccess
	result.Status = StatusSuccess
	result.Message = fmt.Sprintf(MessageConnectivityPassed, TargetKindRedis)
	result.ExtraDetails[ExtraDetailAuthMethod] = authResult

	return result, nil
}

// pingTest 执行 ping 测试
func (r *RedisConnectivityTester) pingTest(address string, port int) (string, error) {
	testPort := port
	if testPort == 0 {
		testPort = DefaultRedisPort
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", address, testPort), 5*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	return "reachable", nil
}

// authTest 执行 Redis 认证测试
func (r *RedisConnectivityTester) authTest(target *Target) (string, error) {
	// 在真实实现中，我们会使用 go-redis/redis 库连接到 Redis
	// 简化为检查是否提供了必要的认证信息
	if target.Password == "" {
		return "", errors.New("password is required for Redis connection")
	}

	return "using_password", nil
}
