package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/crypto/bcrypt" // 导入 bcrypt 包用于密码哈希
	"gorm.io/gorm"

	"ReservoirFloodPrediction/internal/biz" // 导入 biz 包定义的 User 结构体（稍后创建）
)

// User 对应数据库中的 user 表结构
type User struct {
	ID           uint64    `gorm:"primaryKey"`
	Username     string    `gorm:"uniqueIndex;size:50"` // 添加 uniqueIndex 确保用户名唯一
	PasswordHash string    `gorm:"size:255"`
	CreatedAt    time.Time // GORM 会自动处理 created_at
	UpdatedAt    time.Time // GORM 会自动处理 updated_at
}

// TableName 指定 User 模型对应的数据库表名
func (User) TableName() string {
	return "user"
}

// userRepo 实现了 biz.UserRepo 接口
type userRepo struct {
	data *Data // 引用包含 db 连接的 Data 结构
	log  *log.Helper
}

// NewUserRepo .
// 构造函数，用于创建 userRepo 实例
// 注意：这里的返回值类型是 biz.UserRepo，这是一个接口类型
func NewUserRepo(data *Data, logger log.Logger) biz.UserRepo {
	return &userRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/user")),
	}
}

// CreateUser 在数据库中创建一个新用户
func (r *userRepo) CreateUser(ctx context.Context, u *biz.User) (*biz.User, error) {
	// 密码哈希处理
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		r.log.WithContext(ctx).Errorf("failed to hash password for user %s: %v", u.Username, err)
		return nil, err
	}

	// 创建数据模型实例
	dbUser := User{
		Username:     u.Username,
		PasswordHash: string(hashedPassword),
		// ID, CreatedAt, UpdatedAt 由 GORM 或数据库自动处理
	}

	// 使用 GORM 创建记录
	result := r.data.db.WithContext(ctx).Create(&dbUser)
	if result.Error != nil {
		// 这里可以检查是否是唯一约束冲突错误等
		r.log.WithContext(ctx).Errorf("failed to create user %s in db: %v", u.Username, result.Error)
		return nil, result.Error // 返回 GORM 错误
	}

	r.log.WithContext(ctx).Infof("successfully created user %s with id %d", dbUser.Username, dbUser.ID)

	// 将数据库生成的信息（如 ID, CreatedAt）回填到业务对象
	// 注意：我们只返回业务层关心的字段，避免暴露数据库细节
	return &biz.User{
		ID:       dbUser.ID,
		Username: dbUser.Username,
		// 不返回密码哈希
	}, nil
}

// GetUserByUsername 根据用户名查找用户
func (r *userRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	var dbUser User
	// 使用 GORM 查询
	// First 会在找不到记录时返回 gorm.ErrRecordNotFound 错误
	result := r.data.db.WithContext(ctx).Where("username = ?", username).First(&dbUser)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			r.log.WithContext(ctx).Warnf("user not found by username: %s", username)
			// 对于业务逻辑来说，找不到用户通常不是一个系统级错误，
			// 可以在 biz 层处理 ErrRecordNotFound 并返回特定的业务错误
			return nil, biz.ErrUserNotFound // 返回业务层定义的错误（稍后创建）
		}
		// 其他数据库错误
		r.log.WithContext(ctx).Errorf("failed to get user by username %s from db: %v", username, result.Error)
		return nil, result.Error
	}

	r.log.WithContext(ctx).Infof("successfully found user %s by username", username)

	// 将数据库模型转换为业务对象
	return &biz.User{
		ID:           dbUser.ID,
		Username:     dbUser.Username,
		PasswordHash: dbUser.PasswordHash, // 需要返回密码哈希给 biz 层进行验证
	}, nil
}
