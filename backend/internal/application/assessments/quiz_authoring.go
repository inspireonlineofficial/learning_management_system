package assessments

import (
	"context"
	"math"
	"net/url"
	"strings"
	"time"

	"lms-backend/internal/domain/assessments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// Limits that keep a single quiz within sane bounds.
const (
	maxQuestionsPerQuiz   = 300
	maxOptionsPerQuestion = 20
	maxContentLength      = 20000
	maxTitleLength        = 255
)

// CreateQuiz creates a quiz with its questions in one transaction.
func (s *service) CreateQuiz(ctx context.Context, cmd CreateQuizCommand) (*QuizResponse, error) {
	course, err := s.courseRepo.FindByID(ctx, cmd.CourseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(cmd.TeacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to create quiz for this course")
	}
	if err := validateCreateQuiz(cmd); err != nil {
		return nil, err
	}

	now := s.clock().UTC()
	quiz := &assessments.Quiz{
		ID:        uuid.New(),
		CourseID:  cmd.CourseID,
		CreatedAt: now,
	}
	applyQuizSettings(quiz, cmd, now)

	err = s.inTx(ctx, func(ctx context.Context) error {
		if err := s.quizRepo.Create(ctx, quiz); err != nil {
			return err
		}
		for i, qCmd := range cmd.Questions {
			if _, _, err := s.insertQuestion(ctx, quiz.ID, qCmd, i+1, now); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	resp := s.toQuizResponse(quiz)
	resp.QuestionCount = len(cmd.Questions)
	for _, q := range cmd.Questions {
		resp.TotalMarks += normalizeMarks(q.Marks)
	}
	return resp, nil
}

func applyQuizSettings(quiz *assessments.Quiz, cmd CreateQuizCommand, now time.Time) {
	quiz.LessonID = cmd.LessonID
	quiz.Title = strings.TrimSpace(cmd.Title)
	quiz.TimeLimitSeconds = cmd.TimeLimitSeconds
	quiz.MaxAttempts = cmd.MaxAttempts
	quiz.PassingScorePercent = cmd.PassingScorePercent
	quiz.ShuffleQuestions = cmd.ShuffleQuestions
	quiz.ShowAnswersAfterSubmission = cmd.ShowAnswersAfterSubmission
	quiz.IsFree = cmd.IsFree
	quiz.IsPublished = cmd.IsPublished
	quiz.Description = strings.TrimSpace(cmd.Description)
	quiz.Instructions = strings.TrimSpace(cmd.Instructions)
	quiz.Mode = assessments.QuizModeExam
	if cmd.Mode == string(assessments.QuizModePractice) {
		quiz.Mode = assessments.QuizModePractice
	}
	quiz.ShuffleOptions = cmd.ShuffleOptions
	quiz.PartialCredit = cmd.PartialCredit
	quiz.AvailableFrom = cmd.AvailableFrom
	quiz.AvailableUntil = cmd.AvailableUntil
	quiz.CalculatorPolicy = normalizeCalculatorPolicy(cmd.CalculatorPolicy)
	quiz.ReferenceMaterial = strings.TrimSpace(cmd.ReferenceMaterial)
	quiz.UpdatedAt = now
}

func normalizeCalculatorPolicy(p string) string {
	switch p {
	case "basic", "scientific", "graphing":
		return p
	default:
		return "none"
	}
}

func validateCreateQuiz(cmd CreateQuizCommand) error {
	title := strings.TrimSpace(cmd.Title)
	if title == "" {
		return apperrors.NewSimpleValidationError("TITLE_REQUIRED", "quiz title is required")
	}
	if len(title) > maxTitleLength {
		return apperrors.NewSimpleValidationError("TITLE_TOO_LONG", "quiz title is too long")
	}
	// 0 = untimed (practice quizzes); otherwise at most 24 hours.
	if cmd.TimeLimitSeconds < 0 || cmd.TimeLimitSeconds > 24*3600 {
		return apperrors.NewSimpleValidationError("INVALID_TIME_LIMIT", "time limit must be between 0 (untimed) and 24 hours")
	}
	if cmd.MaxAttempts < 0 {
		return apperrors.NewSimpleValidationError("INVALID_MAX_ATTEMPTS", "max attempts cannot be negative")
	}
	if cmd.PassingScorePercent < 0 || cmd.PassingScorePercent > 100 {
		return apperrors.NewSimpleValidationError("INVALID_PASSING_SCORE", "passing score must be between 0 and 100")
	}
	if cmd.Mode != "" && cmd.Mode != string(assessments.QuizModeExam) && cmd.Mode != string(assessments.QuizModePractice) {
		return apperrors.NewSimpleValidationError("INVALID_MODE", "mode must be exam or practice")
	}
	if cmd.AvailableFrom != nil && cmd.AvailableUntil != nil && !cmd.AvailableUntil.After(*cmd.AvailableFrom) {
		return apperrors.NewSimpleValidationError("INVALID_AVAILABILITY", "the closing time must be after the opening time")
	}
	for _, text := range []string{cmd.Description, cmd.Instructions, cmd.ReferenceMaterial} {
		if len(text) > maxContentLength {
			return apperrors.NewSimpleValidationError("CONTENT_TOO_LONG", "quiz text fields are limited to 20,000 characters")
		}
	}
	// Drafts may be saved without questions; publishing needs at least one.
	if cmd.IsPublished && len(cmd.Questions) == 0 {
		return apperrors.NewSimpleValidationError("QUESTIONS_REQUIRED", "at least one question is required to publish")
	}
	if len(cmd.Questions) > maxQuestionsPerQuiz {
		return apperrors.NewSimpleValidationError("TOO_MANY_QUESTIONS", "a quiz can have at most 300 questions")
	}
	for _, question := range cmd.Questions {
		if err := validateQuestion(question); err != nil {
			return err
		}
	}
	return nil
}

func validateQuestion(question CreateQuestionCommand) error {
	if !hasTextOrImage(question.Body, question.ImageURL) {
		return apperrors.NewSimpleValidationError("QUESTION_CONTENT_REQUIRED", "each question must include text or an image")
	}
	if len(question.Body) > maxContentLength || len(question.Explanation) > maxContentLength || len(question.Hint) > maxContentLength {
		return apperrors.NewSimpleValidationError("CONTENT_TOO_LONG", "question text is limited to 20,000 characters")
	}
	if !validMediaURL(question.ImageURL) {
		return apperrors.NewSimpleValidationError("INVALID_IMAGE_URL", "images must use an http(s) URL")
	}
	if question.Marks < 0 || question.Marks > 1000 || math.IsNaN(question.Marks) {
		return apperrors.NewSimpleValidationError("INVALID_MARKS", "marks must be between 0 and 1000")
	}
	if len(question.Options) > maxOptionsPerQuestion {
		return apperrors.NewSimpleValidationError("TOO_MANY_OPTIONS", "a question can have at most 20 options")
	}
	settings := question.Settings
	switch assessments.QuestionType(question.Type) {
	case assessments.QuestionTypeSingle, assessments.QuestionTypeMultiple, assessments.QuestionTypeTrueFalse:
		if len(question.Options) < 2 {
			return apperrors.NewSimpleValidationError("OPTIONS_REQUIRED", "each question must include at least two options")
		}
		correct := 0
		for _, option := range question.Options {
			if !hasTextOrImage(option.Body, option.ImageURL) {
				return apperrors.NewSimpleValidationError("OPTION_CONTENT_REQUIRED", "each answer option must include text or an image")
			}
			if !validMediaURL(option.ImageURL) {
				return apperrors.NewSimpleValidationError("INVALID_IMAGE_URL", "images must use an http(s) URL")
			}
			if option.IsCorrect {
				correct++
			}
		}
		if correct == 0 {
			return apperrors.NewSimpleValidationError("CORRECT_OPTION_REQUIRED", "each question must include a correct option")
		}
		if question.Type != string(assessments.QuestionTypeMultiple) && correct != 1 {
			return apperrors.NewSimpleValidationError("INVALID_CORRECT_OPTIONS", "single-answer questions must include exactly one correct option")
		}
	case assessments.QuestionTypeShort:
		accepted := 0
		for _, option := range question.Options {
			if option.IsCorrect && strings.TrimSpace(option.Body) != "" {
				accepted++
			}
		}
		if accepted == 0 {
			return apperrors.NewSimpleValidationError("CORRECT_TEXT_REQUIRED", "short answer questions must include an expected answer")
		}
	case assessments.QuestionTypeNumeric:
		if settings == nil || settings.NumericAnswer == nil || math.IsNaN(*settings.NumericAnswer) || math.IsInf(*settings.NumericAnswer, 0) {
			return apperrors.NewSimpleValidationError("NUMERIC_ANSWER_REQUIRED", "numeric questions need a correct numeric answer")
		}
		if settings.Tolerance < 0 || math.IsNaN(settings.Tolerance) {
			return apperrors.NewSimpleValidationError("INVALID_TOLERANCE", "tolerance cannot be negative")
		}
		if settings.ToleranceType != "" && settings.ToleranceType != "absolute" && settings.ToleranceType != "percent" {
			return apperrors.NewSimpleValidationError("INVALID_TOLERANCE", "tolerance type must be absolute or percent")
		}
	case assessments.QuestionTypeLong:
		if settings != nil && (settings.MaxWords < 0 || settings.MaxWords > 10000) {
			return apperrors.NewSimpleValidationError("INVALID_WORD_LIMIT", "word limit must be between 0 and 10,000")
		}
	default:
		return apperrors.NewSimpleValidationError("INVALID_QUESTION_TYPE", "question type is invalid")
	}
	return nil
}

func validMediaURL(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "/") {
		return true
	}
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func hasTextOrImage(text, imageURL string) bool {
	return strings.TrimSpace(text) != "" || strings.TrimSpace(imageURL) != ""
}

func normalizeContentType(raw, text, imageURL string) string {
	contentType := strings.TrimSpace(raw)
	if contentType == "text" || contentType == "image" || contentType == "text_image" {
		return contentType
	}
	hasText := strings.TrimSpace(text) != ""
	hasImage := strings.TrimSpace(imageURL) != ""
	if hasText && hasImage {
		return "text_image"
	}
	if hasImage {
		return "image"
	}
	return "text"
}

func normalizeMarks(raw float64) float64 {
	if raw > 0 {
		return round2(raw)
	}
	return 1
}

func settingsFromCommand(qType assessments.QuestionType, cmd *QuestionSettingsCommand) assessments.QuestionSettings {
	if cmd == nil {
		return assessments.QuestionSettings{}
	}
	settings := assessments.QuestionSettings{}
	switch qType {
	case assessments.QuestionTypeNumeric:
		settings.NumericAnswer = cmd.NumericAnswer
		settings.Tolerance = cmd.Tolerance
		settings.ToleranceType = toleranceTypeOrDefault(cmd.ToleranceType)
		settings.Unit = strings.TrimSpace(cmd.Unit)
	case assessments.QuestionTypeLong:
		settings.MaxWords = cmd.MaxWords
	case assessments.QuestionTypeShort:
		settings.CaseSensitive = cmd.CaseSensitive
	}
	return settings
}

func applyQuestionFields(question *assessments.Question, qCmd CreateQuestionCommand, position int, now time.Time) {
	question.Body = qCmd.Body
	question.Type = assessments.QuestionType(qCmd.Type)
	question.ContentType = normalizeContentType(qCmd.ContentType, qCmd.Body, qCmd.ImageURL)
	question.ImageURL = strings.TrimSpace(qCmd.ImageURL)
	question.Marks = normalizeMarks(qCmd.Marks)
	question.IsRequired = qCmd.IsRequired
	question.Position = position
	question.Explanation = qCmd.Explanation
	question.Hint = qCmd.Hint
	question.Settings = settingsFromCommand(question.Type, qCmd.Settings)
	question.UpdatedAt = now
}

func optionFromCommand(questionID uuid.UUID, qType assessments.QuestionType, optCmd CreateQuestionOptionCommand, position int) *assessments.QuestionOption {
	opt := &assessments.QuestionOption{
		ID:          uuid.New(),
		QuestionID:  questionID,
		Body:        optCmd.Body,
		ContentType: normalizeContentType(optCmd.ContentType, optCmd.Body, optCmd.ImageURL),
		ImageURL:    strings.TrimSpace(optCmd.ImageURL),
		IsCorrect:   optCmd.IsCorrect,
		Position:    position,
	}
	if qType == assessments.QuestionTypeShort {
		opt.IsCorrect = true // short-answer "options" are the accepted answers
	}
	return opt
}

// questionOptionsFor keeps only the options a question type uses.
func questionOptionsFor(qCmd CreateQuestionCommand) []CreateQuestionOptionCommand {
	switch assessments.QuestionType(qCmd.Type) {
	case assessments.QuestionTypeNumeric, assessments.QuestionTypeLong:
		return nil
	case assessments.QuestionTypeShort:
		accepted := make([]CreateQuestionOptionCommand, 0, len(qCmd.Options))
		for _, o := range qCmd.Options {
			if o.IsCorrect && strings.TrimSpace(o.Body) != "" {
				accepted = append(accepted, o)
			}
		}
		return accepted
	default:
		return qCmd.Options
	}
}

func (s *service) insertQuestion(ctx context.Context, quizID uuid.UUID, qCmd CreateQuestionCommand, position int, now time.Time) (*assessments.Question, []*assessments.QuestionOption, error) {
	if qCmd.Position > 0 {
		position = qCmd.Position
	}
	question := &assessments.Question{ID: uuid.New(), QuizID: quizID, CreatedAt: now}
	applyQuestionFields(question, qCmd, position, now)
	if err := s.questionRepo.Create(ctx, question); err != nil {
		return nil, nil, err
	}
	options := make([]*assessments.QuestionOption, 0, len(qCmd.Options))
	for i, optCmd := range questionOptionsFor(qCmd) {
		options = append(options, optionFromCommand(question.ID, question.Type, optCmd, i+1))
	}
	if len(options) > 0 {
		if err := s.optionRepo.CreateBatch(ctx, options); err != nil {
			return nil, nil, err
		}
	}
	return question, options, nil
}

// updateQuestionInPlace keeps the question ID and reuses option IDs that are
// still present, so autosaved drafts of in-progress attempts stay valid.
func (s *service) updateQuestionInPlace(ctx context.Context, question *assessments.Question, qCmd CreateQuestionCommand, position int, now time.Time) ([]*assessments.QuestionOption, error) {
	applyQuestionFields(question, qCmd, position, now)
	if err := s.questionRepo.Update(ctx, question); err != nil {
		return nil, err
	}
	existing, err := s.optionRepo.FindByQuestionID(ctx, question.ID)
	if err != nil {
		return nil, err
	}
	existingByID := make(map[uuid.UUID]*assessments.QuestionOption, len(existing))
	for _, o := range existing {
		existingByID[o.ID] = o
	}
	kept := map[uuid.UUID]bool{}
	result := make([]*assessments.QuestionOption, 0, len(qCmd.Options))
	for i, optCmd := range questionOptionsFor(qCmd) {
		if optCmd.ID != nil {
			if current, ok := existingByID[*optCmd.ID]; ok && !kept[current.ID] {
				updated := optionFromCommand(question.ID, question.Type, optCmd, i+1)
				updated.ID = current.ID
				if err := s.optionRepo.Update(ctx, updated); err != nil {
					return nil, err
				}
				kept[current.ID] = true
				result = append(result, updated)
				continue
			}
		}
		created := optionFromCommand(question.ID, question.Type, optCmd, i+1)
		if err := s.optionRepo.Create(ctx, created); err != nil {
			return nil, err
		}
		result = append(result, created)
	}
	for _, o := range existing {
		if !kept[o.ID] {
			if err := s.optionRepo.Delete(ctx, o.ID); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

// UpdateQuiz updates settings and reconciles questions in one transaction.
// Historical attempts are unaffected: each submitted attempt keeps a snapshot
// of the questions it was graded against.
func (s *service) UpdateQuiz(ctx context.Context, cmd UpdateQuizCommand) (*QuizTeacherResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, cmd.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, cmd.TeacherID); err != nil {
		return nil, err
	}
	createCmd := CreateQuizCommand{
		CourseID:                   quiz.CourseID,
		TeacherID:                  cmd.TeacherID,
		LessonID:                   cmd.LessonID,
		Title:                      cmd.Title,
		TimeLimitSeconds:           cmd.TimeLimitSeconds,
		MaxAttempts:                cmd.MaxAttempts,
		PassingScorePercent:        cmd.PassingScorePercent,
		ShuffleQuestions:           cmd.ShuffleQuestions,
		ShowAnswersAfterSubmission: cmd.ShowAnswersAfterSubmission,
		IsFree:                     cmd.IsFree,
		IsPublished:                cmd.IsPublished,
		Description:                cmd.Description,
		Instructions:               cmd.Instructions,
		Mode:                       cmd.Mode,
		ShuffleOptions:             cmd.ShuffleOptions,
		PartialCredit:              cmd.PartialCredit,
		AvailableFrom:              cmd.AvailableFrom,
		AvailableUntil:             cmd.AvailableUntil,
		CalculatorPolicy:           cmd.CalculatorPolicy,
		ReferenceMaterial:          cmd.ReferenceMaterial,
		Questions:                  cmd.Questions,
	}
	if err := validateCreateQuiz(createCmd); err != nil {
		return nil, err
	}

	now := s.clock().UTC()
	applyQuizSettings(quiz, createCmd, now)

	err = s.inTx(ctx, func(ctx context.Context) error {
		if err := s.quizRepo.Update(ctx, quiz); err != nil {
			return err
		}
		existing, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
		if err != nil {
			return err
		}
		existingByID := make(map[uuid.UUID]*assessments.Question, len(existing))
		for _, q := range existing {
			existingByID[q.ID] = q
		}
		kept := map[uuid.UUID]bool{}
		for i, qCmd := range cmd.Questions {
			position := i + 1
			if qCmd.ID != nil {
				if current, ok := existingByID[*qCmd.ID]; ok && !kept[current.ID] {
					if _, err := s.updateQuestionInPlace(ctx, current, qCmd, position, now); err != nil {
						return err
					}
					kept[current.ID] = true
					continue
				}
			}
			qCmd.Position = position
			if _, _, err := s.insertQuestion(ctx, quiz.ID, qCmd, position, now); err != nil {
				return err
			}
		}
		for _, q := range existing {
			if !kept[q.ID] {
				if err := s.questionRepo.Delete(ctx, q.ID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.toQuizTeacherResponse(ctx, quiz)
}

// GetTeacherQuizzes returns all quizzes of a course with answer keys.
func (s *service) GetTeacherQuizzes(ctx context.Context, courseID, teacherID uuid.UUID) ([]QuizTeacherResponse, error) {
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(teacherID) {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to view quizzes for this course")
	}
	quizzes, err := s.quizRepo.FindByCourseID(ctx, courseID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(quizzes))
	for i, q := range quizzes {
		ids[i] = q.ID
	}
	counts, err := s.attemptRepo.CountByQuizIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	responses := make([]QuizTeacherResponse, 0, len(quizzes))
	for _, quiz := range quizzes {
		resp, err := s.toQuizTeacherResponse(ctx, quiz)
		if err != nil {
			return nil, err
		}
		c := counts[quiz.ID]
		resp.AttemptCount, resp.SubmittedCount, resp.InProgress, resp.PendingGrading = c.Attempts, c.Submitted, c.InProgress, c.PendingGrading
		responses = append(responses, *resp)
	}
	return responses, nil
}

func (s *service) GetTeacherQuiz(ctx context.Context, quizID, teacherID uuid.UUID) (*QuizTeacherResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, teacherID); err != nil {
		return nil, err
	}
	resp, err := s.toQuizTeacherResponse(ctx, quiz)
	if err != nil {
		return nil, err
	}
	counts, err := s.attemptRepo.CountByQuizIDs(ctx, []uuid.UUID{quiz.ID})
	if err == nil {
		c := counts[quiz.ID]
		resp.AttemptCount, resp.SubmittedCount, resp.InProgress, resp.PendingGrading = c.Attempts, c.Submitted, c.InProgress, c.PendingGrading
	}
	return resp, nil
}

func (s *service) DeleteQuiz(ctx context.Context, quizID, teacherID uuid.UUID) error {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, teacherID); err != nil {
		return err
	}
	// Deleting would cascade to student attempts and their grades.
	counts, err := s.attemptRepo.CountByQuizIDs(ctx, []uuid.UUID{quiz.ID})
	if err != nil {
		return err
	}
	if counts[quiz.ID].Attempts > 0 {
		return apperrors.NewConflictError("QUIZ_HAS_ATTEMPTS", "this quiz has student attempts; unpublish it instead of deleting it")
	}
	return s.quizRepo.Delete(ctx, quizID)
}

// DuplicateQuiz copies a quiz and its questions as a new unpublished draft.
func (s *service) DuplicateQuiz(ctx context.Context, quizID, teacherID uuid.UUID) (*QuizTeacherResponse, error) {
	source, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, source.CourseID, teacherID); err != nil {
		return nil, err
	}
	questions, err := s.questionRepo.FindByQuizID(ctx, source.ID)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(questions))
	for i, q := range questions {
		ids[i] = q.ID
	}
	optionsMap, err := s.optionRepo.FindByQuestionIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	now := s.clock().UTC()
	copyQuiz := *source
	copyQuiz.ID = uuid.New()
	copyQuiz.Title = truncateTitle("Copy of " + source.Title)
	copyQuiz.IsPublished = false
	copyQuiz.CreatedAt = now
	copyQuiz.UpdatedAt = now

	err = s.inTx(ctx, func(ctx context.Context) error {
		if err := s.quizRepo.Create(ctx, &copyQuiz); err != nil {
			return err
		}
		for _, q := range questions {
			nq := *q
			nq.ID = uuid.New()
			nq.QuizID = copyQuiz.ID
			nq.CreatedAt, nq.UpdatedAt = now, now
			if err := s.questionRepo.Create(ctx, &nq); err != nil {
				return err
			}
			opts := make([]*assessments.QuestionOption, 0, len(optionsMap[q.ID]))
			for _, o := range optionsMap[q.ID] {
				no := *o
				no.ID = uuid.New()
				no.QuestionID = nq.ID
				opts = append(opts, &no)
			}
			if len(opts) > 0 {
				if err := s.optionRepo.CreateBatch(ctx, opts); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.toQuizTeacherResponse(ctx, &copyQuiz)
}

func truncateTitle(t string) string {
	if len(t) <= maxTitleLength {
		return t
	}
	return t[:maxTitleLength]
}

func (s *service) CreateQuestion(ctx context.Context, cmd CreateStandaloneQuestionCommand) (*QuestionTeacherResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, cmd.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, cmd.TeacherID); err != nil {
		return nil, err
	}
	if err := validateQuestion(cmd.Question); err != nil {
		return nil, err
	}
	existing, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	if len(existing) >= maxQuestionsPerQuiz {
		return nil, apperrors.NewSimpleValidationError("TOO_MANY_QUESTIONS", "a quiz can have at most 300 questions")
	}
	position := cmd.Question.Position
	if position <= 0 {
		position = len(existing) + 1
	}
	var question *assessments.Question
	var options []*assessments.QuestionOption
	err = s.inTx(ctx, func(ctx context.Context) error {
		var err error
		question, options, err = s.insertQuestion(ctx, quiz.ID, cmd.Question, position, s.clock().UTC())
		return err
	})
	if err != nil {
		return nil, err
	}
	response := s.toQuestionTeacherResponse(question, options)
	return &response, nil
}

func (s *service) UpdateQuestion(ctx context.Context, cmd UpdateQuestionCommand) (*QuestionTeacherResponse, error) {
	question, err := s.questionRepo.FindByID(ctx, cmd.QuestionID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUESTION_NOT_FOUND", "question not found")
	}
	quiz, err := s.quizRepo.FindByID(ctx, question.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, cmd.TeacherID); err != nil {
		return nil, err
	}
	if err := validateQuestion(cmd.Question); err != nil {
		return nil, err
	}
	position := cmd.Question.Position
	if position <= 0 {
		position = question.Position
	}
	var options []*assessments.QuestionOption
	err = s.inTx(ctx, func(ctx context.Context) error {
		var err error
		options, err = s.updateQuestionInPlace(ctx, question, cmd.Question, position, s.clock().UTC())
		return err
	})
	if err != nil {
		return nil, err
	}
	response := s.toQuestionTeacherResponse(question, options)
	return &response, nil
}

func (s *service) DeleteQuestion(ctx context.Context, questionID, teacherID uuid.UUID) error {
	question, err := s.questionRepo.FindByID(ctx, questionID)
	if err != nil {
		return apperrors.NewNotFoundError("QUESTION_NOT_FOUND", "question not found")
	}
	quiz, err := s.quizRepo.FindByID(ctx, question.QuizID)
	if err != nil {
		return apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, teacherID); err != nil {
		return err
	}
	return s.questionRepo.Delete(ctx, questionID)
}

func (s *service) ensureTeacherOwnsCourse(ctx context.Context, courseID, teacherID uuid.UUID) error {
	course, err := s.courseRepo.FindByID(ctx, courseID)
	if err != nil {
		return apperrors.NewNotFoundError("COURSE_NOT_FOUND", "course not found")
	}
	if !course.IsOwnedBy(teacherID) {
		return apperrors.NewForbiddenError("FORBIDDEN", "not authorized to manage this quiz")
	}
	return nil
}

func (s *service) loadQuestions(ctx context.Context, quizID uuid.UUID) ([]*assessments.Question, map[uuid.UUID][]*assessments.QuestionOption, error) {
	questions, err := s.questionRepo.FindByQuizID(ctx, quizID)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]uuid.UUID, len(questions))
	for i, q := range questions {
		ids[i] = q.ID
	}
	optionsMap, err := s.optionRepo.FindByQuestionIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return questions, optionsMap, nil
}

func (s *service) toQuizTeacherResponse(ctx context.Context, quiz *assessments.Quiz) (*QuizTeacherResponse, error) {
	questions, optionsMap, err := s.loadQuestions(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	resp := &QuizTeacherResponse{QuizResponse: *s.toQuizResponse(quiz)}
	resp.Questions = make([]QuestionTeacherResponse, 0, len(questions))
	for _, question := range questions {
		resp.Questions = append(resp.Questions, s.toQuestionTeacherResponse(question, optionsMap[question.ID]))
		resp.TotalMarks += question.Marks
	}
	resp.QuestionCount = len(questions)
	resp.TotalMarks = round2(resp.TotalMarks)
	return resp, nil
}

func (s *service) quizLifecycle(quiz *assessments.Quiz) string {
	if !quiz.IsPublished {
		return "draft"
	}
	switch quiz.AvailabilityAt(s.clock()) {
	case "not_open":
		return "scheduled"
	case "closed":
		return "closed"
	default:
		return "open"
	}
}

func (s *service) toQuizResponse(quiz *assessments.Quiz) *QuizResponse {
	mode := string(quiz.Mode)
	if mode == "" {
		mode = string(assessments.QuizModeExam)
	}
	return &QuizResponse{
		ID:                         quiz.ID,
		CourseID:                   quiz.CourseID,
		LessonID:                   quiz.LessonID,
		Title:                      quiz.Title,
		TimeLimitSeconds:           quiz.TimeLimitSeconds,
		MaxAttempts:                quiz.MaxAttempts,
		PassingScorePercent:        quiz.PassingScorePercent,
		ShuffleQuestions:           quiz.ShuffleQuestions,
		ShowAnswersAfterSubmission: quiz.ShowAnswersAfterSubmission,
		IsFree:                     quiz.IsFree,
		IsPublished:                quiz.IsPublished,
		CreatedAt:                  quiz.CreatedAt,
		UpdatedAt:                  quiz.UpdatedAt,
		Description:                quiz.Description,
		Instructions:               quiz.Instructions,
		Mode:                       mode,
		ShuffleOptions:             quiz.ShuffleOptions,
		PartialCredit:              quiz.PartialCredit,
		AvailableFrom:              quiz.AvailableFrom,
		AvailableUntil:             quiz.AvailableUntil,
		CalculatorPolicy:           normalizeCalculatorPolicy(quiz.CalculatorPolicy),
		ReferenceMaterial:          quiz.ReferenceMaterial,
		Lifecycle:                  s.quizLifecycle(quiz),
	}
}

func (s *service) toQuestionTeacherResponse(question *assessments.Question, options []*assessments.QuestionOption) QuestionTeacherResponse {
	correctIDs := make([]uuid.UUID, 0)
	optionResponses := make([]QuestionOptionTeacherResponse, 0, len(options))
	for _, opt := range options {
		if opt.IsCorrect {
			correctIDs = append(correctIDs, opt.ID)
		}
		optionResponses = append(optionResponses, QuestionOptionTeacherResponse{
			ID:          opt.ID,
			Body:        opt.Body,
			ContentType: opt.ContentType,
			ImageURL:    opt.ImageURL,
			IsCorrect:   opt.IsCorrect,
			Position:    opt.Position,
		})
	}
	st := question.Settings
	return QuestionTeacherResponse{
		ID:          question.ID,
		QuizID:      question.QuizID,
		Body:        question.Body,
		Type:        string(question.Type),
		ContentType: question.ContentType,
		ImageURL:    question.ImageURL,
		Marks:       question.Marks,
		IsRequired:  question.IsRequired,
		Position:    question.Position,
		Explanation: question.Explanation,
		Hint:        question.Hint,
		Settings: QuestionSettingsResponse{
			NumericAnswer: st.NumericAnswer, Tolerance: st.Tolerance, ToleranceType: st.ToleranceType,
			Unit: st.Unit, MaxWords: st.MaxWords, CaseSensitive: st.CaseSensitive,
		},
		CorrectOptionIDs: correctIDs,
		Options:          optionResponses,
	}
}
