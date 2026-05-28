package cron

import (
	"context"
	"fmt"

	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/database"
	"github.com/geerew/off-course/utils/cardcache"
	"github.com/geerew/off-course/utils/logger"
	"github.com/geerew/off-course/utils/pagination"
	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// cardCacheWarm rebuilds the in-memory card cache from the database and disk
type cardCacheWarm struct {
	db        database.Database
	dao       *dao.DAO
	cardCache *cardcache.CardCache
	logger    *logger.Logger
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func (w *cardCacheWarm) run() error {
	principal := types.Principal{
		UserID: "card-cache-warm",
		Role:   types.UserRoleAdmin,
	}
	ctx := context.WithValue(context.Background(), types.PrincipalContextKey, principal)

	page := 1
	totalPages := 1
	warmed := 0

	for page <= totalPages {
		p := pagination.New(page, 100)
		courses, err := w.dao.ListCourses(ctx, dao.NewOptions().WithPagination(p))
		if err != nil {
			return fmt.Errorf("failed to list courses: %w", err)
		}

		if page == 1 {
			totalPages = p.TotalPages()
		}

		refs := make([]cardcache.CourseCardRef, len(courses))
		for i, course := range courses {
			refs[i] = cardcache.CourseCardRef{
				ID:       course.ID,
				CardPath: course.CardPath,
				CardHash: course.CardHash,
			}
		}

		warmed += w.cardCache.Warm(refs)
		page++
	}

	w.logger.Info().Int("courses", warmed).Msg("Warmed card cache")

	return nil
}
