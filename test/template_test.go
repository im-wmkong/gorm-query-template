package test

import (
	"context"
	"fmt"
	"testing"

	"gorm-query-template/internal/model"
	"gorm-query-template/internal/repository"
	"gorm-query-template/internal/service"
	"gorm-query-template/pkg/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTest 初始化测试环境，返回 context 和 service
// 使用内存数据库，确保每次测试都是独立的
func setupTest(t *testing.T) (context.Context, service.UserService) {
	// Setup DB (In-memory)
	// 使用 file::memory:?cache=shared 模式或者随机文件名确保隔离
	// 修正：使用随机数据库名称避免缓存冲突
	dbName := fmt.Sprintf("file:memdb_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error), // 减少日志噪音
	})
	require.NoError(t, err, "failed to connect database")

	// Migrate
	err = db.AutoMigrate(&model.User{})
	require.NoError(t, err)

	// Initialize components
	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo)

	ctx := context.Background()

	// 填充标准数据
	seedUsers(t, ctx, svc)

	return ctx, svc
}

func seedUsers(t *testing.T, ctx context.Context, svc service.UserService) {
	users := []struct {
		Name  string
		Email string
		Age   int
	}{
		{"Alice", "alice@example.com", 25},
		{"Bob", "bob@example.com", 30},
		{"Charlie", "charlie@example.com", 35},
		{"David", "david@example.com", 20},
		{"admin", "admin@example.com", 40}, // 在某些测试中应被排除 (小写 "admin")
	}

	for _, u := range users {
		err := svc.Create(ctx, &model.User{
			UserName: u.Name,
			Email:    u.Email,
			Age:      u.Age,
			Status:   1,
		})
		require.NoError(t, err, "failed to create user %s", u.Name)
	}
}

// 1. 创建用户测试 (独立测试，不使用 setupTest 的默认数据，而是手动验证创建过程)
func TestCreateUsers(t *testing.T) {
	// 设置新的 DB
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	require.NoError(t, err)
	err = db.AutoMigrate(&model.User{})
	require.NoError(t, err)

	repo := repository.NewUserRepository(db)
	svc := service.NewUserService(repo)
	ctx := context.Background()

	users := []struct {
		Name  string
		Email string
		Age   int
	}{
		{"Alice", "alice@example.com", 25},
		{"Bob", "bob@example.com", 30},
	}

	for _, u := range users {
		err := svc.Create(ctx, &model.User{
			UserName: u.Name,
			Email:    u.Email,
			Age:      u.Age,
			Status:   1,
		})
		require.NoError(t, err, "failed to create user %s", u.Name)
	}

	// 验证创建
	q := query.New()
	count, err := svc.Count(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

// 2. 测试获取活跃用户
func TestGetActiveUsers(t *testing.T) {
	ctx, svc := setupTest(t)

	// 用例 1: 无关键字
	activeUsers, err := svc.GetActiveUsers(ctx, 25, "")
	require.NoError(t, err)

	// 期望 Charlie(35), Bob(30), Alice(25)。"admin" 应被 NotIn 逻辑排除。
	// David(20) 被年龄排除。
	require.Len(t, activeUsers, 3)
	assert.Equal(t, "Charlie", activeUsers[0].UserName)
	assert.Equal(t, "Bob", activeUsers[1].UserName)
	assert.Equal(t, "Alice", activeUsers[2].UserName)

	// 用例 2: 带关键字 "ali" (匹配 Alice)
	activeUsersAli, err := svc.GetActiveUsers(ctx, 25, "ali")
	require.NoError(t, err)
	require.Len(t, activeUsersAli, 1)
	assert.Equal(t, "Alice", activeUsersAli[0].UserName)
}

// 3. 测试通过用户名查找用户
func TestFindUserByName(t *testing.T) {
	ctx, svc := setupTest(t)

	q := query.New().Where(model.UserProps.UserName.Eq("Bob"))
	bob, err := svc.First(ctx, q)

	require.NoError(t, err)
	require.NotNil(t, bob)
	assert.Equal(t, "Bob", bob.UserName)
	assert.Equal(t, "bob@example.com", bob.Email)
}

// 4. 测试类型安全列的使用
func TestTypeSafeColumnUsage(t *testing.T) {
	ctx, svc := setupTest(t)

	q := query.New().Where(model.UserProps.Email.Eq("alice@example.com"))
	fmt.Println("Query built using:", model.UserProps.Email)

	alice, err := svc.First(ctx, q)
	require.NoError(t, err)
	require.NotNil(t, alice)
	assert.Equal(t, "Alice", alice.UserName)
}

// 5. 测试分页
func TestPagination(t *testing.T) {
	ctx, svc := setupTest(t)

	// 排序逻辑:
	// 创建顺序: Alice, Bob, Charlie, David, admin
	// CreatedAt 倒序: admin, David, Charlie, Bob, Alice
	// 查询: 按 CreatedAt 倒序, 第 1 页, 大小 2
	// 期望: admin, David

	q := query.New().
		Order(model.UserProps.CreatedAt, true).
		Page(1, 2)

	pageUsers, err := svc.Find(ctx, q)

	require.NoError(t, err)
	require.Len(t, pageUsers, 2)
	assert.Equal(t, "admin", pageUsers[0].UserName)
	assert.Equal(t, "David", pageUsers[1].UserName)
}

// 6. 测试 BaseService First (通过 ID 获取)
func TestGetByID(t *testing.T) {
	ctx, svc := setupTest(t)

	// ID 1 应该是 Alice
	qID := query.New().Where(model.UserProps.ID.Eq(1))
	user1, err := svc.First(ctx, qID)
	require.NoError(t, err)
	require.NotNil(t, user1)
	assert.Equal(t, "Alice", user1.UserName)

	// 不存在的 ID
	qID999 := query.New().Where(model.UserProps.ID.Eq(999))
	user999, err := svc.First(ctx, qID999)

	require.Error(t, err)
	require.Nil(t, user999)
}

// 7. 测试查询功能 - HasPrefix
func TestQuery_HasPrefix(t *testing.T) {
	ctx, svc := setupTest(t)

	qPrefix := query.New().Where(model.UserProps.UserName.HasPrefix("Al"))
	usersPrefix, err := svc.Find(ctx, qPrefix)
	require.NoError(t, err)
	require.Len(t, usersPrefix, 1)
	assert.Equal(t, "Alice", usersPrefix[0].UserName)
}

// 8. 测试查询功能 - HasSuffix
func TestQuery_HasSuffix(t *testing.T) {
	ctx, svc := setupTest(t)

	qSuffix := query.New().Where(model.UserProps.UserName.HasSuffix("lie"))
	usersSuffix, err := svc.Find(ctx, qSuffix)
	require.NoError(t, err)
	require.Len(t, usersSuffix, 1)
	assert.Equal(t, "Charlie", usersSuffix[0].UserName)
}

// 9. 测试查询功能 - NotLike
func TestQuery_NotLike(t *testing.T) {
	ctx, svc := setupTest(t)

	qNotLike := query.New().Where(model.UserProps.UserName.NotLike("%a%"))
	usersNotLike, err := svc.Find(ctx, qNotLike)
	require.NoError(t, err)
	// Alice(a), Charlie(a), David(a), admin(a). 只有 Bob 没有 'a' (等等, "Bob" 确实没有 'a')
	require.Len(t, usersNotLike, 1)
	assert.Equal(t, "Bob", usersNotLike[0].UserName)
}

// 10. 测试查询功能 - Select 和 Omit
func TestQuery_Select_Omit(t *testing.T) {
	ctx, svc := setupTest(t)

	// 仅选择 UserName
	qSelect := query.New().
		Select(model.UserProps.UserName).
		Where(model.UserProps.UserName.Eq("Bob"))
	userSelect, err := svc.First(ctx, qSelect)
	require.NoError(t, err)
	assert.Equal(t, "Bob", userSelect.UserName)
	assert.Empty(t, userSelect.Email) // Email 应该为空

	// 忽略 Email
	qOmit := query.New().
		Omit(model.UserProps.Email). // 直接传递 Column
		Where(model.UserProps.UserName.Eq("Bob"))
	userOmit, err := svc.First(ctx, qOmit)
	require.NoError(t, err)
	assert.Equal(t, "Bob", userOmit.UserName)
	assert.Empty(t, userOmit.Email) // Email 应该为空
}

// 11. 测试查询功能 - Distinct
func TestQuery_Distinct(t *testing.T) {
	ctx, svc := setupTest(t)

	// 获取不重复的用户名
	qDistinct := query.New().
		Distinct(model.UserProps.UserName). // 直接传递 Column
		Order(model.UserProps.UserName).
		Select(model.UserProps.UserName) // 仅选择 UserName 以避免 ID 唯一性

	// 注意: Distinct 通常配合 Scan 到字符串切片或结构体切片使用。
	// BaseService.Find 扫描到 []*User。
	// 如果我们只选择 UserName，其他字段将为空。
	usersDistinct, err := svc.Find(ctx, qDistinct)
	require.NoError(t, err)
	// 我们插入了 5 个具有唯一名称的用户 (Alice, Bob, Charlie, David, admin)
	require.Len(t, usersDistinct, 5)
	assert.Equal(t, "Alice", usersDistinct[0].UserName)
}

// 12. 测试查询功能 - Between
func TestQuery_Between(t *testing.T) {
	ctx, svc := setupTest(t)

	// 年龄在 20 到 30 之间 (Alice 25, Bob 30, David 20)
	// 应该包含 20 和 30。
	q := query.New().Where(model.UserProps.Age.Between(20, 30))
	users, err := svc.Find(ctx, q)
	require.NoError(t, err)
	// Alice(25), Bob(30), David(20) -> 3 个用户
	require.Len(t, users, 3)

	// 测试以 Column 作为参数
	// 例如 Age Between Age AND Age -> 应该返回所有 (Age = Age)
	// 这测试了 Between 中 Column 类型的处理
	qCol := query.New().Where(model.UserProps.Age.Between(model.UserProps.Age, model.UserProps.Age))
	usersCol, err := svc.Find(ctx, qCol)
	require.NoError(t, err)
	require.Len(t, usersCol, 5) // 所有用户

	// 测试 Like 配合 Column
	// 例如 UserName LIKE UserName -> 匹配所有
	qLikeCol := query.New().Where(model.UserProps.UserName.Like(model.UserProps.UserName))
	usersLikeCol, err := svc.Find(ctx, qLikeCol)
	require.NoError(t, err)
	require.Len(t, usersLikeCol, 5)
}

// 13. 测试删除
func TestDelete(t *testing.T) {
	ctx, svc := setupTest(t)

	// 1. 根据 ID 删除 (通过 Where ID = ? 模拟)
	// 先找到 Alice 获取 ID
	alice, err := svc.First(ctx, query.New().Where(model.UserProps.UserName.Eq("Alice")))
	require.NoError(t, err)

	// 删除 Alice
	err = svc.Delete(ctx, query.New().Where(model.UserProps.ID.Eq(alice.ID)))
	require.NoError(t, err)

	// 验证 Alice 已删除
	_, err = svc.First(ctx, query.New().Where(model.UserProps.ID.Eq(alice.ID)))
	require.Error(t, err) // 应该记录未找到

	// 2. 批量删除 (删除所有剩余年龄 > 30 的用户)
	// 剩余: Bob(30), Charlie(35), David(20), admin(40)
	// Age > 30: Charlie(35), admin(40)
	err = svc.Delete(ctx, query.New().Where(model.UserProps.Age.Gt(30)))
	require.NoError(t, err)

	// 验证
	remaining, err := svc.Find(ctx, query.New())
	require.NoError(t, err)
	// 应该剩余: Bob(30), David(20) -> 2 个用户
	require.Len(t, remaining, 2)
	for _, u := range remaining {
		if u.UserName == "Charlie" || u.UserName == "admin" {
			t.Errorf("User %s should have been deleted", u.UserName)
		}
	}
}

// 14. 测试创建用户
func TestCreateUser(t *testing.T) {
	ctx, svc := setupTest(t)

	// 1. 创建新用户 - 应该成功
	newUser := &model.User{
		UserName: "Eve",
		Email:    "eve@example.com",
		Age:      22,
		Status:   1,
	}
	err := svc.CreateUser(ctx, newUser)
	require.NoError(t, err)

	// 验证创建
	q := query.New().Where(model.UserProps.UserName.Eq("Eve"))
	user, err := svc.First(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, "Eve", user.UserName)

	// 2. 创建重复用户 - 应该失败
	duplicateUser := &model.User{
		UserName: "Eve",
		Email:    "eve.duplicate@example.com",
		Age:      22,
		Status:   1,
	}
	err = svc.CreateUser(ctx, duplicateUser)
	require.Error(t, err)
	assert.Equal(t, service.ErrUserAlreadyExists, err)
}
