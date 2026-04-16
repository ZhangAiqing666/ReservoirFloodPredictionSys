package service

import (
	"context"
	"errors" // 导入 errors

	v1 "ReservoirFloodPrediction/api/user/v1" // 导入生成的 user v1 API 包
	"ReservoirFloodPrediction/internal/biz"   // 导入 biz 包

	"github.com/go-kratos/kratos/v2/log"
)

// UserService is a user service implementation.
type UserService struct {
	v1.UnimplementedUserServiceServer // 必须嵌入，以满足接口要求

	uc  *biz.UserUsecase // 依赖 UserUsecase
	log *log.Helper
}

// NewUserService new a user service.
func NewUserService(uc *biz.UserUsecase, logger log.Logger) *UserService {
	return &UserService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/user")),
	}
}

// Register implements user.UserServiceServer.
func (s *UserService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.RegisterReply, error) {
	// 1. 输入校验 (除了 proto validate 之外的逻辑，比如密码一致性)
	if req.Password != req.ConfirmPassword {
		// Kratos 建议返回标准的 gRPC 错误码或业务错误码
		// 这里暂时返回一个简单的错误
		s.log.WithContext(ctx).Warnf("registration failed for user %s: passwords do not match", req.Username)
		// 可以使用 Kratos 的 errors 包来返回更规范的错误
		// return nil, kerr.InvalidArgument("PASSWORD_MISMATCH", "两次输入的密码不一致")
		return nil, errors.New("两次输入的密码不一致")
	}

	// 2. 调用 biz 层的注册逻辑
	registeredUser, err := s.uc.RegisterUser(ctx, req.Username, req.Password)
	if err != nil {
		s.log.WithContext(ctx).Errorf("failed to register user %s: %v", req.Username, err)
		// 错误处理：可以将 biz 层的错误转换为 API 层的错误
		// 例如：if errors.Is(err, biz.ErrUserAlreadyExists) { return nil, kerr.AlreadyExists(...) }
		return nil, err // 暂时直接返回 biz 层错误
	}

	// 3. 构建并返回响应
	s.log.WithContext(ctx).Infof("user %s registered successfully via API", req.Username)
	return &v1.RegisterReply{
		UserId:  registeredUser.ID, // 从 biz 层返回的结果获取用户 ID
		Message: "注册成功",
	}, nil
}

// Login implements user.UserServiceServer.
func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginReply, error) {
	// 1. 调用 biz 层的登录逻辑
	loggedInUser, err := s.uc.LoginUser(ctx, req.Username, req.Password)
	if err != nil {
		s.log.WithContext(ctx).Warnf("login failed for user %s: %v", req.Username, err)
		// 错误处理：转换为 API 错误
		// 例如：if errors.Is(err, biz.ErrUserNotFound) { return nil, kerr.NotFound(...) }
		//       if errors.Is(err, biz.ErrPasswordIncorrect) { return nil, kerr.Unauthenticated(...) }
		return nil, err // 暂时直接返回 biz 层错误
	}

	// 2. 构建并返回响应
	s.log.WithContext(ctx).Infof("user %s logged in successfully via API", req.Username)
	return &v1.LoginReply{
		UserId:   loggedInUser.ID,
		Username: loggedInUser.Username,
		Message:  "登录成功",
		// Token: "generate_jwt_token_here", // 实际应用中生成并返回 Token
	}, nil
}
