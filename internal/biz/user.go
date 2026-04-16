package biz

import (
	"context"
	"errors" // 导入 errors 包

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt" // 导入 bcrypt
)

// 定义业务错误
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrPasswordIncorrect = errors.New("password is incorrect")
	ErrUserAlreadyExists = errors.New("username already exists") // 可能由 data 层返回的唯一约束错误转换而来
)

// User 是业务层使用的用户模型
// 注意与 data.User 的区别，这里可能不包含 GORM tag 或 PasswordHash
type User struct {
	ID       uint64
	Username string
	Password string // 用于注册或登录时传递明文密码，不在数据库存储
	// 可选：添加其他业务相关字段，如 Role, Email 等
	PasswordHash string // 从 data 层获取，用于密码验证
}

// UserRepo 是数据访问层的接口定义
// biz 层依赖此接口，而不是具体的 data 层实现
type UserRepo interface {
	CreateUser(ctx context.Context, user *User) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	// 未来可能添加其他方法，如 UpdateUser, DeleteUser 等
}

// UserUsecase 包含用户相关的业务逻辑
type UserUsecase struct {
	repo UserRepo // 依赖 UserRepo 接口
	log  *log.Helper
	// 可能依赖其他 usecase，例如发送邮件的 usecase
}

// NewUserUsecase 创建 UserUsecase 实例
func NewUserUsecase(repo UserRepo, logger log.Logger) *UserUsecase {
	return &UserUsecase{
		repo: repo,
		log:  log.NewHelper(log.With(logger, "module", "usecase/user")),
	}
}

// RegisterUser 处理用户注册逻辑
func (uc *UserUsecase) RegisterUser(ctx context.Context, username, password string) (*User, error) {
	// 1. 基本校验 (用户名密码不能为空等，可以在 service 层或 API 层做)
	if username == "" || password == "" {
		// 返回一个更具体的业务错误
		return nil, errors.New("username and password cannot be empty")
	}

	// 2. 检查用户是否已存在 (可选，CreateUser 内部的唯一约束会处理，但提前检查可以返回更友好的错误)
	// _, err := uc.repo.GetUserByUsername(ctx, username)
	// if err == nil {
	// 	return nil, ErrUserAlreadyExists
	// }
	// if err != ErrUserNotFound { // 只处理非 "找不到用户" 的错误
	// 	uc.log.WithContext(ctx).Errorf("error checking if user exists: %v", err)
	// 	return nil, err // 返回底层错误
	// }
	// 如果 err == ErrUserNotFound，说明用户不存在，可以继续注册

	// 3. 创建用户 (密码哈希在 data 层的 CreateUser 中完成)
	newUser := &User{
		Username: username,
		Password: password, // 传递明文密码给 data 层处理
	}
	createdUser, err := uc.repo.CreateUser(ctx, newUser)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to register user %s: %v", username, err)
		return nil, err // 暂时直接返回底层错误
	}

	// 4. 注册成功，返回创建的用户信息（不含密码）
	uc.log.WithContext(ctx).Infof("user %s registered successfully with ID %d", createdUser.Username, createdUser.ID)
	return createdUser, nil // createdUser 由 data 层返回，不含密码哈希
}

// LoginUser 处理用户登录逻辑
func (uc *UserUsecase) LoginUser(ctx context.Context, username, password string) (*User, error) {
	// 1. 基本校验
	if username == "" || password == "" {
		return nil, errors.New("username and password cannot be empty")
	}

	// 2. 根据用户名查找用户
	user, err := uc.repo.GetUserByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) { // 使用 errors.Is 处理包装后的错误
			uc.log.WithContext(ctx).Warnf("login attempt failed for non-existent user: %s", username)
			return nil, ErrUserNotFound // 返回业务错误
		}
		// 其他查找错误
		uc.log.WithContext(ctx).Errorf("failed to get user %s during login: %v", username, err)
		return nil, err // 返回底层错误
	}

	// 3. 验证密码
	// user.PasswordHash 是从 data 层获取的数据库存储的哈希值
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		// 如果 err == bcrypt.ErrMismatchedHashAndPassword，说明密码错误
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			uc.log.WithContext(ctx).Warnf("incorrect password attempt for user: %s", username)
			return nil, ErrPasswordIncorrect // 返回密码错误
		}
		// 其他 bcrypt 错误
		uc.log.WithContext(ctx).Errorf("error comparing password hash for user %s: %v", username, err)
		return nil, err // 返回底层错误
	}

	// 4. 登录成功，返回用户信息（不含密码或哈希）
	uc.log.WithContext(ctx).Infof("user %s logged in successfully", username)
	// 清理敏感信息再返回
	user.PasswordHash = ""
	user.Password = ""
	return user, nil
}
