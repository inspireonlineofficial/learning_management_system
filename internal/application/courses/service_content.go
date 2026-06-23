package courses

import (
	"context"
	"fmt"
	"log"
	"time"

	"lms-backend/internal/domain/courses"
	tsclient "lms-backend/internal/infrastructure/typesense"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// CreateModule creates a new module
func (s *service) CreateModule(ctx context.Context, cmd CreateModuleCommand) (*ModuleResponse, error) {
	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	module := &courses.Module{
		ID:          uuid.New(),
		CourseID:    cmd.CourseID,
		Title:       cmd.Title,
		Description: cmd.Description,
		Position:    cmd.Position,
		IsFree:      cmd.IsFree,
		IsPublished: cmd.IsPublished,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err = s.moduleRepo.Create(ctx, module)
	if err != nil {
		return nil, err
	}

	return &ModuleResponse{
		ID:          module.ID,
		CourseID:    module.CourseID,
		Title:       module.Title,
		Description: module.Description,
		Position:    module.Position,
		IsFree:      module.IsFree,
		IsPublished: module.IsPublished,
		Chapters:    []ChapterResponse{},
		CreatedAt:   module.CreatedAt,
		UpdatedAt:   module.UpdatedAt,
	}, nil
}

// UpdateModule updates an existing module
func (s *service) UpdateModule(ctx context.Context, cmd UpdateModuleCommand) (*ModuleResponse, error) {
	module, err := s.moduleRepo.FindByID(ctx, cmd.ModuleID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "module not found")
	}

	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, module.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	module.Title = cmd.Title
	module.Description = cmd.Description
	module.Position = cmd.Position
	module.IsFree = cmd.IsFree
	module.IsPublished = cmd.IsPublished

	err = s.moduleRepo.Update(ctx, module)
	if err != nil {
		return nil, err
	}

	return &ModuleResponse{
		ID:          module.ID,
		CourseID:    module.CourseID,
		Title:       module.Title,
		Description: module.Description,
		Position:    module.Position,
		IsFree:      module.IsFree,
		IsPublished: module.IsPublished,
		Chapters:    []ChapterResponse{},
		CreatedAt:   module.CreatedAt,
		UpdatedAt:   module.UpdatedAt,
	}, nil
}

// DeleteModule soft-deletes a module and cascades to chapters and lessons
func (s *service) DeleteModule(ctx context.Context, cmd DeleteModuleCommand) error {
	module, err := s.moduleRepo.FindByID(ctx, cmd.ModuleID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "module not found")
	}

	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, module.CourseID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	return s.moduleRepo.CascadeSoftDelete(ctx, cmd.ModuleID)
}

// CreateChapter creates a new chapter
func (s *service) CreateChapter(ctx context.Context, cmd CreateChapterCommand) (*ChapterResponse, error) {
	module, err := s.moduleRepo.FindByID(ctx, cmd.ModuleID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "module not found")
	}

	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, module.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	chapter := &courses.Chapter{
		ID:        uuid.New(),
		ModuleID:  cmd.ModuleID,
		Title:     cmd.Title,
		Position:  cmd.Position,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.chapterRepo.Create(ctx, chapter)
	if err != nil {
		return nil, err
	}

	return &ChapterResponse{
		ID:        chapter.ID,
		ModuleID:  chapter.ModuleID,
		Title:     chapter.Title,
		Position:  chapter.Position,
		Lessons:   []LessonResponse{},
		CreatedAt: chapter.CreatedAt,
		UpdatedAt: chapter.UpdatedAt,
	}, nil
}

// UpdateChapter updates an existing chapter
func (s *service) UpdateChapter(ctx context.Context, cmd UpdateChapterCommand) (*ChapterResponse, error) {
	chapter, err := s.chapterRepo.FindByID(ctx, cmd.ChapterID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "chapter not found")
	}

	module, err := s.moduleRepo.FindByID(ctx, chapter.ModuleID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "module not found")
	}

	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, module.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	chapter.Title = cmd.Title
	chapter.Position = cmd.Position

	err = s.chapterRepo.Update(ctx, chapter)
	if err != nil {
		return nil, err
	}

	return &ChapterResponse{
		ID:        chapter.ID,
		ModuleID:  chapter.ModuleID,
		Title:     chapter.Title,
		Position:  chapter.Position,
		Lessons:   []LessonResponse{},
		CreatedAt: chapter.CreatedAt,
		UpdatedAt: chapter.UpdatedAt,
	}, nil
}

// DeleteChapter soft-deletes a chapter and cascades to lessons
func (s *service) DeleteChapter(ctx context.Context, cmd DeleteChapterCommand) error {
	chapter, err := s.chapterRepo.FindByID(ctx, cmd.ChapterID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "chapter not found")
	}

	module, err := s.moduleRepo.FindByID(ctx, chapter.ModuleID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "module not found")
	}

	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, module.CourseID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	return s.chapterRepo.CascadeSoftDelete(ctx, cmd.ChapterID)
}

// CreateLesson creates a new lesson
func (s *service) CreateLesson(ctx context.Context, cmd CreateLessonCommand) (*LessonResponse, error) {
	chapter, err := s.chapterRepo.FindByID(ctx, cmd.ChapterID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "chapter not found")
	}

	module, err := s.moduleRepo.FindByID(ctx, chapter.ModuleID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "module not found")
	}

	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, module.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	lesson := &courses.Lesson{
		ID:              uuid.New(),
		ChapterID:       cmd.ChapterID,
		Title:           cmd.Title,
		Description:     cmd.Description,
		Type:            courses.LessonType(cmd.Type),
		VideoID:         cmd.VideoID,
		DurationSeconds: cmd.DurationSeconds,
		IsFreePreview:   cmd.IsFreePreview,
		IsFree:          cmd.IsFree,
		IsDownloadable:  cmd.IsDownloadable,
		Position:        cmd.Position,
		Status:          courses.LessonStatus(cmd.Status),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	err = s.lessonRepo.Create(ctx, lesson)
	if err != nil {
		return nil, err
	}

	if lesson.Status == courses.LessonStatusPublished {
		if idxErr := s.indexer.UpsertLesson(ctx, tsclient.LessonDocument{
			ID:            lesson.ID.String(),
			Title:         lesson.Title,
			CourseID:      course.ID.String(),
			CourseTitle:   course.Title,
			IsFreePreview: lesson.IsFreePreview,
			Status:        string(lesson.Status),
		}); idxErr != nil {
			log.Printf("typesense index error: %v", idxErr)
		}
	}

	return &LessonResponse{
		ID:              lesson.ID,
		ChapterID:       lesson.ChapterID,
		Title:           lesson.Title,
		Description:     lesson.Description,
		Type:            string(lesson.Type),
		DurationSeconds: lesson.DurationSeconds,
		IsFreePreview:   lesson.IsFreePreview,
		IsFree:          lesson.IsFree,
		IsDownloadable:  lesson.IsDownloadable,
		Position:        lesson.Position,
		Status:          string(lesson.Status),
		CreatedAt:       lesson.CreatedAt,
		UpdatedAt:       lesson.UpdatedAt,
	}, nil
}

// UpdateLesson updates an existing lesson
func (s *service) UpdateLesson(ctx context.Context, cmd UpdateLessonCommand) (*LessonResponse, error) {
	lesson, err := s.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "lesson not found")
	}

	chapter, err := s.chapterRepo.FindByID(ctx, lesson.ChapterID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "chapter not found")
	}

	module, err := s.moduleRepo.FindByID(ctx, chapter.ModuleID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "module not found")
	}

	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, module.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	lesson.Title = cmd.Title
	lesson.Description = cmd.Description
	lesson.Type = courses.LessonType(cmd.Type)
	if cmd.UpdateVideoID {
		lesson.VideoID = cmd.VideoID
	}
	lesson.DurationSeconds = cmd.DurationSeconds
	lesson.IsFreePreview = cmd.IsFreePreview
	lesson.IsFree = cmd.IsFree
	lesson.IsDownloadable = cmd.IsDownloadable
	lesson.Position = cmd.Position
	lesson.Status = courses.LessonStatus(cmd.Status)

	err = s.lessonRepo.Update(ctx, lesson)
	if err != nil {
		return nil, err
	}

	if lesson.Status == courses.LessonStatusPublished {
		if idxErr := s.indexer.UpsertLesson(ctx, tsclient.LessonDocument{
			ID:            lesson.ID.String(),
			Title:         lesson.Title,
			CourseID:      course.ID.String(),
			CourseTitle:   course.Title,
			IsFreePreview: lesson.IsFreePreview,
			Status:        string(lesson.Status),
		}); idxErr != nil {
			log.Printf("typesense index error: %v", idxErr)
		}
	} else {
		if idxErr := s.indexer.DeleteLesson(ctx, lesson.ID.String()); idxErr != nil {
			log.Printf("typesense index error: %v", idxErr)
		}
	}

	return &LessonResponse{
		ID:              lesson.ID,
		ChapterID:       lesson.ChapterID,
		Title:           lesson.Title,
		Description:     lesson.Description,
		Type:            string(lesson.Type),
		DurationSeconds: lesson.DurationSeconds,
		IsFreePreview:   lesson.IsFreePreview,
		IsFree:          lesson.IsFree,
		IsDownloadable:  lesson.IsDownloadable,
		Position:        lesson.Position,
		Status:          string(lesson.Status),
		CreatedAt:       lesson.CreatedAt,
		UpdatedAt:       lesson.UpdatedAt,
	}, nil
}

// DeleteLesson soft-deletes a lesson
func (s *service) DeleteLesson(ctx context.Context, cmd DeleteLessonCommand) error {
	lesson, err := s.lessonRepo.FindByID(ctx, cmd.LessonID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "lesson not found")
	}

	chapter, err := s.chapterRepo.FindByID(ctx, lesson.ChapterID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "chapter not found")
	}

	module, err := s.moduleRepo.FindByID(ctx, chapter.ModuleID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "module not found")
	}

	// Verify course ownership
	course, err := s.courseRepo.FindByID(ctx, module.CourseID)
	if err != nil {
		return apperrors.NewNotFoundError("NOT_FOUND", "course not found")
	}

	if !course.IsOwnedBy(cmd.TeacherID) {
		return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
	}

	if err := s.lessonRepo.SoftDelete(ctx, cmd.LessonID); err != nil {
		return err
	}

	if idxErr := s.indexer.DeleteLesson(ctx, lesson.ID.String()); idxErr != nil {
		log.Printf("typesense index error: %v", idxErr)
	}

	return nil
}

// ReorderContent atomically updates position values for content items
func (s *service) ReorderContent(ctx context.Context, cmd ReorderContentCommand) error {
	switch cmd.Type {
	case "module":
		// Verify course ownership
		course, err := s.courseRepo.FindByID(ctx, cmd.ParentID)
		if err != nil {
			return apperrors.NewNotFoundError("NOT_FOUND", "course not found")
		}
		if !course.IsOwnedBy(cmd.TeacherID) {
			return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
		}
		return s.moduleRepo.Reorder(ctx, cmd.ParentID, cmd.Positions)

	case "chapter":
		// Verify course ownership via module
		module, err := s.moduleRepo.FindByID(ctx, cmd.ParentID)
		if err != nil {
			return apperrors.NewNotFoundError("NOT_FOUND", "module not found")
		}
		course, err := s.courseRepo.FindByID(ctx, module.CourseID)
		if err != nil {
			return apperrors.NewNotFoundError("NOT_FOUND", "course not found")
		}
		if !course.IsOwnedBy(cmd.TeacherID) {
			return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
		}
		return s.chapterRepo.Reorder(ctx, cmd.ParentID, cmd.Positions)

	case "lesson":
		// Verify course ownership via chapter and module
		chapter, err := s.chapterRepo.FindByID(ctx, cmd.ParentID)
		if err != nil {
			return apperrors.NewNotFoundError("NOT_FOUND", "chapter not found")
		}
		module, err := s.moduleRepo.FindByID(ctx, chapter.ModuleID)
		if err != nil {
			return apperrors.NewNotFoundError("NOT_FOUND", "module not found")
		}
		course, err := s.courseRepo.FindByID(ctx, module.CourseID)
		if err != nil {
			return apperrors.NewNotFoundError("NOT_FOUND", "course not found")
		}
		if !course.IsOwnedBy(cmd.TeacherID) {
			return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to modify this course")
		}
		return s.lessonRepo.Reorder(ctx, cmd.ParentID, cmd.Positions)

	default:
		return fmt.Errorf("invalid content type: %s", cmd.Type)
	}
}
