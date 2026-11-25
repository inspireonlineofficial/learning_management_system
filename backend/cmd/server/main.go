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
	domainassessments "lms-backend/internal/domain/assessments"
	domainenrollments "lms-backend/internal/domain/enrollments"
	infraaudit "lms-backend/internal/infrastructure/audit"
	"lms-backend/internal/infrastructure/certpdf"
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
	"strings"
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
var startedAt = time.Now()

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
	var tsErr error
	for attempt := 1; attempt <= 10; attempt++ {
		tsErr = tsClient.EnsureCollections(ctx)
		if tsErr == nil {
			break
		}
		logger.Warn(ctx, "Typesense not ready, retrying...", "attempt", attempt, "error", tsErr)
		time.Sleep(2 * time.Second)
	}
	if tsErr != nil {
		logger.Error(ctx, "Failed to ensure Typesense collections after retries", "error", tsErr)
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
	ensureStorageCORS(ctx, rustfsClient, cfg)

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
	emailService := email.NewServiceWithConfig(email.Config{
		Host: cfg.SMTPHost, Port: cfg.SMTPPort, TLSMode: cfg.SMTPTLS,
		Username: cfg.SMTPUsername, Password: cfg.SMTPPassword,
		From: cfg.SMTPFrom, FromName: cfg.SMTPFromName,
	})
	if emailService.Configured() {
		logger.Info(ctx, "Email delivery configured", "smtp", emailService.Describe())
	} else {
		logger.Warn(ctx, "EMAIL IS NOT CONFIGURED: set SMTP_HOST (see .env.example). OTP, password-reset and order emails will fail until it is.")
	}
	auditLogger := infraaudit.NewLogger(db)
	auditLogRepo := postgres.NewAuditLogRepository(db)
	logger.Info(ctx, "Infrastructure services initialized")

	// Initialize OAuth providers
	oauthProviders := make(map[string]oauth.Provider)
	if cfg.GoogleClientID != "" {
		oauthProviders["google"] = oauth.NewGoogleProvider(cfg.GoogleClientID, cfg.GoogleClientSecret, cfg.GoogleRedirectURL)
	}
	logger.Info(ctx, "OAuth providers initialized", "count", len(oauthProviders))

	// Initialize OAuth factory
	oauthFactory := oauth.NewFactory(oauthProviders)
	if os.Getenv("OAUTH_TOKEN_ENCRYPTION_KEY") == "" {
		logger.Warn(ctx, "OAUTH_TOKEN_ENCRYPTION_KEY is not set; stored OAuth tokens are encrypted with a key derived from JWT_ISSUER, which is not secret. Set a random 32+ character value in production.")
	}
	if cfg.AdminDevBypass {
		logger.Warn(ctx, "ADMIN_DEV_BYPASS is enabled: admin OTP emails are skipped and a fixed OTP is accepted. Never enable this in production.")
	}
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

	// Initialize notification queue
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
	courseNoteRepo := postgres.NewCourseNoteRepository(db)
	courseCommentRepo := postgres.NewCourseCommentRepository(db)
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
		courseNoteRepo,
		courseCommentRepo,
		enrollmentRepo,
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
		certpdf.NewRenderer(strings.TrimRight(cfg.FrontendBaseURL, "/")+"/verify"),
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
	if setter, ok := enrollmentsService.(enrollments.StreamSigningKeySetter); ok {
		setter.SetStreamSigningKey(jwtService.DeriveKey("hls-playlist"))
	}
	if setter, ok := enrollmentsService.(enrollments.EdgeDeliverySetter); ok && cfg.HLSDeliveryMode == "bunny" {
		if cfg.BunnyCDNHost == "" || cfg.BunnyTokenKey == "" {
			logger.Warn(ctx, "HLS_DELIVERY_MODE=bunny requires BUNNY_CDN_HOST and BUNNY_TOKEN_KEY; falling back to presigned delivery")
		} else {
			setter.SetEdgeDelivery(enrollments.EdgeDeliveryConfig{Mode: "bunny", Host: cfg.BunnyCDNHost, TokenKey: cfg.BunnyTokenKey})
			logger.Info(ctx, "HLS edge delivery enabled", "host", cfg.BunnyCDNHost)
		}
	}
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

	// Assessment engine wiring: transactional quiz edits, serialised attempt
	// starts, and quiz passes awarding gamification points (previously no
	// quiz pass ever reached the points system).
	if configurer, ok := assessmentsService.(assessments.Configurer); ok {
		configurer.SetTxRunner(postgres.NewTxRunner(db))
		configurer.SetQuizPassNotifier(quizPassPoints{points: pointsService, quizzes: quizRepo})
	}
	// Abandoned timed attempts are graded from their autosaved draft once the
	// deadline passes, so teachers see them and students get a result.
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			if n, err := assessmentsService.AutoSubmitExpiredAttempts(context.Background()); err != nil {
				logger.Warn(ctx, "Auto-submitting expired quiz attempts failed", "error", err)
			} else if n > 0 {
				logger.Info(ctx, "Auto-submitted expired quiz attempts", "count", n)
			}
		}
	}()

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
	if configurer, ok := bookshopService.(bookshop.Configurer); ok {
		configurer.SetTxRunner(postgres.NewTxRunner(db))
		configurer.SetRefundRepository(postgres.NewRefundRepository(db))
		configurer.SetRefundRepository(postgres.NewRefundRepository(db))
	}
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

	// Wire the analytics service into the courses service so that when an
	// admin or teacher deletes a course, every enrolled student's cached
	// dashboard snapshot is invalidated immediately rather than waiting
	// for the 5-minute TTL.
	if courseSvc, ok := coursesService.(interface {
		SetDashboardCache(courses.DashboardCache)
	}); ok {
		courseSvc.SetDashboardCache(analyticsService)
	}

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
	if c, ok := purchaseApprovalService.(payments.Configurer); ok {
		c.SetJobQueue(jobQueue)
	}
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
	jobWorker := workers.NewWorker(redisClient, "queue:jobs")
	jobWorker.RegisterHandler("transcode_video", workers.NewTranscodeHandler(workers.TranscodeDeps{
		Storage:     rustfsClient,
		VideoRepo:   videoRepo,
		VideoBucket: cfg.RustFSVideoBucket,
	}))
	jobWorker.RegisterHandler("send_notification", workers.NewSendNotificationJobHandler(notifRepo))
	jobWorker.RegisterHandler("broadcast_notification", workers.NewBroadcastJobHandler(postgres.NewBroadcastNotificationHandler(notifRepo, db)))
	jobWorker.RegisterHandler("send_email", workers.NewSendEmailJobHandler(emailService))
	// Bookshop order events (placed, shipped, delivered, cancelled, refunded)
	// were enqueued with no handler, so students were never told.
	messenger := workers.StudentMessenger{
		Notifications: notifRepo,
		Contacts:      workers.SQLContactLookup{DB: db},
		Queue:         jobQueue,
		FrontendURL:   cfg.FrontendBaseURL,
	}
	bookOrderHandler := workers.NewBookOrderJobHandler(messenger)
	for _, jobType := range []string{"order_placed", "order_shipped", "order_delivered", "order_cancelled", "order_refunded"} {
		jobWorker.RegisterHandler(jobType, bookOrderHandler)
	}
	purchaseDecision := workers.NewPurchaseDecisionJobHandler(messenger)
	jobWorker.RegisterHandler("purchase_approved", purchaseDecision)
	jobWorker.RegisterHandler("purchase_rejected", purchaseDecision)
	refundRequest := workers.NewRefundRequestJobHandler(messenger)
	jobWorker.RegisterHandler("refund_request_received", refundRequest)
	jobWorker.RegisterHandler("refund_request_rejected", refundRequest)
	jobWorker.RegisterHandler("refund_request_admin_alert", workers.NewRefundRequestAdminAlertHandler(messenger, workers.SQLAdminLookup{DB: db}))
	// A long video transcode must not hold up emails and notifications.
	jobWorker.MarkHeavy("transcode_video")
	// Transcodes are expensive and usually fail for a bad input file, so they
	// get one retry instead of four.
	jobWorker.SetMaxAttempts("transcode_video", 2)
	jobWorker.RegisterHandler("forum_post_rejected", workers.NewForumPostRejectedJobHandler(notifRepo))
	jobWorker.RegisterHandler("content_removed", workers.NewContentRemovedJobHandler(db, notifRepo))
	jobWorker.RegisterHandler("generate_certificate", workers.NewGenerateCertificateJobHandler(workers.CertificatePDFFunc(
		func(ctx context.Context, id uuid.UUID) error {
			return certificatesService.GenerateCertificatePDF(ctx, certificates.GenerateCertificatePDFCommand{CertificateID: id})
		})))
	go jobWorker.Run(workerCtx)

	// NotificationQueue (assignment graded, etc.) pushes to "queue:notifications";
	// previously nothing consumed it.
	notificationWorker := workers.NewWorker(redisClient, "notifications")
	notificationWorker.RegisterHandler("send_notification", workers.NewSendNotificationJobHandler(notifRepo))
	go notificationWorker.Run(workerCtx)
	logger.Info(ctx, "Background workers started", "handlers", []string{"transcode_video", "send_notification", "broadcast_notification", "send_email", "forum_post_rejected", "content_removed", "generate_certificate", "order_*"})

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
	server.SetOpsProvider(&opsProvider{
		started:  startedAt,
		db:       db,
		redis:    redisClient,
		workers:  map[string]*workers.Worker{"jobs": jobWorker, "notifications": notificationWorker},
		nWorkers: 2,
		mailer:   emailService,
	})
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

// quizPassPoints adapts the points service to assessments.QuizPassNotifier.
type quizPassPoints struct {
	points  points.Service
	quizzes interface {
		FindByID(ctx context.Context, id uuid.UUID) (*domainassessments.Quiz, error)
	}
}

func (q quizPassPoints) QuizPassed(ctx context.Context, studentID, quizID uuid.UUID, scorePercent float64) error {
	title := ""
	if quiz, err := q.quizzes.FindByID(ctx, quizID); err == nil && quiz != nil {
		title = quiz.Title
	}
	_, err := q.points.AwardQuizPoints(ctx, points.AwardQuizPointsCommand{
		StudentID: studentID, QuizID: quizID, SourceTitle: title, ScorePercent: scorePercent,
	})
	return err
}

// ensureStorageCORS makes sure the books and video buckets let the frontend
// read signed URLs with range requests (PDF.js reader, HLS player). It only
// adds or updates the LMS's own rule, keeps other rules, runs in the
// background and never blocks startup. STORAGE_CORS_AUTO=false turns it off;
// STORAGE_CORS_ORIGINS overrides the origins (default FRONTEND_BASE_URL).
func ensureStorageCORS(ctx context.Context, client *rustfs.Client, cfg *configs.Config) {
	if strings.EqualFold(os.Getenv("STORAGE_CORS_AUTO"), "false") {
		setCORSStatus("automatic setup disabled")
		logger.Info(ctx, "Storage CORS: automatic setup disabled (STORAGE_CORS_AUTO=false)")
		return
	}
	raw := os.Getenv("STORAGE_CORS_ORIGINS")
	if raw == "" {
		raw = cfg.FrontendBaseURL
	}
	origins, err := rustfs.ValidateCORSOrigins(strings.Split(raw, ","))
	if err != nil {
		setCORSStatus("not set: " + err.Error())
		logger.Warn(ctx, "Storage CORS: not set automatically", "reason", err.Error())
		return
	}
	go func() {
		cctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		failed := []string{}
		for _, bucket := range []string{cfg.RustFSBooksBucket, cfg.RustFSVideoBucket} {
			if bucket == "" {
				continue
			}
			changed, err := client.EnsureFrontendCORS(cctx, bucket, origins)
			switch {
			case err != nil:
				failed = append(failed, bucket+": "+err.Error())
				logger.Warn(cctx, "Storage CORS: could not set the rule; the book reader falls back to the browser PDF viewer. Run `make storage-cors` with an account allowed to change bucket CORS.", "bucket", bucket, "error", err.Error())
			case changed:
				logger.Info(cctx, "Storage CORS: rule set", "bucket", bucket, "origins", strings.Join(origins, ","))
			default:
				logger.Info(cctx, "Storage CORS: rule already in place", "bucket", bucket)
			}
		}
		if len(failed) > 0 {
			setCORSStatus("not set: " + strings.Join(failed, "; "))
		} else {
			setCORSStatus("ok")
		}
	}()
}
