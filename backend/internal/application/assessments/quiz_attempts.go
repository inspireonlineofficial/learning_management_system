package assessments

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"math/rand"
	"net/http"
	"sort"
	"strings"

	"lms-backend/internal/domain/assessments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// quizSubmitGrace tolerates network latency between the client timer
// reaching zero and the submit request arriving (see draft_answers.go).

// Draft payload limits (per autosave).
const (
	maxDraftOptionsPerAnswer = 50
)

func (s *service) ensureStudentCanAccessQuiz(ctx context.Context, studentID uuid.UUID, quiz *assessments.Quiz) error {
	if quiz == nil {
		return apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	// Unpublished (draft) quizzes are never available to students.
	if !quiz.IsPublished {
		return apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if quiz.IsFree {
		return nil
	}
	return s.ensureStudentEnrolledInCourse(ctx, studentID, quiz.CourseID)
}

func firstAttempt(attempts []*assessments.QuizAttempt) *assessments.QuizAttempt {
	if len(attempts) == 0 {
		return nil
	}
	return attempts[0]
}

func gradingStatusString(a *assessments.QuizAttempt) string {
	if a.GradingStatus == "" {
		return string(assessments.GradingStatusAuto)
	}
	return string(a.GradingStatus)
}

func toStudentAttemptResult(attempt *assessments.QuizAttempt) *StudentAttemptResult {
	if attempt == nil {
		return nil
	}
	return &StudentAttemptResult{
		ID:               attempt.ID,
		QuizID:           attempt.QuizID,
		Status:           string(attempt.Status),
		StartedAt:        attempt.StartedAt,
		SubmittedAt:      attempt.SubmittedAt,
		ScorePercent:     attempt.ScorePercent,
		Passed:           attempt.Passed,
		TimeTakenSeconds: attempt.TimeTakenSeconds,
		PointsAwarded:    attempt.PointsAwarded,
		GradingStatus:    gradingStatusString(attempt),
		ScorePoints:      attempt.ScorePoints,
		MaxPoints:        attempt.MaxPoints,
	}
}

func decodeDraftAnswers(raw []byte) map[string]interface{} {
	if len(raw) == 0 {
		return nil
	}
	var answers map[string]interface{}
	if err := json.Unmarshal(raw, &answers); err != nil {
		return nil
	}
	return answers
}

// ---------------------------------------------------------------------------
// Catalog

func (s *service) studentSummary(ctx context.Context, quiz *assessments.Quiz, studentID uuid.UUID, courseTitles map[uuid.UUID]string) (StudentQuizSummaryResponse, error) {
	attempts, err := s.attemptRepo.FindByQuizAndStudent(ctx, quiz.ID, studentID)
	if err != nil {
		return StudentQuizSummaryResponse{}, err
	}
	questions, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
	if err != nil {
		return StudentQuizSummaryResponse{}, err
	}
	totalMarks := 0.0
	for _, q := range questions {
		totalMarks += q.Marks
	}
	title, ok := courseTitles[quiz.CourseID]
	if !ok && s.courseRepo != nil {
		if course, err := s.courseRepo.FindByID(ctx, quiz.CourseID); err == nil && course != nil {
			title = course.Title
		}
		courseTitles[quiz.CourseID] = title
	}

	summary := StudentQuizSummaryResponse{
		ID:                  quiz.ID,
		CourseID:            quiz.CourseID,
		LessonID:            quiz.LessonID,
		Title:               quiz.Title,
		TimeLimitSeconds:    quiz.TimeLimitSeconds,
		MaxAttempts:         quiz.MaxAttempts,
		PassingScorePercent: quiz.PassingScorePercent,
		IsFree:              quiz.IsFree,
		AttemptsUsed:        len(attempts),
		LatestAttempt:       toStudentAttemptResult(firstAttempt(attempts)),
		CourseTitle:         title,
		Description:         quiz.Description,
		Instructions:        quiz.Instructions,
		Mode:                string(quizModeOf(quiz)),
		QuestionCount:       len(questions),
		TotalMarks:          round2(totalMarks),
		AvailableFrom:       quiz.AvailableFrom,
		AvailableUntil:      quiz.AvailableUntil,
		Availability:        quiz.AvailabilityAt(s.clock()),
		CalculatorPolicy:    normalizeCalculatorPolicy(quiz.CalculatorPolicy),
		ReferenceMaterial:   quiz.ReferenceMaterial,
	}
	if quiz.MaxAttempts > 0 {
		remaining := quiz.MaxAttempts - len(attempts)
		if remaining < 0 {
			remaining = 0
		}
		summary.RemainingAttempts = &remaining
	}
	for _, a := range attempts {
		if a.IsInProgress() && !a.ExpiredAt(quiz, s.clock()) {
			id := a.ID
			summary.InProgressAttemptID = &id
			if summary.RemainingAttempts != nil {
				// Resuming does not use another attempt.
				r := *summary.RemainingAttempts + 1
				summary.RemainingAttempts = &r
			}
		}
		if a.ScorePercent != nil && a.GradingStatus != assessments.GradingStatusPending {
			if summary.BestScorePercent == nil || *a.ScorePercent > *summary.BestScorePercent {
				v := *a.ScorePercent
				summary.BestScorePercent = &v
			}
		}
	}
	return summary, nil
}

func quizModeOf(quiz *assessments.Quiz) assessments.QuizMode {
	if quiz.Mode == assessments.QuizModePractice {
		return assessments.QuizModePractice
	}
	return assessments.QuizModeExam
}

// ListStudentQuizzes returns published quizzes across the student's courses.
func (s *service) ListStudentQuizzes(ctx context.Context, studentID uuid.UUID) ([]StudentQuizSummaryResponse, error) {
	courseIDs, err := s.listStudentCourseIDs(ctx, studentID)
	if err != nil {
		return nil, err
	}
	titles := map[uuid.UUID]string{}
	responses := make([]StudentQuizSummaryResponse, 0)
	for _, courseID := range courseIDs {
		quizzes, err := s.quizRepo.FindByCourseID(ctx, courseID)
		if err != nil {
			return nil, err
		}
		for _, quiz := range quizzes {
			// Drafts were previously listed to students (titles leaked).
			if !quiz.IsPublished {
				continue
			}
			summary, err := s.studentSummary(ctx, quiz, studentID, titles)
			if err != nil {
				return nil, err
			}
			responses = append(responses, summary)
		}
	}
	return responses, nil
}

// GetStudentQuizDetail returns a student-visible quiz summary without answer keys.
func (s *service) GetStudentQuizDetail(ctx context.Context, quizID, studentID uuid.UUID) (*StudentQuizDetailResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, studentID, quiz); err != nil {
		return nil, err
	}
	summary, err := s.studentSummary(ctx, quiz, studentID, map[uuid.UUID]string{})
	if err != nil {
		return nil, err
	}
	return &StudentQuizDetailResponse{
		StudentQuizSummaryResponse: summary,
		ShowAnswersAfterSubmission: quiz.ShowAnswersAfterSubmission,
	}, nil
}

// ---------------------------------------------------------------------------
// Attempt lifecycle

// StartAttempt resumes the student's in-progress attempt or starts a new one.
// Starts are serialised per student and quiz, so double clicks or two tabs
// cannot create two attempts or exceed the attempt limit.
func (s *service) StartAttempt(ctx context.Context, cmd StartAttemptCommand) (*QuizAttemptResponse, error) {
	quiz, err := s.quizRepo.FindByID(ctx, cmd.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, cmd.StudentID, quiz); err != nil {
		return nil, err
	}

	var attempt *assessments.QuizAttempt
	var expired *assessments.QuizAttempt
	err = s.inTx(ctx, func(ctx context.Context) error {
		if locker, ok := s.attemptRepo.(AttemptLocker); ok {
			if err := locker.LockStudentQuiz(ctx, quiz.ID, cmd.StudentID); err != nil {
				return err
			}
		}
		existing, err := s.attemptRepo.FindByQuizAndStudent(ctx, cmd.QuizID, cmd.StudentID)
		if err != nil {
			return err
		}
		now := s.clock()
		if latest := firstAttempt(existing); latest != nil && latest.IsInProgress() {
			if !latest.ExpiredAt(quiz, now) {
				attempt = latest
				return nil
			}
			expired = latest
		}
		if availability := quiz.AvailabilityAt(now); availability != "open" {
			if availability == "not_open" {
				return apperrors.NewForbiddenError("ASSESSMENT_NOT_OPEN", "this assessment is not open yet")
			}
			return apperrors.NewForbiddenError("ASSESSMENT_CLOSED", "this assessment is closed")
		}
		used := len(existing)
		if !quiz.CanAttempt(used) {
			return apperrors.NewValidationErrorWithDetails("MAX_ATTEMPTS_REACHED", "maximum attempts reached for this quiz", nil)
		}
		attempt = &assessments.QuizAttempt{
			ID:            uuid.New(),
			QuizID:        cmd.QuizID,
			StudentID:     cmd.StudentID,
			StartedAt:     now,
			Status:        assessments.QuizAttemptStatusInProgress,
			GradingStatus: assessments.GradingStatusAuto,
			ShuffleSeed:   rand.Int63(),
		}
		return s.attemptRepo.Create(ctx, attempt)
	})
	if expired != nil {
		// Finalise the expired attempt from its saved draft (outside the
		// start transaction; Finalize is idempotent).
		if _, ferr := s.finalizeAttempt(ctx, quiz, expired, answersFromDraft(expired.DraftAnswers), assessments.QuizAttemptStatusAutoSubmitted); ferr != nil {
			return nil, ferr
		}
	}
	if err != nil {
		return nil, err
	}
	return s.buildQuizAttemptResponse(ctx, quiz, attempt, true)
}

// shuffleSeed returns the attempt's seed; attempts created before the seed
// existed derive a stable one from their ID.
func shuffleSeed(attempt *assessments.QuizAttempt) int64 {
	if attempt.ShuffleSeed != 0 {
		return attempt.ShuffleSeed
	}
	return int64(binary.BigEndian.Uint64(attempt.ID[:8]))
}

// orderForAttempt returns the questions (and options) in the order this
// attempt shows them. The order is a pure function of the attempt's seed, so
// it is identical after a refresh or on another device.
func orderForAttempt(quiz *assessments.Quiz, attempt *assessments.QuizAttempt, questions []*assessments.Question, optionsMap map[uuid.UUID][]*assessments.QuestionOption) ([]*assessments.Question, map[uuid.UUID][]*assessments.QuestionOption) {
	ordered := make([]*assessments.Question, len(questions))
	copy(ordered, questions)
	rng := rand.New(rand.NewSource(shuffleSeed(attempt)))
	if quiz.ShuffleQuestions {
		rng.Shuffle(len(ordered), func(i, j int) { ordered[i], ordered[j] = ordered[j], ordered[i] })
	}
	if !quiz.ShuffleOptions {
		return ordered, optionsMap
	}
	shuffled := make(map[uuid.UUID][]*assessments.QuestionOption, len(optionsMap))
	// Iterate in a fixed (position) order so the RNG sequence is stable.
	byPosition := make([]*assessments.Question, len(questions))
	copy(byPosition, questions)
	sort.SliceStable(byPosition, func(i, j int) bool { return byPosition[i].Position < byPosition[j].Position })
	for _, q := range byPosition {
		opts := make([]*assessments.QuestionOption, len(optionsMap[q.ID]))
		copy(opts, optionsMap[q.ID])
		if q.Type != assessments.QuestionTypeTrueFalse {
			rng.Shuffle(len(opts), func(i, j int) { opts[i], opts[j] = opts[j], opts[i] })
		}
		shuffled[q.ID] = opts
	}
	return ordered, shuffled
}

func (s *service) toQuestionStudentResponse(quiz *assessments.Quiz, question *assessments.Question, options []*assessments.QuestionOption) QuestionStudentResponse {
	optionResponses := make([]QuestionOptionStudentResponse, 0, len(options))
	if question.Type.UsesOptions() {
		for _, opt := range options {
			optionResponses = append(optionResponses, QuestionOptionStudentResponse{
				ID:          opt.ID,
				Body:        opt.Body,
				ContentType: opt.ContentType,
				ImageURL:    opt.ImageURL,
				Position:    opt.Position,
			})
		}
	}
	resp := QuestionStudentResponse{
		ID:          question.ID,
		QuizID:      question.QuizID,
		Body:        question.Body,
		Type:        string(question.Type),
		ContentType: question.ContentType,
		ImageURL:    question.ImageURL,
		Marks:       question.Marks,
		IsRequired:  question.IsRequired,
		Position:    question.Position,
		Options:     optionResponses,
		Settings: StudentQuestionSettings{
			Unit:     question.Settings.Unit,
			MaxWords: question.Settings.MaxWords,
		},
	}
	if quiz.IsPractice() {
		resp.Hint = question.Hint
	}
	return resp
}

func (s *service) buildQuizAttemptResponse(ctx context.Context, quiz *assessments.Quiz, attempt *assessments.QuizAttempt, withQuestions bool) (*QuizAttemptResponse, error) {
	resp := &QuizAttemptResponse{
		ID:                attempt.ID,
		QuizID:            attempt.QuizID,
		StartedAt:         attempt.StartedAt,
		ExpiresAt:         attempt.Deadline(quiz),
		Questions:         []QuestionStudentResponse{},
		Answers:           decodeDraftAnswers(attempt.DraftAnswers),
		ServerNow:         s.clock().UTC(),
		Status:            string(attempt.Status),
		DraftRevision:     attempt.DraftRevision,
		QuizTitle:         quiz.Title,
		Mode:              string(quizModeOf(quiz)),
		Instructions:      quiz.Instructions,
		TimeLimitSeconds:  quiz.TimeLimitSeconds,
		CalculatorPolicy:  normalizeCalculatorPolicy(quiz.CalculatorPolicy),
		ReferenceMaterial: quiz.ReferenceMaterial,
	}
	if !withQuestions {
		return resp, nil
	}
	questions, optionsMap, err := s.loadQuestions(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	ordered, orderedOptions := orderForAttempt(quiz, attempt, questions, optionsMap)
	for _, question := range ordered {
		resp.Questions = append(resp.Questions, s.toQuestionStudentResponse(quiz, question, orderedOptions[question.ID]))
		resp.TotalMarks += question.Marks
	}
	resp.TotalMarks = round2(resp.TotalMarks)
	return resp, nil
}

// sanitizeDraft keeps only answers to this quiz's questions in the shapes the
// grader understands, and caps their size.
func sanitizeDraft(answers map[string]interface{}, valid map[string]bool) map[string]interface{} {
	clean := make(map[string]interface{}, len(answers))
	for key, value := range answers {
		if !valid[key] {
			continue
		}
		switch v := value.(type) {
		case string:
			clean[key] = truncate(v)
		case float64:
			clean[key] = v
		case []interface{}:
			ids := make([]interface{}, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok && len(ids) < maxDraftOptionsPerAnswer {
					if _, err := uuid.Parse(str); err == nil {
						ids = append(ids, str)
					}
				}
			}
			clean[key] = ids
		case map[string]interface{}:
			obj := map[string]interface{}{}
			if text, ok := v["text"].(string); ok {
				obj["text"] = truncate(text)
			}
			if img, ok := v["image_url"].(string); ok && validMediaURL(img) {
				obj["image_url"] = strings.TrimSpace(img)
			}
			clean[key] = obj
		}
	}
	return clean
}

// SaveAttemptAnswers autosaves the draft. With a revision, older saves that
// arrive late are ignored (saved=false) instead of overwriting newer answers.
func (s *service) SaveAttemptAnswers(ctx context.Context, cmd SaveAttemptAnswersCommand) (*QuizAttemptResponse, error) {
	attempt, err := s.attemptRepo.FindByID(ctx, cmd.AttemptID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if attempt.StudentID != cmd.StudentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to update this attempt")
	}
	if !attempt.IsInProgress() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_ALREADY_SUBMITTED", "attempt has already been submitted", nil)
	}
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, cmd.StudentID, quiz); err != nil {
		return nil, err
	}
	if attempt.ExpiredAt(quiz, s.clock()) {
		if _, err := s.finalizeAttempt(ctx, quiz, attempt, answersFromDraft(attempt.DraftAnswers), assessments.QuizAttemptStatusAutoSubmitted); err != nil {
			return nil, err
		}
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_EXPIRED", "quiz attempt has expired", nil)
	}
	questions, err := s.questionRepo.FindByQuizID(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	valid := make(map[string]bool, len(questions))
	for _, q := range questions {
		valid[q.ID.String()] = true
	}
	clean := sanitizeDraft(cmd.Answers, valid)
	raw, err := json.Marshal(clean)
	if err != nil {
		return nil, apperrors.NewSimpleValidationError("INVALID_ANSWERS", "answers must be valid JSON")
	}

	saved := true
	if cmd.Revision > 0 {
		saved, err = s.attemptRepo.SaveDraftRevision(ctx, attempt.ID, raw, cmd.Revision)
		if err != nil {
			return nil, err
		}
		if saved {
			attempt.DraftAnswers = raw
			attempt.DraftRevision = cmd.Revision
		} else if current, err := s.attemptRepo.FindByID(ctx, attempt.ID); err == nil {
			if !current.IsInProgress() {
				return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_ALREADY_SUBMITTED", "attempt has already been submitted", nil)
			}
			attempt = current
		}
	} else {
		if err := s.attemptRepo.SaveDraftAnswers(ctx, attempt.ID, raw); err != nil {
			return nil, err
		}
		attempt.DraftAnswers = raw
	}
	resp, err := s.buildQuizAttemptResponse(ctx, quiz, attempt, false)
	if err != nil {
		return nil, err
	}
	resp.Saved = &saved
	return resp, nil
}

// SubmitAttempt grades and records a submission. It is idempotent: replays
// and double clicks return the already-recorded result.
func (s *service) SubmitAttempt(ctx context.Context, cmd SubmitAttemptCommand) (*SubmitAttemptResponse, error) {
	attempt, err := s.attemptRepo.FindByID(ctx, cmd.AttemptID)
	if err != nil {
		return nil, err
	}
	if attempt.StudentID != cmd.StudentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to submit this attempt")
	}
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if !attempt.IsInProgress() {
		return s.submitResponse(quiz, attempt), nil
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, cmd.StudentID, quiz); err != nil {
		return nil, err
	}

	now := s.clock()
	status := assessments.QuizAttemptStatusSubmitted
	var answers map[uuid.UUID]AnswerInput
	if len(cmd.Answers) > 0 {
		answers = answersFromCommands(cmd.Answers)
	} else {
		answers = answersFromDraft(attempt.DraftAnswers)
	}
	if attempt.ExpiredAt(quiz, now) {
		status = assessments.QuizAttemptStatusAutoSubmitted
		// Answers sent after the deadline (plus a short grace period for
		// network latency) are not accepted: the attempt is scored from the
		// last draft saved while the timer was running.
		if deadline := attempt.Deadline(quiz); deadline != nil && now.After(deadline.Add(quizSubmitGrace)) {
			answers = answersFromDraft(attempt.DraftAnswers)
		}
	}
	return s.finalizeAttempt(ctx, quiz, attempt, answers, status)
}

// AutoSubmitAttempt finalises an expired attempt from its saved draft.
func (s *service) AutoSubmitAttempt(ctx context.Context, attemptID uuid.UUID) (*SubmitAttemptResponse, error) {
	attempt, err := s.attemptRepo.FindByID(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if !attempt.IsInProgress() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_ALREADY_SUBMITTED", "attempt has already been submitted", nil)
	}
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, err
	}
	if !attempt.ExpiredAt(quiz, s.clock()) {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_NOT_EXPIRED", "attempt has not expired", nil)
	}
	return s.finalizeAttempt(ctx, quiz, attempt, answersFromDraft(attempt.DraftAnswers), assessments.QuizAttemptStatusAutoSubmitted)
}

// AutoSubmitExpiredAttempts finalises abandoned attempts whose deadline has
// passed, so they are graded and visible to teachers even if the student
// never returns.
func (s *service) AutoSubmitExpiredAttempts(ctx context.Context) (int, error) {
	attempts, err := s.attemptRepo.FindInProgressAttempts(ctx)
	if err != nil {
		return 0, err
	}
	quizzes := map[uuid.UUID]*assessments.Quiz{}
	now := s.clock()
	done := 0
	for _, attempt := range attempts {
		quiz, ok := quizzes[attempt.QuizID]
		if !ok {
			quiz, err = s.quizRepo.FindByID(ctx, attempt.QuizID)
			if err != nil {
				continue
			}
			quizzes[attempt.QuizID] = quiz
		}
		if !attempt.ExpiredAt(quiz, now) {
			continue
		}
		if _, err := s.finalizeAttempt(ctx, quiz, attempt, answersFromDraft(attempt.DraftAnswers), assessments.QuizAttemptStatusAutoSubmitted); err == nil {
			done++
		}
	}
	return done, nil
}

// gradeAttempt grades every question in position order.
func (s *service) gradeAttempt(ctx context.Context, quiz *assessments.Quiz, answers map[uuid.UUID]AnswerInput) ([]QuestionOutcome, error) {
	questions, optionsMap, err := s.loadQuestions(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	outcomes := make([]QuestionOutcome, 0, len(questions))
	for _, q := range questions {
		outcomes = append(outcomes, gradeQuestion(q, optionsMap[q.ID], answers[q.ID], quiz.PartialCredit))
	}
	return outcomes, nil
}

// finalizeAttempt grades and records the attempt exactly once. When another
// request already finalised it, the stored result is returned unchanged.
func (s *service) finalizeAttempt(ctx context.Context, quiz *assessments.Quiz, attempt *assessments.QuizAttempt, answers map[uuid.UUID]AnswerInput, status assessments.QuizAttemptStatus) (*SubmitAttemptResponse, error) {
	outcomes, err := s.gradeAttempt(ctx, quiz, answers)
	if err != nil {
		return nil, err
	}
	awarded, max, pending := gradeTotals(outcomes)
	percent := scorePercent(awarded, max)
	now := s.clock()
	timeTaken := int(now.Sub(attempt.StartedAt).Seconds())
	if deadline := attempt.Deadline(quiz); deadline != nil && now.After(*deadline) {
		timeTaken = int(deadline.Sub(attempt.StartedAt).Seconds())
	}

	resultsJSON, err := json.Marshal(outcomes)
	if err != nil {
		return nil, apperrors.NewInternalError("GRADING_FAILED", "failed to record grading")
	}

	finalized := *attempt
	finalized.SubmittedAt = &now
	finalized.TimeTakenSeconds = &timeTaken
	finalized.Status = status
	finalized.Answers = answersJSON(answers)
	finalized.Results = resultsJSON
	finalized.ScorePoints = &awarded
	finalized.MaxPoints = &max
	if pending {
		// Marks are not final: no percentage, pass/fail or points yet.
		finalized.GradingStatus = assessments.GradingStatusPending
		finalized.ScorePercent = nil
		finalized.Passed = nil
		finalized.PointsAwarded = 0
	} else {
		passed := quiz.IsPassing(percent)
		finalized.GradingStatus = assessments.GradingStatusAuto
		finalized.ScorePercent = &percent
		finalized.Passed = &passed
		finalized.PointsAwarded = 0
		if passed && status == assessments.QuizAttemptStatusSubmitted {
			points, err := s.firstPassPoints(ctx, quiz, attempt.StudentID, percent)
			if err != nil {
				return nil, err
			}
			finalized.PointsAwarded = points
		}
	}

	won, err := s.attemptRepo.Finalize(ctx, &finalized)
	if err != nil {
		return nil, err
	}
	if !won {
		stored, err := s.attemptRepo.FindByID(ctx, attempt.ID)
		if err != nil {
			return nil, err
		}
		return s.submitResponse(quiz, stored), nil
	}
	if !pending && finalized.Passed != nil && *finalized.Passed {
		s.notifyPass(ctx, finalized.StudentID, quiz.ID, percent)
	}
	return s.submitResponse(quiz, &finalized), nil
}

// firstPassPoints keeps the legacy per-attempt points: 20 for the first
// passing attempt, +10 for a perfect score.
func (s *service) firstPassPoints(ctx context.Context, quiz *assessments.Quiz, studentID uuid.UUID, percent float64) (int, error) {
	highest, err := s.attemptRepo.GetHighestScore(ctx, quiz.ID, studentID)
	if err != nil {
		return 0, err
	}
	if highest != nil && quiz.IsPassing(*highest) {
		return 0, nil
	}
	points := 20
	if percent == 100.0 {
		points += 10
	}
	return points, nil
}

func (s *service) notifyPass(ctx context.Context, studentID, quizID uuid.UUID, percent float64) {
	if s.passNotifier == nil {
		return
	}
	// Points are a side effect; a failure must not undo the recorded grade.
	_ = s.passNotifier.QuizPassed(ctx, studentID, quizID, percent)
}

func (s *service) submitResponse(quiz *assessments.Quiz, attempt *assessments.QuizAttempt) *SubmitAttemptResponse {
	outcomes := decodeOutcomes(attempt.Results)
	resp := &SubmitAttemptResponse{
		AttemptID:       attempt.ID,
		QuizID:          attempt.QuizID,
		Status:          string(attempt.Status),
		GradingStatus:   gradingStatusString(attempt),
		PointsAwarded:   attempt.PointsAwarded,
		AnswersReleased: quiz.ShowAnswersAfterSubmission,
	}
	if attempt.TimeTakenSeconds != nil {
		resp.TimeTakenSeconds = *attempt.TimeTakenSeconds
	}
	if attempt.ScorePoints != nil {
		resp.ScorePoints = *attempt.ScorePoints
	}
	if attempt.MaxPoints != nil {
		resp.MaxPoints = *attempt.MaxPoints
	}
	if attempt.ScorePercent != nil {
		resp.ScorePercent = *attempt.ScorePercent
	} else if attempt.MaxPoints != nil {
		// Provisional (auto-graded part only) while grading is pending.
		resp.ScorePercent = scorePercent(resp.ScorePoints, resp.MaxPoints)
	}
	if attempt.Passed != nil {
		resp.Passed = *attempt.Passed
	}
	resp.Review = buildReview(outcomes, quiz.ShowAnswersAfterSubmission)
	if quiz.ShowAnswersAfterSubmission {
		for _, o := range outcomes {
			resp.QuestionResults = append(resp.QuestionResults, QuestionResultResponse{
				QuestionID:      o.QuestionID,
				IsCorrect:       o.Status == OutcomeCorrect,
				SelectedOptions: o.SelectedOptionIDs,
				CorrectOptions:  o.CorrectOptionIDs,
				TextAnswer:      o.TextAnswer,
				ImageURL:        o.AnswerImageURL,
				Explanation:     o.Explanation,
			})
		}
	}
	return resp
}

// buildReview applies the answer-release policy. Students always see the
// questions and their own answers; marks, correctness, answer keys and
// explanations are included only when the teacher releases answers.
func buildReview(outcomes []QuestionOutcome, released bool) []QuestionReview {
	if len(outcomes) == 0 {
		return nil
	}
	review := make([]QuestionReview, 0, len(outcomes))
	for _, o := range outcomes {
		r := QuestionReview{
			QuestionID:        o.QuestionID,
			Position:          o.Position,
			Type:              o.Type,
			Body:              o.Body,
			ContentType:       o.ContentType,
			ImageURL:          o.ImageURL,
			Marks:             o.Marks,
			Unit:              o.Unit,
			SelectedOptionIDs: o.SelectedOptionIDs,
			TextAnswer:        o.TextAnswer,
			AnswerImageURL:    o.AnswerImageURL,
			Answered:          o.Answered,
			Pending:           o.Status == OutcomePending,
		}
		for _, opt := range o.Options {
			copyOpt := opt
			if !released {
				copyOpt.IsCorrect = false
			}
			r.Options = append(r.Options, copyOpt)
		}
		if released {
			awarded := o.Awarded
			if o.Status != OutcomePending {
				r.Awarded = &awarded
			}
			r.Status = o.Status
			r.CorrectOptionIDs = o.CorrectOptionIDs
			r.AcceptedAnswers = o.AcceptedAnswers
			r.NumericAnswer = o.NumericAnswer
			r.Tolerance = o.Tolerance
			r.ToleranceType = o.ToleranceType
			r.Explanation = o.Explanation
			r.Feedback = o.Feedback
		} else if o.Feedback != "" && o.Status == OutcomeGraded {
			// Teacher feedback on a written answer is not an answer key.
			r.Feedback = o.Feedback
		}
		review = append(review, r)
	}
	return review
}

func (s *service) attemptResult(quiz *assessments.Quiz, attempt *assessments.QuizAttempt) *StudentAttemptResult {
	result := toStudentAttemptResult(attempt)
	result.QuizTitle = quiz.Title
	result.PassingScorePercent = quiz.PassingScorePercent
	result.AnswersReleased = quiz.ShowAnswersAfterSubmission
	if attempt.IsSubmitted() {
		outcomes := decodeOutcomes(attempt.Results)
		result.ReviewAvailable = len(outcomes) > 0
		result.Review = buildReview(outcomes, quiz.ShowAnswersAfterSubmission)
	}
	return result
}

// GetStudentQuizAttemptResult returns a submitted attempt with its review.
func (s *service) GetStudentQuizAttemptResult(ctx context.Context, quizID, attemptID, studentID uuid.UUID) (*StudentAttemptResult, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, studentID, quiz); err != nil {
		return nil, err
	}
	attempt, err := s.attemptRepo.FindByID(ctx, attemptID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if attempt.QuizID != quizID || attempt.StudentID != studentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to access this attempt")
	}
	if !attempt.IsSubmitted() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_IN_PROGRESS", "quiz attempt is still in progress", nil)
	}
	return s.attemptResult(quiz, attempt), nil
}

// GetStudentAttempt returns a student's attempt by ID (status lookup; the
// review is included once submitted).
func (s *service) GetStudentAttempt(ctx context.Context, attemptID, studentID uuid.UUID) (*StudentAttemptResult, error) {
	attempt, err := s.attemptRepo.FindByID(ctx, attemptID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if attempt.StudentID != studentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to access this attempt")
	}
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, studentID, quiz); err != nil {
		return nil, err
	}
	return s.attemptResult(quiz, attempt), nil
}

// CheckAnswer gives instant feedback on one answer. Practice mode only: in
// exam mode nothing about correctness is revealed before submission.
func (s *service) CheckAnswer(ctx context.Context, cmd CheckAnswerCommand) (*CheckAnswerResponse, error) {
	attempt, err := s.attemptRepo.FindByID(ctx, cmd.AttemptID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if attempt.StudentID != cmd.StudentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to access this attempt")
	}
	if !attempt.IsInProgress() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_ALREADY_SUBMITTED", "attempt has already been submitted", nil)
	}
	quiz, err := s.quizRepo.FindByID(ctx, attempt.QuizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureStudentCanAccessQuiz(ctx, cmd.StudentID, quiz); err != nil {
		return nil, err
	}
	if !quiz.IsPractice() {
		return nil, apperrors.NewForbiddenError("CHECK_NOT_ALLOWED", "answer checking is only available in practice mode")
	}
	question, err := s.questionRepo.FindByID(ctx, cmd.QuestionID)
	if err != nil || question.QuizID != quiz.ID {
		return nil, apperrors.NewNotFoundError("QUESTION_NOT_FOUND", "question not found")
	}
	options, err := s.optionRepo.FindByQuestionID(ctx, question.ID)
	if err != nil {
		return nil, err
	}
	raw, _ := json.Marshal(map[string]interface{}{question.ID.String(): cmd.Answer})
	answer := answersFromDraft(raw)[question.ID]
	outcome := gradeQuestion(question, options, answer, quiz.PartialCredit)
	return &CheckAnswerResponse{
		QuestionID:       question.ID,
		Status:           outcome.Status,
		Correct:          outcome.Status == OutcomeCorrect,
		CorrectOptionIDs: outcome.CorrectOptionIDs,
		AcceptedAnswers:  outcome.AcceptedAnswers,
		NumericAnswer:    outcome.NumericAnswer,
		Explanation:      outcome.Explanation,
	}, nil
}

// maxAnswerImageBytes caps student answer images.
const maxAnswerImageBytes = 5 << 20

// UploadAnswerImageCommand attaches an image to a written answer.
type UploadAnswerImageCommand struct {
	AttemptID uuid.UUID
	StudentID uuid.UUID
	FileName  string
	Size      int64
	Content   []byte
}

// UploadAnswerImageResponse returns the stable media path to store in the answer.
type UploadAnswerImageResponse struct {
	FileURL string `json:"file_url"`
}

// UploadAnswerImage stores an image for the student's own in-progress
// attempt (JPEG, PNG or WebP, at most 5 MB, type checked from the bytes).
func (s *service) UploadAnswerImage(ctx context.Context, cmd UploadAnswerImageCommand) (*UploadAnswerImageResponse, error) {
	attempt, err := s.attemptRepo.FindByID(ctx, cmd.AttemptID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
	}
	if attempt.StudentID != cmd.StudentID {
		return nil, apperrors.NewForbiddenError("FORBIDDEN", "not authorized to update this attempt")
	}
	if !attempt.IsInProgress() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_ALREADY_SUBMITTED", "attempt has already been submitted", nil)
	}
	if cmd.Size <= 0 || cmd.Size > maxAnswerImageBytes || int64(len(cmd.Content)) != cmd.Size {
		return nil, apperrors.NewSimpleValidationError("INVALID_FILE_SIZE", "images must be at most 5 MB")
	}
	ext := ""
	switch http.DetectContentType(cmd.Content) {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	default:
		return nil, apperrors.NewSimpleValidationError("INVALID_FILE_TYPE", "only JPEG, PNG or WebP images are accepted")
	}
	if s.storageClient == nil {
		return nil, apperrors.NewInternalError("STORAGE_NOT_CONFIGURED", "upload storage is not configured")
	}
	key := "quiz-answers/" + attempt.ID.String() + "/" + uuid.New().String() + ext
	contentType := strings.TrimPrefix(ext, ".")
	if contentType == "jpg" {
		contentType = "jpeg"
	}
	if err := s.storageClient.PutObject(ctx, s.filesBucket, key, bytes.NewReader(cmd.Content), cmd.Size, "image/"+contentType); err != nil {
		return nil, apperrors.NewInternalError("FILE_UPLOAD_FAILED", "failed to store image")
	}
	return &UploadAnswerImageResponse{FileURL: "/v1/media/" + key}, nil
}
