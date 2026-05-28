package coursescan

// TODO support author.txt/md for course
// TODO support description.txt/md for course

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/geerew/off-course/dao"
	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils"
	"github.com/geerew/off-course/utils/coursemetadata"
	"github.com/geerew/off-course/utils/media/probe"
	"github.com/geerew/off-course/utils/types"
	"github.com/spf13/afero"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type AssetsByChapterPrefix map[string]map[int][]*models.Asset
type AttachmentsByChapterPrefix map[string]map[int][]*models.Attachment

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// Processor scans a course to identify assets and attachments
//
// It can be passed to coursescan.Worker
func Processor(ctx context.Context, s *CourseScan, scanState *ScanState) error {
	if scanState == nil {
		return ErrNilScan
	}

	extractedKeyframes := make(map[string][]float64)

	startTime := time.Now()

	courseID := scanState.CourseID
	s.logger.Info().Str("course_id", courseID).Msg("Starting scan for course")

	scanState.UpdateStatusAndMessage(types.ScanStatusProcessing, "Initializing scan")

	course, err := fetchCourse(ctx, s, courseID)
	if err != nil {
		s.logger.Error().Err(err).Str("course_id", courseID).Msg("Failed to fetch course")
		return err
	}

	if course == nil {
		return nil
	}

	coursePath := course.Path
	courseTitle := course.Title
	s.logger.Debug().
		Str("course_id", courseID).
		Str("course_path", coursePath).
		Str("course_title", courseTitle).
		Msg("Found course")

	defer func() {
		cleanupCtx := context.Background()
		if err := clearCourseMaintenanceWithRetry(cleanupCtx, s, courseID, coursePath); err != nil {
			s.logger.Error().
				Err(err).
				Str("course_id", courseID).
				Str("course_path", coursePath).
				Msg("Failed to clear course from maintenance mode after retries - course may remain locked")
		}
	}()

	available, err := checkAndSetCourseAvailability(ctx, s, course)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Msg("Failed to check course availability")
		return err
	}

	if !available {
		return nil
	}

	if err := enableCourseMaintenance(ctx, s, course); err != nil {
		s.logger.Error().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Msg("Failed to enable course maintenance mode")
		return err
	}

	if _, err := s.fs.Stat(course.Path); err != nil {
		if os.IsNotExist(err) {
			s.logger.Warn().
				Str("course_id", courseID).
				Str("course_path", coursePath).
				Msg("Course path does not exist, skipping scan")
			return nil
		}

		s.logger.Error().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Msg("Failed to access course path")
		return fmt.Errorf("failed to access course path %s: %w", course.Path, err)
	}

	// Read course metadata if it exists
	scanState.UpdateMessage("Reading course metadata")
	metadata, err := coursemetadata.ReadMetadata(s.fs, course.Path)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Msg("Failed to read oc.json, continuing without metadata")
		metadata = nil
	}

	scanState.UpdateMessage("Scanning course directory")

	scanned, err := scanFiles(s, course)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Msg("Failed to scan course directory")
		return err
	}

	scannedAttachments, scannedAssets := flatAttachmentsAndAssets(scanned.lessons)

	s.logger.Info().
		Str("course_id", courseID).
		Str("course_path", coursePath).
		Int("lessons_count", len(scanned.lessons)).
		Int("attachments_count", len(scannedAttachments)).
		Int("assets_count", len(scannedAssets)).
		Str("card_path", scanned.cardPath).
		Msg("Found lessons")

	// List the assets that already exist in the database for this course
	// Include asset metadata so we can calculate course duration correctly when deleting lessons
	dbOpts := dao.NewOptions().
		WithWhere(squirrel.Eq{models.ASSET_COURSE_ID: course.ID}).
		WithAssetMetadata()
	existingGroups, err := s.dao.ListLessons(ctx, dbOpts)
	if err != nil {
		s.logger.Error().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Msg("Failed to list existing lessons")
		return err
	}

	existingAttachments, existingAssets := flatAttachmentsAndAssets(existingGroups)

	if err := populateHashesIfChanged(ctx, s, scannedAssets, existingAssets, course, scanState); err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return err
		}
		s.logger.Error().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Msg("Failed to populate asset hashes")
		return err
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	s.logger.Info().
		Str("course_id", courseID).
		Str("course_path", coursePath).
		Int("scanned_assets", len(scannedAssets)).
		Int("existing_assets", len(existingAssets)).
		Int("scanned_attachments", len(scannedAttachments)).
		Int("existing_attachments", len(existingAttachments)).
		Msg("Generating reconciliation operations")

	groupOps := reconcileLessons(scanned.lessons, existingGroups)
	assetOps := reconcileAssets(scannedAssets, existingAssets)
	attachmentOps := reconcileAttachments(scannedAttachments, existingAttachments)

	s.logger.Info().
		Str("course_id", courseID).
		Str("course_path", coursePath).
		Int("lesson_ops", len(groupOps)).
		Int("asset_ops", len(assetOps)).
		Int("attachment_ops", len(attachmentOps)).
		Msg("Generated reconciliation operations")

	assetMetadataByPath := probeVideos(ctx, s, assetOps, course, extractedKeyframes, scanState)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Handle card changes
	cardChanged, err := handleCourseCard(ctx, s, course, scanned.cardPath, courseID, coursePath, scanState)
	if err != nil {
		return err
	}

	updatedCourse := cardChanged

	opCounts := countOperations(groupOps, assetOps, attachmentOps)

	scanState.UpdateMessage("Applying database changes")
	s.logger.Info().
		Str("course_id", courseID).
		Str("course_path", coursePath).
		Msg("Applying database changes in transaction")

	err = s.db.RunInTransaction(ctx, func(txCtx context.Context) error {
		if updated, err := applyLessonCreateUpdateOps(txCtx, s, groupOps); err != nil {
			return err
		} else if updated {
			updatedCourse = true
		}

		if updated, err := applyAttachmentOps(txCtx, s, attachmentOps); err != nil {
			return err
		} else if updated {
			updatedCourse = true
		}

		if updated, err := applyAssetOps(txCtx, s, course, assetOps, assetMetadataByPath, extractedKeyframes); err != nil {
			return err
		} else if updated {
			updatedCourse = true
		}

		if updated, err := applyLessonDeleteOps(txCtx, s, groupOps, course); err != nil {
			return err
		} else if updated {
			updatedCourse = true
		}

		// Apply tags (if metadata exists)
		if metadata != nil && len(metadata.Tags) > 0 {
			if err := applyTagsFromMetadata(txCtx, s, course.ID, metadata.Tags); err != nil {
				s.logger.Warn().
					Err(err).
					Str("course_id", courseID).
					Str("course_path", coursePath).
					Msg("Failed to apply tags from oc.json")
				// Don't fail the scan if tag application fails
			}
		} else if metadata != nil {
			// Metadata exists but has no tags, remove all tags for this course
			if err := removeAllCourseTags(txCtx, s, course.ID); err != nil {
				s.logger.Warn().
					Err(err).
					Str("course_id", courseID).
					Str("course_path", coursePath).
					Msg("Failed to remove tags from course")
			}
		}

		// Apply description from metadata
		// If metadata exists, use its description (even if empty string)
		// If metadata doesn't exist, set description to empty string
		newDescription := ""
		if metadata != nil {
			newDescription = metadata.Description
		}

		// Only update if description changed
		if course.Description != newDescription {
			course.Description = newDescription
			updatedCourse = true
		}

		if updatedCourse {
			// Recalculate course duration from scratch to ensure accuracy, preventing accumulation errors
			// from incremental updates
			scanState.UpdateMessage("Recalculating course duration")
			recalculatedDuration, err := recalculateCourseDuration(txCtx, s, course.ID)
			if err != nil {
				s.logger.Warn().
					Err(err).
					Str("course_id", courseID).
					Str("course_path", coursePath).
					Msg("Failed to recalculate course duration, using existing value")
			} else {
				course.Duration = recalculatedDuration
			}

			if course.Duration < 0 {
				s.logger.Warn().
					Str("course_id", courseID).
					Str("course_path", coursePath).
					Int("negative_duration", course.Duration).
					Msg("Course duration became negative, resetting to 0")
				course.Duration = 0
			}

			course.InitialScan = true
			s.logger.Debug().
				Str("course_id", courseID).
				Str("course_path", coursePath).
				Int("duration", course.Duration).
				Msg("Updating course metadata")

			return s.dao.UpdateCourse(txCtx, course)
		}

		return nil
	})

	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded || isCancellationError(err) {
			return err
		}
		s.logger.Error().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Msg("Failed to apply changes")
		return err
	}

	duration := time.Since(startTime)

	// When not testing, ensure minimum scan duration of 2 seconds to allow frontend to see the changes
	isTesting := flag.Lookup("test.v") != nil
	if !isTesting {
		const minScanDuration = 2 * time.Second
		if duration < minScanDuration {
			remaining := minScanDuration - duration
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(remaining):
			}
			duration = time.Since(startTime)
		}
	}

	scanState.UpdateMessage("Scan complete")
	s.logger.Info().
		Str("course_id", courseID).
		Str("course_path", coursePath).
		Dur("duration", duration).
		Int("lessons_created", opCounts.LessonsCreated).
		Int("lessons_updated", opCounts.LessonsUpdated).
		Int("lessons_deleted", opCounts.LessonsDeleted).
		Int("assets_created", opCounts.AssetsCreated).
		Int("assets_updated", opCounts.AssetsUpdated).
		Int("assets_deleted", opCounts.AssetsDeleted).
		Int("attachments_created", opCounts.AttachmentsCreated).
		Int("attachments_deleted", opCounts.AttachmentsDeleted).
		Msg("Completed scan for course")

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// operationCounts tracks the number of each type of operation
type operationCounts struct {
	LessonsCreated     int
	LessonsUpdated     int
	LessonsDeleted     int
	AssetsCreated      int
	AssetsUpdated      int
	AssetsDeleted      int
	AttachmentsCreated int
	AttachmentsDeleted int
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// countOperations counts the number of each type of operation
func countOperations(groupOps, assetOps, attachmentOps []Op) operationCounts {
	counts := operationCounts{}

	for _, op := range groupOps {
		switch op.(type) {
		case CreateLessonOp:
			counts.LessonsCreated++
		case UpdateLessonOp:
			counts.LessonsUpdated++
		case DeleteLessonOp:
			counts.LessonsDeleted++
		}
	}

	for _, op := range assetOps {
		switch op.(type) {
		case CreateAssetOp:
			counts.AssetsCreated++
		case UpdateAssetOp:
			counts.AssetsUpdated++
		case ReplaceAssetOp, SwapAssetOp, OverwriteAssetOp:
			counts.AssetsUpdated++
		case DeleteAssetOp:
			counts.AssetsDeleted++
		}
	}

	for _, op := range attachmentOps {
		switch op.(type) {
		case CreateAttachmentOp:
			counts.AttachmentsCreated++
		case DeleteAttachmentOp:
			counts.AttachmentsDeleted++
		}
	}

	return counts
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// handleCourseCard handles course card changes: deletion, addition, or modification
// Returns true if the card changed, and an error if context was canceled during optimization
func handleCourseCard(
	ctx context.Context,
	s *CourseScan,
	course *models.Course,
	scannedCardPath string,
	courseID, coursePath string,
	scanState *ScanState,
) (bool, error) {
	pathChanged := course.CardPath != scannedCardPath

	// Card was deleted from disk
	if scannedCardPath == "" {
		if course.CardPath == "" {
			return false, nil
		}

		if err := s.cardCache.Delete(course.ID); err != nil {
			s.logger.Warn().
				Err(err).
				Str("course_id", courseID).
				Str("course_path", coursePath).
				Msg("Failed to delete optimized card")
		}

		course.CardPath = ""
		course.CardHash = ""
		course.CardModTime = ""
		return true, nil
	}

	// Calculate card hash and mod time for the scanned source file
	cardHash, err := hashFilePartial(s.fs, scannedCardPath, 1024*1024)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Str("card_path", scannedCardPath).
			Msg("Failed to calculate card hash, continuing without optimization")
		hadMetadata := course.CardHash != "" || course.CardModTime != ""
		course.CardPath = scannedCardPath
		course.CardHash = ""
		course.CardModTime = ""
		return pathChanged || hadMetadata, nil
	}

	stat, err := s.fs.Stat(scannedCardPath)
	if err != nil {
		s.logger.Warn().
			Err(err).
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Str("card_path", scannedCardPath).
			Msg("Failed to get card mod time, continuing without optimization")
		hadMetadata := course.CardHash != "" || course.CardModTime != ""
		course.CardPath = scannedCardPath
		course.CardHash = ""
		course.CardModTime = ""
		return pathChanged || hadMetadata, nil
	}

	cardModTime := stat.ModTime().UTC().Format(time.RFC3339Nano)
	contentChanged := course.CardHash != cardHash || course.CardModTime != cardModTime

	course.CardPath = scannedCardPath
	course.CardHash = cardHash
	course.CardModTime = cardModTime

	if !pathChanged && !contentChanged {
		return false, nil
	}

	scanState.UpdateMessage("Optimizing course card")
	err = s.cardCache.OptimizeCard(ctx, course.ID, scannedCardPath, cardHash)
	if err != nil {
		if err == context.Canceled || err == context.DeadlineExceeded {
			return true, err
		}
	}

	return true, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// fetchCourse retrieves the course from the database
func fetchCourse(ctx context.Context, s *CourseScan, courseID string) (*models.Course, error) {
	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: courseID})
	course, err := s.dao.GetCourse(ctx, dbOpts)
	if err != nil {
		return nil, err
	}

	if course == nil {
		s.logger.Debug().
			Str("course_id", courseID).
			Msg("Ignoring scan job as the course no longer exists")
		return nil, nil
	}

	return course, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// checkAndSetCourseAvailability checks if the course is available and updates its status
// accordingly
func checkAndSetCourseAvailability(ctx context.Context, s *CourseScan, course *models.Course) (bool, error) {
	_, err := s.fs.Stat(course.Path)
	if os.IsNotExist(err) {
		s.logger.Debug().
			Str("course_id", course.ID).
			Str("course_path", course.Path).
			Msg("Skipping unavailable course")

		if course.Available {
			course.Available = false
			return false, s.dao.UpdateCourse(ctx, course)
		}

		return false, nil
	}

	if err != nil {
		return false, err
	}

	if !course.Available {
		course.Available = true
		return true, s.dao.UpdateCourse(ctx, course)
	}

	return true, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// enableCourseMaintenance sets the course to maintenance mode if it is not already
func enableCourseMaintenance(ctx context.Context, s *CourseScan, course *models.Course) error {
	if course.Maintenance {
		return nil
	}

	course.Maintenance = true
	if err := s.dao.UpdateCourse(ctx, course); err != nil {
		return err
	}

	s.logger.Debug().
		Str("course_id", course.ID).
		Str("course_path", course.Path).
		Msg("Set course to maintenance mode")
	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// clearCourseMaintenanceWithRetry attempts to clear maintenance mode with retry logic
func clearCourseMaintenanceWithRetry(ctx context.Context, s *CourseScan, courseID, coursePath string) error {
	const maxRetries = 3
	const initialDelay = 100 * time.Millisecond

	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TABLE_ID: courseID})
	course, err := s.dao.GetCourse(ctx, dbOpts)
	if err != nil {
		return fmt.Errorf("failed to fetch course for maintenance cleanup: %w", err)
	}

	if course == nil {
		return nil
	}

	if !course.Maintenance {
		return nil
	}

	var lastErr error
	delay := initialDelay
	for attempt := 0; attempt < maxRetries; attempt++ {
		course, err := s.dao.GetCourse(ctx, dbOpts)
		if err != nil {
			lastErr = fmt.Errorf("failed to fetch course (attempt %d/%d): %w", attempt+1, maxRetries, err)
			if attempt < maxRetries-1 {
				time.Sleep(delay)
				delay *= 2
			}
			continue
		}

		if course == nil || !course.Maintenance {
			return nil
		}

		course.Maintenance = false
		if err := s.dao.UpdateCourse(ctx, course); err != nil {
			lastErr = fmt.Errorf("failed to update course (attempt %d/%d): %w", attempt+1, maxRetries, err)
			if attempt < maxRetries-1 {
				time.Sleep(delay)
				delay *= 2
			}
			continue
		}

		s.logger.Debug().
			Str("course_id", courseID).
			Str("course_path", coursePath).
			Int("attempt", attempt+1).
			Msg("Cleared course from maintenance mode")
		return nil
	}

	return fmt.Errorf("failed to clear maintenance mode after %d attempts: %w", maxRetries, lastErr)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// lessonBucket accumulates files for a given module & prefix
type lessonBucket struct {
	groupedFiles []*parsedFile
	soloFiles    []*parsedFile
	attachFiles  []*parsedFile
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// scannedResults holds all lessons and the card image path
type scannedResults struct {
	lessons  []*models.Lesson
	cardPath string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// scanFiles scans the course directory for files. It will return a list of grouped assets,
// and a card path, if found.
func scanFiles(s *CourseScan, course *models.Course) (*scannedResults, error) {
	s.logger.Info().
		Str("course_id", course.ID).
		Str("course_path", course.Path).
		Msg("Scanning course directory")

	files, err := s.fs.ReadDirFlat(course.Path, 2)
	if err != nil {
		return nil, err
	}

	// A bucket is module => prefix => fileBucket
	buckets := map[string]map[int]*lessonBucket{}
	cardPath := ""

	// Scan files on disk and categorize them into buckets
	for _, fp := range files {
		normalizedPath := utils.NormalizeWindowsDrive(fp)
		filename := filepath.Base(normalizedPath)
		dir := filepath.Dir(normalizedPath)
		inRoot := dir == utils.NormalizeWindowsDrive(course.Path)

		module := ""
		if !inRoot {
			module = filepath.Base(dir)
		}

		parsed := parseFilename(normalizedPath, filename)
		category := categorizeFile(parsed)

		if category == Ignore {
			continue
		}

		if buckets[module] == nil {
			buckets[module] = map[int]*lessonBucket{}
		}

		prefix := parsed.Prefix
		if buckets[module][prefix] == nil {
			buckets[module][prefix] = &lessonBucket{}
		}

		bucket := buckets[module][prefix]

		switch category {
		case Card:
			if inRoot && cardPath == "" {
				cardPath = normalizedPath
			}

		case GroupedAsset:
			bucket.groupedFiles = append(bucket.groupedFiles, parsed)

		case Asset:
			bucket.soloFiles = append(bucket.soloFiles, parsed)

		case Attachment:
			bucket.attachFiles = append(bucket.attachFiles, parsed)
		}
	}

	var lessons []*models.Lesson
	for module, prefixMap := range buckets {
		for prefix, bucket := range prefixMap {
			sort.Slice(bucket.groupedFiles, func(i, j int) bool {
				return *bucket.groupedFiles[i].SubPrefix < *bucket.groupedFiles[j].SubPrefix
			})

			lesson := &models.Lesson{
				CourseID: course.ID,
				Module:   module,
				Prefix:   sql.NullInt16{Int16: int16(prefix), Valid: true},
			}

			// Set the lesson title from the first grouped file (if any)
			if len(bucket.groupedFiles) > 0 {
				lesson.Title = bucket.groupedFiles[0].Title
			}

			if len(bucket.groupedFiles) > 0 {
				for _, parsedFile := range bucket.groupedFiles {
					asset, err := parsedFile.toAsset(s.fs, module, course.ID)
					if err != nil {
						return nil, err
					}

					asset.LessonID = lesson.ID
					lesson.Assets = append(lesson.Assets, asset)
				}

				for _, parsedFile := range bucket.soloFiles {
					lesson.Attachments = append(lesson.Attachments, parsedFile.toAttachment())
				}
			} else if len(bucket.soloFiles) > 0 {
				if len(bucket.soloFiles) > 1 {
					idx := pickBest(bucket.soloFiles)
					pf := bucket.soloFiles[idx]

					asset, err := pf.toAsset(s.fs, module, course.ID)
					if err != nil {
						return nil, err
					}
					asset.LessonID = lesson.ID
					lesson.Assets = append(lesson.Assets, asset)

					lesson.Title = pf.Title

					for i, other := range bucket.soloFiles {
						if i == idx {
							continue
						}
						lesson.Attachments = append(lesson.Attachments, other.toAttachment())
					}
				} else {
					asset, err := bucket.soloFiles[0].toAsset(s.fs, module, course.ID)
					if err != nil {
						return nil, err
					}
					asset.LessonID = lesson.ID
					lesson.Assets = append(lesson.Assets, asset)

					lesson.Title = bucket.soloFiles[0].Title
				}

				for _, parsedFile := range bucket.attachFiles {
					lesson.Attachments = append(lesson.Attachments, parsedFile.toAttachment())
				}

			}

			if len(lesson.Assets) > 0 {
				lessons = append(lessons, lesson)
			}
		}
	}

	return &scannedResults{
		lessons:  lessons,
		cardPath: cardPath,
	}, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// flatAttachmentsAndAssets returns a flat list of attachments and assets from a list of lessons
func flatAttachmentsAndAssets(lessons []*models.Lesson) ([]*models.Attachment, []*models.Asset) {
	var attachments []*models.Attachment
	var assets []*models.Asset
	for _, g := range lessons {
		assets = append(assets, g.Assets...)
		attachments = append(attachments, g.Attachments...)
	}

	return attachments, assets
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// probeVideos probes videos assets that match the operations create, replace, swap, or overwrite
func probeVideos(ctx context.Context, s *CourseScan, ops []Op, course *models.Course, extractedKeyframes map[string][]float64, scanState *ScanState) map[string]*models.AssetMetadata {
	var targets []*models.Asset
	for _, op := range ops {
		switch v := op.(type) {
		case CreateAssetOp:
			if v.New.Type.IsVideo() {
				targets = append(targets, v.New)
			}
		case ReplaceAssetOp:
			if v.New.Type.IsVideo() {
				targets = append(targets, v.New)
			}
		case SwapAssetOp:
			if v.NewA.Type.IsVideo() {
				targets = append(targets, v.NewA)
			}
			if v.NewB.Type.IsVideo() {
				targets = append(targets, v.NewB)
			}
		case OverwriteAssetOp:
			if v.Renamed.Type.IsVideo() {
				targets = append(targets, v.Renamed)
			}
		}
	}

	if len(targets) == 0 {
		return map[string]*models.AssetMetadata{}
	}

	s.logger.Info().
		Str("course_id", course.ID).
		Str("course_path", course.Path).
		Int("video_assets_count", len(targets)).
		Msg("Starting video probing and keyframe extraction")

	scanState.UpdateMessage("Extracting video keyframes")
	mediaProbe := probe.MediaProbe{Tools: s.tools}
	assetMetadataByPath := make(map[string]*models.AssetMetadata)
	totalVideos := len(targets)

	cancelled := false
	for i, asset := range targets {
		if ctx.Err() != nil {
			s.logger.Info().
				Str("course_id", course.ID).
				Str("course_path", course.Path).
				Msg("Video probing cancelled")
			cancelled = true
			break
		}

		scanState.UpdateMessage(fmt.Sprintf("Extracting video keyframes (%d/%d)", i+1, totalVideos))

		info, _, err := mediaProbe.ProbeVideo(ctx, asset.Path)
		if err != nil {
			if ctx.Err() != nil || isCancellationError(err) {
				cancelled = true
				break
			}
			s.logger.Warn().
				Err(err).
				Str("course_id", course.ID).
				Str("course_path", course.Path).
				Str("video_path", asset.Path).
				Msg("Failed to probe video file")
			continue
		}

		var keyframes []float64
		if kf, err := mediaProbe.ExtractKeyframesForVideo(ctx, asset.Path); err == nil {
			keyframes = kf
			s.logger.Debug().
				Str("course_id", course.ID).
				Str("course_path", course.Path).
				Int("keyframes_count", len(keyframes)).
				Str("video_path", asset.Path).
				Msg("Extracted keyframes for video")
		} else {
			if ctx.Err() != nil || isCancellationError(err) {
				cancelled = true
				break
			}
			s.logger.Warn().
				Err(err).
				Str("course_id", course.ID).
				Str("course_path", course.Path).
				Str("video_path", asset.Path).
				Msg("Failed to extract keyframes for video")
		}

		assetMetadataByPath[asset.Path] = &models.AssetMetadata{
			VideoMetadata: &models.VideoMetadata{
				DurationSec: info.DurationSec,
				Container:   info.File.Container,
				MIMEType:    info.File.MIMEType,
				SizeBytes:   info.File.SizeBytes,
				OverallBPS:  info.File.OverallBPS,
				VideoCodec:  info.Video.Codec,
				Width:       info.Video.Width,
				Height:      info.Video.Height,
				FPSNum:      info.Video.FPSNum,
				FPSDen:      info.Video.FPSDen,
			},
			AudioMetadata: &models.AudioMetadata{
				Language:      info.Audio.Language,
				Codec:         info.Audio.Codec,
				Profile:       info.Audio.Profile,
				Channels:      info.Audio.Channels,
				ChannelLayout: info.Audio.ChannelLayout,
				SampleRate:    info.Audio.SampleRate,
				BitRate:       info.Audio.BitRate,
			},
		}
		if len(keyframes) > 0 {
			extractedKeyframes[asset.Path] = keyframes
		}
	}

	if !cancelled {
		s.logger.Info().
			Str("course_id", course.ID).
			Str("course_path", course.Path).
			Int("videos_processed", len(assetMetadataByPath)).
			Int("total_videos", totalVideos).
			Msg("Completed video probing and keyframe extraction")
	}

	return assetMetadataByPath
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// isCancellationError checks if an error is related to cancellation
func isCancellationError(err error) bool {
	if err == nil {
		return false
	}

	if err == context.Canceled || err == context.DeadlineExceeded {
		return true
	}

	errStr := err.Error()
	return strings.Contains(errStr, "context canceled") ||
		strings.Contains(errStr, "signal: killed") ||
		strings.Contains(errStr, "context deadline exceeded")
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// applyLessonChanges applies lesson create/update operations. Delete is done after all
// asset and attachment operations have been applied
func applyLessonCreateUpdateOps(
	ctx context.Context,
	s *CourseScan,
	ops []Op,
) (bool, error) {
	if len(ops) == 0 {
		return false, nil
	}

	for _, op := range ops {
		switch v := op.(type) {
		case CreateLessonOp:
			if err := s.dao.CreateLesson(ctx, v.New); err != nil {
				return false, err
			}

			for _, asset := range v.New.Assets {
				asset.LessonID = v.New.ID
			}

			for _, att := range v.New.Attachments {
				att.CourseID = v.New.CourseID
				att.LessonID = v.New.ID
			}

		case NoLessonOp:
			// Ensure all new assets and attachments have the lesson ID. When a new
			// asset/attachment is added to an existing lesson, we need to ensure
			// that the lesson ID is set
			for _, asset := range v.New.Assets {
				asset.LessonID = v.Existing.ID
			}

			for _, att := range v.New.Attachments {
				att.CourseID = v.Existing.CourseID
				att.LessonID = v.Existing.ID
			}

		case UpdateLessonOp:
			// Update an existing lesson by giving the new lesson the ID of the existing
			// lesson, then calling update
			v.New.ID = v.Existing.ID

			if err := s.dao.UpdateLesson(ctx, v.New); err != nil {
				return false, err
			}

			for _, asset := range v.New.Assets {
				asset.LessonID = v.New.ID
			}

			for _, att := range v.New.Attachments {
				att.CourseID = v.New.CourseID
				att.LessonID = v.New.ID
			}
		}
	}

	return true, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// applyLessonDeleteOps applies lesson deletion operations
func applyLessonDeleteOps(
	ctx context.Context,
	s *CourseScan,
	ops []Op,
	course *models.Course,
) (bool, error) {
	if len(ops) == 0 {
		return false, nil
	}

	for _, op := range ops {
		switch v := op.(type) {

		case DeleteLessonOp:
			// Subtract durations of all video assets in the deleted lesson
			for _, asset := range v.Deleted.Assets {
				if asset.AssetMetadata != nil && asset.AssetMetadata.VideoMetadata != nil {
					course.Duration -= asset.AssetMetadata.VideoMetadata.DurationSec
				}
			}

			// Delete an existing lesson
			dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.LESSON_TABLE_ID: v.Deleted.ID})
			if err := s.dao.DeleteLessons(ctx, dbOpts); err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// applyAssetOps applies the changes to the assets in the database by creating, renaming,
// replacing, swapping, or deleting them as needed
func applyAssetOps(
	ctx context.Context,
	s *CourseScan,
	course *models.Course,
	ops []Op,
	assetMetadataByPath map[string]*models.AssetMetadata,
	extractedKeyframes map[string][]float64,
) (bool, error) {
	if len(ops) == 0 {
		return false, nil
	}

	for _, op := range ops {
		switch v := op.(type) {
		case CreateAssetOp:
			// Create an asset that was found on disk and does not exist in the database
			if err := s.dao.CreateAsset(ctx, v.New); err != nil {
				return false, err
			}

			if metadata := assetMetadataByPath[v.New.Path]; metadata != nil && v.New.Type.IsVideo() {
				metadata.AssetID = v.New.ID

				if err := s.dao.CreateAssetMetadata(ctx, metadata); err != nil {
					return false, err
				}

				if keyframes, exists := extractedKeyframes[v.New.Path]; exists {
					if len(keyframes) > 0 {
						assetKeyframes := &models.AssetKeyframes{
							AssetID:    v.New.ID,
							Keyframes:  types.Keyframes(keyframes),
							IsComplete: true,
						}

						if err := s.dao.CreateAssetKeyframes(ctx, assetKeyframes); err != nil {
							s.logger.Error().
								Err(err).
								Str("course_id", course.ID).
								Str("course_path", course.Path).
								Str("asset_id", v.New.ID).
								Str("asset_path", v.New.Path).
								Msg("Failed to store keyframes for asset")
						}
					}

					delete(extractedKeyframes, v.New.Path)
				}

				course.Duration += metadata.VideoMetadata.DurationSec
			}

		case UpdateAssetOp:
			// Update an existing asset by giving the new asset the ID of the existing asset and lesson,
			// then calling update
			//
			// This happens when the metadata (title, path, prefix, etc) changes but the
			// contents of the asset have not
			//
			// Asset progress will be preserved
			v.New.ID = v.Existing.ID

			// When the update is because of a change to `description path`, the actual asset will not
			// have changed and therefore a hash will not have been generated. In this case, just take
			// the existing hash
			if v.New.Hash == "" {
				v.New.Hash = v.Existing.Hash
			}

			if err := s.dao.UpdateAsset(ctx, v.New); err != nil {
				return false, err
			}

		case ReplaceAssetOp:
			// Replace an existing asset with a new asset by first deleting the existing asset, then
			// creating the new asset. Take the existing lesson ID
			//
			// This happens when the contents of an existing asset have changed but the metadata
			// (title, path, prefix, etc) has not
			//
			// Asset progress will be lost
			dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: v.Existing.ID})
			if err := s.dao.DeleteAssets(ctx, dbOpts); err != nil {
				return false, err
			}

			if v.Existing.AssetMetadata != nil && v.Existing.AssetMetadata.VideoMetadata != nil {
				course.Duration -= v.Existing.AssetMetadata.VideoMetadata.DurationSec
			}

			v.New.LessonID = v.Existing.LessonID
			if err := s.dao.CreateAsset(ctx, v.New); err != nil {
				return false, err
			}

			if metadata := assetMetadataByPath[v.New.Path]; metadata != nil && v.New.Type.IsVideo() {
				metadata.AssetID = v.New.ID

				if err := s.dao.CreateAssetMetadata(ctx, metadata); err != nil {
					return false, err
				}

				course.Duration += metadata.VideoMetadata.DurationSec
			}

		case OverwriteAssetOp:
			// Overwrite an existing but now deleted asset with another still existing asset by first
			// taking the ID of the existing asset, the lesson ID of the nonexisting asset, then
			// deleting the nonexisting asset, and finally calling update on the renamed asset to update
			// its metadata (title, path, prefix, etc)
			//
			// This happens when an asset has been renamed to that of another, now deleted, asset
			//
			// Asset progress will be preserved
			dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: v.Deleted.ID})
			if err := s.dao.DeleteAssets(ctx, dbOpts); err != nil {
				return false, err
			}

			// Subtract the deleted asset's duration if it was a video
			if v.Deleted.AssetMetadata != nil && v.Deleted.AssetMetadata.VideoMetadata != nil {
				course.Duration -= v.Deleted.AssetMetadata.VideoMetadata.DurationSec
			}

			// Subtract the existing asset's duration if it was a video (we're replacing it)
			if v.Existing.AssetMetadata != nil && v.Existing.AssetMetadata.VideoMetadata != nil {
				course.Duration -= v.Existing.AssetMetadata.VideoMetadata.DurationSec
			}

			v.Renamed.ID = v.Existing.ID
			v.Renamed.LessonID = v.Deleted.LessonID

			if err := s.dao.UpdateAsset(ctx, v.Renamed); err != nil {
				return false, err
			}

			if metadata := assetMetadataByPath[v.Renamed.Path]; metadata != nil && v.Renamed.Type.IsVideo() {
				metadata.AssetID = v.Renamed.ID

				if err := s.dao.CreateAssetMetadata(ctx, metadata); err != nil {
					return false, err
				}

				if keyframes, exists := extractedKeyframes[v.Renamed.Path]; exists {
					if len(keyframes) > 0 {
						assetKeyframes := &models.AssetKeyframes{
							AssetID:    v.Renamed.ID,
							Keyframes:  types.Keyframes(keyframes),
							IsComplete: true,
						}

						if err := s.dao.CreateAssetKeyframes(ctx, assetKeyframes); err != nil {
							s.logger.Error().
								Err(err).
								Str("course_id", course.ID).
								Str("course_path", course.Path).
								Str("asset_id", v.Renamed.ID).
								Str("asset_path", v.Renamed.Path).
								Msg("Failed to store keyframes for overwritten asset")
						}
					}
					delete(extractedKeyframes, v.Renamed.Path)
				}

				course.Duration += metadata.VideoMetadata.DurationSec
			}

		case SwapAssetOp:
			// Swap two assets by first deleting the existing assets, then recreating the assets.
			// The lesson IDs of the new assets will be swapped
			//
			// This happens when two existing assets swap paths
			//
			// Asset progress will be lost
			for _, existing := range []*models.Asset{v.ExistingA, v.ExistingB} {
				dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: existing.ID})
				if err := s.dao.DeleteAssets(ctx, dbOpts); err != nil {
					return false, err
				}

				if existing.AssetMetadata != nil && existing.AssetMetadata.VideoMetadata != nil {
					course.Duration -= existing.AssetMetadata.VideoMetadata.DurationSec
				}
			}

			// Swap the new lesson IDs
			v.NewA.LessonID = v.ExistingB.LessonID
			v.NewB.LessonID = v.ExistingA.LessonID

			for _, newAsset := range []*models.Asset{v.NewA, v.NewB} {
				if err := s.dao.CreateAsset(ctx, newAsset); err != nil {
					return false, err
				}

				if metadata := assetMetadataByPath[newAsset.Path]; metadata != nil && newAsset.Type.IsVideo() {
					metadata.AssetID = newAsset.ID

					if err := s.dao.CreateAssetMetadata(ctx, metadata); err != nil {
						return false, err
					}

					course.Duration += metadata.VideoMetadata.DurationSec
				}
			}

		case DeleteAssetOp:
			// Delete an asset that no longer exists on disk
			dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.ASSET_TABLE_ID: v.Deleted.ID})
			if err := s.dao.DeleteAssets(ctx, dbOpts); err != nil {
				return false, err
			}

			if v.Deleted.AssetMetadata != nil && v.Deleted.AssetMetadata.VideoMetadata != nil {
				course.Duration -= v.Deleted.AssetMetadata.VideoMetadata.DurationSec
			}
		}
	}

	return true, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// recalculateCourseDuration recalculates the course duration by summing all video asset durations
func recalculateCourseDuration(ctx context.Context, s *CourseScan, courseID string) (int, error) {
	dbOpts := dao.NewOptions().
		WithWhere(squirrel.Eq{models.ASSET_COURSE_ID: courseID}).
		WithAssetMetadata()

	assets, err := s.dao.ListAssets(ctx, dbOpts)
	if err != nil {
		return 0, fmt.Errorf("failed to list assets for duration calculation: %w", err)
	}

	totalDuration := 0
	for _, asset := range assets {
		if asset.AssetMetadata != nil && asset.AssetMetadata.VideoMetadata != nil {
			totalDuration += asset.AssetMetadata.VideoMetadata.DurationSec
		}
	}

	return totalDuration, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// applyAttachmentOps applies the changes to the attachments in the database by creating
// or deleting them as needed
func applyAttachmentOps(
	ctx context.Context,
	s *CourseScan,
	ops []Op,

) (bool, error) {
	if len(ops) == 0 {
		return false, nil
	}

	for _, op := range ops {
		switch v := op.(type) {
		case CreateAttachmentOp:
			if err := s.dao.CreateAttachment(ctx, v.New); err != nil {
				return false, err
			}
		case DeleteAttachmentOp:
			dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.ATTACHMENT_TABLE_ID: v.Deleted.ID})
			if err := s.dao.DeleteAttachments(ctx, dbOpts); err != nil {
				return false, err
			}
		}
	}

	return true, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// A priority list for assets when picking a single asset
var assetPriority = []types.AssetType{
	types.AssetVideo,
	types.AssetPDF,
	types.AssetMarkdown,
	types.AssetText,
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// pickBest returns the index in candidates of the highest‐priority asset
func pickBest(assets []*parsedFile) int {
	bestIdx := 0
	bestRank := len(assetPriority)
	for i, asset := range assets {
		for rank, at := range assetPriority {
			if asset.AssetType.IsValid() && asset.AssetType == at {
				if rank < bestRank {
					bestRank = rank
					bestIdx = i
				}

				break
			}
		}
	}

	return bestIdx
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// FileCategory enumerates what parse+classify says we’ve got.
type FileCategory int

const (
	Ignore FileCategory = iota
	Card
	Asset
	GroupedAsset
	Attachment
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// categorizeFile inspects a parsedFile and tells you which it is
func categorizeFile(p *parsedFile) FileCategory {
	// Ignore
	if p == nil {
		return Ignore
	}

	// Card
	if p.IsCard {
		return Card
	}

	// Asset || grouped asset
	if p.AssetType.IsValid() && p.Title != "" {
		if p.SubPrefix != nil {
			return GroupedAsset
		}

		return Asset
	}

	// Attachment
	return Attachment
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// populateHashesIfChanged populates the hashes of the scanned assets if they have changed
// Uses parallel processing with a worker pool to improve performance
func populateHashesIfChanged(ctx context.Context, s *CourseScan, scanned []*models.Asset, existing []*models.Asset, course *models.Course, scanState *ScanState) error {
	s.logger.Info().
		Str("course_id", course.ID).
		Str("course_path", course.Path).
		Int("assets_to_hash", len(scanned)).
		Msg("Calculating asset hashes")

	existingMap := make(map[string]*models.Asset)
	for _, e := range existing {
		existingMap[e.Path] = e
	}

	// Identify assets that need hashing
	var toHash []*models.Asset
	for _, s := range scanned {
		e := existingMap[s.Path]
		if e == nil || e.FileSize != s.FileSize || e.ModTime != s.ModTime {
			toHash = append(toHash, s)
		}
	}

	if len(toHash) == 0 {
		return nil
	}

	// Use parallel hashing with worker pool
	const numWorkers = 4 // Limit concurrency to avoid overwhelming I/O
	return populateHashesParallel(ctx, s, toHash, course, numWorkers, scanState)
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// populateHashesParallel hashes assets in parallel using a worker pool
func populateHashesParallel(ctx context.Context, s *CourseScan, assets []*models.Asset, course *models.Course, numWorkers int, scanState *ScanState) error {
	if len(assets) == 0 {
		return nil
	}

	// Create work channel
	workChan := make(chan *models.Asset, len(assets))
	for _, asset := range assets {
		workChan <- asset
	}
	close(workChan)

	// Track progress and errors
	var (
		wg          sync.WaitGroup
		mu          sync.Mutex
		hashedCount int
		firstError  error
		totalToHash = len(assets)
	)

	// Start workers
	for i := 0; i < numWorkers && i < totalToHash; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for asset := range workChan {
				// Check for context cancellation
				if ctx.Err() != nil {
					mu.Lock()
					if firstError == nil {
						firstError = ctx.Err()
					}
					mu.Unlock()
					return
				}

				hash, err := hashFilePartial(s.fs, asset.Path, 1024*1024)
				if err != nil {
					mu.Lock()
					if firstError == nil {
						firstError = err
					}
					mu.Unlock()
					s.logger.Error().
						Err(err).
						Str("course_id", course.ID).
						Str("course_path", course.Path).
						Str("asset_path", asset.Path).
						Msg("Failed to hash asset")
					continue
				}

				mu.Lock()
				asset.Hash = hash
				hashedCount++
				current := hashedCount
				mu.Unlock()

				// Update status directly (thread-safe via mutex in ScanState)
				scanState.UpdateMessage(fmt.Sprintf("Calculating asset hashes (%d/%d)", current, totalToHash))
			}
		}()
	}

	// Wait for all workers to complete
	wg.Wait()

	// Return first error encountered, if any
	if firstError != nil {
		// Check if this is a cancellation (not a real error)
		if firstError == context.Canceled || firstError == context.DeadlineExceeded {
			return firstError
		}
		return fmt.Errorf("failed to hash one or more assets: %w", firstError)
	}

	s.logger.Info().
		Str("course_id", course.ID).
		Str("course_path", course.Path).
		Int("hashed_count", hashedCount).
		Int("total_count", totalToHash).
		Msg("Completed calculating asset hashes")

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// hashFilePartial computes the SHA-256 hash of a file by reading it in chunks
func hashFilePartial(fs afero.Fs, path string, chunkSize int64) (string, error) {
	file, err := fs.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	size := info.Size()

	hasher := sha256.New()
	readChunk := func(offset int64) error {
		buf := make([]byte, chunkSize)
		_, err := file.Seek(offset, io.SeekStart)
		if err != nil {
			return err
		}
		n, err := io.ReadFull(file, buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			return err
		}
		hasher.Write(buf[:n])
		return nil
	}

	// Head
	if err := readChunk(0); err != nil {
		return "", err
	}

	// Middle
	if size > chunkSize*2 {
		middle := size / 2
		if err := readChunk(middle); err != nil {
			return "", err
		}
	}

	// Tail
	if size > chunkSize {
		tail := size - chunkSize
		if err := readChunk(tail); err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// getInt safely dereferences *int
func getInt(ptr *int) int {
	if ptr != nil {
		return *ptr
	}
	return 0
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parsedFile represents a parsed file name with its components
type parsedFile struct {
	// 1, 2, 03, etc
	Prefix int

	// Text before `{`
	Title string

	// If {N ...}
	SubPrefix *int

	// If {... subtitle}
	SubTitle string

	// Lowercase extension (without dot)
	Ext string

	// Non-empty when type is one of video, pdf, markdown, text
	AssetType types.AssetType

	// True when the file is a card
	IsCard bool

	// The original filename
	Original string

	// Path to the file
	NormalizedPath string
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// A regex for parsing a file name into a prefix, title, sub-prefix (optional), sub-title (optional) and extension
var filenameRegex = regexp.MustCompile(
	`^` +
		// Prefix
		//   - Ex: 1, 01, 123
		`(?P<Prefix>\d+)` +

		// Optional spacer followed by an optional title
		//   - Spacer: zero or more spaces, zero or more hyphens, zero or more spaces
		//   - Title: any chars up to “{” (non‐greedy)
		//
		//   - Ex:
		// 		`- title`,
		// 		`  -  title`,
		// 		`title`,
		// 		`--- title`
		`(?:(?:\s+|\s*-+\s*)(?P<Title>[^{]*?))?` +

		// Optional sub‐group in braces { … }
		//   - SubPrefix: any number of digits (non‐greedy)
		//   - Spacer: zero or more spaces, zero or more hyphens, zero or more spaces
		//   - SubTitle: any chars up to `}` (non‐greedy)
		//
		//  - Ex:
		// 		`{2}`,
		// 		`{2 - subtitle}`,
		// 		`{ 2 - subtitle}`,
		// 		`{ 2 }`,
		// 		`{2 - subtitle}`,
		`(?:\s*\{(?:(?P<SubPrefix>\d+)(?:\s*(?:-+\s*)?(?P<SubTitle>[^}]*))?)?\})?` +

		// Optional extension
		//   - Ex: `.jpg`, `.png`, `.pdf`
		`(?:\.(?P<Ext>\w+))?` +

		// End of string
		`$`,
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// parseFilename parses a filename into its constituent parts
func parseFilename(normalizedPath, filename string) *parsedFile {
	// Quick check for card
	if utils.IsCard(filename) {
		return &parsedFile{
			Title:          "card",
			Ext:            strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), ".")),
			AssetType:      types.AssetType(""), // Empty for cards
			IsCard:         true,
			Original:       filename,
			NormalizedPath: normalizedPath,
		}
	}

	matches := filenameRegex.FindStringSubmatch(filename)
	if matches == nil {
		return nil
	}

	// Prefix
	prefix, err := strconv.Atoi(matches[filenameRegex.SubexpIndex("Prefix")])
	if err != nil {
		return nil
	}

	// Title without leading hyphens and spaces
	title := strings.TrimLeft(strings.TrimSpace(matches[filenameRegex.SubexpIndex("Title")]), " -")

	// Ext
	ext := strings.ToLower(matches[filenameRegex.SubexpIndex("Ext")])

	// Asset type
	var assetType types.AssetType
	if ext != "" {
		if at, err := types.NewAsset(ext); err == nil {
			assetType = at
		}
	}

	var subPrefix *int
	if sp := matches[filenameRegex.SubexpIndex("SubPrefix")]; sp != "" {
		if v, err := strconv.Atoi(sp); err == nil {
			subPrefix = &v
		}
	}

	subTitle := strings.Trim(strings.TrimSpace(matches[filenameRegex.SubexpIndex("SubTitle")]), " -")

	return &parsedFile{
		Prefix:         prefix,
		Title:          title,
		SubPrefix:      subPrefix,
		SubTitle:       subTitle,
		Ext:            ext,
		AssetType:      assetType,
		IsCard:         false,
		Original:       filename,
		NormalizedPath: normalizedPath,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// toAsset converts a parsedFile to a models.Asset
func (p *parsedFile) toAsset(fs afero.Fs, module, courseID string) (*models.Asset, error) {
	stat, err := fs.Stat(p.NormalizedPath)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", p.NormalizedPath, err)
	}
	mtime := stat.ModTime().UTC().Format(time.RFC3339Nano)

	asset := &models.Asset{
		CourseID:  courseID,
		Module:    module,
		Title:     p.Title,
		Prefix:    sql.NullInt16{Int16: int16(p.Prefix), Valid: true},
		SubPrefix: sql.NullInt16{Int16: int16(getInt(p.SubPrefix)), Valid: p.SubPrefix != nil},
		SubTitle:  p.SubTitle,
		Type:      p.AssetType,
		Path:      p.NormalizedPath,
		FileSize:  stat.Size(),
		ModTime:   mtime,
		Weight:    1,
	}

	return asset, nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// toAttachment converts a parsedFile to a models.Attachment
func (p *parsedFile) toAttachment() *models.Attachment {
	return &models.Attachment{
		Title: p.Title + filepath.Ext(p.Original),
		Path:  p.NormalizedPath,
	}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// applyTagsFromMetadata applies tags from oc.json to the database.
// Creates tags that don't exist and associates them with the course.
// Removes tags from the course that are not in the metadata.
func applyTagsFromMetadata(ctx context.Context, s *CourseScan, courseID string, tags []string) error {
	// Get existing course tags
	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courseID})
	existingCourseTags, err := s.dao.ListCourseTags(ctx, dbOpts)
	if err != nil {
		return fmt.Errorf("failed to list existing course tags: %w", err)
	}

	// Create a map of existing tag names (lowercase for case-insensitive comparison)
	existingTagMap := make(map[string]*models.CourseTag)
	for _, ct := range existingCourseTags {
		existingTagMap[strings.ToLower(ct.Tag)] = ct
	}

	// Create a map of desired tag names (lowercase)
	desiredTagMap := make(map[string]string)
	for _, tag := range tags {
		if tag != "" {
			desiredTagMap[strings.ToLower(tag)] = tag
		}
	}

	// Remove tags that are not in the desired list
	for lowerTag, courseTag := range existingTagMap {
		if _, exists := desiredTagMap[lowerTag]; !exists {
			deleteOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_ID: courseTag.ID})
			if err := s.dao.DeleteCourseTags(ctx, deleteOpts); err != nil {
				return fmt.Errorf("failed to delete course tag: %w", err)
			}
		}
	}

	// Add tags that are in the desired list but not in existing list
	for lowerTag, tagName := range desiredTagMap {
		if _, exists := existingTagMap[lowerTag]; !exists {
			courseTag := &models.CourseTag{
				CourseID: courseID,
				Tag:      tagName,
			}
			if err := s.dao.CreateCourseTag(ctx, courseTag); err != nil {
				// Ignore duplicate tag errors (could happen in race conditions)
				if !strings.Contains(err.Error(), "UNIQUE constraint failed") {
					return fmt.Errorf("failed to create course tag: %w", err)
				}
			}
		}
	}

	return nil
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// removeAllCourseTags removes all tags associated with a course
func removeAllCourseTags(ctx context.Context, s *CourseScan, courseID string) error {
	dbOpts := dao.NewOptions().WithWhere(squirrel.Eq{models.COURSE_TAG_TABLE_COURSE_ID: courseID})
	return s.dao.DeleteCourseTags(ctx, dbOpts)
}
