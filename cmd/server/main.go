package main

import (
	"context"
	"fmt"
	"lms-backend/configs"
	"lms-backend/internal/application/analytics"
	"lms-backend/internal/application/assessments"
	appaudit "lms-backend/internal/application/audit"
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
	domainenrollments "lms-backend/internal/domain/enrollments"
	infraaudit "lms-backend/internal/infrastructure/audit"
	"lms-backend/internal/infrastructure/email"
	"lms-backend/internal/infrastructure/jwt"
	"lms-backend/internal/infrastructure/oauth"
	"lms-backend/internal/infrastructure/postgres"
	"lms-backend/internal/infrastructure/redis"
	"lms-backend/internal/infrastructure/rustfs"
	tsclient "lms-backend/internal/infrastructure/typesense"
	"lms-backend/internal/infrastructure/workers"
	httpServer "lms-backend/internal/interfaces/http"
	"lms-backend/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"

	_ "lms-backend/docs"
)

// @title           LMS Backend API
// @version         1.0.0
// @description     REST API for the LMS platform covering auth, courses, enrollments, assessments, payments, and more.
// @host            localhost:8080
// @BasePath        /
// @schemes         http https
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := configs.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Get("http://127.0.0.1:" + cfg.ServerPort + "/health")
		if err != nil {
			fmt.Fprintf(os.Stderr, "healthcheck failed: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			fmt.Fprintf(os.Stderr, "healthcheck failed: unexpected status %d\n", resp.StatusCode)
			os.Exit(1)
		}
		os.Exit(0)
	}

	logger.Info(ctx, "Starting LMS Backend Server")

	// Initialize PostgreSQL
	db, err := postgres.NewDB(cfg.DatabaseDSN)
	if err != nil {
		logger.Error(ctx, "Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info(ctx, "Connected to PostgreSQL")

	// Run migrations
	if err := postgres.RunMigrations(db, "migrations"); err != nil {
		logger.Error(ctx, "Failed to run migrations", "error", err)
		os.Exit(1)
	}
	logger.Info(ctx, "Migrations applied successfully")

	// Initialize Redis
	redisClient, err := redis.NewClient(cfg.RedisURL)
	if err != nil {
		logger.Error(ctx, "Failed to connect to Redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()
	logger.Info(ctx, "Connected to Redis")

	// Initialize Typesense client
	tsClient, err := tsclient.NewClient(cfg.TypesenseHost, cfg.TypesensePort, cfg.TypesenseAPIKey)
	if err != nil {
		logger.Error(ctx, "Failed to initialize Typesense client", "error", err)
		os.Exit(1)
	}
	if err := tsClient.EnsureCollections(ctx); err != nil {
		logger.Error(ctx, "Failed to ensure Typesense collections", "error", err)
		os.Exit(1)
	}
	logger.Info(ctx, "Typesense client initialized")

	// Initialize RustFS
	rustfsClient, err := rustfs.NewClient(
		cfg.RustFSEndpoint,
		cfg.RustFSAccessKey,
		cfg.RustFSSecretKey,
		cfg.RustFSRegion,
	)
	if err != nil {
		logger.Error(ctx, "Failed to initialize RustFS client", "error", err)
		os.Exit(1)
	}
	logger.Info(ctx, "RustFS client initialized")

	// Initialize JWT service
	jwtService, err := jwt.NewJWTService(cfg.JWTPrivateKeyPath, cfg.JWTPublicKeyPath, cfg.JWTIssuer)
	if err != nil {
		logger.Error(ctx, "Failed to initialize JWT service", "error", err)
		os.Exit(1)
	}
	logger.Info(ctx, "JWT service initialized")

	// Initialize repositories
	userRepo := postgres.NewUserRepository(db)
	userSettingsRepo := postgres.NewUserSettingsRepository(db)
	otpRepo := postgres.NewOTPRepository(db)
	passwordResetRepo := postgres.NewPasswordResetRepository(db)
	oauthProviderRepo := postgres.NewOAuthProviderRepository(db)
	studentProfileRepo := postgres.NewStudentProfileRepository(db)
	tokenStore := redis.NewTokenStore(redisClient)
	logger.Info(ctx, "Repositories initialized")

	// Initialize infrastructure services
	emailService := email.NewService(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPFrom)
	auditLogger := infraaudit.NewLogger(db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	logger.Info(ctx, "Infrastructure services initialized")

	// Initialize OAuth providers
	oauthProviders := make(map[string]oauth.Provider)
	if cfg.GoogleClientID != "" {
		oauthProviders["google"] = oauth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	}
	if cfg.GitHubClientID != "" {
		oauthProviders["github"] = oauth.NewGitHubProvider(cfg.GitHubClientID, cfg.GitHubClientSecret, cfg.GitHubRedirectURL)
	}
	if cfg.MicrosoftClientID != "" {
		oauthProviders["microsoft"] = oauth.NewMicrosoftProvider(cfg.MicrosoftClientID, cfg.MicrosoftClientSecret, cfg.MicrosoftRedirectURL)
	}
	logger.Info(ctx, "OAuth providers initialized", "count", len(oauthProviders))

	// Initialize OAuth factory
	oauthFactory := oauth.NewFactory(oauthProviders)
	tokenEncryptor := oauth.NewTokenEncryptor(cfg.OAuthTokenEncryptionKey)

	// Initialize auth service
	authService := auth.NewService(auth.ServiceDeps{
		UserRepo:          userRepo,
		UserSettingsRepo:  userSettingsRepo,
		OTPRepo:           otpRepo,
		PasswordResetRepo: passwordResetRepo,
		OAuthProviderRepo: oauthProviderRepo,
		TokenStore:        tokenStore,
		JWTService:        jwtService,
		OAuthFactory:      &oauthFactoryAdapter{factory: oauthFactory},
		TokenEncryptor:    tokenEncryptor,
		EmailService:      emailService,
		RedisClient:       redisClient,
		AuditLogger:       auditLogger,
		FrontendBaseURL:   cfg.FrontendBaseURL,
		AdminDevBypass:    cfg.AdminDevBypass,
		AdminDevOTP:       cfg.AdminDevOTP,
	})
	logger.Info(ctx, "Auth service initialized")

	// Initialize notification queue (placeholder for now)
	notificationQueue := workers.NewNotificationQueue(redisClient)

	// Initialize a general-purpose job queue for background jobs (e.g. certificate PDF generation)
	jobQueue := workers.NewRedisQueue(redisClient, "queue:jobs")

	// Initialize users service
	usersService := users.NewService(users.ServiceDeps{
		UserRepo:          userRepo,
		ProfileRepo:       studentProfileRepo,
		TokenStore:        tokenStore,
		EmailService:      emailService,
		NotificationQueue: notificationQueue,
		AuditLogger:       auditLogger,
	})
	logger.Info(ctx, "Users service initialized")

	// Initialize courses repositories
	courseRepo := postgres.NewCourseRepository(db)
	moduleRepo := postgres.NewModuleRepository(db)
	chapterRepo := postgres.NewChapterRepository(db)
	lessonRepo := postgres.NewLessonRepository(db)
	videoRepo := postgres.NewVideoRepository(db)
	courseReviewRepo := postgres.NewCourseReviewRepository(db)
	logger.Info(ctx, "Courses repositories initialized")

	// Initialize enrollments repositories
	enrollmentRepo := postgres.NewEnrollmentRepository(db)
	lessonProgressRepo := postgres.NewLessonProgressRepository(db)
	signingKeyStore := redis.NewSigningKeyStore(redisClient)
	logger.Info(ctx, "Enrollments repositories initialized")

	// Initialize courses service
	coursesService := courses.NewServiceWithUploadDeps(
		courseRepo,
		moduleRepo,
		chapterRepo,
		lessonRepo,
		videoRepo,
		courseReviewRepo,
		tsClient,
		rustfsClient,
		jobQueue,
		cfg.RustFSVideoBucket,
		cfg.RustFSFilesBucket,
	)
	logger.Info(ctx, "Courses service initialized")

	// Initialize certificates repository and service before enrollments so course
	// completion can issue certificates immediately.
	certRepo := postgres.NewCertificateRepository(db)
	certificatesService := certificates.NewService(
		certRepo,
		jobQueue,
		rustfsClient,
		nil, // PDFRenderer: nil for now (async PDF generation is best-effort)
		cfg.RustFSCertificatesBucket,
	)
	logger.Info(ctx, "Certificates service initialized")

	// Initialize enrollments service
	enrollmentsService := enrollments.NewService(
		enrollmentRepo,
		lessonProgressRepo,
		courseRepo,
		lessonRepo,
		videoRepo,
		userRepo,
		signingKeyStore,
		rustfsClient,
		cfg.RustFSVideoBucket,
		certificatesService,
	)
	logger.Info(ctx, "Enrollments service initialized")

	// Initialize assessments repositories
	quizRepo := postgres.NewQuizRepository(db)
	questionRepo := postgres.NewQuestionRepository(db)
	questionOptionRepo := postgres.NewQuestionOptionRepository(db)
	quizAttemptRepo := postgres.NewQuizAttemptRepository(db)
	assignmentRepo := postgres.NewAssignmentRepository(db)
	assignmentSubmissionRepo := postgres.NewAssignmentSubmissionRepository(db)
	submissionFileRepo := postgres.NewSubmissionFileRepository(db)
	submissionGradeRepo := postgres.NewSubmissionGradeRepository(db)
	logger.Info(ctx, "Assessments repositories initialized")

	// Initialize assessments service
	assessmentsService := assessments.NewService(
		quizRepo,
		questionRepo,
		questionOptionRepo,
		quizAttemptRepo,
		assignmentRepo,
		assignmentSubmissionRepo,
		submissionFileRepo,
		submissionGradeRepo,
		enrollmentRepo,
		courseRepo,
		rustfsClient,
		notificationQueue,
		cfg.RustFSFilesBucket,
	)
	logger.Info(ctx, "Assessments service initialized")

	// Initialize points repositories and service
	pointEventRepo := postgres.NewPointEventRepository(db)
	pointsConfigRepo := postgres.NewPointsConfigRepository(db)
	pointsRankRepo := postgres.NewPointsRankRepository(db)
	leaderboardStore := redis.NewLeaderboardStore(redisClient)
	nameResolver := postgres.NewStudentNameResolver(db)
	pointsService := points.NewService(
		pointEventRepo,
		pointsConfigRepo,
		pointsRankRepo,
		leaderboardStore,
		nameResolver,
		auditLogger,
	)
	logger.Info(ctx, "Points service initialized")

	// Initialize bookshop repositories and service
	bookRepo := postgres.NewBookRepository(db)
	orderRepo := postgres.NewOrderRepository(db)
	bookmarkRepo := postgres.NewBookBookmarkRepository(db)
	bookshopIdempotency := redis.NewIdempotencyStore(redisClient)
	bookshopService := bookshop.NewService(
		bookRepo,
		orderRepo,
		bookmarkRepo,
		jobQueue,
		rustfsClient,
		auditLogger,
		bookshopIdempotency,
		cfg.RustFSBooksBucket,
		tsClient,
	)
	logger.Info(ctx, "Bookshop service initialized")

	// Initialize forum repositories and service
	forumPostRepo := postgres.NewForumPostRepository(db)
	forumCommentRepo := postgres.NewForumCommentRepository(db)
	forumUpvoteRepo := postgres.NewPostUpvoteRepository(db)
	forumFlagRepo := postgres.NewContentFlagRepository(db)
	forumPostReviewRepo := postgres.NewForumPostReviewRepository(db)
	forumService := forum.NewServiceWithReviewRepo(
		forumPostRepo,
		forumCommentRepo,
		forumUpvoteRepo,
		forumFlagRepo,
		forumPostReviewRepo,
		jobQueue,
		auditLogger,
		nil, // UserDeactivator: wired via usersService adapter in a later phase
		tsClient,
	)
	logger.Info(ctx, "Forum service initialized")

	// Initialize notifications repositories and service
	notifRepo := postgres.NewNotificationRepository(db)
	notifTemplateRepo := postgres.NewNotificationTemplateRepository(db)
	notificationsService := notifications.NewService(notifRepo, notifTemplateRepo, jobQueue, auditLogger, auditLogRepo)
	logger.Info(ctx, "Notifications service initialized")

	// Initialize analytics repositories, cache, and service
	analyticsRepo := postgres.NewAnalyticsRepository(db)
	analyticsLiveRepo := postgres.NewAnalyticsLiveRepository(db)
	analyticsCache := redis.NewAnalyticsCache(redisClient)
	analyticsService := analytics.NewService(analyticsRepo, analyticsLiveRepo, analyticsCache)
	logger.Info(ctx, "Analytics service initialized")

	// Initialize purchase approval repositories and service
	purchaseRequestRepo := postgres.NewPurchaseRequestRepository(db)
	txRunner := postgres.NewTxRunner(db)
	purchaseApprovalService := payments.NewService(payments.ServiceDeps{
		RequestRepo:    purchaseRequestRepo,
		UserRepo:       userRepo,
		CourseRepo:     courseRepo,
		BookRepo:       bookRepo,
		EnrollmentRepo: enrollmentRepo,
		OrderRepo:      orderRepo,
		TxRunner:       txRunner,
		AuditLogger:    auditLogger,
	})
	logger.Info(ctx, "Purchase approval service initialized")

	// Initialize system config repositories and service
	systemSettingRepo := postgres.NewSystemSettingRepository(db)
	systemSettingHistoryRepo := postgres.NewSystemSettingHistoryRepository(db)
	systemConfigService := appsysconfig.NewService(appsysconfig.ServiceDeps{
		SettingRepo: systemSettingRepo,
		HistoryRepo: systemSettingHistoryRepo,
		AuditLogger: auditLogger,
	})
	logger.Info(ctx, "System config service initialized")

	// Initialize live sessions repositories and service
	liveSessionRepo := postgres.NewLiveSessionRepository(db)
	attendanceRepo := postgres.NewAttendanceRepository(db)
	liveSessionsService := livesessions.NewServiceWithTimezone(
		liveSessionRepo,
		attendanceRepo,
		&enrollmentChecker{repo: enrollmentRepo},
		nil, // notifier: wired via notification queue adapter in a later phase
		nil, // recorder: wired via recording worker in a later phase
		courseRepo,
		&timezoneProvider{service: systemConfigService},
	)
	logger.Info(ctx, "Live sessions service initialized")

	// Initialize background workers
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	// Start analytics pre-aggregation worker (hourly)
	analyticsWorker := workers.NewAnalyticsWorker(analyticsRepo, time.Hour)
	go analyticsWorker.Run(workerCtx)

	// Start background workers
	// TODO: Fix worker initialization in later phases
	// worker := workers.NewWorker(redisClient)
	// worker.RegisterHandler("send_email", workers.NewEmailJobHandler(emailService))
	// Additional job handlers will be registered in later phases
	// go worker.Run(workerCtx)
	logger.Info(ctx, "Background workers started (placeholder)")

	// Initialize RBAC repository and service
	rbacRepo := postgres.NewRBACRepository(db)
	rbacService := rbac.NewService(rbac.ServiceDeps{
		PermissionRepo: rbacRepo,
		AuditLogger:    auditLogger,
	})
	logger.Info(ctx, "RBAC service initialized")

	// Initialize Audit Log repository and service
	auditService := appaudit.NewService(auditLogRepo)
	logger.Info(ctx, "Audit service initialized")

	// Initialize Search service (uses raw DB for cross-context queries)
	searchService := search.NewService(tsClient, db)
	logger.Info(ctx, "Search service initialized")

	// Initialize promotional slide service
	slideRepo := postgres.NewPromotionalSlideRepository(db)
	slidesService := slides.NewService(slideRepo, rustfsClient, auditLogger, cfg.RustFSFilesBucket)
	logger.Info(ctx, "Promotional slides service initialized")

	// Initialize idempotency store (shared between payments and bookshop)
	idempotencyStore := redis.NewIdempotencyStore(redisClient)

	// Initialize HTTP server
	server := httpServer.NewServer(redisClient, authService, usersService, coursesService, enrollmentsService, assessmentsService, pointsService, certificatesService, bookshopService, forumService, notificationsService, analyticsService, purchaseApprovalService, systemConfigService, liveSessionsService, rbacService, auditService, searchService, slidesService, idempotencyStore, jwtService, cfg.FrontendBaseURL)
	server.RegisterRoutes()

	// Start HTTP server
	httpSrv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      server.Handler(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		logger.Info(ctx, "HTTP server listening", "port", cfg.ServerPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error(ctx, "HTTP server error", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info(ctx, "Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		logger.Error(ctx, "Server forced to shutdown", "error", err)
	}

	logger.Info(ctx, "Server exited")

	// Suppress unused variable warnings for now
	_ = rustfsClient
	_ = workerCtx
	_ = oauthFactory
}

// enrollmentChecker adapts enrollments.EnrollmentRepository to livesessions.EnrollmentChecker.
type enrollmentChecker struct {
	repo interface {
		Exists(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
		FindByStudentID(ctx context.Context, studentID uuid.UUID, page, limit int) ([]*domainenrollments.Enrollment, int, error)
	}
}

type timezoneProvider struct {
	service appsysconfig.Service
}

func (p *timezoneProvider) DefaultTimezone(ctx context.Context) string {
	settings, err := p.service.GetSettings(ctx)
	if err != nil || settings.DefaultTimezone == "" {
		return "UTC"
	}
	return settings.DefaultTimezone
}

func (e *enrollmentChecker) IsEnrolled(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	return e.repo.Exists(ctx, studentID, courseID)
}

func (e *enrollmentChecker) ListStudentCourseIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	page := 1
	limit := 100
	courseIDs := make([]uuid.UUID, 0)

	for {
		enrollmentList, total, err := e.repo.FindByStudentID(ctx, studentID, page, limit)
		if err != nil {
			return nil, err
		}

		for _, enrollment := range enrollmentList {
			if enrollment != nil && enrollment.Status == domainenrollments.EnrollmentStatusActive {
				courseIDs = append(courseIDs, enrollment.CourseID)
			}
		}

		if page*limit >= total || len(enrollmentList) == 0 {
			break
		}
		page++
	}

	return courseIDs, nil
}

type oauthFactoryAdapter struct {
	factory *oauth.Factory
}

func (a *oauthFactoryAdapter) GetProvider(name string) (auth.OAuthProvider, error) {
	return a.factory.GetProvider(name)
}

func (a *oauthFactoryAdapter) IsProviderEnabled(name string) bool {
	return a.factory.IsProviderEnabled(name)
}
