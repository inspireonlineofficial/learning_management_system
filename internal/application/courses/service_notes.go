package courses

import (
	"context"
	"strings"
	"time"

	"lms-backend/internal/domain/courses"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

func (s *service) CreateCourseNote(ctx context.Context, cmd CreateCourseNoteCommand) (*NoteResponse, error) {
	if s.noteRepo == nil {
		return nil, apperrors.NewInternalError("NOTES_UNAVAILABLE", "course notes are not configured")
	}
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you can only add notes to your own courses")
	}

	moduleID, lessonID, err := s.validateNoteTarget(ctx, course.ID, cmd.ModuleID, cmd.LessonID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, apperrors.NewSimpleValidationError("NOTE_TITLE_REQUIRED", "note title is required")
	}
	if strings.TrimSpace(cmd.Content) == "" && strings.TrimSpace(cmd.FileURL) == "" {
		return nil, apperrors.NewSimpleValidationError("NOTE_CONTENT_REQUIRED", "note content or file URL is required")
	}

	now := time.Now().UTC()
	note := &courses.CourseNote{
		ID:          uuid.New(),
		CourseID:    course.ID,
		ModuleID:    moduleID,
		LessonID:    lessonID,
		Title:       title,
		Content:     strings.TrimSpace(cmd.Content),
		FileURL:     strings.TrimSpace(cmd.FileURL),
		IsFree:      cmd.IsFree,
		IsPublished: cmd.IsPublished,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.noteRepo.Create(ctx, note); err != nil {
		return nil, err
	}
	return ptrNoteResponse(s.toNoteResponse(note)), nil
}

func (s *service) UpdateCourseNote(ctx context.Context, cmd UpdateCourseNoteCommand) (*NoteResponse, error) {
	if s.noteRepo == nil {
		return nil, apperrors.NewInternalError("NOTES_UNAVAILABLE", "course notes are not configured")
	}
	note, err := s.noteRepo.FindByID(ctx, cmd.NoteID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("NOTE_NOT_FOUND", "note not found")
	}
	course, err := s.courseRepo.FindByID(ctx, note.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "you can only edit notes on your own courses")
	}

	moduleID, lessonID, err := s.validateNoteTarget(ctx, course.ID, cmd.ModuleID, cmd.LessonID)
	if err != nil {
		return nil, err
	}
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return nil, apperrors.NewSimpleValidationError("NOTE_TITLE_REQUIRED", "note title is required")
	}
	if strings.TrimSpace(cmd.Content) == "" && strings.TrimSpace(cmd.FileURL) == "" {
		return nil, apperrors.NewSimpleValidationError("NOTE_CONTENT_REQUIRED", "note content or file URL is required")
	}

	note.ModuleID = moduleID
	note.LessonID = lessonID
	note.Title = title
	note.Content = strings.TrimSpace(cmd.Content)
	note.FileURL = strings.TrimSpace(cmd.FileURL)
	note.IsFree = cmd.IsFree
	note.IsPublished = cmd.IsPublished
	if err := s.noteRepo.Update(ctx, note); err != nil {
		return nil, err
	}
	return ptrNoteResponse(s.toNoteResponse(note)), nil
}

func (s *service) DeleteCourseNote(ctx context.Context, cmd DeleteCourseNoteCommand) error {
	if s.noteRepo == nil {
		return apperrors.NewInternalError("NOTES_UNAVAILABLE", "course notes are not configured")
	}
	note, err := s.noteRepo.FindByID(ctx, cmd.NoteID)
	if err != nil {
		return apperrors.NewNotFoundError("NOTE_NOT_FOUND", "note not found")
	}
	course, err := s.courseRepo.FindByID(ctx, note.CourseID)
	if err != nil {
		return apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(cmd.TeacherID) {
		return apperrors.NewForbiddenError("FORBIDDEN", "you can only delete notes on your own courses")
	}
	return s.noteRepo.SoftDelete(ctx, cmd.NoteID)
}

func (s *service) validateNoteTarget(ctx context.Context, courseID uuid.UUID, moduleID, lessonID *uuid.UUID) (*uuid.UUID, *uuid.UUID, error) {
	if moduleID != nil {
		module, err := s.moduleRepo.FindByID(ctx, *moduleID)
		if err != nil {
			return nil, nil, apperrors.NewNotFoundError("MODULE_NOT_FOUND", "module not found")
		}
		if module.CourseID != courseID {
			return nil, nil, apperrors.NewSimpleValidationError("INVALID_MODULE", "module does not belong to this course")
		}
	}

	if lessonID == nil {
		return moduleID, nil, nil
	}

	lesson, err := s.lessonRepo.FindByID(ctx, *lessonID)
	if err != nil {
		return nil, nil, apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson not found")
	}
	chapter, err := s.chapterRepo.FindByID(ctx, lesson.ChapterID)
	if err != nil {
		return nil, nil, apperrors.NewNotFoundError("CHAPTER_NOT_FOUND", "chapter not found")
	}
	module, err := s.moduleRepo.FindByID(ctx, chapter.ModuleID)
	if err != nil {
		return nil, nil, apperrors.NewNotFoundError("MODULE_NOT_FOUND", "module not found")
	}
	if module.CourseID != courseID {
		return nil, nil, apperrors.NewSimpleValidationError("INVALID_LESSON", "lesson does not belong to this course")
	}
	if moduleID != nil && *moduleID != module.ID {
		return nil, nil, apperrors.NewSimpleValidationError("INVALID_NOTE_TARGET", "lesson does not belong to the selected module")
	}
	return &module.ID, lessonID, nil
}

func ptrNoteResponse(resp NoteResponse) *NoteResponse {
	return &resp
}
