package session

import (
	"bytes"
	"context"
	"encoding/gob"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// SqliteStorage implements the `fiber.Storage` interface
type SqliteStorage struct {
	dao        *dao.DAO
	gcInterval time.Duration
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// NewSqliteStorage creates a new sqlite storage
func NewSqliteStorage(db database.Database, gcInterval time.Duration) *SqliteStorage {
	storage := &SqliteStorage{
		dao:        dao.New(db),
		gcInterval: gcInterval,
	}

	go storage.gcTicker()

	return storage
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Get gets a session by ID
func (s *SqliteStorage) Get(id string) ([]byte, error) {
	if id == "" {
		return nil, utils.ErrId
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.SESSION_TABLE_ID: id})
	session, err := s.dao.GetSession(context.Background(), dbOpts)
	if err != nil {
		return nil, err
	}

	if session == nil || (session.Expires != 0 && session.Expires <= time.Now().Unix()) {
		return nil, nil
	}

	return session.Data, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Set will create or replace an existing session by ID
func (s *SqliteStorage) Set(id string, data []byte, exp time.Duration) error {
	if id == "" || len(data) <= 0 {
		return utils.ErrId
	}

	var expSeconds int64
	if exp != 0 {
		expSeconds = time.Now().Add(exp).Unix()
	}

	userID, ok := extractUserId(data)
	if !ok {
		return utils.ErrUserId
	}

	session := &models.Session{
		ID:      id,
		Data:    data,
		Expires: expSeconds,
		UserId:  userID,
	}

	return s.dao.CreateOrReplaceSession(context.Background(), session)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Delete deletes a session by ID
func (s *SqliteStorage) Delete(id string) error {
	if id == "" {
		return utils.ErrId
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.SESSION_TABLE_ID: id})
	return s.dao.DeleteSessions(context.Background(), dbOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// DeleteUser deletes all sessions for a user
func (s *SqliteStorage) DeleteUser(id string) error {
	if id == "" {
		return utils.ErrUserId
	}

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.SESSION_TABLE_USER_ID: id})
	return s.dao.DeleteSessions(context.Background(), dbOpts)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Reset deletes all session for all users
func (s *SqliteStorage) Reset() error {
	return s.dao.DeleteAllSessions(context.Background())
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Close is unimplemented for this storage
func (s *SqliteStorage) Close() error {
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// gcTicker runs a garbage collector that deletes expired sessions at a set interval
func (s *SqliteStorage) gcTicker() {
	ticker := time.NewTicker(s.gcInterval)
	ctx := context.Background()
	defer ticker.Stop()

	for t := range ticker.C {
		dbOpts := dao.NewOptions().
			WithWhere(squirrel.And{
				squirrel.LtOrEq{models.SESSION_TABLE_EXPIRES: t.Unix()},
				squirrel.NotEq{models.SESSION_TABLE_EXPIRES: 0},
			})
		s.dao.DeleteSessions(ctx, dbOpts)
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// extractUserId takes a slice of bytes, decodes it and attempts to extract the user ID
//
// It returns a string, and a boolean indicating whether the user ID was found
func extractUserId(b []byte) (string, bool) {
	var m map[string]any
	buf := bytes.NewBuffer(b)
	dec := gob.NewDecoder(buf)

	if err := dec.Decode(&m); err != nil {
		return "", false
	}

	if v, ok := m["id"]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s, true
		}
	}

	return "", false
}
