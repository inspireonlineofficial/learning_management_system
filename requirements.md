# Requirements Document: LMS Enhancements Phase 2

## Codebase Analysis Baseline

Analysis date: 2026-05-29.

The repository is an LMS with a Go API backend and a Vite/React frontend.

- Backend follows clean architecture under `internal/domain`, `internal/application`, `internal/infrastructure`, and `internal/interfaces/http`.
- HTTP routes are registered in `internal/interfaces/http/server.go` under the `/v1` prefix.
- PostgreSQL migrations live in `migrations`; Redis, Typesense, RustFS/S3-style storage, JWT auth, RBAC, audit logs, and workers are already present.
- Frontend lives in `frontend`, uses React 19, React Router 7, Tailwind CSS, local UI primitives, and API adapters under `frontend/src/lib/api`.
- Existing frontend shells are `PublicShell`, `StudentShell`, `TeacherShell`, and `AdminShell`.
- Existing course free/paid fields, bookshop admin APIs, points APIs, live-session APIs, and flagged-content moderation APIs are partially or fully implemented.
- Current root docs were inconsistent before this rewrite: `requirements.md` contained design material, `design.md` contained requirements material, and `task.md` was empty.

## Scope

This future task adds backend capabilities where contracts are missing and updates the React frontend so users can actually operate the new features without placeholder or misleading UI.

## Non-Goals

- Do not replace the current Go monolith with microservices.
- Do not add a second frontend framework.
- Do not bypass existing JWT, RBAC, audit, pagination, or error response conventions.
- Do not add UI controls that persist data unless the matching backend route exists.

## Glossary

- **Admin**: A user with platform administration privileges.
- **Teacher**: A user who creates and manages courses.
- **Student**: A user enrolled in courses.
- **Promotional Slide**: One active or inactive homepage carousel item.
- **Content Builder**: Teacher UI for modules, chapters, lessons, media, and quiz attachment.
- **Forum Review Queue**: Admin queue for pre-publication forum post review.
- **Flag Moderation Queue**: Existing admin queue for user-flagged posts or comments.
- **Public Bookshop**: Auth-optional book catalog route visible from public navigation.

## Requirements

### Requirement 1: Promotional Slider Management

**User Story:** As an admin, I want to manage homepage promotional slides, so public users see timely platform announcements and featured content.

#### Acceptance Criteria

1. THE System SHALL store promotional slide metadata with ID, title, optional body, CTA label, CTA URL, media storage key, display duration, position, active status, and audit timestamps.
2. THE System SHALL provide admin-only create, update, reorder, and deactivate endpoints for promotional slides.
3. THE System SHALL validate uploaded slide media by size, declared content type, and magic bytes.
4. THE System SHALL accept JPEG, PNG, WebP, and GIF slide media up to 5 MB.
5. THE System SHALL return active slides from a public endpoint ordered by position.
6. THE System SHALL return presigned media URLs for slide display and SHALL NOT expose raw RustFS/S3 object keys.
7. THE Admin frontend SHALL provide a slide management screen with upload, preview, duration, active/inactive, and reorder controls.
8. THE Landing page SHALL render the active slide carousel from the public endpoint and SHALL show a static fallback when no active slides exist.
9. WHEN slide CRUD or reorder succeeds, THE System SHALL write an audit log entry with actor ID, action, target ID, and metadata.

### Requirement 2: Public Navigation and Bookshop Entry Points

**User Story:** As a visitor or student, I want the bookshop to be reachable from main navigation, so I can browse educational materials without knowing a protected route.

#### Acceptance Criteria

1. THE System SHALL expose a public navigation endpoint or stable frontend navigation config containing Home, Courses, and Bookshop entries.
2. THE Bookshop public route SHALL be `/bookshop`.
3. THE PublicShell frontend SHALL show a Bookshop link on desktop and mobile navigation.
4. THE frontend SHALL add a public `/bookshop` route backed by `GET /v1/bookshop/books`.
5. THE public `/bookshop` route SHALL not request protected order-history or digital-access endpoints.
6. THE StudentShell SHALL include a Bookshop navigation item that routes to `/student/bookshop`.
7. THE AdminShell SHALL include a Bookshop Management navigation item that routes to `/admin/bookshop`.
8. THE Admin bookshop page SHALL set `activeKey` to the matching admin navigation key.

### Requirement 3: Persisted Teacher Content Builder

**User Story:** As a teacher, I want to create and edit course modules, chapters, lessons, files, videos, and quizzes from the content builder, so I can prepare a course without backend-only tools.

#### Acceptance Criteria

1. THE System SHALL register HTTP routes for the existing module, chapter, lesson, and reorder service methods.
2. THE System SHALL enforce teacher ownership of the parent course before mutating modules, chapters, lessons, videos, files, or quiz links.
3. THE System SHALL support lesson types `video`, `text`, and `attachment` without breaking existing data.
4. THE System SHALL support attaching quizzes to a lesson using the existing nullable quiz `lesson_id` contract.
5. THE System SHALL support PDF and general file attachments with file metadata and presigned access URLs.
6. THE Content Builder frontend SHALL replace read-only notices with real add, edit, delete, publish/draft, preview, and reorder controls after backend routes are registered.
7. THE Content Builder frontend SHALL call the registered routes through `frontend/src/lib/api` adapters and SHALL display request state, validation errors, and reload behavior consistently.
8. THE frontend SHALL poll video processing status after upload until `ready` or `failed`.
9. THE frontend SHALL not expose raw video IDs or storage keys on public course detail responses.

### Requirement 4: Upload Pipeline Hardening

**User Story:** As a teacher, I want video and file uploads to be validated and processed reliably, so students receive safe downloadable or streamable course content.

#### Acceptance Criteria

1. THE `/v1/uploads/video` handler SHALL parse actual multipart form data instead of hardcoded file metadata.
2. THE `/v1/uploads/file` handler SHALL parse actual multipart form data instead of hardcoded file metadata.
3. THE System SHALL enforce video file type allow-list MP4, WebM, and MOV.
4. THE System SHALL enforce PDF uploads as `application/pdf` with PDF magic-byte validation.
5. THE System SHALL enforce general file upload size limits and block executable or unsafe types.
6. THE System SHALL write uploaded objects with server-generated keys under scoped prefixes.
7. THE System SHALL enqueue asynchronous video transcoding work and expose processing status.
8. THE frontend SHALL show upload progress where browser APIs allow it, blocked-state messages where they do not, and retry controls on failure.
9. THE System SHALL log upload success and failure with request ID, actor ID, size, type, and target course where available.

### Requirement 5: Smart Calendar and Today's Classes

**User Story:** As a student, I want my calendar to show scheduled live classes and highlight today's classes, so I can quickly find my current schedule.

#### Acceptance Criteria

1. THE System SHALL support retrieving student live sessions for a date range.
2. THE live-session list response SHALL include an `is_today` boolean calculated from the platform timezone configured in system settings.
3. THE response SHALL include title, course ID, scheduled time, duration, status, and room/join availability.
4. THE response SHALL order sessions by scheduled time ascending.
5. THE Calendar frontend SHALL consume the real student live-session API and remove the outdated "not registered" notice.
6. THE Calendar frontend SHALL render month cells with session counts, status badges, and a distinct today treatment.
7. THE Calendar frontend SHALL provide a list of today's classes with direct detail and join actions when allowed.
8. WHEN a student has no sessions for a date range, THE Calendar frontend SHALL render an empty state instead of a manual session-ID workaround.

### Requirement 6: Forum Pre-Publication Review

**User Story:** As an admin, I want forum posts reviewed before publication, so community content quality is controlled before posts are visible.

#### Acceptance Criteria

1. THE System SHALL add forum post statuses `pending` and `rejected` without breaking existing `active` and `removed` records.
2. WHEN a user creates a forum post, THE System SHALL default the post status to `pending`.
3. THE public forum listing SHALL return only active, non-deleted posts.
4. THE System SHALL provide admin endpoints to list pending, active, rejected, and removed posts with pagination and status filters.
5. THE System SHALL provide admin endpoints to approve or reject a pending post.
6. WHEN an admin approves a post, THE System SHALL set status to `active`.
7. WHEN an admin rejects a post, THE System SHALL require a rejection reason and set status to `rejected`.
8. WHEN a post is rejected, THE System SHALL notify the author with the rejection reason.
9. THE Admin moderation frontend SHALL support both pending-post review and the existing flagged-content queue without mixing their actions.
10. THE System SHALL audit every approve, reject, remove, dismiss, and ban action.

### Requirement 7: Points and Leaderboard Frontend Completion

**User Story:** As a student, I want clear points, weekly progress, rank, and opt-out controls, so I understand my gamification status.

#### Acceptance Criteria

1. THE Student dashboard SHALL show total points, today's points, this week's points, global rank, and weekly rank from `GET /v1/student/points`.
2. THE Points Progress page SHALL show points history from `GET /v1/student/points/history`.
3. THE Leaderboard page SHALL support weekly and all-time periods from `GET /v1/leaderboard`.
4. THE frontend SHALL expose the leaderboard opt-out control using `PATCH /v1/student/leaderboard/opt-out`.
5. WHEN a student has no points, THE frontend SHALL show zero-value states without treating them as errors.
6. THE frontend SHALL preserve existing privacy behavior for opted-out students.

### Requirement 8: Course Monetization Hardening

**User Story:** As a teacher, I want course pricing rules to be enforced consistently, so students do not hit payment paths for free courses or free-enrollment paths for paid courses.

#### Acceptance Criteria

1. THE System SHALL default new courses to `price_type = paid` when no price type is supplied.
2. THE System SHALL require `price = 0` when `price_type = free`.
3. THE System SHALL require paid courses to have a positive price and valid currency.
4. THE System SHALL prevent changing `price_type` after a course has any enrollment.
5. THE public course list SHALL continue to support `price_type` filtering.
6. THE frontend course creation and edit forms SHALL expose free/paid selection and enforce client-side hints matching backend validation.
7. THE Course Detail frontend SHALL keep free enrollment on `POST /v1/enrollments` and paid checkout on `/student/checkout/:courseId`.
8. THE frontend SHALL render clear labels for free, paid, enrolled, and checkout-required states.

### Requirement 9: Bookshop Management and Public Catalog

**User Story:** As an admin, I want to manage bookshop catalog items with images and availability, while visitors and students can browse a clear catalog.

#### Acceptance Criteria

1. THE System SHALL support cover image storage for books without exposing raw storage keys.
2. THE System SHALL validate cover images as JPEG, PNG, or WebP up to 2 MB.
3. THE Admin bookshop frontend SHALL support create, edit, activate/deactivate, cover upload, and inventory fields.
4. THE public Bookshop frontend SHALL support subject, class grade, format, and search filters using `GET /v1/bookshop/books`.
5. THE Student bookshop route SHALL keep authenticated order history, digital preview, digital access, and bookmarks separate from the public catalog route.
6. WHEN a book is inactive, THE public and student catalog SHALL hide it.
7. WHEN a book is created or updated, THE System SHALL audit the action.

### Requirement 10: Documentation, Tests, and Accessibility

**User Story:** As a maintainer, I want these changes documented and tested, so future work can be implemented safely.

#### Acceptance Criteria

1. THE OpenAPI/Swagger docs SHALL be updated for all new or changed endpoints.
2. THE backend SHALL include service, handler, repository, and migration tests where behavior changes.
3. THE frontend SHALL pass lint and production build.
4. THE frontend SHALL include accessible labels, keyboard-usable controls, visible focus states, and useful empty/loading/error states.
5. THE implementation SHALL preserve existing route names unless this document explicitly introduces a new route.
6. THE implementation SHALL not remove unrelated user or generated changes from the working tree.
