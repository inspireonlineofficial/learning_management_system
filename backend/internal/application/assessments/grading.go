package assessments

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"

	"lms-backend/internal/domain/assessments"

	"github.com/google/uuid"
)

// This file is the single grading engine used by submission, auto-submit,
// practice-mode answer checks and manual grading. It is pure (no I/O) so it
// can be tested exhaustively.

// Outcome statuses for one question.
const (
	OutcomeCorrect    = "correct"
	OutcomePartial    = "partial"
	OutcomeIncorrect  = "incorrect"
	OutcomeUnanswered = "unanswered"
	OutcomePending    = "pending" // waiting for manual marks
	OutcomeGraded     = "graded"  // manual marks awarded
)

// AnswerInput is one student answer, whatever the question type.
type AnswerInput struct {
	SelectedOptionIDs []uuid.UUID
	Text              string
	ImageURL          string
}

// OutcomeOption is an option as it was when the attempt was graded.
type OutcomeOption struct {
	ID          uuid.UUID `json:"id"`
	Body        string    `json:"body"`
	ContentType string    `json:"content_type,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	IsCorrect   bool      `json:"is_correct"`
	Position    int       `json:"position"`
}

// QuestionOutcome is the per-question grading record stored on the attempt
// (quiz_attempts.results). It snapshots the question so later edits to the
// quiz never change what a historical attempt shows or how it was scored.
type QuestionOutcome struct {
	QuestionID       uuid.UUID       `json:"question_id"`
	Position         int             `json:"position"`
	Type             string          `json:"type"`
	Body             string          `json:"body"`
	ContentType      string          `json:"content_type,omitempty"`
	ImageURL         string          `json:"image_url,omitempty"`
	Marks            float64         `json:"marks"`
	Options          []OutcomeOption `json:"options,omitempty"`
	CorrectOptionIDs []uuid.UUID     `json:"correct_option_ids"`
	AcceptedAnswers  []string        `json:"accepted_answers,omitempty"`
	NumericAnswer    *float64        `json:"numeric_answer,omitempty"`
	Tolerance        float64         `json:"tolerance,omitempty"`
	ToleranceType    string          `json:"tolerance_type,omitempty"`
	Unit             string          `json:"unit,omitempty"`
	Explanation      string          `json:"explanation,omitempty"`

	SelectedOptionIDs []uuid.UUID `json:"selected_option_ids"`
	TextAnswer        string      `json:"text_answer,omitempty"`
	AnswerImageURL    string      `json:"answer_image_url,omitempty"`
	Answered          bool        `json:"answered"`

	Awarded  float64    `json:"awarded"`
	Status   string     `json:"status"`
	Feedback string     `json:"feedback,omitempty"`
	GradedBy *uuid.UUID `json:"graded_by,omitempty"`
	GradedAt *time.Time `json:"graded_at,omitempty"`
}

// maxAnswerTextLength caps a single text answer (long answers included).
const maxAnswerTextLength = 20000

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// gradeQuestion grades one answer. partialCredit enables proportional marks
// for multiple-select questions: (correct picks - wrong picks) / correct
// options, never below zero.
func gradeQuestion(q *assessments.Question, options []*assessments.QuestionOption, answer AnswerInput, partialCredit bool) QuestionOutcome {
	marks := q.Marks
	if marks <= 0 {
		marks = 1
	}
	outcome := QuestionOutcome{
		QuestionID:        q.ID,
		Position:          q.Position,
		Type:              string(q.Type),
		Body:              q.Body,
		ContentType:       q.ContentType,
		ImageURL:          q.ImageURL,
		Marks:             marks,
		CorrectOptionIDs:  []uuid.UUID{},
		Explanation:       q.Explanation,
		SelectedOptionIDs: []uuid.UUID{},
	}

	correctSet := map[uuid.UUID]bool{}
	validOption := map[uuid.UUID]bool{}
	for _, opt := range options {
		validOption[opt.ID] = true
		if q.Type.UsesOptions() {
			outcome.Options = append(outcome.Options, OutcomeOption{
				ID: opt.ID, Body: opt.Body, ContentType: opt.ContentType, ImageURL: opt.ImageURL,
				IsCorrect: opt.IsCorrect, Position: opt.Position,
			})
		}
		if opt.IsCorrect {
			correctSet[opt.ID] = true
			if q.Type.UsesOptions() {
				outcome.CorrectOptionIDs = append(outcome.CorrectOptionIDs, opt.ID)
			} else if q.Type == assessments.QuestionTypeShort {
				outcome.AcceptedAnswers = append(outcome.AcceptedAnswers, opt.Body)
			}
		}
	}

	switch q.Type {
	case assessments.QuestionTypeSingle, assessments.QuestionTypeTrueFalse, assessments.QuestionTypeMultiple:
		// Only options that belong to this question count; duplicates are ignored.
		// A client may send a single-choice option ID as text; accept it.
		selected := answer.SelectedOptionIDs
		if len(selected) == 0 {
			if id, err := uuid.Parse(strings.TrimSpace(answer.Text)); err == nil {
				selected = []uuid.UUID{id}
			}
		}
		seen := map[uuid.UUID]bool{}
		for _, id := range selected {
			if validOption[id] && !seen[id] {
				seen[id] = true
				outcome.SelectedOptionIDs = append(outcome.SelectedOptionIDs, id)
			}
		}
		if q.Type != assessments.QuestionTypeMultiple && len(outcome.SelectedOptionIDs) > 1 {
			// A single-answer question with several picks is never correct.
			outcome.Answered = true
			outcome.Status = OutcomeIncorrect
			return outcome
		}
		if len(outcome.SelectedOptionIDs) == 0 {
			outcome.Status = OutcomeUnanswered
			return outcome
		}
		outcome.Answered = true
		right, wrong := 0, 0
		for _, id := range outcome.SelectedOptionIDs {
			if correctSet[id] {
				right++
			} else {
				wrong++
			}
		}
		switch {
		case wrong == 0 && right == len(correctSet) && right > 0:
			outcome.Awarded = marks
			outcome.Status = OutcomeCorrect
		case q.Type == assessments.QuestionTypeMultiple && partialCredit && len(correctSet) > 0:
			fraction := float64(right-wrong) / float64(len(correctSet))
			if fraction > 0 {
				outcome.Awarded = round2(marks * fraction)
				outcome.Status = OutcomePartial
			} else {
				outcome.Status = OutcomeIncorrect
			}
		default:
			outcome.Status = OutcomeIncorrect
		}

	case assessments.QuestionTypeShort:
		outcome.TextAnswer = truncate(answer.Text)
		outcome.AnswerImageURL = strings.TrimSpace(answer.ImageURL)
		normalized := normalizeShortAnswer(outcome.TextAnswer, q.Settings.CaseSensitive)
		if normalized == "" && outcome.AnswerImageURL == "" {
			outcome.Status = OutcomeUnanswered
			return outcome
		}
		outcome.Answered = true
		outcome.Status = OutcomeIncorrect
		for _, accepted := range outcome.AcceptedAnswers {
			if normalized != "" && normalizeShortAnswer(accepted, q.Settings.CaseSensitive) == normalized {
				outcome.Awarded = marks
				outcome.Status = OutcomeCorrect
				break
			}
		}

	case assessments.QuestionTypeNumeric:
		outcome.TextAnswer = truncate(strings.TrimSpace(answer.Text))
		outcome.NumericAnswer = q.Settings.NumericAnswer
		outcome.Tolerance = q.Settings.Tolerance
		outcome.ToleranceType = toleranceTypeOrDefault(q.Settings.ToleranceType)
		outcome.Unit = q.Settings.Unit
		if outcome.TextAnswer == "" {
			outcome.Status = OutcomeUnanswered
			return outcome
		}
		outcome.Answered = true
		outcome.Status = OutcomeIncorrect
		value, ok := ParseNumericAnswer(outcome.TextAnswer, q.Settings.Unit)
		if ok && q.Settings.NumericAnswer != nil &&
			withinTolerance(value, *q.Settings.NumericAnswer, q.Settings.Tolerance, outcome.ToleranceType) {
			outcome.Awarded = marks
			outcome.Status = OutcomeCorrect
		}

	case assessments.QuestionTypeLong:
		outcome.TextAnswer = truncate(answer.Text)
		outcome.AnswerImageURL = strings.TrimSpace(answer.ImageURL)
		if strings.TrimSpace(outcome.TextAnswer) == "" && outcome.AnswerImageURL == "" {
			outcome.Status = OutcomeUnanswered
			return outcome
		}
		outcome.Answered = true
		outcome.Status = OutcomePending

	default:
		outcome.Status = OutcomeUnanswered
	}
	return outcome
}

func truncate(text string) string {
	if len(text) <= maxAnswerTextLength {
		return text
	}
	// Cut on a rune boundary.
	cut := maxAnswerTextLength
	for cut > 0 && !utf8RuneStart(text[cut]) {
		cut--
	}
	return text[:cut]
}

func utf8RuneStart(b byte) bool { return b&0xC0 != 0x80 }

func normalizeShortAnswer(value string, caseSensitive bool) string {
	value = strings.TrimSpace(value)
	if !caseSensitive {
		value = strings.ToLower(value)
	}
	return strings.Join(strings.Fields(value), " ")
}

func toleranceTypeOrDefault(t string) string {
	if t == "percent" {
		return "percent"
	}
	return "absolute"
}

func withinTolerance(value, expected, tolerance float64, toleranceType string) bool {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return false
	}
	if tolerance < 0 {
		tolerance = 0
	}
	allowed := tolerance
	if toleranceType == "percent" {
		allowed = math.Abs(expected) * tolerance / 100
	}
	// Floating-point slack so 0.1+0.2 style representations still match.
	slack := 1e-9 * math.Max(1, math.Abs(expected))
	return math.Abs(value-expected) <= allowed+slack
}

// ParseNumericAnswer accepts integers, decimals, negatives, scientific
// notation (1.5e-3, 1.5E3, 1.5×10^-3, 1.5x10^3, 1.5*10^3), a Unicode minus
// sign, spaces/thin spaces as digit-group separators, and an optional unit
// suffix matching the configured unit. Commas are rejected because they are
// ambiguous (thousands separator vs decimal comma).
func ParseNumericAnswer(raw, unit string) (float64, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	if u := strings.TrimSpace(unit); u != "" {
		lower := strings.ToLower(s)
		if strings.HasSuffix(lower, strings.ToLower(u)) {
			s = strings.TrimSpace(s[:len(s)-len(u)])
		}
	}
	replacer := strings.NewReplacer(
		"−", "-", // minus sign
		"–", "-", // en dash typed as minus
		" ", "", " ", "", " ", "", " ", "",
		"×", "x", // multiplication sign
		"·", "x", // middle dot
	)
	s = replacer.Replace(s)
	if strings.Contains(s, ",") {
		return 0, false
	}
	lower := strings.ToLower(s)
	for _, sep := range []string{"x10^", "*10^", "x10**", "*10**"} {
		if idx := strings.Index(lower, sep); idx > 0 {
			mantissa := lower[:idx]
			exponent := strings.Trim(lower[idx+len(sep):], "()")
			s = mantissa + "e" + exponent
			lower = s
			break
		}
	}
	for _, r := range s {
		if !(unicode.IsDigit(r) || r == '.' || r == '-' || r == '+' || r == 'e' || r == 'E') {
			return 0, false
		}
	}
	value, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

// gradeTotals sums an attempt. pending is true while any answered
// manually-graded question has no marks yet.
func gradeTotals(outcomes []QuestionOutcome) (awarded, max float64, pending bool) {
	for _, o := range outcomes {
		max += o.Marks
		awarded += o.Awarded
		if o.Status == OutcomePending {
			pending = true
		}
	}
	return round2(awarded), round2(max), pending
}

func scorePercent(awarded, max float64) float64 {
	if max <= 0 {
		return 0
	}
	return round2(awarded / max * 100)
}

// answersFromCommands converts submitted answers into grading inputs.
func answersFromCommands(commands []QuizAnswerCommand) map[uuid.UUID]AnswerInput {
	answers := make(map[uuid.UUID]AnswerInput, len(commands))
	for _, c := range commands {
		answers[c.QuestionID] = AnswerInput{
			SelectedOptionIDs: c.SelectedOptions,
			Text:              c.TextAnswer,
			ImageURL:          c.ImageURL,
		}
	}
	return answers
}

// answersFromDraft parses the autosaved draft (see draftAnswersToCommands).
func answersFromDraft(raw []byte) map[uuid.UUID]AnswerInput {
	return answersFromCommands(draftAnswersToCommands(raw))
}

// answersJSON stores the final answers in the same shape as the draft so a
// submitted attempt can be shown with the same client code.
func answersJSON(answers map[uuid.UUID]AnswerInput) []byte {
	out := make(map[string]any, len(answers))
	for id, a := range answers {
		switch {
		case len(a.SelectedOptionIDs) > 0:
			ids := make([]string, len(a.SelectedOptionIDs))
			for i, v := range a.SelectedOptionIDs {
				ids[i] = v.String()
			}
			out[id.String()] = ids
		case a.Text != "" || a.ImageURL != "":
			out[id.String()] = map[string]string{"text": truncate(a.Text), "image_url": a.ImageURL}
		}
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return []byte("{}")
	}
	return raw
}

func decodeOutcomes(raw []byte) []QuestionOutcome {
	if len(raw) == 0 {
		return nil
	}
	var outcomes []QuestionOutcome
	if err := json.Unmarshal(raw, &outcomes); err != nil {
		return nil
	}
	return outcomes
}
