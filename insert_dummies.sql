DO $$
DECLARE
    v_teacher_id UUID;
    v_student_id UUID;
    v_admin_id UUID;
    v_course_id UUID;
    v_module_id UUID;
    v_chapter_id UUID;
    v_video_id UUID;
    v_lesson_id UUID;
    v_book_id UUID;
    v_quiz_id UUID;
    v_question_id UUID;
    v_assignment_id UUID;
BEGIN
    SELECT id INTO v_teacher_id FROM users WHERE email='teacher@example.com' LIMIT 1;
    SELECT id INTO v_student_id FROM users WHERE email='student@example.com' LIMIT 1;
    SELECT id INTO v_admin_id   FROM users WHERE email='admin@example.com' LIMIT 1;

    -- Add a second teacher to have a mix if needed, or stick to the ones we have
    
    -- COURSE
    INSERT INTO courses (teacher_id, title, slug, short_description, description, subject, level, price_type, price, currency, status)
    VALUES (v_teacher_id, 'Mastering Go Programming', 'mastering-go-' || extract(epoch from now()), 'An advanced Go development course', 'Deep dive into Goroutines, Channels, and standard library.', 'Programming', 'advanced', 'paid', 5000, 'BDT', 'published')
    RETURNING id INTO v_course_id;

    -- MODULE
    INSERT INTO modules (course_id, title, position)
    VALUES (v_course_id, 'Concurrency in Go', 1)
    RETURNING id INTO v_module_id;

    -- CHAPTER
    INSERT INTO chapters (module_id, title, position)
    VALUES (v_module_id, 'Goroutines Basics', 1)
    RETURNING id INTO v_chapter_id;

    -- VIDEO
    INSERT INTO videos (course_id, uploader_id, rustfs_key, status, duration_seconds)
    VALUES (v_course_id, v_teacher_id, 'videos/goroutines.mp4', 'ready', 1500)
    RETURNING id INTO v_video_id;

    -- LESSON
    INSERT INTO lessons (chapter_id, title, type, video_id, duration_seconds, is_free_preview, position, status)
    VALUES (v_chapter_id, 'Introduction to Goroutines', 'video', v_video_id, 1500, true, 1, 'published')
    RETURNING id INTO v_lesson_id;

    -- ENROLLMENT
    INSERT INTO enrollments (student_id, course_id, enrollment_type, status)
    VALUES (v_student_id, v_course_id, 'paid', 'active');

    -- QUIZ
    INSERT INTO quizzes (course_id, lesson_id, title, time_limit_seconds, max_attempts)
    VALUES (v_course_id, v_lesson_id, 'Goroutines Quiz 1', 600, 3)
    RETURNING id INTO v_quiz_id;

    -- QUESTION
    INSERT INTO questions (quiz_id, body, type, position)
    VALUES (v_quiz_id, 'What is a goroutine?', 'single', 1)
    RETURNING id INTO v_question_id;

    INSERT INTO question_options (question_id, body, is_correct, position) VALUES 
    (v_question_id, 'A lightweight thread managed by the Go runtime', true, 1),
    (v_question_id, 'An OS-level thread', false, 2);

    -- QUIZ ATTEMPT
    INSERT INTO quiz_attempts (quiz_id, student_id, score_percent, passed, status)
    VALUES (v_quiz_id, v_student_id, 100.00, true, 'submitted');

    -- ASSIGNMENT
    INSERT INTO assignments (course_id, title, due_at, submission_type, total_marks)
    VALUES (v_course_id, 'Build a Concurrent Web Scraper', NOW() + INTERVAL '7 days', 'both', 100)
    RETURNING id INTO v_assignment_id;

    -- BOOK
    INSERT INTO books (title, author, subject, format, price, physical_stock, is_active)
    VALUES ('The Go Programming Language', 'Alan A. A. Donovan', 'Programming', 'physical', 4000, 50, true)
    RETURNING id INTO v_book_id;

    -- FORUM POST
    INSERT INTO forum_posts (id, author_id, course_id, title, body_markdown, body_html, status)
    VALUES (gen_random_uuid(), v_student_id, v_course_id, 'Having trouble with Select statements', 'Can someone explain how `select` blocks?', '<p>Can someone explain how <code>select</code> blocks?</p>', 'active');

    RAISE NOTICE 'Dummy data inserted successfully into all key tables!';
END $$;
