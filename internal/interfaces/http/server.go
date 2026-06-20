package http

import (
	"context"
	"lms-backend/internal/application/analytics"
	"lms-backend/internal/application/assessments"
	"lms-backend/internal/application/audit"
	"lms-backend/internal/application/auth"
	"lms-backend/internal/application/bookshop"
	"lms-backend/internal/application/certificates"
	"lms-backend/internal/application/courses"
	"lms-backend/internal/application/enrollments"
	"lms-backend/internal/application/forum"
	livesessions "lms-backend/internal/application/live_sessions"
	"lms-backend/internal/application/notifications"
	"lms-backend/internal/application/payments"
	"lms-backend/internal/application/points"
	"lms-backend/internal/application/rbac"
	"lms-backend/internal/application/search"
	"lms-backend/internal/application/slides"
	appsysconfig "lms-backend/internal/application/system_config"
	"lms-backend/internal/application/users"
	"lms-backend/internal/infrastructure/jwt"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/internal/interfaces/http/handlers"
	"lms-backend/internal/interfaces/http/middleware"
	"net/http"
	"net/url"
	"strings"

	httpSwagger "github.com/swaggo/http-swagger"
)

// Server holds the HTTP server configuration
type Server struct {
	mux                  *http.ServeMux
	rateLimiter          *middleware.RateLimiter
	authHandler          *handlers.AuthHandler
	usersHandler         *handlers.UsersHandler
	coursesHandler       *handlers.CoursesHandler
	enrollmentsHandler   *handlers.EnrollmentsHandler
	assessmentsHandler   *handlers.AssessmentsHandler
	pointsHandler        *handlers.PointsHandler
	certificatesHandler  *handlers.CertificatesHandler
	bookshopHandler      *handlers.BookshopHandler
	forumHandler         *handlers.ForumHandler
	notificationsHandler *handlers.NotificationsHandler
	analyticsHandler     *handlers.AnalyticsHandler
	paymentsHandler      *handlers.PurchaseApprovalsHandler
	systemConfigHandler  *handlers.SystemConfigHandler
	liveSessionsHandler  *handlers.LiveSessionsHandler
	rbacHandler          *handlers.RBACHandler
	auditHandler         *handlers.AuditHandler
	searchHandler        *handlers.SearchHandler
	slidesHandler        *handlers.SlidesHandler
	jwksHandler          *handlers.JWKSHandler
	jwtService           *jwt.JWTService
	systemConfigService  appsysconfig.Service
	idempotencyStore     middleware.IdempotencyStore
	frontendBaseURL      string
}

// NewServer creates a new HTTP server
func NewServer(redisClient *redis.Client, authService auth.Service, usersService users.Service, coursesService courses.Service, enrollmentsService enrollments.Service, assessmentsService assessments.Service, pointsService points.Service, certificatesService certificates.Service, bookshopService bookshop.Service, forumService forum.Service, notificationsService notifications.Service, analyticsService analytics.Service, paymentsService payments.Service, systemConfigService appsysconfig.Service, liveSessionsService livesessions.Service, rbacService rbac.Service, auditService audit.Service, searchService search.Service, slidesService slides.Service, idempotencyStore middleware.IdempotencyStore, jwtService *jwt.JWTService, frontendBaseURL string) *Server {
	mux := http.NewServeMux()
	rateLimiter := middleware.NewRateLimiter(redisClient)
	authHandler := handlers.NewAuthHandler(authService, frontendBaseURL)
	usersHandler := handlers.NewUsersHandler(usersService)
	coursesHandler := handlers.NewCoursesHandler(coursesService)
	enrollmentsHandler := handlers.NewEnrollmentsHandler(enrollmentsService)
	assessmentsHandler := handlers.NewAssessmentsHandler(assessmentsService)
	pointsHandler := handlers.NewPointsHandler(pointsService)
	certificatesHandler := handlers.NewCertificatesHandler(certificatesService)
	bookshopHandler := handlers.NewBookshopHandler(bookshopService, redisClient)
	forumHandler := handlers.NewForumHandler(forumService)
	notificationsHandler := handlers.NewNotificationsHandler(notificationsService)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)
	paymentsHandler := handlers.NewPaymentsHandler(paymentsService)
	systemConfigHandler := handlers.NewSystemConfigHandler(systemConfigService)
	liveSessionsHandler := handlers.NewLiveSessionsHandler(liveSessionsService, redisClient)
	rbacHandler := handlers.NewRBACHandler(rbacService)
	auditHandler := handlers.NewAuditHandler(auditService)
	searchHandler := handlers.NewSearchHandler(searchService)
	slidesHandler := handlers.NewSlidesHandler(slidesService)
	jwksHandler := handlers.NewJWKSHandler(jwtService)

	return &Server{
		mux:                  mux,
		rateLimiter:          rateLimiter,
		authHandler:          authHandler,
		usersHandler:         usersHandler,
		coursesHandler:       coursesHandler,
		enrollmentsHandler:   enrollmentsHandler,
		assessmentsHandler:   assessmentsHandler,
		pointsHandler:        pointsHandler,
		certificatesHandler:  certificatesHandler,
		bookshopHandler:      bookshopHandler,
		forumHandler:         forumHandler,
		notificationsHandler: notificationsHandler,
		analyticsHandler:     analyticsHandler,
		paymentsHandler:      paymentsHandler,
		systemConfigHandler:  systemConfigHandler,
		liveSessionsHandler:  liveSessionsHandler,
		rbacHandler:          rbacHandler,
		auditHandler:         auditHandler,
		searchHandler:        searchHandler,
		slidesHandler:        slidesHandler,
		jwksHandler:          jwksHandler,
		jwtService:           jwtService,
		systemConfigService:  systemConfigService,
		idempotencyStore:     idempotencyStore,
		frontendBaseURL:      frontendBaseURL,
	}
}

// Handler returns the configured HTTP handler with middleware chain
func (s *Server) Handler() http.Handler {
	// Apply global middleware chain (in reverse order)
	var handler http.Handler = s.mux

	// Middleware chain
	handler = s.rateLimiter.Limit(handler)
	handler = middleware.CORS(corsAllowedOrigins(s.frontendBaseURL, []string{
		"https://app.example.com",
		"https://admin.example.com",
		"http://localhost:3000",
		"http://127.0.0.1:3000",
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}))(handler)
	handler = middleware.SecurityHeaders(handler)
	// Maintenance mode: returns 503 for all non-admin endpoints when enabled (Requirement 25.5)
	handler = middleware.MaintenanceMode(func() bool {
		return s.systemConfigService.IsMaintenanceMode(context.Background())
	})(handler)
	handler = middleware.StructuredLog(handler)
	handler = middleware.RequestID(handler)

	return handler
}

func corsAllowedOrigins(frontendBaseURL string, defaults []string) []string {
	seen := map[string]bool{}
	var origins []string
	for _, origin := range append(defaults, frontendBaseURL) {
		origin = strings.TrimRight(strings.TrimSpace(origin), "/")
		if origin == "" || seen[origin] {
			continue
		}
		if parsed, err := url.Parse(origin); err == nil && parsed.Scheme != "" && parsed.Host != "" {
			origins = append(origins, origin)
			seen[origin] = true
		}
	}
	return origins
}

// RegisterRoutes registers all application routes
func (s *Server) RegisterRoutes() {
	// Health check endpoint
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Swagger UI (public, no auth middleware)
	s.mux.Handle("/swagger/", httpSwagger.WrapHandler)

	// JWKS endpoint (public)
	s.mux.HandleFunc("/v1/.well-known/jwks.json", s.jwksHandler.GetJWKS)

	// Auth endpoints (public)
	s.mux.HandleFunc("/v1/auth/register", s.authHandler.Register)
	s.mux.HandleFunc("/v1/auth/verify-otp", s.authHandler.VerifyOTP)
	s.mux.HandleFunc("/v1/auth/resend-otp", s.authHandler.ResendOTP)
	s.mux.HandleFunc("/v1/auth/login", s.authHandler.Login)
	s.mux.HandleFunc("/v1/auth/refresh", s.authHandler.RefreshToken)
	s.mux.HandleFunc("/v1/auth/logout", s.authHandler.Logout)
	s.mux.HandleFunc("/v1/auth/forgot-password", s.authHandler.ForgotPassword)
	s.mux.HandleFunc("/v1/auth/reset-password", s.authHandler.ResetPassword)

	// Admin auth endpoints (public)
	s.mux.HandleFunc("/v1/auth/admin/login", s.authHandler.AdminLogin)
	s.mux.HandleFunc("/v1/auth/admin/verify-otp", s.authHandler.AdminVerifyOTP)
	s.mux.HandleFunc("/v1/auth/admin/resend-otp", s.authHandler.AdminResendOTP)

	// OAuth endpoints (public)
	s.mux.HandleFunc("/v1/auth/oauth/google", s.authHandler.OAuthRedirect)
	s.mux.HandleFunc("/v1/auth/oauth/google/callback", s.authHandler.OAuthCallback)
	s.mux.HandleFunc("/v1/auth/oauth/github", s.authHandler.OAuthRedirect)
	s.mux.HandleFunc("/v1/auth/oauth/github/callback", s.authHandler.OAuthCallback)
	s.mux.HandleFunc("/v1/auth/oauth/microsoft", s.authHandler.OAuthRedirect)
	s.mux.HandleFunc("/v1/auth/oauth/microsoft/callback", s.authHandler.OAuthCallback)

	// Authenticated user endpoints (require JWT)
	s.mux.HandleFunc("/v1/auth/me", s.withAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.authHandler.GetProfile(w, r)
			return
		}
		s.authHandler.UpdateProfile(w, r)
	}))
	s.mux.HandleFunc("/v1/auth/me/settings", s.withAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			s.authHandler.GetUserSettings(w, r)
			return
		}
		s.authHandler.UpdateUserSettings(w, r)
	}))
	s.mux.HandleFunc("/v1/auth/me/change-password", s.withAuth(s.authHandler.ChangePassword))
	s.mux.HandleFunc("/v1/auth/me/oauth/connect", s.withAuth(s.authHandler.ConnectProvider))
	s.mux.HandleFunc("/v1/auth/me/oauth/providers", s.withAuth(s.authHandler.ListProviders))
	s.mux.HandleFunc("/v1/auth/me/oauth/google", s.withAuth(s.authHandler.DisconnectProvider))
	s.mux.HandleFunc("/v1/auth/me/oauth/github", s.withAuth(s.authHandler.DisconnectProvider))
	s.mux.HandleFunc("/v1/auth/me/oauth/microsoft", s.withAuth(s.authHandler.DisconnectProvider))

	// Student onboarding endpoints (require JWT + student role)
	s.mux.HandleFunc("/v1/onboarding/student-profile", s.withAuthAndRole("student", s.usersHandler.SubmitStudentProfile, s.usersHandler.GetStudentProfile))

	// Admin user management endpoints (require JWT + admin role)
	s.mux.HandleFunc("/v1/admin/users", s.withAuthAndRole("admin", s.usersHandler.CreateUser, s.usersHandler.ListUsers))
	s.mux.HandleFunc("/v1/admin/users/", s.handleAdminUserRoutes)

	// Public course endpoints
	s.mux.HandleFunc("GET /v1/courses", s.coursesHandler.ListPublishedCourses)
	s.mux.HandleFunc("GET /v1/courses/{courseId}", s.coursesHandler.GetCourseDetail)
	s.mux.HandleFunc("GET /v1/courses/{courseId}/reviews", s.coursesHandler.ListCourseReviews)
	s.mux.HandleFunc("POST /v1/courses/{courseId}/reviews", s.withAuthAndRole("student", s.coursesHandler.UpsertCourseReview, nil))
	s.mux.HandleFunc("DELETE /v1/courses/{courseId}/reviews/me", s.withAuthAndRole("student", s.coursesHandler.DeleteCourseReview, nil))
	s.mux.HandleFunc("GET /v1/courses/{courseId}/comments", s.coursesHandler.ListCourseComments)
	s.mux.HandleFunc("POST /v1/courses/{courseId}/comments", s.withAuth(s.coursesHandler.CreateCourseComment))
	s.mux.HandleFunc("PATCH /v1/courses/comments/{commentId}", s.withAuth(s.coursesHandler.UpdateCourseComment))
	s.mux.HandleFunc("DELETE /v1/courses/comments/{commentId}", s.withAuth(s.coursesHandler.DeleteCourseComment))
	s.mux.HandleFunc("GET /v1/public/slides", s.slidesHandler.ListPublicSlides)

	// Teacher course endpoints (require JWT + teacher role)
	s.mux.HandleFunc("GET /v1/teacher/courses", s.withAuthAndRole("teacher", nil, s.coursesHandler.ListTeacherCourses))
	s.mux.HandleFunc("POST /v1/teacher/courses", s.withAuthAndRole("teacher", s.coursesHandler.CreateCourse, nil))
	s.mux.HandleFunc("PATCH /v1/teacher/courses/{courseId}", s.withAuthAndRole("teacher", s.coursesHandler.UpdateCourse, nil))
	s.mux.HandleFunc("DELETE /v1/teacher/courses/{courseId}", s.withAuthAndRole("teacher", s.coursesHandler.DeleteCourse, nil))
	s.mux.HandleFunc("POST /v1/teacher/courses/{courseId}/submit", s.withAuthAndRole("teacher", s.coursesHandler.SubmitCourse, nil))
	s.mux.HandleFunc("GET /v1/teacher/courses/{courseId}/preview", s.withAuthAndRole("teacher", nil, s.coursesHandler.GetTeacherCoursePreview))
	s.mux.HandleFunc("POST /v1/teacher/courses/{courseId}/modules", s.withAuthAndRole("teacher", s.coursesHandler.CreateModule, nil))
	s.mux.HandleFunc("PATCH /v1/teacher/modules/{moduleId}", s.withAuthAndRole("teacher", s.coursesHandler.UpdateModule, nil))
	s.mux.HandleFunc("DELETE /v1/teacher/modules/{moduleId}", s.withAuthAndRole("teacher", s.coursesHandler.DeleteModule, nil))
	s.mux.HandleFunc("POST /v1/teacher/modules/{moduleId}/chapters", s.withAuthAndRole("teacher", s.coursesHandler.CreateChapter, nil))
	s.mux.HandleFunc("PATCH /v1/teacher/chapters/{chapterId}", s.withAuthAndRole("teacher", s.coursesHandler.UpdateChapter, nil))
	s.mux.HandleFunc("DELETE /v1/teacher/chapters/{chapterId}", s.withAuthAndRole("teacher", s.coursesHandler.DeleteChapter, nil))
	s.mux.HandleFunc("POST /v1/teacher/chapters/{chapterId}/lessons", s.withAuthAndRole("teacher", s.coursesHandler.CreateLesson, nil))
	s.mux.HandleFunc("PATCH /v1/teacher/lessons/{lessonId}", s.withAuthAndRole("teacher", s.coursesHandler.UpdateLesson, nil))
	s.mux.HandleFunc("DELETE /v1/teacher/lessons/{lessonId}", s.withAuthAndRole("teacher", s.coursesHandler.DeleteLesson, nil))
	s.mux.HandleFunc("POST /v1/teacher/courses/{courseId}/notes", s.withAuthAndRole("teacher", s.coursesHandler.CreateCourseNote, nil))
	s.mux.HandleFunc("PATCH /v1/teacher/notes/{noteId}", s.withAuthAndRole("teacher", s.coursesHandler.UpdateCourseNote, nil))
	s.mux.HandleFunc("DELETE /v1/teacher/notes/{noteId}", s.withAuthAndRole("teacher", s.coursesHandler.DeleteCourseNote, nil))
	s.mux.HandleFunc("PATCH /v1/teacher/content/reorder", s.withAuthAndRole("teacher", s.coursesHandler.ReorderContent, nil))

	// Admin course endpoints (require JWT + admin role)
	s.mux.HandleFunc("GET /v1/admin/courses", s.withAuthAndRole("admin", nil, s.coursesHandler.ListPendingCourses))
	s.mux.HandleFunc("GET /v1/admin/courses/{courseId}", s.withAuthAndRole("admin", nil, s.coursesHandler.GetAdminCourseDetail))
	s.mux.HandleFunc("POST /v1/admin/courses/{courseId}/review", s.withAuthAndRole("admin", s.coursesHandler.ReviewCourse, nil))
	s.mux.HandleFunc("GET /v1/admin/slides", s.withAuthAndRole("admin", nil, s.slidesHandler.ListAdminSlides))
	s.mux.HandleFunc("POST /v1/admin/slides", s.withAuthAndRole("admin", s.slidesHandler.CreateSlide, nil))
	s.mux.HandleFunc("PATCH /v1/admin/slides/{slideId}", s.withAuthAndRole("admin", s.slidesHandler.UpdateSlide, nil))
	s.mux.HandleFunc("POST /v1/admin/slides/reorder", s.withAuthAndRole("admin", s.slidesHandler.ReorderSlides, nil))
	s.mux.HandleFunc("POST /v1/admin/slides/{slideId}/deactivate", s.withAuthAndRole("admin", s.slidesHandler.DeactivateSlide, nil))

	// Upload endpoints (require JWT)
	s.mux.HandleFunc("POST /v1/uploads/video", s.withAuth(s.coursesHandler.UploadVideo))
	s.mux.HandleFunc("GET /v1/uploads/video/{videoId}/status", s.withAuth(s.coursesHandler.GetVideoStatus))
	s.mux.HandleFunc("POST /v1/uploads/file", s.withAuth(s.coursesHandler.UploadFile))

	// Enrollment endpoints (require JWT + student role)
	s.mux.HandleFunc("POST /v1/enrollments", s.withAuthAndRole("student", s.enrollmentsHandler.CreateEnrollment, nil))
	s.mux.HandleFunc("GET /v1/student/enrollments", s.withAuthAndRole("student", nil, s.enrollmentsHandler.ListStudentEnrollments))

	// Streaming endpoints (require JWT)
	s.mux.HandleFunc("GET /v1/stream/lessons/{lessonId}/signed-url", s.withAuth(s.enrollmentsHandler.GetStreamingSignedURL))

	// Progress tracking endpoints (require JWT + student role)
	s.mux.HandleFunc("POST /v1/enrollments/{courseId}/lessons/{lessonId}/progress", s.withAuthAndRole("student", s.enrollmentsHandler.UpdateLessonProgress, nil))
	s.mux.HandleFunc("GET /v1/enrollments/{courseId}/lessons/{lessonId}/progress", s.withAuthAndRole("student", nil, s.enrollmentsHandler.GetLessonProgress))

	// Teacher quiz endpoints (require JWT + teacher role)
	s.mux.HandleFunc("POST /v1/teacher/courses/{courseId}/quizzes", s.withAuthAndRole("teacher", s.assessmentsHandler.CreateQuiz, nil))
	s.mux.HandleFunc("GET /v1/teacher/courses/{courseId}/quizzes", s.withAuthAndRole("teacher", nil, s.assessmentsHandler.GetTeacherQuizzes))
	s.mux.HandleFunc("GET /v1/teacher/quizzes/{quizId}", s.withAuthAndRole("teacher", nil, s.assessmentsHandler.GetTeacherQuiz))
	s.mux.HandleFunc("PATCH /v1/teacher/quizzes/{quizId}", s.withAuthAndRole("teacher", s.assessmentsHandler.UpdateTeacherQuiz, nil))
	s.mux.HandleFunc("DELETE /v1/teacher/quizzes/{quizId}", s.withAuthAndRole("teacher", s.assessmentsHandler.DeleteTeacherQuiz, nil))
	s.mux.HandleFunc("POST /v1/teacher/quizzes/{quizId}/questions", s.withAuthAndRole("teacher", s.assessmentsHandler.CreateTeacherQuestion, nil))
	s.mux.HandleFunc("PATCH /v1/teacher/questions/{questionId}", s.withAuthAndRole("teacher", s.assessmentsHandler.UpdateTeacherQuestion, nil))
	s.mux.HandleFunc("DELETE /v1/teacher/questions/{questionId}", s.withAuthAndRole("teacher", s.assessmentsHandler.DeleteTeacherQuestion, nil))

	// Teacher assignment endpoints (require JWT + teacher role)
	s.mux.HandleFunc("POST /v1/teacher/courses/{courseId}/assignments", s.withAuthAndRole("teacher", s.assessmentsHandler.CreateAssignment, nil))
	s.mux.HandleFunc("GET /v1/teacher/courses/{courseId}/assignments", s.withAuthAndRole("teacher", nil, s.assessmentsHandler.ListTeacherCourseAssignments))
	s.mux.HandleFunc("GET /v1/teacher/assignments/{assignmentId}", s.withAuthAndRole("teacher", nil, s.assessmentsHandler.GetTeacherAssignment))
	s.mux.HandleFunc("PATCH /v1/teacher/assignments/{assignmentId}", s.withAuthAndRole("teacher", s.assessmentsHandler.UpdateTeacherAssignment, nil))
	s.mux.HandleFunc("GET /v1/teacher/assignments/{assignmentId}/submissions", s.withAuthAndRole("teacher", nil, s.assessmentsHandler.ListTeacherAssignmentSubmissions))
	s.mux.HandleFunc("GET /v1/teacher/assignments/{assignmentId}/submissions/{submissionId}", s.withAuthAndRole("teacher", nil, s.assessmentsHandler.GetTeacherAssignmentSubmission))
	s.mux.HandleFunc("POST /v1/teacher/assignments/{assignmentId}/submissions/{submissionId}/grade", s.withAuthAndRole("teacher", s.assessmentsHandler.GradeSubmission, nil))

	// Student quiz endpoints (require JWT + student role)
	s.mux.HandleFunc("GET /v1/student/assessments", s.withAuthAndRole("student", nil, s.assessmentsHandler.ListStudentQuizzes))
	s.mux.HandleFunc("GET /v1/student/assessments/attempts/{attemptId}", s.withAuthAndRole("student", nil, s.assessmentsHandler.GetStudentAttempt))
	s.mux.HandleFunc("GET /v1/student/assessments/{quizId}", s.withAuthAndRole("student", nil, s.assessmentsHandler.GetStudentQuizDetail))
	s.mux.HandleFunc("GET /v1/student/assessments/{quizId}/attempts/{attemptId}", s.withAuthAndRole("student", nil, s.assessmentsHandler.GetStudentQuizAttemptResult))
	s.mux.HandleFunc("POST /v1/quizzes/{quizId}/attempts", s.withAuthAndRole("student", s.assessmentsHandler.StartQuizAttempt, nil))
	s.mux.HandleFunc("PATCH /v1/quizzes/{quizId}/attempts/{attemptId}", s.withAuthAndRole("student", s.assessmentsHandler.SaveQuizAttemptAnswers, nil))
	s.mux.HandleFunc("POST /v1/quizzes/{quizId}/attempts/{attemptId}/submit", s.withAuthAndRole("student", s.assessmentsHandler.SubmitQuizAttempt, nil))

	// Student assignment endpoints (require JWT + student role)
	s.mux.HandleFunc("GET /v1/student/assignments", s.withAuthAndRole("student", nil, s.assessmentsHandler.ListStudentAssignments))
	s.mux.HandleFunc("GET /v1/student/assignments/{assignmentId}", s.withAuthAndRole("student", nil, s.assessmentsHandler.GetStudentAssignmentDetail))
	s.mux.HandleFunc("POST /v1/assignments/{assignmentId}/submissions", s.withAuthAndRole("student", s.assessmentsHandler.SubmitAssignment, nil))

	// Points endpoints
	// Student: GET /v1/student/points, GET /v1/student/points/history, PATCH /v1/student/leaderboard/opt-out
	s.mux.HandleFunc("GET /v1/student/points", s.withAuthAndRole("student", nil, s.pointsHandler.GetStudentPoints))
	s.mux.HandleFunc("GET /v1/student/points/history", s.withAuthAndRole("student", nil, s.pointsHandler.GetPointsHistory))
	s.mux.HandleFunc("PATCH /v1/student/leaderboard/opt-out", s.withAuthAndRole("student", s.pointsHandler.ToggleLeaderboardOptOut, nil))

	// Leaderboard: any authenticated user
	s.mux.HandleFunc("GET /v1/leaderboard", s.withAuth(s.pointsHandler.GetLeaderboard))

	// Admin: PATCH /v1/admin/points/config
	s.mux.HandleFunc("GET /v1/admin/points/config", s.withAuthAndRole("admin", nil, s.pointsHandler.GetPointsConfig))
	s.mux.HandleFunc("PATCH /v1/admin/points/config", s.withAuthAndRole("admin", s.pointsHandler.UpdatePointsConfig, nil))

	// Certificate endpoints
	// Student: GET /v1/student/certificates/:courseId (requires JWT + student role)
	s.mux.HandleFunc("GET /v1/student/certificates/{courseId}", s.withAuthAndRole("student", nil, s.certificatesHandler.GetStudentCertificate))

	// Public: GET /v1/certificates/verify/:verificationId (no auth required)
	s.mux.HandleFunc("GET /v1/certificates/verify/{verificationId}", s.certificatesHandler.VerifyCertificate)

	// ─── Bookshop routes ──────────────────────────────────────────────────────
	// Public: GET /v1/bookshop/books, GET /v1/bookshop/books/:bookId/preview
	s.mux.HandleFunc("GET /v1/bookshop/books", s.bookshopHandler.ListBooks)
	s.mux.HandleFunc("GET /v1/bookshop/books/{bookId}", s.bookshopHandler.GetBookDetail)
	s.mux.HandleFunc("GET /v1/bookshop/books/{bookId}/preview", s.withAuth(s.bookshopHandler.GetBookPreview))

	// Student: digital access, bookmark, orders
	s.mux.HandleFunc("GET /v1/bookshop/cart", s.withAuthAndRole("student", nil, s.bookshopHandler.GetCart))
	s.mux.HandleFunc("POST /v1/bookshop/cart/items", s.withAuthAndRole("student", s.bookshopHandler.AddCartItem, nil))
	s.mux.HandleFunc("PATCH /v1/bookshop/cart/items/{itemId}", s.withAuthAndRole("student", s.bookshopHandler.UpdateCartItem, nil))
	s.mux.HandleFunc("DELETE /v1/bookshop/cart/items/{itemId}", s.withAuthAndRole("student", s.bookshopHandler.RemoveCartItem, nil))
	s.mux.HandleFunc("POST /v1/bookshop/orders", s.withAuthAndRole("student", s.bookshopHandler.PlaceOrder, nil))
	s.mux.HandleFunc("POST /v1/bookshop/checkout", s.withAuthAndRole("student", s.bookshopHandler.Checkout, nil))
	s.mux.HandleFunc("GET /v1/student/bookshop/reader/{bookId}/access", s.withAuthAndRole("student", nil, s.bookshopHandler.GetDigitalBookAccess))
	s.mux.HandleFunc("POST /v1/student/bookshop/reader/{bookId}/bookmark", s.withAuthAndRole("student", s.bookshopHandler.UpsertBookmark, nil))
	s.mux.HandleFunc("GET /v1/student/bookshop/library", s.withAuthAndRole("student", nil, s.bookshopHandler.ListStudentLibrary))
	s.mux.HandleFunc("GET /v1/student/bookshop/orders", s.withAuthAndRole("student", nil, s.bookshopHandler.ListStudentOrders))
	s.mux.HandleFunc("GET /v1/student/bookshop/orders/{orderId}", s.withAuthAndRole("student", nil, s.bookshopHandler.GetStudentOrder))

	// Admin: book management, order fulfilment, refunds
	s.mux.HandleFunc("GET /v1/admin/bookshop/books", s.withAuthAndRole("admin", nil, s.bookshopHandler.ListAdminBooks))
	s.mux.HandleFunc("POST /v1/admin/bookshop/books", s.withAuthAndRole("admin", s.bookshopHandler.CreateBook, nil))
	s.mux.HandleFunc("PATCH /v1/admin/bookshop/books/{bookId}", s.withAuthAndRole("admin", s.bookshopHandler.UpdateBook, nil))
	s.mux.HandleFunc("POST /v1/admin/bookshop/books/{bookId}/cover", s.withAuthAndRole("admin", s.bookshopHandler.UploadBookCover, nil))
	s.mux.HandleFunc("PATCH /v1/admin/bookshop/orders/{orderId}", s.withAuthAndRole("admin", s.bookshopHandler.FulfilOrder, nil))
	s.mux.HandleFunc("GET /v1/admin/bookshop/orders", s.withAuthAndRole("admin", nil, s.bookshopHandler.ListAdminOrders))
	s.mux.HandleFunc("GET /v1/admin/bookshop/refunds", s.withAuthAndRole("admin", nil, s.bookshopHandler.ListRefunds))
	// Idempotency middleware applied to refunds (Requirements: 20.4)
	s.mux.HandleFunc("POST /v1/admin/bookshop/refunds", s.withAuthAndRole("admin",
		middleware.Idempotency(s.idempotencyStore)(http.HandlerFunc(s.bookshopHandler.ProcessRefund)).ServeHTTP, nil))

	// ─── Forum routes ─────────────────────────────────────────────────────────
	// Public: GET /v1/forum/posts, GET /v1/forum/posts/:postId/comments
	s.mux.HandleFunc("GET /v1/forum/posts", s.forumHandler.ListPosts)
	s.mux.HandleFunc("GET /v1/forum/posts/{postId}", s.forumHandler.GetPost)
	s.mux.HandleFunc("GET /v1/forum/posts/{postId}/comments", s.forumHandler.ListComments)

	// Authenticated: post CRUD, comment CRUD, upvote, flag
	s.mux.HandleFunc("POST /v1/forum/posts", s.withAuth(s.forumHandler.CreatePost))
	s.mux.HandleFunc("PATCH /v1/forum/posts/{postId}", s.withAuth(s.forumHandler.UpdatePost))
	s.mux.HandleFunc("DELETE /v1/forum/posts/{postId}", s.withAuth(s.forumHandler.DeletePost))
	s.mux.HandleFunc("POST /v1/forum/posts/{postId}/comments", s.withAuth(s.forumHandler.CreateComment))
	s.mux.HandleFunc("PATCH /v1/forum/posts/{postId}/comments/{commentId}", s.withAuth(s.forumHandler.UpdateComment))
	s.mux.HandleFunc("DELETE /v1/forum/posts/{postId}/comments/{commentId}", s.withAuth(s.forumHandler.DeleteComment))
	s.mux.HandleFunc("POST /v1/forum/posts/{postId}/upvote", s.withAuth(s.forumHandler.ToggleUpvote))
	s.mux.HandleFunc("POST /v1/forum/posts/{postId}/flag", s.withAuth(s.forumHandler.FlagPost))

	// Admin: moderation queue and actions
	s.mux.HandleFunc("GET /v1/admin/moderation", s.withAuthAndRole("admin", nil, s.forumHandler.GetModerationQueue))
	s.mux.HandleFunc("POST /v1/admin/moderation/{flagId}/action", s.withAuthAndRole("admin", s.forumHandler.ModerateContent, nil))
	s.mux.HandleFunc("GET /v1/admin/forum/posts", s.withAuthAndRole("admin", nil, s.forumHandler.ListPostsForReview))
	s.mux.HandleFunc("POST /v1/admin/forum/posts/{postId}/approve", s.withAuthAndRole("admin", s.forumHandler.ApprovePost, nil))
	s.mux.HandleFunc("POST /v1/admin/forum/posts/{postId}/reject", s.withAuthAndRole("admin", s.forumHandler.RejectPost, nil))

	// ─── Notifications routes ─────────────────────────────────────────────────
	// Authenticated: list, mark read, mark all read
	s.mux.HandleFunc("GET /v1/notifications", s.withAuth(s.notificationsHandler.ListNotifications))
	s.mux.HandleFunc("PATCH /v1/notifications/{notificationId}/read", s.withAuth(s.notificationsHandler.MarkRead))
	s.mux.HandleFunc("PATCH /v1/notifications/read-all", s.withAuth(s.notificationsHandler.MarkAllRead))

	// Admin: list and update templates, send broadcast
	s.mux.HandleFunc("GET /v1/admin/notifications/templates", s.withAuthAndRole("admin", nil, s.notificationsHandler.ListTemplates))
	s.mux.HandleFunc("PATCH /v1/admin/notifications/templates/{templateId}", s.withAuthAndRole("admin", s.notificationsHandler.UpdateTemplate, nil))
	s.mux.HandleFunc("POST /v1/admin/notifications/broadcast", s.withAuthAndRole("admin", s.notificationsHandler.SendBroadcast, nil))
	s.mux.HandleFunc("GET /v1/admin/notifications/broadcasts", s.withAuthAndRole("admin", nil, s.notificationsHandler.ListBroadcasts))

	// ─── Analytics routes ─────────────────────────────────────────────────────
	// Admin: platform-wide and per-course analytics (Requirement 23.1–23.4)
	s.mux.HandleFunc("GET /v1/admin/analytics/overview", s.withAuthAndRole("admin", nil, s.analyticsHandler.GetAdminOverview))
	s.mux.HandleFunc("GET /v1/admin/analytics/courses/{courseId}", s.withAuthAndRole("admin", nil, s.analyticsHandler.GetCourseAnalytics))
	s.mux.HandleFunc("GET /v1/admin/analytics/courses/{courseId}/students", s.withAuthAndRole("admin", nil, s.analyticsHandler.GetCourseStudents))
	s.mux.HandleFunc("GET /v1/admin/analytics/students/{studentId}", s.withAuthAndRole("admin", nil, s.analyticsHandler.GetStudentAnalytics))
	s.mux.HandleFunc("GET /v1/admin/stats", s.withAuthAndRole("admin", nil, s.analyticsHandler.GetAdminStats))
	s.mux.HandleFunc("GET /v1/admin/analytics/courses", s.withAuthAndRole("admin", nil, s.analyticsHandler.ListCoursesAnalytics))
	s.mux.HandleFunc("GET /v1/admin/analytics/students", s.withAuthAndRole("admin", nil, s.analyticsHandler.ListStudentsAnalytics))
	s.mux.HandleFunc("GET /v1/student/dashboard", s.withAuthAndRole("student", nil, s.analyticsHandler.GetStudentDashboard))

	// Teacher: scoped exclusively to own courses (Requirement 23.5)
	s.mux.HandleFunc("GET /v1/teacher/analytics", s.withAuthAndRole("teacher", nil, s.analyticsHandler.GetTeacherAnalytics))
	s.mux.HandleFunc("GET /v1/teacher/courses/{courseId}/students", s.withAuthAndRole("teacher", nil, s.analyticsHandler.GetTeacherCourseStudents))
	s.mux.HandleFunc("GET /v1/teacher/analytics/students/{studentId}", s.withAuthAndRole("teacher", nil, s.analyticsHandler.GetTeacherStudentAnalytics))

	// ─── Purchase approval routes ─────────────────────────────────────────────
	s.mux.HandleFunc("POST /v1/purchase-requests", s.withAuthAndRole("student", s.paymentsHandler.CreatePurchaseRequest, nil))
	s.mux.HandleFunc("GET /v1/student/purchase-requests", s.withAuthAndRole("student", nil, s.paymentsHandler.ListMyPurchaseRequests))
	s.mux.HandleFunc("GET /v1/admin/purchase-requests", s.withAuthAndRole("admin", nil, s.paymentsHandler.ListAdminPurchaseRequests))
	s.mux.HandleFunc("GET /v1/admin/purchase-requests/export", s.withAuthAndRole("admin", nil, s.paymentsHandler.ExportAdminPurchaseRequests))
	s.mux.HandleFunc("POST /v1/admin/purchase-requests/{requestId}/approve", s.withAuthAndRole("admin", s.paymentsHandler.ApprovePurchaseRequest, nil))
	s.mux.HandleFunc("POST /v1/admin/purchase-requests/{requestId}/reject", s.withAuthAndRole("admin", s.paymentsHandler.RejectPurchaseRequest, nil))

	// Compatibility routes for student requests and admin approvals
	s.mux.HandleFunc("GET /v1/admin/approvals", s.withAuthAndRole("admin", nil, s.paymentsHandler.ListApprovalsCompatibility))
	s.mux.HandleFunc("POST /v1/admin/approvals/{id}/approve", s.withAuthAndRole("admin", s.paymentsHandler.ApproveApprovalCompatibility, nil))
	s.mux.HandleFunc("POST /v1/admin/approvals/{id}/reject", s.withAuthAndRole("admin", s.paymentsHandler.RejectApprovalCompatibility, nil))
	s.mux.HandleFunc("POST /v1/student/requests", s.withAuthAndRole("student", s.paymentsHandler.CreatePurchaseRequestCompatibility, nil))
	s.mux.HandleFunc("GET /v1/student/requests", s.withAuthAndRole("student", nil, s.paymentsHandler.ListMyRequestsCompatibility))

	// ─── System Config routes ─────────────────────────────────────────────────
	// Admin: GET/PATCH settings, history, rollback (Requirements: 25.1–25.4)
	s.mux.HandleFunc("GET /v1/admin/system/settings", s.withAuthAndRole("admin", nil, s.systemConfigHandler.GetSettings))
	s.mux.HandleFunc("PATCH /v1/admin/system/settings", s.withAuthAndRole("admin", s.systemConfigHandler.UpdateSettings, nil))
	s.mux.HandleFunc("GET /v1/admin/system/settings/history", s.withAuthAndRole("admin", nil, s.systemConfigHandler.GetSettingsHistory))
	s.mux.HandleFunc("POST /v1/admin/system/settings/rollback/{historyId}", s.withAuthAndRole("admin", s.systemConfigHandler.RollbackSettings, nil))
	s.mux.HandleFunc("GET /v1/admin/system/health", s.withAuthAndRole("admin", nil, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","db_status":"ok","queue_depth":0,"worker_count":0,"cache_hit_rate":0}`))
	}))

	// ─── Live Sessions routes ─────────────────────────────────────────────────
	// Teacher: schedule, start, end, reschedule/cancel, attendance (Requirements: 16.1, 16.3)
	s.mux.HandleFunc("GET /v1/teacher/live-sessions", s.withAuthAndRole("teacher", nil, s.liveSessionsHandler.ListTeacherSessions))
	s.mux.HandleFunc("POST /v1/teacher/live-sessions", s.withAuthAndRole("teacher", s.liveSessionsHandler.ScheduleSession, nil))
	s.mux.HandleFunc("GET /v1/teacher/live-sessions/{sessionId}", s.withAuthAndRole("teacher", nil, s.liveSessionsHandler.GetTeacherSession))
	s.mux.HandleFunc("POST /v1/teacher/live-sessions/{sessionId}/start", s.withAuthAndRole("teacher", s.liveSessionsHandler.StartSession, nil))
	s.mux.HandleFunc("POST /v1/teacher/live-sessions/{sessionId}/end", s.withAuthAndRole("teacher", s.liveSessionsHandler.EndSession, nil))
	s.mux.HandleFunc("PATCH /v1/teacher/live-sessions/{sessionId}", s.withAuthAndRole("teacher", s.liveSessionsHandler.RescheduleOrCancelSession, nil))
	s.mux.HandleFunc("GET /v1/teacher/live-sessions/{sessionId}/attendance", s.withAuthAndRole("teacher", nil, s.liveSessionsHandler.GetAttendance))
	s.mux.HandleFunc("GET /v1/teacher/live-sessions/{sessionId}/chat", s.withAuthAndRole("teacher", nil, s.liveSessionsHandler.ListChat))
	s.mux.HandleFunc("POST /v1/teacher/live-sessions/{sessionId}/chat", s.withAuthAndRole("teacher", s.liveSessionsHandler.PostChat, nil))
	s.mux.HandleFunc("GET /v1/teacher/live-sessions/{sessionId}/participants", s.withAuthAndRole("teacher", nil, s.liveSessionsHandler.ListParticipants))
	s.mux.HandleFunc("POST /v1/teacher/live-sessions/{sessionId}/participants/{participantId}/mute", s.withAuthAndRole("teacher", s.liveSessionsHandler.MuteParticipant, nil))
	s.mux.HandleFunc("DELETE /v1/teacher/live-sessions/{sessionId}/participants/{participantId}", s.withAuthAndRole("teacher", s.liveSessionsHandler.RemoveParticipant, nil))
	s.mux.HandleFunc("POST /v1/teacher/live-sessions/{sessionId}/participants/{participantId}/lower-hand", s.withAuthAndRole("teacher", s.liveSessionsHandler.LowerHand, nil))
	s.mux.HandleFunc("POST /v1/teacher/live-sessions/{sessionId}/recording", s.withAuthAndRole("teacher", s.liveSessionsHandler.SetRecording, nil))

	// Student: join (Requirements: 16.3)
	s.mux.HandleFunc("GET /v1/student/live-sessions", s.withAuthAndRole("student", nil, s.liveSessionsHandler.ListStudentSessions))
	s.mux.HandleFunc("GET /v1/student/live-sessions/{sessionId}", s.withAuthAndRole("student", nil, s.liveSessionsHandler.GetStudentSession))
	s.mux.HandleFunc("POST /v1/live-sessions/{sessionId}/join", s.withAuthAndRole("student", s.liveSessionsHandler.JoinSession, nil))

	// ─── RBAC routes ──────────────────────────────────────────────────────────
	// Admin: list roles, update role permissions (Requirements: 9.2, 9.3)
	s.mux.HandleFunc("GET /v1/admin/rbac/roles", s.withAuthAndRole("admin", nil, s.rbacHandler.ListRoles))
	s.mux.HandleFunc("PATCH /v1/admin/rbac/roles/{roleId}", s.withAuthAndRole("admin", s.rbacHandler.UpdateRolePermissions, nil))

	// ─── Audit Log routes ─────────────────────────────────────────────────────
	// Admin: paginated, filterable audit log (Requirements: 9.5)
	s.mux.HandleFunc("GET /v1/admin/audit-logs", s.withAuthAndRole("admin", nil, s.auditHandler.ListAuditLogs))

	// ─── Global Search route ──────────────────────────────────────────────────
	// Public/authenticated: search across courses, lessons, forum, books (Requirements: 26.1–26.5)
	s.mux.HandleFunc("GET /v1/search", s.searchHandler.Search)

	// API version prefix - all routes under /v1/
	s.mux.HandleFunc("/v1/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"NOT_FOUND","message":"Endpoint not found"}}`))
	})
}

// withAuth wraps a handler with JWT authentication middleware
func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authMiddleware := middleware.NewAuthenticateMiddleware(s.jwtService)
		authMiddleware.Authenticate(http.HandlerFunc(next)).ServeHTTP(w, r)
	}
}

// withAuthAndRole wraps handlers with JWT authentication and role authorization middleware
// It routes based on HTTP method (POST/PUT -> writeHandler, GET -> readHandler)
func (s *Server) withAuthAndRole(requiredRole string, writeHandler, readHandler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authMiddleware := middleware.NewAuthenticateMiddleware(s.jwtService)
		authorizeMiddleware := middleware.Authorize(requiredRole)

		// Choose handler based on method
		var handler http.HandlerFunc
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodPatch || r.Method == http.MethodDelete {
			if writeHandler == nil {
				http.NotFound(w, r)
				return
			}
			handler = writeHandler
		} else {
			if readHandler == nil {
				http.NotFound(w, r)
				return
			}
			handler = readHandler
		}

		// Apply middleware chain
		finalHandler := authMiddleware.Authenticate(authorizeMiddleware(http.HandlerFunc(handler)))
		finalHandler.ServeHTTP(w, r)
	}
}

// handleAdminUserRoutes handles all /v1/admin/users/:userId/* routes
func (s *Server) handleAdminUserRoutes(w http.ResponseWriter, r *http.Request) {
	authMiddleware := middleware.NewAuthenticateMiddleware(s.jwtService)
	authorizeMiddleware := middleware.Authorize("admin")

	// Determine which handler to use based on the path
	path := r.URL.Path

	var handler http.HandlerFunc
	if r.Method == http.MethodPost && len(path) > len("/v1/admin/users/") {
		// Check if it's force-password-reset
		if len(path) > len("/v1/admin/users/") &&
			path[len(path)-len("/force-password-reset"):] == "/force-password-reset" {
			handler = s.usersHandler.ForcePasswordReset
		} else if len(path) > len("/v1/admin/users/") &&
			path[len(path)-len("/impersonate"):] == "/impersonate" {
			handler = s.authHandler.ImpersonateUser
		} else {
			http.NotFound(w, r)
			return
		}
	} else if r.Method == http.MethodPatch && len(path) > len("/v1/admin/users/") {
		// Check if it's student-profile update
		if len(path) > len("/v1/admin/users/") &&
			path[len(path)-len("/student-profile"):] == "/student-profile" {
			handler = s.usersHandler.UpdateStudentProfile
		} else {
			handler = s.usersHandler.UpdateUser
		}
	} else if r.Method == http.MethodGet {
		handler = s.usersHandler.GetUser
	} else {
		http.NotFound(w, r)
		return
	}

	// Apply middleware chain
	finalHandler := authMiddleware.Authenticate(authorizeMiddleware(http.HandlerFunc(handler)))
	finalHandler.ServeHTTP(w, r)
}
