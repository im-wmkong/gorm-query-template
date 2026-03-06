package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"gorm-query-template/internal/model"
	"gorm-query-template/internal/repository"
	"gorm-query-template/internal/service"
	"gorm-query-template/pkg/db"
	"gorm-query-template/pkg/query"
	"gorm-query-template/pkg/transaction"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// setupTest 初始化测试环境，返回 context, service 和 repository
// 使用内存数据库，确保每次测试都是独立的
func setupTest(t *testing.T) (context.Context, service.UserService, repository.UserRepository) {
	// Setup DB (In-memory)
	// 使用 file::memory:?cache=shared 模式或者随机文件名确保隔离
	// 修正：使用随机数据库名称避免缓存冲突
	dbName := fmt.Sprintf("file:memdb_%s?mode=memory&cache=shared", t.Name())
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second,  // Slow SQL threshold
			LogLevel:                  logger.Error, // Log level
			IgnoreRecordNotFoundError: true,         // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,        // Disable color
		},
	)
	gormDB, err := gorm.Open(sqlite.Open(dbName), &gorm.Config{
		Logger: newLogger, // 减少日志噪音
	})
	require.NoError(t, err, "failed to connect database")

	// Migrate
	err = gormDB.AutoMigrate(&model.User{})
	require.NoError(t, err)

	// Initialize components
	connector := db.NewClient(gormDB)
	tm := transaction.NewManager(connector)
	repo := repository.NewUserRepository(connector)
	svc := service.NewUserService(repo, tm)

	ctx := context.Background()

	// 填充标准数据
	seedUsers(t, ctx, repo)

	return ctx, svc, repo
}

func seedUsers(t *testing.T, ctx context.Context, repo repository.UserRepository) {
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
		err := repo.Create(ctx, &model.User{
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
	gormDB, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Error),
	})
	require.NoError(t, err)
	err = gormDB.AutoMigrate(&model.User{})
	require.NoError(t, err)

	connector := db.NewClient(gormDB)
	repo := repository.NewUserRepository(connector)
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
		err := repo.Create(ctx, &model.User{
			UserName: u.Name,
			Email:    u.Email,
			Age:      u.Age,
			Status:   1,
		})
		require.NoError(t, err, "failed to create user %s", u.Name)
	}

	// 验证创建
	q := query.New()
	count, err := repo.Count(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, int64(2), count)
}

// 2. 测试获取活跃用户
func TestGetActiveUsers(t *testing.T) {
	ctx, svc, _ := setupTest(t)

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
	ctx, _, repo := setupTest(t)

	q := query.New().Where(model.UserProps.UserName.Eq("Bob"))
	bob, err := repo.First(ctx, q)

	require.NoError(t, err)
	require.NotNil(t, bob)
	assert.Equal(t, "Bob", bob.UserName)
	assert.Equal(t, "bob@example.com", bob.Email)
}

// 4. 测试类型安全列的使用
func TestTypeSafeColumnUsage(t *testing.T) {
	ctx, _, repo := setupTest(t)

	q := query.New().Where(model.UserProps.Email.Eq("alice@example.com"))

	alice, err := repo.First(ctx, q)
	require.NoError(t, err)
	require.NotNil(t, alice)
	assert.Equal(t, "Alice", alice.UserName)
}

// 5. 测试分页
func TestPagination(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 排序逻辑:
	// 创建顺序: Alice, Bob, Charlie, David, admin
	// CreatedAt 倒序: admin, David, Charlie, Bob, Alice
	// 查询: 按 CreatedAt 倒序, 第 1 页, 大小 2
	// 期望: admin, David

	q := query.New().
		Order(model.UserProps.CreatedAt, true).
		Page(1, 2)

	pageUsers, err := repo.Find(ctx, q)

	require.NoError(t, err)
	require.Len(t, pageUsers, 2)
	assert.Equal(t, "admin", pageUsers[0].UserName)
	assert.Equal(t, "David", pageUsers[1].UserName)
}

// 6. 测试 BaseService First (通过 ID 获取)
func TestGetByID(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// ID 1 应该是 Alice
	qID := query.New().Where(model.UserProps.ID.Eq(1))
	user1, err := repo.First(ctx, qID)
	require.NoError(t, err)
	require.NotNil(t, user1)
	assert.Equal(t, "Alice", user1.UserName)

	// 不存在的 ID
	qID999 := query.New().Where(model.UserProps.ID.Eq(999))
	user999, err := repo.First(ctx, qID999)

	require.Error(t, err)
	require.Nil(t, user999)
}

// 7. 测试查询功能 - HasPrefix
func TestQuery_HasPrefix(t *testing.T) {
	ctx, _, repo := setupTest(t)

	qPrefix := query.New().Where(model.UserProps.UserName.HasPrefix("Al"))
	usersPrefix, err := repo.Find(ctx, qPrefix)
	require.NoError(t, err)
	require.Len(t, usersPrefix, 1)
	assert.Equal(t, "Alice", usersPrefix[0].UserName)
}

// 8. 测试查询功能 - HasSuffix
func TestQuery_HasSuffix(t *testing.T) {
	ctx, _, repo := setupTest(t)

	qSuffix := query.New().Where(model.UserProps.UserName.HasSuffix("lie"))
	usersSuffix, err := repo.Find(ctx, qSuffix)
	require.NoError(t, err)
	require.Len(t, usersSuffix, 1)
	assert.Equal(t, "Charlie", usersSuffix[0].UserName)
}

// 9. 测试查询功能 - NotLike
func TestQuery_NotLike(t *testing.T) {
	ctx, _, repo := setupTest(t)

	qNotLike := query.New().Where(model.UserProps.UserName.NotLike("%a%"))
	usersNotLike, err := repo.Find(ctx, qNotLike)
	require.NoError(t, err)
	// Alice(a), Charlie(a), David(a), admin(a). 只有 Bob 没有 'a' (等等, "Bob" 确实没有 'a')
	require.Len(t, usersNotLike, 1)
	assert.Equal(t, "Bob", usersNotLike[0].UserName)
}

// 10. 测试查询功能 - Select 和 Omit
func TestQuery_Select_Omit(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 仅选择 UserName
	qSelect := query.New().
		Select(model.UserProps.UserName).
		Where(model.UserProps.UserName.Eq("Bob"))
	userSelect, err := repo.First(ctx, qSelect)
	require.NoError(t, err)
	assert.Equal(t, "Bob", userSelect.UserName)
	assert.Empty(t, userSelect.Email) // Email 应该为空

	// 忽略 Email
	qOmit := query.New().
		Omit(model.UserProps.Email). // 直接传递 Column
		Where(model.UserProps.UserName.Eq("Bob"))
	userOmit, err := repo.First(ctx, qOmit)
	require.NoError(t, err)
	assert.Equal(t, "Bob", userOmit.UserName)
	assert.Empty(t, userOmit.Email) // Email 应该为空
}

// 11. 测试查询功能 - Distinct
func TestQuery_Distinct(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 获取不重复的用户名
	qDistinct := query.New().
		Distinct(model.UserProps.UserName). // 直接传递 Column
		Order(model.UserProps.UserName).
		Select(model.UserProps.UserName) // 仅选择 UserName 以避免 ID 唯一性

	// 注意: Distinct 通常配合 Scan 到字符串切片或结构体切片使用。
	// BaseService.Find 扫描到 []*User。
	// 如果我们只选择 UserName，其他字段将为空。
	usersDistinct, err := repo.Find(ctx, qDistinct)
	require.NoError(t, err)
	// 我们插入了 5 个具有唯一名称的用户 (Alice, Bob, Charlie, David, admin)
	require.Len(t, usersDistinct, 5)
	assert.Equal(t, "Alice", usersDistinct[0].UserName)
}

// 12. 测试查询功能 - Between
func TestQuery_Between(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 年龄在 20 到 30 之间 (Alice 25, Bob 30, David 20)
	// 应该包含 20 和 30。
	q := query.New().Where(model.UserProps.Age.Between(20, 30))
	users, err := repo.Find(ctx, q)
	require.NoError(t, err)
	// Alice(25), Bob(30), David(20) -> 3 个用户
	require.Len(t, users, 3)

	// 测试以 Column 作为参数
	// 例如 Age Between Age AND Age -> 应该返回所有 (Age = Age)
	// 这测试了 Between 中 Column 类型的处理
	qCol := query.New().Where(model.UserProps.Age.Between(model.UserProps.Age, model.UserProps.Age))
	usersCol, err := repo.Find(ctx, qCol)
	require.NoError(t, err)
	require.Len(t, usersCol, 5) // 所有用户

	// 测试 Like 配合 Column
	// 例如 UserName LIKE UserName -> 匹配所有
	qLikeCol := query.New().Where(model.UserProps.UserName.Like(model.UserProps.UserName))
	usersLikeCol, err := repo.Find(ctx, qLikeCol)
	require.NoError(t, err)
	require.Len(t, usersLikeCol, 5)
}

// 13. 测试删除
func TestDelete(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 1. 根据 ID 删除 (通过 Where ID = ? 模拟)
	// 先找到 Alice 获取 ID
	alice, err := repo.First(ctx, query.New().Where(model.UserProps.UserName.Eq("Alice")))
	require.NoError(t, err)

	// 删除 Alice
	err = repo.Delete(ctx, query.New().Where(model.UserProps.ID.Eq(alice.ID)))
	require.NoError(t, err)

	// 验证 Alice 已删除
	_, err = repo.First(ctx, query.New().Where(model.UserProps.ID.Eq(alice.ID)))
	require.Error(t, err) // 应该记录未找到

	// 2. 批量删除 (删除所有剩余年龄 > 30 的用户)
	// 剩余: Bob(30), Charlie(35), David(20), admin(40)
	// Age > 30: Charlie(35), admin(40)
	err = repo.Delete(ctx, query.New().Where(model.UserProps.Age.Gt(30)))
	require.NoError(t, err)

	// 验证
	remaining, err := repo.Find(ctx, query.New())
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
	ctx, svc, repo := setupTest(t)

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
	user, err := repo.First(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, "Eve", user.UserName)

	// 2. 创建重复用户 - 应该失败
	duplicateUser := &model.User{
		UserName: "Eve",
		Email:    "eve@example.com",
		Age:      22,
		Status:   1,
	}
	err = svc.CreateUser(ctx, duplicateUser)
	require.Error(t, err)
	assert.Equal(t, service.ErrUserAlreadyExists, err)
}

// 15. 测试查询功能 - Neq (Not Equal)
func TestQuery_Neq(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 获取不是 Alice 的用户
	qNeq := query.New().Where(model.UserProps.UserName.Neq("Alice"))
	usersNeq, err := repo.Find(ctx, qNeq)
	require.NoError(t, err)
	// Alice, Bob, Charlie, David, admin -> 5 users
	// Expect 4
	require.Len(t, usersNeq, 4)

	for _, u := range usersNeq {
		assert.NotEqual(t, "Alice", u.UserName)
	}

	// Test Column comparison: Age <> Age -> False for all (should return 0 results if Age is not null)
	// But Age is not null. So WHERE Age <> Age matches nothing.
	qCol := query.New().Where(model.UserProps.Age.Neq(model.UserProps.Age))
	usersCol, err := repo.Find(ctx, qCol)
	require.NoError(t, err)
	require.Empty(t, usersCol)
}

// 16. 测试查询功能 - Gt, Gte, Lt, Lte
func TestQuery_Comparison(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// Gt: Age > 30 (Charlie 35, admin 40)
	qGt := query.New().Where(model.UserProps.Age.Gt(30))
	usersGt, err := repo.Find(ctx, qGt)
	require.NoError(t, err)
	require.Len(t, usersGt, 2)

	// Gte: Age >= 30 (Bob 30, Charlie 35, admin 40)
	qGte := query.New().Where(model.UserProps.Age.Gte(30))
	usersGte, err := repo.Find(ctx, qGte)
	require.NoError(t, err)
	require.Len(t, usersGte, 3)

	// Lt: Age < 25 (David 20)
	qLt := query.New().Where(model.UserProps.Age.Lt(25))
	usersLt, err := repo.Find(ctx, qLt)
	require.NoError(t, err)
	require.Len(t, usersLt, 1)
	assert.Equal(t, "David", usersLt[0].UserName)

	// Lte: Age <= 25 (Alice 25, David 20)
	qLte := query.New().Where(model.UserProps.Age.Lte(25))
	usersLte, err := repo.Find(ctx, qLte)
	require.NoError(t, err)
	require.Len(t, usersLte, 2)

	// Column comparison
	// Age < Age -> False
	qLtCol := query.New().Where(model.UserProps.Age.Lt(model.UserProps.Age))
	usersLtCol, err := repo.Find(ctx, qLtCol)
	require.NoError(t, err)
	require.Empty(t, usersLtCol)

	// Age <= Age -> True (All)
	qLteCol := query.New().Where(model.UserProps.Age.Lte(model.UserProps.Age))
	usersLteCol, err := repo.Find(ctx, qLteCol)
	require.NoError(t, err)
	require.Len(t, usersLteCol, 5)

	// Age > Age -> False
	qGtCol := query.New().Where(model.UserProps.Age.Gt(model.UserProps.Age))
	usersGtCol, err := repo.Find(ctx, qGtCol)
	require.NoError(t, err)
	require.Empty(t, usersGtCol)

	// Age >= Age -> True (All)
	qGteCol := query.New().Where(model.UserProps.Age.Gte(model.UserProps.Age))
	usersGteCol, err := repo.Find(ctx, qGteCol)
	require.NoError(t, err)
	require.Len(t, usersGteCol, 5)
}

// 17. 测试查询功能 - In, NotIn
func TestQuery_In_NotIn(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// In: Alice, Bob
	names := []string{"Alice", "Bob"}
	qIn := query.New().Where(model.UserProps.UserName.In(names))
	usersIn, err := repo.Find(ctx, qIn)
	require.NoError(t, err)
	require.Len(t, usersIn, 2)

	// NotIn: Alice, Bob -> Charlie, David, admin
	qNotIn := query.New().Where(model.UserProps.UserName.NotIn(names))
	usersNotIn, err := repo.Find(ctx, qNotIn)
	require.NoError(t, err)
	require.Len(t, usersNotIn, 3)
}

// 18. 测试查询功能 - IsNull, IsNotNull
func TestQuery_Null_NotNull(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// Create a user with NULL Email (Assuming Email can be null, let's check model)
	// model.User struct has string for Email, usually empty string in Go is not NULL in DB unless pointer.
	// Looking at model/user.go: Email string `gorm:"uniqueIndex;size:128"`
	// It's a string, not *string, so it will be empty string "", not NULL.
	// But let's check if we can insert a NULL using map or raw SQL if we want to test IsNull.
	// Or we can test on DeletedAt which is gorm.DeletedAt (Time pointer wrapper).

	// For this test, let's use DeletedAt field which is nullable.

	// Default users are not deleted, so DeletedAt IS NULL
	qIsNull := query.New().Where(model.UserProps.DeletedAt.IsNull())
	usersIsNull, err := repo.Find(ctx, qIsNull)
	require.NoError(t, err)
	require.Len(t, usersIsNull, 5)

	// Soft delete Alice
	alice, _ := repo.First(ctx, query.New().Where(model.UserProps.UserName.Eq("Alice")))
	repo.Delete(ctx, query.New().Where(model.UserProps.ID.Eq(alice.ID)))

	// Now query Unscoped to find deleted user
	qIsNotNull := query.New().
		Unscoped().
		Where(model.UserProps.DeletedAt.IsNotNull())

	usersIsNotNull, err := repo.Find(ctx, qIsNotNull)
	require.NoError(t, err)
	require.Len(t, usersIsNotNull, 1)
	assert.Equal(t, "Alice", usersIsNotNull[0].UserName)
}

// 19. 测试查询功能 - Or
func TestQuery_Or(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// Name = Alice OR Name = Bob
	// 使用 Or 方法
	qOr := query.New().Where(
		model.UserProps.UserName.Eq("Alice"),
	).Or(
		model.UserProps.UserName.Eq("Bob"),
	)

	usersOr, err := repo.Find(ctx, qOr)
	require.NoError(t, err)
	// Alice, Bob
	require.Len(t, usersOr, 2)
}

// 20. 测试查询功能 - Group & Having
func TestQuery_Group_Having(t *testing.T) {
	// 准备数据: 添加另一个 Age=25 的用户，以便分组测试
	ctx, _, repo := setupTest(t)
	repo.Create(ctx, &model.User{
		UserName: "Frank",
		Email:    "frank@example.com",
		Age:      25,
		Status:   1,
	})

	// Group by Age, Having Count(*) > 1
	// 应该找到 Age=25 (Alice, Frank)

	qGroup := query.New().
		Select(model.UserProps.Age).
		Group(model.UserProps.Age).
		Having("count(*) > ?", 1)

	// 注意: Find 的结果将只有 Age 字段被填充
	usersGroup, err := repo.Find(ctx, qGroup)
	require.NoError(t, err)

	require.Len(t, usersGroup, 1)
	assert.Equal(t, 25, usersGroup[0].Age)
}

// 21. 测试查询功能 - Preload
func TestQuery_Preload(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// User 模型目前没有定义关联关系。
	// 我们仅仅调用一下 Preload 看看是否 panic，或者 SQL 是否生成 (虽然没有任何效果)
	// 或者我们可以假装有一个关联 "Profile"

	// 这是一个简单的 smoke test，确保代码路径被覆盖
	qPreload := query.New().Preload("Orders") // 假设有 Orders
	// 这会报错: model.User' does not have relation 'Orders'
	// 所以我们应该期望错误

	_, err := repo.Find(ctx, qPreload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Orders")
}

// 22. 测试查询功能 - Limit & Offset & Page
func TestQuery_Limit_Offset(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// Limit 2
	qLimit := query.New().Limit(2).Order(model.UserProps.ID)
	usersLimit, err := repo.Find(ctx, qLimit)
	require.NoError(t, err)
	require.Len(t, usersLimit, 2)
	assert.Equal(t, "Alice", usersLimit[0].UserName)
	assert.Equal(t, "Bob", usersLimit[1].UserName)

	// Limit 2 Offset 2
	qOffset := query.New().Limit(2).Offset(2).Order(model.UserProps.ID)
	usersOffset, err := repo.Find(ctx, qOffset)
	require.NoError(t, err)
	require.Len(t, usersOffset, 2)
	assert.Equal(t, "Charlie", usersOffset[0].UserName)
	assert.Equal(t, "David", usersOffset[1].UserName)

	// Page edge cases
	qPage0 := query.New().Page(0, 0) // Should default to 1, 10
	usersPage0, err := repo.Find(ctx, qPage0)
	require.NoError(t, err)
	require.Len(t, usersPage0, 5) // Alice, Bob, Charlie, David, admin
}

// 23. 测试查询功能 - Scope
func TestQuery_Scope(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 定义一个 Scope
	activeScope := func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", 1)
	}

	qScope := query.New().Scope(activeScope)
	usersScope, err := repo.Find(ctx, qScope)
	require.NoError(t, err)

	// All seeded users have status 1
	require.Len(t, usersScope, 5)
}

// 24. 测试查询功能 - Unscoped
func TestQuery_Unscoped(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 删除 Alice
	alice, _ := repo.First(ctx, query.New().Where(model.UserProps.UserName.Eq("Alice")))
	repo.Delete(ctx, query.New().Where(model.UserProps.ID.Eq(alice.ID)))

	// Normal find - Alice missing
	users, _ := repo.Find(ctx, query.New())
	require.Len(t, users, 4)

	// Unscoped find - Alice present
	usersUnscoped, err := repo.Find(ctx, query.New().Unscoped())
	require.NoError(t, err)
	require.Len(t, usersUnscoped, 5)
}

// 25. 测试 Order (String vs Column)
func TestQuery_Order_Variants(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// String "id DESC"
	qStr := query.New().Order("id DESC")
	usersStr, _ := repo.Find(ctx, qStr)
	assert.Equal(t, "admin", usersStr[0].UserName) // ID 5

	// Column Asc
	qColAsc := query.New().Order(model.UserProps.ID)
	usersColAsc, _ := repo.Find(ctx, qColAsc)
	assert.Equal(t, "Alice", usersColAsc[0].UserName) // ID 1

	// Column Desc
	qColDesc := query.New().Order(model.UserProps.ID, true)
	usersColDesc, _ := repo.Find(ctx, qColDesc)
	assert.Equal(t, "admin", usersColDesc[0].UserName) // ID 5
}

// 26. 测试 Update
func TestUpdate(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 更新 Alice 的 Age 为 26
	q := query.New().Where(model.UserProps.UserName.Eq("Alice"))
	err := repo.Update(ctx, q, "age", 26)
	require.NoError(t, err)

	// 验证
	alice, err := repo.First(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 26, alice.Age)
}

// 27. 测试 Updates
func TestUpdates(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 更新 Bob 的 Age 和 Status
	q := query.New().Where(model.UserProps.UserName.Eq("Bob"))
	updates := map[string]interface{}{
		"age":    31,
		"status": 2,
	}
	err := repo.Updates(ctx, q, updates)
	require.NoError(t, err)

	// 验证
	bob, err := repo.First(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 31, bob.Age)
	assert.Equal(t, 2, bob.Status)
}

// 28. 测试 Save
func TestSave(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 获取 Charlie
	q := query.New().Where(model.UserProps.UserName.Eq("Charlie"))
	charlie, err := repo.First(ctx, q)
	require.NoError(t, err)

	// 修改
	charlie.Age = 36
	charlie.Status = 0

	// 保存
	err = repo.Save(ctx, charlie)
	require.NoError(t, err)

	// 验证
	charlieNew, err := repo.First(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 36, charlieNew.Age)
	assert.Equal(t, 0, charlieNew.Status)
}

// 29. 测试 Joins
func TestJoins(t *testing.T) {
	ctx, _, repo := setupTest(t)
	// 虽然没有关联表，但我们可以测试生成的 SQL 不报错，或者测试 self-join 语法
	// SELECT users.* FROM users JOIN users as u2 ON users.id = u2.id

	// 使用 func(db *gorm.DB) *gorm.DB 作为 Where 参数
	q := query.New().Joins("JOIN users as u2 ON users.id = u2.id").Where(func(db *gorm.DB) *gorm.DB {
		return db.Where("u2.user_name = ?", "Alice")
	})
	users, err := repo.Find(ctx, q)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, "Alice", users[0].UserName)
}

// 30. 测试 Group 字符串参数
func TestGroupString(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// Group("age") instead of Column
	q := query.New().Select("age").Group("age").Having("count(*) > ?", 0)
	users, err := repo.Find(ctx, q)
	require.NoError(t, err)
	// Should have distinct ages: 20, 25, 30, 35, 40
	require.Len(t, users, 5)
}

// 31. 测试 Coverage 补充用例
func TestCoverageSupplement(t *testing.T) {
	ctx, _, repo := setupTest(t)

	// 1. Select String
	qSelect := query.New().Select("user_name").Where(model.UserProps.UserName.Eq("Alice"))
	uSelect, err := repo.First(ctx, qSelect)
	require.NoError(t, err)
	assert.Equal(t, "Alice", uSelect.UserName)

	// 2. Omit String
	qOmit := query.New().Omit("email").Where(model.UserProps.UserName.Eq("Alice"))
	uOmit, err := repo.First(ctx, qOmit)
	require.NoError(t, err)
	assert.Empty(t, uOmit.Email)

	// 3. Order Invalid Type (should be ignored)
	qOrder := query.New().Order(123).Where(model.UserProps.UserName.Eq("Alice"))
	uOrder, err := repo.First(ctx, qOrder)
	require.NoError(t, err)
	assert.Equal(t, "Alice", uOrder.UserName)

	// 4. Eq(Column)
	// UserName = UserName -> All
	qEqCol := query.New().Where(model.UserProps.UserName.Eq(model.UserProps.UserName))
	usersEqCol, err := repo.Find(ctx, qEqCol)
	require.NoError(t, err)
	require.Len(t, usersEqCol, 5)

	// 5. NotLike(Column)
	// UserName NOT LIKE UserName -> None
	qNotLikeCol := query.New().Where(model.UserProps.UserName.NotLike(model.UserProps.UserName))
	usersNotLikeCol, err := repo.Find(ctx, qNotLikeCol)
	require.NoError(t, err)
	require.Len(t, usersNotLikeCol, 0)
}
