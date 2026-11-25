package assessments

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// quizSubmitGrace tolerates network latency between the client timer reaching
// zero and the submit request arriving.
const quizSubmitGrace = 30 * time.Second

// draftAnswersToCommands converts the auto-saved draft (as stored by
// PATCH /v1/quizzes/{quizId}/attempts/{attemptId}) into scoring commands.
//
// The draft is a JSON object keyed by question ID whose values mirror the
// client state:
//   - "option-id"                        single choice
//   - ["option-id", ...]                 multiple choice
//   - {"text": "...", "image_url": "..."} short, numeric or long answer
//   - "text" (not an option ID) or a JSON number: text/numeric answer
func draftAnswersToCommands(raw []byte) []QuizAnswerCommand {
	if len(raw) == 0 {
		return []QuizAnswerCommand{}
	}
	var draft map[string]json.RawMessage
	if err := json.Unmarshal(raw, &draft); err != nil {
		return []QuizAnswerCommand{}
	}
	answers := make([]QuizAnswerCommand, 0, len(draft))
	for questionKey, value := range draft {
		questionID, err := uuid.Parse(questionKey)
		if err != nil {
			continue
		}
		answer := QuizAnswerCommand{QuestionID: questionID, SelectedOptions: []uuid.UUID{}}

		var single string
		var number json.Number
		var multiple []string
		var short struct {
			Text     string `json:"text"`
			ImageURL string `json:"image_url"`
		}
		switch {
		case json.Unmarshal(value, &single) == nil:
			if id, err := uuid.Parse(single); err == nil {
				answer.SelectedOptions = []uuid.UUID{id}
			} else {
				answer.TextAnswer = single
			}
		case json.Unmarshal(value, &number) == nil && number != "":
			answer.TextAnswer = number.String()
		case json.Unmarshal(value, &multiple) == nil:
			for _, item := range multiple {
				if id, err := uuid.Parse(item); err == nil {
					answer.SelectedOptions = append(answer.SelectedOptions, id)
				}
			}
		case json.Unmarshal(value, &short) == nil:
			answer.TextAnswer = short.Text
			answer.ImageURL = short.ImageURL
		default:
			continue
		}
		answers = append(answers, answer)
	}
	return answers
}
