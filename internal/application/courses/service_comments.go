package courses

import (
	"context"
	"strings"
	"time"

	"lms-backend/internal/domain/courses"
	domainenrollments "lms-backend/internal/domain/enrollments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

func (s *service) ListCourseComments(ctx context.Context, courseID uuid.UUID, page, limit int) (*CommentsResponse, error) {
	if s.commentRepo == nil {
		return nil, apperrors.NewInternalError("COMMENTS_NOT_CONFIGURED", "course comments are not configured")
	}
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	comments, total, err := s.commentRepo.FindByCourseID(ctx, courseID, page, limit)
	if err != nil {
		return nil, err
	}
	responses := make([]CommentResponse, 0, len(comments))
	for _, comment := range comments {
		responses = append(responses, toCommentResponse(comment))
	}
	totalPages := (total + limit - 1) / limit
	return &CommentsResponse{
		Comments: responses,
		Meta: map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}, nil
}

func (s *service) CreateCourseComment(ctx context.Context, cmd CreateCourseCommentCommand) (*CommentResponse, error) {
	if s.commentRepo == nil {
		return nil, apperrors.NewInternalError("COMMENTS_NOT_CONFIGURED", "course comments are not configured")
	}
	content := strings.TrimSpace(cmd.Content)
	if content == "" {
		return nil, apperrors.NewSimpleValidationError("COMMENT_REQUIRED", "comment cannot be empty")
	}
	if len(content) > 2000 {
		return nil, apperrors.NewSimpleValidationError("COMMENT_TOO_LONG", "comment must be 2000 characters or less")
	}
	if err := s.validateCourseCommentTarget(ctx, cmd.CourseID, cmd.ModuleID, cmd.LessonID); err != nil {
		return nil, err
	}
	if cmd.ParentCommentID != nil {
		parent, err := s.commentRepo.FindByID(ctx, *cmd.ParentCommentID)
		if err != nil {
			return nil, apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "parent comment not found")
		}
		if parent.CourseID != cmd.CourseID {
			return nil, apperrors.NewSimpleValidationError("INVALID_PARENT_COMMENT", "parent comment belongs to a different course")
		}
	}
	if err := s.ensureCanParticipateInCourseDiscussion(ctx, cmd.UserID, cmd.Role, cmd.CourseID); err != nil {
		return nil, err
	}
	now := time.Now()
	comment := &courses.CourseComment{
		ID:              uuid.New(),
		CourseID:        cmd.CourseID,
		ModuleID:        cmd.ModuleID,
		LessonID:        cmd.LessonID,
		QuizID:          cmd.QuizID,
		UserID:          cmd.UserID,
		ParentCommentID: cmd.ParentCommentID,
		Content:         content,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.commentRepo.Create(ctx, comment); err != nil {
		return nil, err
	}
	response := toCommentResponse(comment)
	return &response, nil
}

func (s *service) UpdateCourseComment(ctx context.Context, cmd UpdateCourseCommentCommand) (*CommentResponse, error) {
	if s.commentRepo == nil {
		return nil, apperrors.NewInternalError("COMMENTS_NOT_CONFIGURED", "course comments are not configured")
	}
	comment, err := s.commentRepo.FindByID(ctx, cmd.CommentID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "comment not found")
	}
	isModerator, err := s.canModerateCourseComment(ctx, cmd.UserID, cmd.Role, comment.CourseID)
	if err != nil {
		return nil, err
	}
	if comment.UserID != cmd.UserID && !isModerator {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to update this comment")
	}
	if cmd.IsPinned != nil {
		if !isModerator {
			return nil, apperrors.NewForbiddenError("FORBIDDEN", "only the course teacher or admin can pin comments")
		}
		comment.IsPinned = *cmd.IsPinned
	}
	if strings.TrimSpace(cmd.Content) != "" {
		content := strings.TrimSpace(cmd.Content)
		if len(content) > 2000 {
			return nil, apperrors.NewSimpleValidationError("COMMENT_TOO_LONG", "comment must be 2000 characters or less")
		}
		comment.Content = content
	}
	comment.UpdatedAt = time.Now()
	if err := s.commentRepo.Update(ctx, comment); err != nil {
		return nil, err
	}
	response := toCommentResponse(comment)
	return &response, nil
}

func (s *service) DeleteCourseComment(ctx context.Context, cmd DeleteCourseCommentCommand) error {
	if s.commentRepo == nil {
		return apperrors.NewInternalError("COMMENTS_NOT_CONFIGURED", "course comments are not configured")
	}
	comment, err := s.commentRepo.FindByID(ctx, cmd.CommentID)
	if err != nil {
		return apperrors.NewNotFoundError("COMMENT_NOT_FOUND", "comment not found")
	}
	isModerator, err := s.canModerateCourseComment(ctx, cmd.UserID, cmd.Role, comment.CourseID)
	if err != nil {
		return err
	}
	if comment.UserID != cmd.UserID && !isModerator {
		return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to delete this comment")
	}
	return s.commentRepo.SoftDelete(ctx, cmd.CommentID)
}

func (s *service) validateCourseCommentTarget(ctx context.Context, courseID uuid.UUID, moduleID, lessonID *uuid.UUID) error {
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil || course == nil {
		return apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if moduleID != nil {
		module, err := s.moduleRepo.FindByID(ctx, *moduleID)
		if err != nil || module == nil {
			return apperrors.NewNotFoundError("MODULE_NOT_FOUND", "module not found")
		}
		if module.CourseID != courseID {
			return apperrors.NewSimpleValidationError("MODULE_MISMATCH", "module does not belong to this course")
		}
	}
	if lessonID != nil {
		lesson, err := s.lessonRepo.FindByID(ctx, *lessonID)
		if err != nil || lesson == nil {
			return apperrors.NewNotFoundError("LESSON_NOT_FOUND", "lesson not found")
		}
		chapter, err := s.chapterRepo.FindByID(ctx, lesson.ChapterID)
		if err != nil || chapter == nil {
			return apperrors.NewNotFoundError("CHAPTER_NOT_FOUND", "chapter not found")
		}
		module, err := s.moduleRepo.FindByID(ctx, chapter.ModuleID)
		if err != nil || module == nil {
			return apperrors.NewNotFoundError("MODULE_NOT_FOUND", "module not found")
		}
		if module.CourseID != courseID {
			return apperrors.NewSimpleValidationError("LESSON_MISMATCH", "lesson does not belong to this course")
		}
	}
	return nil
}

func (s *service) ensureCanParticipateInCourseDiscussion(ctx context.Context, userID uuid.UUID, role string, courseID uuid.UUID) error {
	if role == "admin" {
		return nil
	}
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil || course == nil {
		return apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if role == "teacher" && course.TeacherID == userID {
		return nil
	}
	if role == "student" {
		return s.ensureStudentHasCourseAccess(ctx, userID, courseID)
	}
	return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to comment on this course")
}

func (s *service) canModerateCourseComment(ctx context.Context, userID uuid.UUID, role string, courseID uuid.UUID) (bool, error) {
	if role == "admin" {
		return true, nil
	}
	if role != "teacher" {
		return false, nil
	}
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil || course == nil {
		return false, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	return course.TeacherID == userID, nil
}

func (s *service) ensureStudentHasCourseAccess(ctx context.Context, studentID, courseID uuid.UUID) error {
	if s.enrollmentRepo == nil {
		return apperrors.NewForbiddenError("NOT_ENROLLED", "course access is required")
	}
	enrollment, err := s.enrollmentRepo.FindByStudentAndCourse(ctx, studentID, courseID)
	if err != nil || enrollment == nil || enrollment.Status != domainenrollments.EnrollmentStatusActive {
		return apperrors.NewForbiddenError("NOT_ENROLLED", "you must be enrolled or approved before using this course feature")
	}
	return nil
}

func toCommentResponse(comment *courses.CourseComment) CommentResponse {
	return CommentResponse{
		ID:              comment.ID,
		CourseID:        comment.CourseID,
		ModuleID:        comment.ModuleID,
		LessonID:        comment.LessonID,
		QuizID:          comment.QuizID,
		UserID:          comment.UserID,
		ParentCommentID: comment.ParentCommentID,
		Content:         comment.Content,
		IsPinned:        comment.IsPinned,
		CreatedAt:       comment.CreatedAt,
		UpdatedAt:       comment.UpdatedAt,
	}
}
