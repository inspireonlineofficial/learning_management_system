package assessments

import (
	"context"
	"encoding/json"
	"math"
	"sort"
	"strings"

	"lms-backend/internal/domain/assessments"
	domainenrollments "lms-backend/internal/domain/enrollments"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// AttemptListFilter filters and pages the teacher monitoring table.
type AttemptListFilter struct {
	Status string // "", in_progress, submitted, pending_grading, graded
	Search string // student name or email
	Sort   string // started_at (default) | score | student | submitted_at
	Order  string // desc (default) | asc
	Page   int
	Limit  int
}

// EnrollmentCounter counts active enrollments (optional dependency).
type EnrollmentCounter interface {
	CountActiveByCourse(ctx context.Context, courseID uuid.UUID) (int, error)
}

func (s *service) teacherQuiz(ctx context.Context, quizID, teacherID uuid.UUID) (*assessments.Quiz, error) {
	quiz, err := s.quizRepo.FindByID(ctx, quizID)
	if err != nil {
		return nil, apperrors.NewNotFoundError("QUIZ_NOT_FOUND", "quiz not found")
	}
	if err := s.ensureTeacherOwnsCourse(ctx, quiz.CourseID, teacherID); err != nil {
		return nil, err
	}
	return quiz, nil
}

func toTeacherRow(a *assessments.QuizAttemptWithStudent) TeacherAttemptRow {
	return TeacherAttemptRow{
		AttemptID:        a.ID,
		StudentID:        a.StudentID,
		StudentName:      a.StudentName,
		StudentEmail:     a.StudentEmail,
		AttemptNumber:    a.AttemptNumber,
		Status:           string(a.Status),
		GradingStatus:    gradingStatusString(&a.QuizAttempt),
		StartedAt:        a.StartedAt,
		SubmittedAt:      a.SubmittedAt,
		TimeTakenSeconds: a.TimeTakenSeconds,
		ScorePercent:     a.ScorePercent,
		ScorePoints:      a.ScorePoints,
		MaxPoints:        a.MaxPoints,
		Passed:           a.Passed,
	}
}

func (s *service) countEnrolled(ctx context.Context, courseID uuid.UUID) int {
	if counter, ok := s.enrollmentRepo.(EnrollmentCounter); ok {
		if n, err := counter.CountActiveByCourse(ctx, courseID); err == nil {
			return n
		}
	}
	return 0
}

// ListQuizAttempts is the teacher monitoring view: summary plus a filtered,
// sorted, paged attempt table. Only the course owner may call it.
func (s *service) ListQuizAttempts(ctx context.Context, quizID, teacherID uuid.UUID, filter AttemptListFilter) (*QuizAttemptsResponse, error) {
	quiz, err := s.teacherQuiz(ctx, quizID, teacherID)
	if err != nil {
		return nil, err
	}
	attempts, err := s.attemptRepo.ListByQuiz(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}

	summary := QuizAttemptSummary{EnrolledStudents: s.countEnrolled(ctx, quiz.CourseID)}
	students := map[uuid.UUID]bool{}
	// Latest final score per student (attempts are newest first).
	latestFinal := map[uuid.UUID]*assessments.QuizAttemptWithStudent{}
	for _, a := range attempts {
		students[a.StudentID] = true
		summary.TotalAttempts++
		switch {
		case a.IsInProgress():
			summary.InProgress++
		default:
			summary.Submitted++
			if a.GradingStatus == assessments.GradingStatusPending {
				summary.PendingGrading++
			} else if a.ScorePercent != nil {
				summary.GradedAttempts++
				if _, seen := latestFinal[a.StudentID]; !seen {
					latestFinal[a.StudentID] = a
				}
			}
		}
	}
	summary.StudentsAttempted = len(students)
	if summary.EnrolledStudents > summary.StudentsAttempted {
		summary.StudentsNotStarted = summary.EnrolledStudents - summary.StudentsAttempted
	}
	if len(latestFinal) > 0 {
		total, passed := 0.0, 0
		high, low := -1.0, 101.0
		for _, a := range latestFinal {
			v := *a.ScorePercent
			total += v
			high = math.Max(high, v)
			low = math.Min(low, v)
			if a.Passed != nil && *a.Passed {
				passed++
			}
		}
		avg := round2(total / float64(len(latestFinal)))
		rate := round2(float64(passed) / float64(len(latestFinal)) * 100)
		summary.AverageScore, summary.HighestScore, summary.LowestScore, summary.PassRate = &avg, &high, &low, &rate
	}

	// Filter.
	search := strings.ToLower(strings.TrimSpace(filter.Search))
	filtered := make([]*assessments.QuizAttemptWithStudent, 0, len(attempts))
	for _, a := range attempts {
		switch filter.Status {
		case "in_progress":
			if !a.IsInProgress() {
				continue
			}
		case "submitted":
			if a.IsInProgress() {
				continue
			}
		case "pending_grading":
			if a.IsInProgress() || a.GradingStatus != assessments.GradingStatusPending {
				continue
			}
		case "graded":
			if a.IsInProgress() || a.GradingStatus == assessments.GradingStatusPending {
				continue
			}
		}
		if search != "" && !strings.Contains(strings.ToLower(a.StudentName), search) &&
			!strings.Contains(strings.ToLower(a.StudentEmail), search) {
			continue
		}
		filtered = append(filtered, a)
	}

	// Sort.
	asc := filter.Order == "asc"
	less := func(i, j int) bool {
		a, b := filtered[i], filtered[j]
		switch filter.Sort {
		case "score":
			av, bv := -1.0, -1.0
			if a.ScorePercent != nil {
				av = *a.ScorePercent
			}
			if b.ScorePercent != nil {
				bv = *b.ScorePercent
			}
			return av < bv
		case "student":
			return strings.ToLower(a.StudentName) < strings.ToLower(b.StudentName)
		case "submitted_at":
			if a.SubmittedAt == nil || b.SubmittedAt == nil {
				return a.SubmittedAt == nil && b.SubmittedAt != nil
			}
			return a.SubmittedAt.Before(*b.SubmittedAt)
		default:
			return a.StartedAt.Before(b.StartedAt)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if asc {
			return less(i, j)
		}
		return less(j, i)
	})

	// Page.
	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	total := len(filtered)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}
	rows := make([]TeacherAttemptRow, 0, end-start)
	for _, a := range filtered[start:end] {
		rows = append(rows, toTeacherRow(a))
	}
	resp := &QuizAttemptsResponse{
		Quiz:     *s.toQuizResponse(quiz),
		Summary:  summary,
		Attempts: rows,
		Meta:     PageMeta{Page: page, Limit: limit, Total: total, TotalPages: (total + limit - 1) / limit},
	}
	return resp, nil
}

func (s *service) findTeacherAttempt(ctx context.Context, quiz *assessments.Quiz, attemptID uuid.UUID) (*assessments.QuizAttemptWithStudent, error) {
	attempts, err := s.attemptRepo.ListByQuiz(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	for _, a := range attempts {
		if a.ID == attemptID {
			return a, nil
		}
	}
	return nil, apperrors.NewNotFoundError("ATTEMPT_NOT_FOUND", "quiz attempt not found")
}

func (s *service) teacherAttemptDetail(quiz *assessments.Quiz, a *assessments.QuizAttemptWithStudent) *TeacherAttemptDetailResponse {
	outcomes := decodeOutcomes(a.Results)
	detail := &TeacherAttemptDetailResponse{
		TeacherAttemptRow: toTeacherRow(a),
		QuizTitle:         quiz.Title,
		ReviewAvailable:   len(outcomes) > 0,
		Results:           outcomes,
	}
	if a.IsInProgress() {
		detail.DraftAnswers = decodeDraftAnswers(a.DraftAnswers)
	}
	if detail.Results == nil {
		detail.Results = []QuestionOutcome{}
	}
	return detail
}

// GetTeacherAttempt returns one attempt with every question outcome.
func (s *service) GetTeacherAttempt(ctx context.Context, quizID, attemptID, teacherID uuid.UUID) (*TeacherAttemptDetailResponse, error) {
	quiz, err := s.teacherQuiz(ctx, quizID, teacherID)
	if err != nil {
		return nil, err
	}
	a, err := s.findTeacherAttempt(ctx, quiz, attemptID)
	if err != nil {
		return nil, err
	}
	return s.teacherAttemptDetail(quiz, a), nil
}

// GradeAttempt records manual marks. Long answers move from pending to
// graded; short answers may be re-marked (for spelling the key missed).
// The attempt becomes fully graded only when no pending answer remains.
func (s *service) GradeAttempt(ctx context.Context, cmd GradeAttemptCommand) (*TeacherAttemptDetailResponse, error) {
	quiz, err := s.teacherQuiz(ctx, cmd.QuizID, cmd.TeacherID)
	if err != nil {
		return nil, err
	}
	if len(cmd.Grades) == 0 {
		return nil, apperrors.NewSimpleValidationError("GRADES_REQUIRED", "provide at least one grade")
	}
	a, err := s.findTeacherAttempt(ctx, quiz, cmd.AttemptID)
	if err != nil {
		return nil, err
	}
	if a.IsInProgress() {
		return nil, apperrors.NewValidationErrorWithDetails("ATTEMPT_IN_PROGRESS", "the attempt has not been submitted yet", nil)
	}
	outcomes := decodeOutcomes(a.Results)
	if len(outcomes) == 0 {
		return nil, apperrors.NewValidationErrorWithDetails("REVIEW_UNAVAILABLE", "this attempt predates per-question grading and cannot be re-marked", nil)
	}
	index := map[uuid.UUID]int{}
	for i, o := range outcomes {
		index[o.QuestionID] = i
	}
	now := s.clock().UTC()
	teacher := cmd.TeacherID
	for _, g := range cmd.Grades {
		i, ok := index[g.QuestionID]
		if !ok {
			return nil, apperrors.NewSimpleValidationError("QUESTION_NOT_IN_ATTEMPT", "question is not part of this attempt")
		}
		o := &outcomes[i]
		if o.Type != string(assessments.QuestionTypeLong) && o.Type != string(assessments.QuestionTypeShort) {
			return nil, apperrors.NewSimpleValidationError("NOT_MANUALLY_GRADABLE", "only written answers can be graded manually")
		}
		if math.IsNaN(g.Awarded) || g.Awarded < 0 || g.Awarded > o.Marks {
			return nil, apperrors.NewSimpleValidationError("INVALID_MARKS", "awarded marks must be between 0 and the question's marks")
		}
		if len(g.Feedback) > maxContentLength {
			return nil, apperrors.NewSimpleValidationError("CONTENT_TOO_LONG", "feedback is limited to 20,000 characters")
		}
		o.Awarded = round2(g.Awarded)
		o.Feedback = strings.TrimSpace(g.Feedback)
		o.GradedBy = &teacher
		o.GradedAt = &now
		if o.Type == string(assessments.QuestionTypeLong) {
			o.Status = OutcomeGraded
		} else {
			switch {
			case o.Awarded >= o.Marks:
				o.Status = OutcomeCorrect
			case o.Awarded > 0:
				o.Status = OutcomePartial
			default:
				o.Status = OutcomeIncorrect
			}
		}
	}

	awarded, max, pending := gradeTotals(outcomes)
	raw, err := json.Marshal(outcomes)
	if err != nil {
		return nil, apperrors.NewInternalError("GRADING_FAILED", "failed to record grading")
	}
	updated := a.QuizAttempt
	wasPassed := updated.Passed != nil && *updated.Passed
	updated.Results = raw
	updated.ScorePoints = &awarded
	updated.MaxPoints = &max
	updated.GradedAt = &now
	updated.GradedBy = &teacher
	var newlyPassed bool
	var percent float64
	if pending {
		updated.GradingStatus = assessments.GradingStatusPending
		updated.ScorePercent = nil
		updated.Passed = nil
	} else {
		percent = scorePercent(awarded, max)
		passed := quiz.IsPassing(percent)
		updated.GradingStatus = assessments.GradingStatusGraded
		updated.ScorePercent = &percent
		updated.Passed = &passed
		newlyPassed = passed && !wasPassed
		if newlyPassed && updated.Status == assessments.QuizAttemptStatusSubmitted && updated.PointsAwarded == 0 {
			points, err := s.firstPassPoints(ctx, quiz, updated.StudentID, percent)
			if err != nil {
				return nil, err
			}
			updated.PointsAwarded = points
		}
	}
	if err := s.attemptRepo.SaveGrading(ctx, &updated); err != nil {
		return nil, err
	}
	if newlyPassed {
		s.notifyPass(ctx, updated.StudentID, quiz.ID, percent)
	}
	if !pending && s.notificationQueue != nil {
		_ = s.notificationQueue.EnqueueNotification(ctx, updated.StudentID, "quiz_graded",
			"Quiz graded", "Your attempt at \""+quiz.Title+"\" has been graded.")
	}
	a.QuizAttempt = updated
	return s.teacherAttemptDetail(quiz, a), nil
}

// GetQuizAnalytics computes question-level statistics from the stored
// per-question outcomes of submitted attempts. Nothing is estimated.
func (s *service) GetQuizAnalytics(ctx context.Context, quizID, teacherID uuid.UUID) (*QuizAnalyticsResponse, error) {
	quiz, err := s.teacherQuiz(ctx, quizID, teacherID)
	if err != nil {
		return nil, err
	}
	attempts, err := s.attemptRepo.ListByQuiz(ctx, quiz.ID)
	if err != nil {
		return nil, err
	}
	resp := &QuizAnalyticsResponse{QuizID: quiz.ID, ScoreDistribution: make([]ScoreBucket, 10)}
	for i := range resp.ScoreDistribution {
		resp.ScoreDistribution[i] = ScoreBucket{From: i * 10, To: (i + 1) * 10}
	}
	resp.ScoreDistribution[9].To = 101

	type acc struct {
		qa       QuestionAnalytics
		awarded  float64
		options  map[uuid.UUID]*OptionAnalytic
		wrongTxt map[string]int
	}
	byQuestion := map[uuid.UUID]*acc{}
	var scores []float64
	timeTotal, timeCount := 0.0, 0
	for _, a := range attempts {
		if a.IsInProgress() {
			continue
		}
		resp.SubmittedAttempts++
		if a.GradingStatus != assessments.GradingStatusPending && a.ScorePercent != nil {
			resp.GradedAttempts++
			scores = append(scores, *a.ScorePercent)
			bucket := int(*a.ScorePercent / 10)
			if bucket > 9 {
				bucket = 9
			}
			if bucket < 0 {
				bucket = 0
			}
			resp.ScoreDistribution[bucket].Count++
		}
		if a.TimeTakenSeconds != nil {
			timeTotal += float64(*a.TimeTakenSeconds)
			timeCount++
		}
		for _, o := range decodeOutcomes(a.Results) {
			q, ok := byQuestion[o.QuestionID]
			if !ok {
				q = &acc{options: map[uuid.UUID]*OptionAnalytic{}, wrongTxt: map[string]int{}}
				byQuestion[o.QuestionID] = q
			}
			// Keep the latest snapshot's text (attempts are newest first).
			if q.qa.Responses == 0 {
				q.qa.QuestionID, q.qa.Position, q.qa.Type, q.qa.Body, q.qa.Marks = o.QuestionID, o.Position, o.Type, o.Body, o.Marks
				for _, opt := range o.Options {
					q.options[opt.ID] = &OptionAnalytic{OptionID: opt.ID, Body: opt.Body, IsCorrect: opt.IsCorrect}
				}
			}
			q.qa.Responses++
			q.awarded += o.Awarded
			switch o.Status {
			case OutcomeCorrect:
				q.qa.Correct++
			case OutcomePartial:
				q.qa.Partial++
			case OutcomeIncorrect:
				q.qa.Incorrect++
				if t := normalizeShortAnswer(o.TextAnswer, false); t != "" {
					q.wrongTxt[t]++
				}
			case OutcomeUnanswered:
				q.qa.Unanswered++
			case OutcomePending:
				q.qa.Pending++
			case OutcomeGraded:
				if o.Awarded >= o.Marks {
					q.qa.Correct++
				} else if o.Awarded > 0 {
					q.qa.Partial++
				} else {
					q.qa.Incorrect++
				}
			}
			for _, id := range o.SelectedOptionIDs {
				if opt, ok := q.options[id]; ok {
					opt.Count++
				}
			}
		}
	}
	resp.SmallSample = resp.GradedAttempts < 5
	if len(scores) > 0 {
		sum := 0.0
		for _, v := range scores {
			sum += v
		}
		avg := round2(sum / float64(len(scores)))
		resp.AverageScore = &avg
		sorted := append([]float64(nil), scores...)
		sort.Float64s(sorted)
		var median float64
		if n := len(sorted); n%2 == 1 {
			median = sorted[n/2]
		} else {
			median = (sorted[n/2-1] + sorted[n/2]) / 2
		}
		median = round2(median)
		resp.MedianScore = &median
	}
	if timeCount > 0 {
		avg := round2(timeTotal / float64(timeCount))
		resp.AverageTimeSeconds = &avg
	}
	resp.Questions = make([]QuestionAnalytics, 0, len(byQuestion))
	for _, q := range byQuestion {
		if q.qa.Responses > 0 {
			q.qa.AverageAwarded = round2(q.awarded / float64(q.qa.Responses))
			if graded := q.qa.Responses - q.qa.Pending; graded > 0 {
				acc := round2(float64(q.qa.Correct) / float64(graded) * 100)
				q.qa.AccuracyPercent = &acc
			}
		}
		for _, opt := range q.options {
			q.qa.OptionCounts = append(q.qa.OptionCounts, *opt)
		}
		sort.Slice(q.qa.OptionCounts, func(i, j int) bool { return q.qa.OptionCounts[i].Count > q.qa.OptionCounts[j].Count })
		for text, count := range q.wrongTxt {
			q.qa.CommonWrong = append(q.qa.CommonWrong, TextAnalytic{Answer: text, Count: count})
		}
		sort.Slice(q.qa.CommonWrong, func(i, j int) bool { return q.qa.CommonWrong[i].Count > q.qa.CommonWrong[j].Count })
		if len(q.qa.CommonWrong) > 5 {
			q.qa.CommonWrong = q.qa.CommonWrong[:5]
		}
		resp.Questions = append(resp.Questions, q.qa)
	}
	sort.Slice(resp.Questions, func(i, j int) bool { return resp.Questions[i].Position < resp.Questions[j].Position })
	return resp, nil
}

var _ domainenrollments.EnrollmentRepository = nil
