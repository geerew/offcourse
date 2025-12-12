package api

import (
	"sort"
	"strings"

	"github.com/geerew/off-course/models"
	"github.com/geerew/off-course/utils/coursescan"
	"github.com/geerew/off-course/utils/types"
)

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// File System
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type fileSystemResponse struct {
	Count       int                 `json:"count"`
	Directories []*fileInfoResponse `json:"directories"`
	Files       []*fileInfoResponse `json:"files"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type fileInfoResponse struct {
	Title string `json:"title"`
	Path  string `json:"path"`

	// Only relevant when a directory
	Classification types.PathClassification `json:"classification"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Course
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type courseRequest struct {
	Title string `json:"title"`
	Path  string `json:"path"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type courseProgressResponse struct {
	Started     bool           `json:"started"`
	StartedAt   types.DateTime `json:"startedAt"`
	Percent     int            `json:"percent"`
	CompletedAt types.DateTime `json:"completedAt"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type courseResponse struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Path        string         `json:"path,omitempty"`
	HasCard     bool           `json:"hasCard"`
	CardHash    string         `json:"cardHash,omitempty"`
	Available   bool           `json:"available"`
	Duration    int            `json:"duration"`
	InitialScan *bool          `json:"initialScan,omitempty"`
	Maintenance bool           `json:"maintenance"`
	Description string         `json:"description"`
	CreatedAt   types.DateTime `json:"createdAt"`
	UpdatedAt   types.DateTime `json:"updatedAt"`

	// Scan status
	ScanStatus string `json:"scanStatus,omitempty"`

	// Progress
	Progress *courseProgressResponse `json:"progress,omitempty"`

	// Favourite
	Favourited bool `json:"favourited,omitempty"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func courseResponseHelper(courses []*models.Course, isAdmin bool) []*courseResponse {
	responses := []*courseResponse{}

	for _, course := range courses {
		// Course progress
		var progress *courseProgressResponse
		if course.Progress != nil {
			progress = &courseProgressResponse{
				Started:     course.Progress.Started,
				StartedAt:   course.Progress.StartedAt,
				Percent:     course.Progress.Percent,
				CompletedAt: course.Progress.CompletedAt,
			}
		}

		// Get first 12 chars. The frontend can use this to cache bust the card image
		cardHash := ""
		if course.CardPath == "" {
			cardHash = "fallback"
		} else if course.CardHash != "" && len(course.CardHash) >= 12 {
			cardHash = course.CardHash[:12]
		} else {
			cardHash = "pending"
		}

		response := &courseResponse{
			ID:          course.ID,
			Title:       course.Title,
			HasCard:     course.CardPath != "",
			CardHash:    cardHash,
			Available:   course.Available,
			Duration:    course.Duration,
			Maintenance: course.Maintenance,
			Description: course.Description,
			CreatedAt:   course.CreatedAt,
			UpdatedAt:   course.UpdatedAt,

			// Progress
			Progress: progress,

			// Favourite
			Favourited: course.Favourited,
		}

		if isAdmin {
			response.Path = course.Path
			response.InitialScan = &course.InitialScan
		}

		responses = append(responses, response)
	}

	return responses
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Course Tag
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type courseTagResponse struct {
	ID  string `json:"id"`
	Tag string `json:"tag"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func courseTagResponseHelper(courseTags []*models.CourseTag) []*courseTagResponse {
	responses := []*courseTagResponse{}
	for _, tag := range courseTags {
		responses = append(responses, &courseTagResponse{
			ID:  tag.ID,
			Tag: tag.Tag,
		})
	}

	return responses
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Lesson
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type lessonResponse struct {
	ID        string         `json:"id"`
	CourseID  string         `json:"courseId"`
	Prefix    int            `json:"prefix"`
	Title     string         `json:"title"`
	Module    string         `json:"module"`
	CreatedAt types.DateTime `json:"createdAt"`
	UpdatedAt types.DateTime `json:"updatedAt"`

	// Relations
	Assets      []*assetResponse      `json:"assets"`
	Attachments []*attachmentResponse `json:"attachments"`

	// Generated during the response helper (when assets include progress)
	Started            bool `json:"started"`
	Completed          bool `json:"completed"`
	AssetsCompleted    int  `json:"assetsCompleted"`
	TotalVideoDuration int  `json:"totalVideoDuration"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func lessonResponseHelper(lessons []*models.Lesson) []*lessonResponse {
	responses := []*lessonResponse{}
	for _, lesson := range lessons {
		response := &lessonResponse{
			ID:        lesson.ID,
			CourseID:  lesson.CourseID,
			Title:     lesson.Title,
			Prefix:    int(lesson.Prefix.Int16),
			Module:    lesson.Module,
			CreatedAt: lesson.CreatedAt,
			UpdatedAt: lesson.UpdatedAt,

			Assets:      assetResponseHelper(lesson.Assets),
			Attachments: attachmentResponseHelper(lesson.Attachments),
		}

		// Counts + Duration
		for _, a := range lesson.Assets {
			// Set the total duration
			if a.AssetMetadata != nil && a.AssetMetadata.VideoMetadata != nil {
				response.TotalVideoDuration += a.AssetMetadata.VideoMetadata.DurationSec
			}

			// Set the number of completed assets and whether the lesson has started
			if a.Progress != nil {
				if a.Progress.Completed {
					response.AssetsCompleted++
				}

				if a.Progress.Completed || a.Progress.Position > 0 {
					response.Started = true
				}
			}

			// Set lesson as completed if all assets are completed (and there is at least one asset)
			response.Completed = len(lesson.Assets) > 0 && response.AssetsCompleted == len(lesson.Assets)
		}

		responses = append(responses, response)
	}

	return responses
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Module
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type moduleResponse struct {
	Prefix  int              `json:"prefix"`
	Module  string           `json:"module"`
	Lessons []lessonResponse `json:"lessons"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
type modulesResponse struct {
	Modules []moduleResponse `json:"modules"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func modulesResponseHelper(lessons []*models.Lesson) modulesResponse {
	const noChapter = "(no chapter)"

	modulesMap := make(map[string][]lessonResponse)
	order := []string{}

	for _, g := range lessons {
		moduleName := strings.TrimSpace(g.Module)
		if moduleName == "" {
			moduleName = noChapter
		}

		lesson := lessonResponseHelper([]*models.Lesson{g})[0]

		if _, ok := modulesMap[moduleName]; !ok {
			order = append(order, moduleName)
			modulesMap[moduleName] = []lessonResponse{*lesson}
		} else {
			modulesMap[moduleName] = append(modulesMap[moduleName], *lesson)
		}
	}

	// Build ordered modules with 1-based index, ensuring lessons are ordered by prefix
	modules := make([]moduleResponse, 0, len(order))
	for i, name := range order {
		lessons := modulesMap[name]
		sort.SliceStable(lessons, func(i, j int) bool { return lessons[i].Prefix < lessons[j].Prefix })

		modules = append(modules, moduleResponse{
			Prefix:  i + 1,
			Module:  name,
			Lessons: lessons,
		})
	}

	return modulesResponse{Modules: modules}
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Asset
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type assetProgressRequest struct {
	Position  int  `json:"position"`
	Completed bool `json:"completed"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type assetProgressResponse struct {
	Position    int            `json:"position"`
	Completed   bool           `json:"completed"`
	CompletedAt types.DateTime `json:"completedAt"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type assetVideoMetadataResponse struct {
	DurationSec int    `json:"durationSec"`
	Container   string `json:"container"`
	MIMEType    string `json:"mimeType"`
	SizeBytes   int64  `json:"sizeBytes"`
	OverallBPS  int    `json:"overallBps"`
	VideoCodec  string `json:"videoCodec"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	FPSNum      int    `json:"fpsNum"`
	FPSDen      int    `json:"fpsDen"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type assetAudioMetadataResponse struct {
	Language      string `json:"language"`
	Codec         string `json:"codec"`
	Profile       string `json:"profile"`
	Channels      int    `json:"channels"`
	ChannelLayout string `json:"channelLayout"`
	SampleRate    int    `json:"sampleRate"`
	BitRate       int    `json:"bitRate"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type assetMetadataResponse struct {
	Video assetVideoMetadataResponse `json:"video,omitempty"`
	Audio assetAudioMetadataResponse `json:"audio,omitempty"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type assetResponse struct {
	ID        string          `json:"id"`
	CourseID  string          `json:"courseId"`
	LessonID  string          `json:"lessonId"`
	Title     string          `json:"title"`
	Prefix    int             `json:"prefix"`
	SubPrefix int             `json:"subPrefix,omitempty"`
	SubTitle  string          `json:"subTitle,omitempty"`
	Module    string          `json:"module"`
	Path      string          `json:"path"`
	Type      types.AssetType `json:"type"`
	CreatedAt types.DateTime  `json:"createdAt"`
	UpdatedAt types.DateTime  `json:"updatedAt"`

	// Relations
	Metadata *assetMetadataResponse `json:"metadata"`
	Progress *assetProgressResponse `json:"progress,omitempty"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func assetResponseHelper(assets []*models.Asset) []*assetResponse {
	responses := []*assetResponse{}
	for _, asset := range assets {

		// Asset metadata
		assetMetadata := &assetMetadataResponse{}
		if asset.AssetMetadata != nil {
			if asset.AssetMetadata.VideoMetadata != nil {
				assetMetadata.Video = assetVideoMetadataResponse{
					DurationSec: asset.AssetMetadata.VideoMetadata.DurationSec,
					Container:   asset.AssetMetadata.VideoMetadata.Container,
					MIMEType:    asset.AssetMetadata.VideoMetadata.MIMEType,
					SizeBytes:   asset.AssetMetadata.VideoMetadata.SizeBytes,
					OverallBPS:  asset.AssetMetadata.VideoMetadata.OverallBPS,
					VideoCodec:  asset.AssetMetadata.VideoMetadata.VideoCodec,
					Width:       asset.AssetMetadata.VideoMetadata.Width,
					Height:      asset.AssetMetadata.VideoMetadata.Height,
					FPSNum:      asset.AssetMetadata.VideoMetadata.FPSNum,
					FPSDen:      asset.AssetMetadata.VideoMetadata.FPSDen,
				}
			}

			if asset.AssetMetadata.AudioMetadata != nil {
				assetMetadata.Audio = assetAudioMetadataResponse{
					Language:      asset.AssetMetadata.AudioMetadata.Language,
					Codec:         asset.AssetMetadata.AudioMetadata.Codec,
					Profile:       asset.AssetMetadata.AudioMetadata.Profile,
					Channels:      asset.AssetMetadata.AudioMetadata.Channels,
					ChannelLayout: asset.AssetMetadata.AudioMetadata.ChannelLayout,
					SampleRate:    asset.AssetMetadata.AudioMetadata.SampleRate,
					BitRate:       asset.AssetMetadata.AudioMetadata.BitRate,
				}
			}
		}

		// Asset progress
		var progress *assetProgressResponse
		if asset.Progress != nil {
			progress = &assetProgressResponse{
				Position:    asset.Progress.Position,
				Completed:   asset.Progress.Completed,
				CompletedAt: asset.Progress.CompletedAt,
			}
		}

		response := &assetResponse{
			ID:        asset.ID,
			CourseID:  asset.CourseID,
			LessonID:  asset.LessonID,
			Title:     asset.Title,
			Prefix:    int(asset.Prefix.Int16),
			Module:    asset.Module,
			Path:      asset.Path,
			Type:      asset.Type,
			CreatedAt: asset.CreatedAt,
			UpdatedAt: asset.UpdatedAt,

			Metadata: assetMetadata,
			Progress: progress,
		}

		// Set sub-prefix and sub-title if available
		if asset.SubPrefix.Valid {
			response.SubPrefix = int(asset.SubPrefix.Int16)
			response.SubTitle = asset.SubTitle
		}

		responses = append(responses, response)
	}

	return responses
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// 	Attachment
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type attachmentResponse struct {
	ID        string         `json:"id"`
	LessonID  string         `json:"lessonId"`
	Title     string         `json:"title"`
	Path      string         `json:"path"`
	CreatedAt types.DateTime `json:"createdAt"`
	UpdatedAt types.DateTime `json:"updatedAt"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func attachmentResponseHelper(attachments []*models.Attachment) []*attachmentResponse {
	if len(attachments) == 0 {
		return []*attachmentResponse{}
	}

	responses := []*attachmentResponse{}
	for _, attachment := range attachments {
		responses = append(responses, &attachmentResponse{
			ID:        attachment.ID,
			LessonID:  attachment.LessonID,
			Title:     attachment.Title,
			Path:      attachment.Path,
			CreatedAt: attachment.CreatedAt,
			UpdatedAt: attachment.UpdatedAt,
		})
	}

	return responses
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type ScanRequest struct {
	CourseID string `json:"courseId"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type scanResponse struct {
	ID          string               `json:"id"`
	CourseID    string               `json:"courseId"`
	CourseTitle string               `json:"courseTitle"`
	Status      types.ScanStatusType `json:"status"`
	Message     string               `json:"message"`
	CreatedAt   types.DateTime       `json:"createdAt"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func scanResponseHelper(scanStates []*coursescan.ScanState, isAdmin bool) []*scanResponse {
	responses := []*scanResponse{}
	for _, scanState := range scanStates {
		response := &scanResponse{
			ID:          scanState.ID,
			CourseID:    scanState.CourseID,
			CourseTitle: scanState.CourseTitle,
			Status:      scanState.GetStatus(),
			Message:     scanState.GetMessage(),
			CreatedAt:   types.DateTime(scanState.CreatedAt),
		}

		responses = append(responses, response)
	}

	return responses
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type tagRequest struct {
	Tag string `json:"tag"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type tagResponse struct {
	ID          string         `json:"id"`
	Tag         string         `json:"tag"`
	CourseCount int            `json:"courseCount"`
	CreatedAt   types.DateTime `json:"createdAt"`
	UpdatedAt   types.DateTime `json:"updatedAt"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func tagResponseHelper(tags []*models.Tag) []*tagResponse {
	responses := []*tagResponse{}

	for _, tag := range tags {
		t := &tagResponse{
			ID:          tag.ID,
			Tag:         tag.Tag,
			CreatedAt:   tag.CreatedAt,
			UpdatedAt:   tag.UpdatedAt,
			CourseCount: tag.CourseCount,
		}

		// Add basic course information
		responses = append(responses, t)
	}

	return responses
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// User
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type userRequest struct {
	Username        string `json:"username"`
	DisplayName     string `json:"displayName"`
	CurrentPassword string `json:"currentPassword"`
	Password        string `json:"password"`
	Role            string `json:"role"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type userResponse struct {
	ID          string         `json:"id"`
	Username    string         `json:"username"`
	DisplayName string         `json:"displayName"`
	Role        types.UserRole `json:"role"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func userResponseHelper(users []*models.User) []*userResponse {
	responses := []*userResponse{}

	for _, user := range users {
		responses = append(responses, &userResponse{
			ID:          user.ID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
		})
	}

	return responses
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
// Auth
// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type signupStatusResponse struct {
	Enabled bool `json:"enabled"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type registerRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type selfUpdateRequest struct {
	DisplayName     string `json:"displayName"`
	CurrentPassword string `json:"currentPassword"`
	Password        string `json:"password"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type selfDeleteRequest struct {
	CurrentPassword string `json:"currentPassword"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

type logResponse struct {
	ID        string         `json:"id"`
	Level     int            `json:"level"`
	Message   string         `json:"message"`
	Data      types.JsonMap  `json:"data"`
	CreatedAt types.DateTime `json:"createdAt"`
}

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

func logsResponseHelper(logs []*models.Log) []*logResponse {
	responses := []*logResponse{}

	for _, log := range logs {
		responses = append(responses, &logResponse{
			ID:        log.ID,
			Level:     log.Level,
			Message:   log.Message,
			Data:      log.Data,
			CreatedAt: log.CreatedAt,
		})
	}

	return responses
}
