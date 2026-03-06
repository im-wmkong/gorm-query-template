package base

// Service 定义了通用的 Service 接口
type Service interface {
}

var _ Service = (*BaseService)(nil)

// BaseService 实现了通用的业务逻辑
type BaseService struct {
}

// NewService 创建一个新的 BaseService
func NewService() *BaseService {
	return &BaseService{}
}
