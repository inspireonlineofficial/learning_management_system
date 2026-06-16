package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	livesessions "lms-backend/internal/application/live_sessions"
	"lms-backend/pkg/apperrors"

	"github.com/google/uuid"
)

// LiveSessionsHandler handles HTTP requests for the live sessions bounded context.
type LiveSessionsHandler struct {
	service   livesessions.Service
	roomStore stateStore
}

// NewLiveSessionsHandler creates a new LiveSessionsHandler.
func NewLiveSessionsHandler(service livesessions.Service, stores ...stateStore) *LiveSessionsHandler {
	var roomStore stateStore
	if len(stores) > 0 {
		roomStore = stores[0]
	}
	return &LiveSessionsHandler{service: service, roomStore: roomStore}
}

type liveRoomMessage struct {
	ID         uuid.UUID `json:"id"`
	AuthorID   uuid.UUID `json:"author_id"`
	AuthorName string    `json:"author_name"`
	AuthorRole string    `json:"author_role"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
}

type liveRoomParticipant struct {
	ID         uuid.UUID `json:"id"`
	FullName   string    `json:"full_name"`
	Role       string    `json:"role"`
	Status     string    `json:"status"`
	Muted      bool      `json:"muted"`
	HandRaised bool      `json:"hand_raised"`
	JoinedAt   time.Time `json:"joined_at"`
}

type liveRoomState struct {
	Messages     []liveRoomMessage
	Participants map[string]liveRoomParticipant
	Recording    bool
}

// ListTeacherSessions handles GET /v1/teacher/live-sessions.
func (h *LiveSessionsHandler) ListTeacherSessions(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	result, err := h.service.ListTeacherSessions(r.Context(), livesessions.ListTeacherSessionsCommand{
		TeacherID: userID,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetTeacherSession handles GET /v1/teacher/live-sessions/:sessionId.
func (h *LiveSessionsHandler) GetTeacherSession(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("sessionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid session ID"))
		return
	}

	result, err := h.service.GetTeacherSession(r.Context(), livesessions.GetTeacherSessionCommand{
		SessionID: sessionID,
		TeacherID: userID,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ScheduleSession handles POST /v1/teacher/live-sessions
// Requirement 16.1: Create a live session and notify enrolled students.
//
// @Summary      Schedule a live session
// @Description  Creates a new live session for a course and notifies enrolled students
// @Tags         live-sessions
// @Accept       json
// @Produce      json
// @Param        body  body      object{course_id=string,title=string,scheduled_at=string,duration_minutes=int,record_session=bool}  true  "Session details"
// @Success      201   {object}  live_sessions.LiveSessionResponse
// @Failure      400   {object}  ValidationErrorResponse
// @Failure      401   {object}  ErrorResponse
// @Failure      403   {object}  ErrorResponse
// @Failure      404   {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/live-sessions [post]
func (h *LiveSessionsHandler) ScheduleSession(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		CourseID        string `json:"course_id"`
		Title           string `json:"title"`
		ScheduledAt     string `json:"scheduled_at"`
		DurationMinutes int    `json:"duration_minutes"`
		RecordSession   bool   `json:"record_session"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	courseID, err := uuid.Parse(req.CourseID)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_COURSE_ID", "invalid course_id"))
		return
	}

	scheduledAt, err := time.Parse(time.RFC3339, req.ScheduledAt)
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_DATE", "scheduled_at must be ISO 8601 UTC"))
		return
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("MISSING_TITLE", "title is required"))
		return
	}
	if !scheduledAt.After(time.Now().UTC()) {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_DATE", "scheduled_at must be in the future"))
		return
	}
	if req.DurationMinutes <= 0 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_DURATION", "duration_minutes must be positive"))
		return
	}

	cmd := livesessions.ScheduleSessionCommand{
		TeacherID:       userID,
		CourseID:        courseID,
		Title:           title,
		ScheduledAt:     scheduledAt.UTC(),
		DurationMinutes: req.DurationMinutes,
		RecordSession:   req.RecordSession,
	}

	result, err := h.service.ScheduleSession(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, result)
}

// StartSession handles POST /v1/teacher/live-sessions/:sessionId/start
// Requirement 16.2: Transition session to live, return teacher room token.
//
// @Summary      Start a live session
// @Description  Transitions a scheduled session to live status and returns a teacher room token
// @Tags         live-sessions
// @Produce      json
// @Param        sessionId  path      string  true  "Session ID"
// @Success      200        {object}  live_sessions.StartSessionResponse
// @Failure      400        {object}  ValidationErrorResponse
// @Failure      401        {object}  ErrorResponse
// @Failure      403        {object}  ErrorResponse
// @Failure      404        {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/live-sessions/{sessionId}/start [post]
func (h *LiveSessionsHandler) StartSession(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("sessionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid session ID"))
		return
	}

	cmd := livesessions.StartSessionCommand{
		SessionID: sessionID,
		TeacherID: userID,
	}

	result, err := h.service.StartSession(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// EndSession handles POST /v1/teacher/live-sessions/:sessionId/end
// Requirement 16.5: Transition to ended, record attendee_count, enqueue recording job.
//
// @Summary      End a live session
// @Description  Transitions a live session to ended status and enqueues a recording job if applicable
// @Tags         live-sessions
// @Produce      json
// @Param        sessionId  path      string  true  "Session ID"
// @Success      200        {object}  live_sessions.LiveSessionResponse
// @Failure      400        {object}  ValidationErrorResponse
// @Failure      401        {object}  ErrorResponse
// @Failure      403        {object}  ErrorResponse
// @Failure      404        {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/live-sessions/{sessionId}/end [post]
func (h *LiveSessionsHandler) EndSession(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("sessionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid session ID"))
		return
	}

	cmd := livesessions.EndSessionCommand{
		SessionID: sessionID,
		TeacherID: userID,
	}

	result, err := h.service.EndSession(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// RescheduleOrCancelSession handles PATCH /v1/teacher/live-sessions/:sessionId
// Requirement 16.6: Update or cancel a session and notify enrolled students.
//
// @Summary      Reschedule or cancel a live session
// @Description  Updates a session's schedule details or cancels it, notifying enrolled students
// @Tags         live-sessions
// @Accept       json
// @Produce      json
// @Param        sessionId  path      string                                                                          true  "Session ID"
// @Param        body       body      object{title=string,scheduled_at=string,duration_minutes=int,cancel=bool}       true  "Update details"
// @Success      200        {object}  live_sessions.LiveSessionResponse
// @Failure      400        {object}  ValidationErrorResponse
// @Failure      401        {object}  ErrorResponse
// @Failure      403        {object}  ErrorResponse
// @Failure      404        {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/live-sessions/{sessionId} [patch]
func (h *LiveSessionsHandler) RescheduleOrCancelSession(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("sessionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid session ID"))
		return
	}

	var req struct {
		Title           *string `json:"title"`
		ScheduledAt     *string `json:"scheduled_at"`
		DurationMinutes *int    `json:"duration_minutes"`
		Cancel          bool    `json:"cancel"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	cmd := livesessions.RescheduleOrCancelSessionCommand{
		SessionID:       sessionID,
		TeacherID:       userID,
		Title:           req.Title,
		DurationMinutes: req.DurationMinutes,
		Cancel:          req.Cancel,
	}

	if req.ScheduledAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ScheduledAt)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_DATE", "scheduled_at must be ISO 8601 UTC"))
			return
		}
		utc := t.UTC()
		cmd.ScheduledAt = &utc
	}

	result, err := h.service.RescheduleOrCancelSession(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// GetAttendance handles GET /v1/teacher/live-sessions/:sessionId/attendance
// Requirement 16.7: Return per-student attendance records.
//
// @Summary      Get session attendance
// @Description  Returns per-student attendance records for a live session
// @Tags         live-sessions
// @Produce      json
// @Param        sessionId  path      string  true  "Session ID"
// @Success      200        {object}  live_sessions.AttendanceResponse
// @Failure      400        {object}  ValidationErrorResponse
// @Failure      401        {object}  ErrorResponse
// @Failure      403        {object}  ErrorResponse
// @Failure      404        {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/teacher/live-sessions/{sessionId}/attendance [get]
func (h *LiveSessionsHandler) GetAttendance(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("sessionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid session ID"))
		return
	}

	cmd := livesessions.GetAttendanceCommand{
		SessionID: sessionID,
		TeacherID: userID,
	}

	result, err := h.service.GetAttendance(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func (h *LiveSessionsHandler) ListChat(w http.ResponseWriter, r *http.Request) {
	sessionID, err := parseLiveSessionID(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var since *time.Time
	if raw := r.URL.Query().Get("since"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_DATE", "since must be RFC3339"))
			return
		}
		since = &parsed
	}

	room, err := h.readLiveRoom(r.Context(), sessionID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	messages := make([]liveRoomMessage, 0, len(room.Messages))
	for _, message := range room.Messages {
		if since == nil || message.CreatedAt.After(*since) {
			messages = append(messages, message)
		}
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"data": messages})
}

func (h *LiveSessionsHandler) PostChat(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	sessionID, err := parseLiveSessionID(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}
	text := strings.TrimSpace(req.Text)
	if text == "" {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("EMPTY_MESSAGE", "message text is required"))
		return
	}
	if len(text) > 2000 {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("MESSAGE_TOO_LONG", "message text must be 2000 characters or fewer"))
		return
	}

	now := time.Now().UTC()
	message := liveRoomMessage{
		ID:         uuid.New(),
		AuthorID:   userID,
		AuthorName: liveRoomDisplayName(r),
		AuthorRole: liveRoomRole(r),
		Text:       text,
		CreatedAt:  now,
	}

	room, err := h.readLiveRoom(r.Context(), sessionID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	room.Messages = append(room.Messages, message)
	room.Participants[userID.String()] = liveRoomParticipant{
		ID:         userID,
		FullName:   message.AuthorName,
		Role:       message.AuthorRole,
		Status:     "joined",
		JoinedAt:   now,
		HandRaised: false,
	}
	if err := h.writeLiveRoom(r.Context(), sessionID, room); err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusCreated, message)
}

func (h *LiveSessionsHandler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	sessionID, err := parseLiveSessionID(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	room, err := h.readLiveRoom(r.Context(), sessionID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	if _, ok := room.Participants[userID.String()]; !ok {
		room.Participants[userID.String()] = liveRoomParticipant{
			ID:       userID,
			FullName: liveRoomDisplayName(r),
			Role:     liveRoomRole(r),
			Status:   "joined",
			JoinedAt: time.Now().UTC(),
		}
	}
	participants := make([]liveRoomParticipant, 0, len(room.Participants))
	for _, participant := range room.Participants {
		participants = append(participants, participant)
	}
	if err := h.writeLiveRoom(r.Context(), sessionID, room); err != nil {
		writeErrorResponse(w, err)
		return
	}

	sort.Slice(participants, func(i, j int) bool {
		return participants[i].JoinedAt.Before(participants[j].JoinedAt)
	})
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"data": participants})
}

func (h *LiveSessionsHandler) MuteParticipant(w http.ResponseWriter, r *http.Request) {
	h.updateParticipant(w, r, func(participant liveRoomParticipant) liveRoomParticipant {
		participant.Muted = true
		return participant
	})
}

func (h *LiveSessionsHandler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	h.updateParticipant(w, r, func(participant liveRoomParticipant) liveRoomParticipant {
		participant.Status = "left"
		return participant
	})
}

func (h *LiveSessionsHandler) LowerHand(w http.ResponseWriter, r *http.Request) {
	h.updateParticipant(w, r, func(participant liveRoomParticipant) liveRoomParticipant {
		participant.HandRaised = false
		return participant
	})
}

func (h *LiveSessionsHandler) SetRecording(w http.ResponseWriter, r *http.Request) {
	sessionID, err := parseLiveSessionID(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	var req struct {
		Recording bool `json:"recording"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_JSON", "invalid request body"))
		return
	}

	room, err := h.readLiveRoom(r.Context(), sessionID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	room.Recording = req.Recording
	if err := h.writeLiveRoom(r.Context(), sessionID, room); err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "recording": req.Recording})
}

// JoinSession handles POST /v1/live-sessions/:sessionId/join
// Requirement 16.3, 16.4: Verify live status, record attendance, return student room token.
//
// @Summary      Join a live session
// @Description  Verifies the session is live, records student attendance, and returns a room token
// @Tags         live-sessions
// @Produce      json
// @Param        sessionId  path      string  true  "Session ID"
// @Success      200        {object}  live_sessions.JoinSessionResponse
// @Failure      400        {object}  ValidationErrorResponse
// @Failure      401        {object}  ErrorResponse
// @Failure      403        {object}  ErrorResponse
// @Failure      404        {object}  ErrorResponse
// @Security     BearerAuth
// @Router       /v1/live-sessions/{sessionId}/join [post]
func (h *LiveSessionsHandler) JoinSession(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("sessionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid session ID"))
		return
	}

	cmd := livesessions.JoinSessionCommand{
		SessionID: sessionID,
		StudentID: userID,
	}

	result, err := h.service.JoinSession(r.Context(), cmd)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// ListStudentSessions handles GET /v1/student/live-sessions.
func (h *LiveSessionsHandler) ListStudentSessions(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	from, err := parseOptionalTimeQuery(r, "from")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	to, err := parseOptionalTimeQuery(r, "to")
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	result, err := h.service.ListStudentSessions(r.Context(), livesessions.ListStudentSessionsCommand{
		StudentID: userID,
		From:      from,
		To:        to,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func parseOptionalTimeQuery(r *http.Request, key string) (*time.Time, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return &parsed, nil
	}
	return nil, apperrors.NewSimpleValidationError("INVALID_DATE", key+" must be YYYY-MM-DD or RFC3339")
}

// GetStudentSession handles GET /v1/student/live-sessions/:sessionId.
func (h *LiveSessionsHandler) GetStudentSession(w http.ResponseWriter, r *http.Request) {
	userID, err := getUserIDFromContext(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	sessionID, err := uuid.Parse(r.PathValue("sessionId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_ID", "invalid session ID"))
		return
	}

	result, err := h.service.GetStudentSession(r.Context(), livesessions.GetStudentSessionCommand{
		SessionID: sessionID,
		StudentID: userID,
	})
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

func (h *LiveSessionsHandler) updateParticipant(w http.ResponseWriter, r *http.Request, update func(liveRoomParticipant) liveRoomParticipant) {
	sessionID, err := parseLiveSessionID(r)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	participantID, err := uuid.Parse(r.PathValue("participantId"))
	if err != nil {
		writeErrorResponse(w, apperrors.NewSimpleValidationError("INVALID_PARTICIPANT_ID", "invalid participant ID"))
		return
	}

	room, err := h.readLiveRoom(r.Context(), sessionID)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}
	participant, ok := room.Participants[participantID.String()]
	if !ok {
		participant = liveRoomParticipant{
			ID:       participantID,
			FullName: "Participant",
			Role:     "student",
			Status:   "joined",
			JoinedAt: time.Now().UTC(),
		}
	}
	participant = update(participant)
	room.Participants[participantID.String()] = participant
	if err := h.writeLiveRoom(r.Context(), sessionID, room); err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{"ok": true, "participant": participant})
}

const liveRoomTTL = 24 * time.Hour

func (h *LiveSessionsHandler) readLiveRoom(ctx context.Context, sessionID uuid.UUID) (*liveRoomState, error) {
	if h.roomStore == nil {
		return nil, apperrors.NewInternalError("LIVE_ROOM_STORE_UNAVAILABLE", "live room store is not configured")
	}
	raw, err := h.roomStore.Get(ctx, liveRoomStoreKey(sessionID))
	if err != nil {
		if stateStoreMiss(err) {
			return newLiveRoomState(), nil
		}
		return nil, apperrors.NewInternalError("LIVE_ROOM_READ_FAILED", "failed to read live room state")
	}
	var room liveRoomState
	if err := json.Unmarshal([]byte(raw), &room); err != nil {
		return nil, apperrors.NewInternalError("LIVE_ROOM_READ_FAILED", "failed to read live room state")
	}
	if room.Messages == nil {
		room.Messages = make([]liveRoomMessage, 0)
	}
	if room.Participants == nil {
		room.Participants = make(map[string]liveRoomParticipant)
	}
	return &room, nil
}

func (h *LiveSessionsHandler) writeLiveRoom(ctx context.Context, sessionID uuid.UUID, room *liveRoomState) error {
	if h.roomStore == nil {
		return apperrors.NewInternalError("LIVE_ROOM_STORE_UNAVAILABLE", "live room store is not configured")
	}
	data, err := json.Marshal(room)
	if err != nil {
		return apperrors.NewInternalError("LIVE_ROOM_WRITE_FAILED", "failed to write live room state")
	}
	if err := h.roomStore.Set(ctx, liveRoomStoreKey(sessionID), string(data), liveRoomTTL); err != nil {
		return apperrors.NewInternalError("LIVE_ROOM_WRITE_FAILED", "failed to write live room state")
	}
	return nil
}

func newLiveRoomState() *liveRoomState {
	return &liveRoomState{
		Messages:     make([]liveRoomMessage, 0),
		Participants: make(map[string]liveRoomParticipant),
	}
}

func liveRoomStoreKey(sessionID uuid.UUID) string {
	return "live-room:" + sessionID.String()
}

func parseLiveSessionID(r *http.Request) (uuid.UUID, error) {
	sessionID, err := uuid.Parse(r.PathValue("sessionId"))
	if err != nil {
		return uuid.Nil, apperrors.NewSimpleValidationError("INVALID_ID", "invalid session ID")
	}
	return sessionID, nil
}

func liveRoomRole(r *http.Request) string {
	if role, ok := r.Context().Value("role").(string); ok && role == "teacher" {
		return "host"
	}
	return "student"
}

func liveRoomDisplayName(r *http.Request) string {
	if name, ok := r.Context().Value("user_name").(string); ok && strings.TrimSpace(name) != "" {
		return strings.TrimSpace(name)
	}
	if email, ok := r.Context().Value("email").(string); ok && strings.TrimSpace(email) != "" {
		return strings.TrimSpace(email)
	}
	return "Participant"
}
