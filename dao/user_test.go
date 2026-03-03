package dao

import (
	"fmt"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_CreateUser(t *testing.T) {
	// Test successfully creating a user record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		user := &models.User{Username: "admin", DisplayName: "Admin", PasswordHash: "password", Role: types.UserRoleAdmin}
		require.NoError(t, dao.CreateUser(ctx, user))
	})

	// Test error due to duplicate record
	t.Run("duplicate", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		user := &models.User{Username: "test-user", DisplayName: "Test User", PasswordHash: "password", Role: types.UserRoleUser}
		require.NoError(t, dao.CreateUser(ctx, user))

		require.ErrorContains(t, dao.CreateUser(ctx, user), "UNIQUE constraint failed: "+models.USER_TABLE_USERNAME)
	})

	// Test error due to nil pointer
	t.Run("nil pointer", func(t *testing.T) {
		dao, ctx := setup(t)
		require.ErrorIs(t, dao.CreateUser(ctx, nil), utils.ErrNilPtr)
	})

	// Test error due to empty username
	t.Run("empty username", func(t *testing.T) {
		dao, ctx := setup(t)

		user := &models.User{Username: "", DisplayName: "Admin", PasswordHash: "password", Role: types.UserRoleAdmin}
		require.ErrorIs(t, dao.CreateUser(ctx, user), utils.ErrUsername)
	})

	// Test error due to empty password
	t.Run("empty password", func(t *testing.T) {
		dao, ctx := setup(t)

		user := &models.User{Username: "admin", DisplayName: "Admin", PasswordHash: "", Role: types.UserRoleAdmin}
		require.ErrorIs(t, dao.CreateUser(ctx, user), utils.ErrUserPassword)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_GetUser(t *testing.T) {
	// Test successfully retrieving a user record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		user := &models.User{Username: "admin", DisplayName: "Admin", PasswordHash: "password", Role: types.UserRoleAdmin}
		require.NoError(t, dao.CreateUser(ctx, user))

		dbOpts = NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: user.ID})
		record, err := dao.GetUser(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, user.ID, record.ID)
	})

	// Test no error when retrieving a non-existent user record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		record, err := dao.GetUser(ctx, nil)
		require.Nil(t, err)
		require.Nil(t, record)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_ListUsers(t *testing.T) {
	// Test successfully retrieving all user records
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		users := []*models.User{}

		for i := range 3 {
			user := &models.User{
				Username:     fmt.Sprintf("user%d", i),
				DisplayName:  fmt.Sprintf("User %d", i),
				PasswordHash: "password",
				Role:         types.UserRoleUser,
			}
			users = append(users, user)
			require.NoError(t, dao.CreateUser(ctx, user))

			time.Sleep(1 * time.Millisecond)
		}

		records, err := dao.ListUsers(ctx, nil)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, users[i].ID, record.ID)
		}
	})

	// Test successfully retrieving no user records
	t.Run("empty", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		records, err := dao.ListUsers(ctx, nil)
		require.Nil(t, err)
		require.Empty(t, records)
	})

	// Test successfully retrieving ordered user records
	t.Run("order by", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		users := []*models.User{}
		for i := range 3 {
			user := &models.User{
				Username:     fmt.Sprintf("user%d", i),
				DisplayName:  fmt.Sprintf("User %d", i),
				PasswordHash: "password",
				Role:         types.UserRoleUser,
			}
			users = append(users, user)
			require.NoError(t, dao.CreateUser(ctx, user))
			time.Sleep(1 * time.Millisecond)
		}

		// Descending order by created_at
		opts := NewOptions().WithOrderBy(models.USER_TABLE_CREATED_AT + " DESC")

		records, err := dao.ListUsers(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, users[2-i].ID, record.ID)
		}

		// Ascending order by created_at
		opts = NewOptions().WithOrderBy(models.USER_TABLE_CREATED_AT + " ASC")

		records, err = dao.ListUsers(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 3)

		for i, record := range records {
			require.Equal(t, users[i].ID, record.ID)
		}
	})

	// Test successfully retrieving selected user records
	t.Run("where", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		user := &models.User{
			Username:     "test-user",
			DisplayName:  "Test User",
			PasswordHash: "password",
			Role:         types.UserRoleUser,
		}
		require.NoError(t, dao.CreateUser(ctx, user))

		opts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: user.ID})
		records, err := dao.ListUsers(ctx, opts)
		require.Nil(t, err)
		require.Len(t, records, 1)
		require.Equal(t, user.ID, records[0].ID)
	})

	// Test successfully retrieving paginated user records
	t.Run("pagination", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		users := []*models.User{}
		for i := range 17 {
			user := &models.User{
				Username:     fmt.Sprintf("user%d", i),
				DisplayName:  fmt.Sprintf("User %d", i),
				PasswordHash: "password",
				Role:         types.UserRoleUser,
			}
			users = append(users, user)
			require.NoError(t, dao.CreateUser(ctx, user))
			time.Sleep(1 * time.Millisecond)
		}

		// First page with 10 records
		p := NewOptions().WithPagination(pagination.New(1, 10))
		records, err := dao.ListUsers(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 10)
		require.Equal(t, users[0].ID, records[0].ID)
		require.Equal(t, users[9].ID, records[9].ID)

		// Second page with remaining 7 records
		p = NewOptions().WithPagination(pagination.New(2, 10))
		records, err = dao.ListUsers(ctx, p)
		require.Nil(t, err)
		require.Len(t, records, 7)
		require.Equal(t, users[10].ID, records[0].ID)
		require.Equal(t, users[16].ID, records[6].ID)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_UpdateUser(t *testing.T) {
	// Test successfully updating a user record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		OriginalUser := &models.User{Username: "Admin", DisplayName: "Michael", Role: types.UserRoleAdmin, PasswordHash: "password"}
		require.NoError(t, dao.CreateUser(ctx, OriginalUser))

		time.Sleep(1 * time.Millisecond)

		updatedUser := &models.User{
			Base:         OriginalUser.Base,
			Username:     "nimda",            // Immutable
			DisplayName:  "Bob",              // Mutable
			Role:         types.UserRoleUser, // Mutable
			PasswordHash: "new password",     // Mutable
		}
		require.NoError(t, dao.UpdateUser(ctx, updatedUser))

		dbOpts = NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: OriginalUser.ID})
		record, err := dao.GetUser(ctx, dbOpts)
		require.Nil(t, err)
		require.Equal(t, OriginalUser.ID, record.ID)                     // No change
		require.Equal(t, OriginalUser.Username, record.Username)         // No change
		require.True(t, record.CreatedAt.Equal(OriginalUser.CreatedAt))  // No change
		require.Equal(t, updatedUser.DisplayName, record.DisplayName)    // Changed
		require.Equal(t, updatedUser.PasswordHash, record.PasswordHash)  // Changed
		require.Equal(t, updatedUser.Role, record.Role)                  // Changed
		require.False(t, record.UpdatedAt.Equal(OriginalUser.UpdatedAt)) // Changed
	})

	// Test error due to invalid user
	t.Run("invalid", func(t *testing.T) {
		dao, ctx := setup(t)

		user := &models.User{Username: "Admin", DisplayName: "Michael", Role: types.UserRoleAdmin, PasswordHash: "password"}
		require.NoError(t, dao.CreateUser(ctx, user))

		// Empty ID
		user.ID = ""
		require.ErrorIs(t, dao.UpdateUser(ctx, user), utils.ErrId)

		// Nil Model
		require.ErrorIs(t, dao.UpdateUser(ctx, nil), utils.ErrNilPtr)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func Test_DeleteUsers(t *testing.T) {
	// Test successfully deleting a user record
	t.Run("success", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		user := &models.User{Username: "test-user", DisplayName: "Test User", PasswordHash: "password", Role: types.UserRoleUser}
		require.NoError(t, dao.CreateUser(ctx, user))

		opts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: user.ID})
		require.Nil(t, dao.DeleteUsers(ctx, opts))

		records, err := dao.ListUsers(ctx, opts)
		require.NoError(t, err)
		require.Empty(t, records)
	})

	// Test no error when deleting a non-existent user record
	t.Run("not found", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		user := &models.User{Username: "test-user", DisplayName: "Test User", PasswordHash: "password", Role: types.UserRoleUser}
		require.NoError(t, dao.CreateUser(ctx, user))

		opts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_ID: "non-existent"})
		require.Nil(t, dao.DeleteUsers(ctx, opts))

		records, err := dao.ListUsers(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, user.ID, records[0].ID)
	})

	// Test error due to missing where clause
	t.Run("missing where", func(t *testing.T) {
		dao, ctx := setup(t)

		// Remove the existing admin user
		dbOpts := NewOptions().WithWhere(squirrel.Eq{models.USER_TABLE_USERNAME: "test-user"})
		require.NoError(t, dao.DeleteUsers(ctx, dbOpts))

		user := &models.User{Username: "test-user", DisplayName: "Test User", PasswordHash: "password", Role: types.UserRoleUser}
		require.NoError(t, dao.CreateUser(ctx, user))

		require.ErrorIs(t, dao.DeleteUsers(ctx, nil), utils.ErrWhere)

		records, err := dao.ListUsers(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, user.ID, records[0].ID)
	})
}
