package session

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"

	"github.com/geerew/off-course/utils"
	"github.com/stretchr/testify/require"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqlite_Set(t *testing.T) {
	// Test successfully creating a session
	t.Run("success", func(t *testing.T) {
		db, ctx := setup(t)

		storage := NewSqliteStorage(db, time.Hour)

		payload := map[string]any{
			"id":   "user-123",
			"role": "admin",
		}
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(payload)
		require.NoError(t, err)

		err = storage.Set("key", buf.Bytes(), time.Second)
		require.NoError(t, err)

		records, err := storage.dao.ListSessions(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)

		require.Equal(t, "key", records[0].ID)
		require.Equal(t, "user-123", records[0].UserId)
		require.Greater(t, records[0].Expires, time.Now().Unix()-1)
		require.NotEmpty(t, records[0].Data)
	})

	// Test successfully replacing an existing session
	t.Run("replace", func(t *testing.T) {
		db, ctx := setup(t)

		storage := NewSqliteStorage(db, time.Hour)

		// First payload
		payload := map[string]any{
			"id":   "user-123",
			"role": "user",
			"msg":  "value",
		}
		var session1 bytes.Buffer
		require.NoError(t, gob.NewEncoder(&session1).Encode(payload))

		err := storage.Set("key", session1.Bytes(), time.Second)
		require.NoError(t, err)

		// Second payload (same key, different content)
		payload = map[string]any{
			"id":   "user-123",
			"role": "admin",
			"msg":  "new value",
		}
		var session2 bytes.Buffer
		require.NoError(t, gob.NewEncoder(&session2).Encode(payload))

		err = storage.Set("key", session2.Bytes(), time.Second)
		require.NoError(t, err)

		records, err := storage.dao.ListSessions(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)
		require.Equal(t, "key", records[0].ID)
		require.Equal(t, "user-123", records[0].UserId)

		require.NotEqual(t, session1.Bytes(), records[0].Data)
		require.Equal(t, session2.Bytes(), records[0].Data)
	})

	// Test error due to no ID
	t.Run("no id", func(t *testing.T) {
		db, ctx := setup(t)

		storage := NewSqliteStorage(db, time.Millisecond)

		err := storage.Set("", []byte("value"), time.Second)
		require.ErrorIs(t, err, utils.ErrId)

		records, err := storage.dao.ListSessions(ctx, nil)
		require.NoError(t, err)
		require.Zero(t, records)
	})

	// Test error due to no user ID
	t.Run("no user ID", func(t *testing.T) {
		db, ctx := setup(t)

		storage := NewSqliteStorage(db, time.Millisecond)

		err := storage.Set("key", []byte("value"), time.Second)
		require.ErrorIs(t, err, utils.ErrUserId)

		records, err := storage.dao.ListSessions(ctx, nil)
		require.NoError(t, err)
		require.Zero(t, records)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqlite_Get(t *testing.T) {
	// Test successfully getting a session
	t.Run("success", func(t *testing.T) {
		db, _ := setup(t)

		storage := NewSqliteStorage(db, time.Hour)

		payload := map[string]any{
			"id":   "user-123",
			"role": "admin",
		}
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(payload)
		require.NoError(t, err)

		err = storage.Set("key", buf.Bytes(), time.Second)
		require.NoError(t, err)

		_, err = storage.Get("key")
		require.NoError(t, err)
	})

	// Test no error when garbage collection deletes a session
	t.Run("gc", func(t *testing.T) {
		db, _ := setup(t)

		storage := NewSqliteStorage(db, time.Millisecond)

		payload := map[string]any{
			"id":   "user-123",
			"role": "admin",
		}
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(payload)
		require.NoError(t, err)

		err = storage.Set("key", buf.Bytes(), time.Millisecond)
		require.NoError(t, err)

		time.Sleep(5 * time.Millisecond)

		res, err := storage.Get("key")
		require.NoError(t, err)
		require.Nil(t, res)
	})

	// Test error due to no ID
	t.Run("no id", func(t *testing.T) {
		db, _ := setup(t)

		storage := NewSqliteStorage(db, time.Hour)

		res, err := storage.Get("")
		require.ErrorIs(t, err, utils.ErrId)
		require.Nil(t, res)
	})

	// Test error due to expired session
	t.Run("expired", func(t *testing.T) {
		db, _ := setup(t)

		storage := NewSqliteStorage(db, time.Hour)

		payload := map[string]any{
			"id":   "user-123",
			"role": "admin",
		}
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(payload)
		require.NoError(t, err)

		err = storage.Set("key", buf.Bytes(), time.Millisecond)
		require.NoError(t, err)

		time.Sleep(2 * time.Millisecond)

		res, err := storage.Get("key")
		require.NoError(t, err)
		require.Nil(t, res)
	})

}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func TestSqlite_Delete(t *testing.T) {
	// Test successfully deleting a session
	t.Run("success", func(t *testing.T) {
		db, ctx := setup(t)

		storage := NewSqliteStorage(db, time.Hour)

		payload := map[string]any{
			"id":   "user-123",
			"role": "admin",
		}
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(payload)
		require.NoError(t, err)

		err = storage.Set("key", buf.Bytes(), time.Second)
		require.NoError(t, err)

		err = storage.Delete("key")
		require.NoError(t, err)

		records, err := storage.dao.ListSessions(ctx, nil)
		require.NoError(t, err)
		require.Zero(t, len(records))
	})

	// Test no error when garbage collection deletes a session
	t.Run("gc", func(t *testing.T) {
		db, _ := setup(t)

		storage := NewSqliteStorage(db, time.Millisecond)

		payload := map[string]any{
			"id":   "user-123",
			"role": "admin",
		}
		var buf bytes.Buffer
		err := gob.NewEncoder(&buf).Encode(payload)
		require.NoError(t, err)

		err = storage.Set("key", buf.Bytes(), time.Millisecond)
		require.NoError(t, err)

		time.Sleep(2 * time.Millisecond)

		err = storage.Delete("key")
		require.NoError(t, err)
	})

	// Test error due to no ID
	t.Run("no id", func(t *testing.T) {
		db, _ := setup(t)

		storage := NewSqliteStorage(db, time.Hour)

		err := storage.Delete("")
		require.ErrorIs(t, err, utils.ErrId)
	})

}

func TestSqlite_DeleteUser(t *testing.T) {
	// Test successfully deleting all sessions for a user
	t.Run("success", func(t *testing.T) {
		db, ctx := setup(t)

		storage := NewSqliteStorage(db, time.Millisecond)

		encode := func(m map[string]any) []byte {
			var b bytes.Buffer
			require.NoError(t, gob.NewEncoder(&b).Encode(m))
			return b.Bytes()
		}

		// Set two sessions for the same user
		p1 := map[string]any{"id": "user-123", "role": "admin"}
		require.NoError(t, storage.Set("key1", encode(p1), time.Second))
		require.NoError(t, storage.Set("key2", encode(p1), time.Second))

		// Set a session for a different user
		p2 := map[string]any{"id": "user-456", "role": "admin"}
		require.NoError(t, storage.Set("key3", encode(p2), time.Second))

		require.NoError(t, storage.DeleteUser("user-123"))

		records, err := storage.dao.ListSessions(ctx, nil)
		require.NoError(t, err)
		require.Len(t, records, 1)

		require.Equal(t, "key3", records[0].ID)
		require.Equal(t, "user-456", records[0].UserId)
		require.Greater(t, records[0].Expires, time.Now().Unix()-1)
		require.NotEmpty(t, records[0].Data)
	})

	// Test error due to no ID
	t.Run("no id", func(t *testing.T) {
		db, _ := setup(t)

		storage := NewSqliteStorage(db, time.Hour)

		err := storage.DeleteUser("")
		require.ErrorIs(t, err, utils.ErrUserId)
	})
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Test successfully deleting all sessions for all users
func TestSqlite_Reset(t *testing.T) {
	db, ctx := setup(t)

	storage := NewSqliteStorage(db, time.Hour)

	payload := map[string]any{
		"id":   "user-123",
		"role": "admin",
	}
	var buf bytes.Buffer
	err := gob.NewEncoder(&buf).Encode(payload)
	require.NoError(t, err)

	err = storage.Set("key 1", buf.Bytes(), time.Second)
	require.NoError(t, err)

	err = storage.Set("key 2", buf.Bytes(), time.Second)
	require.NoError(t, err)

	err = storage.Set("key 3", buf.Bytes(), time.Second)
	require.NoError(t, err)

	err = storage.Reset()
	require.NoError(t, err)

	records, err := storage.dao.ListSessions(ctx, nil)
	require.NoError(t, err)
	require.Zero(t, len(records))
}
