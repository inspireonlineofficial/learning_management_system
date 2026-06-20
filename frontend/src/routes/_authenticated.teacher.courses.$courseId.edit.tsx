import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link, useNavigate } from "@tanstack/react-router";
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  FileText,
  Plus,
  Save,
  Trash2,
  Upload,
  Users,
} from "lucide-react";
import { useEffect, useState } from "react";
import type { ChangeEvent, DragEvent } from "react";
import { toast } from "sonner";

import { AppShell, SectionHeading } from "@/components/layout/app-shell";
import { Input } from "@/components/ui/input";
import {
  createChapter,
  createLesson,
  createModule,
  createCourseNote,
  deleteChapter,
  deleteLesson,
  deleteModule,
  deleteCourseNote,
  getTeacherCourse,
  listCourseStudents,
  reorderContent,
  uploadLessonFile,
  uploadLessonVideo,
  updateCourse,
  updateChapter,
  updateLesson,
  updateModule,
  updateCourseNote,
} from "@/lib/api/teacher";
import { createQuiz } from "@/lib/api/teacher-quizzes";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/edit")({
  component: EditCoursePage,
});

function EditCoursePage() {
  const { courseId } = Route.useParams();
  return <CourseEditor courseId={courseId} />;
}

export function CourseEditor({ courseId }: { courseId: string }) {
  const qc = useQueryClient();
  const [tab, setTab] = useState<"content" | "notes" | "details" | "students">("content");

  const course = useQuery({
    queryKey: ["teacher-course", courseId],
    queryFn: () => getTeacherCourse(courseId),
  });

  if (course.isLoading) {
    return (
      <AppShell>
        <div className="h-12 w-1/2 bg-brand/10 animate-pulse mb-6" />
        <div className="h-64 bg-brand/5 animate-pulse" />
      </AppShell>
    );
  }

  if (course.isError || !course.data) {
    return (
      <AppShell title="Course unavailable">
        <p className="text-sm text-brand/60">{(course.error as Error)?.message ?? "Not found"}</p>
      </AppShell>
    );
  }

  const c = course.data;

  return (
    <AppShell>
      <Link
        to="/teacher"
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back to courses
      </Link>

      <div className="flex items-center justify-between mb-2 flex-wrap gap-3">
        <h1 className="font-serif text-4xl lg:text-5xl text-balance">{c.title}</h1>
        <span className="px-3 py-1 text-[11px] font-medium border border-brand/15 capitalize text-brand/60">
          {c.status}
        </span>
      </div>
      {c.subtitle && <p className="text-brand/65 mb-6">{c.subtitle}</p>}

      <div className="flex gap-2 border-b border-brand/10 mb-8">
        {(["content", "notes", "details", "students"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-4 py-3 text-sm font-medium capitalize border-b-2 -mb-px ${
              tab === t
                ? "border-brand text-brand"
                : "border-transparent text-brand/55 hover:text-brand"
            }`}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "content" && (
        <ContentEditor
          courseId={courseId}
          modules={c.modules ?? []}
          onChange={() => qc.invalidateQueries({ queryKey: ["teacher-course", courseId] })}
        />
      )}
      {tab === "notes" && (
        <NotesEditor
          courseId={courseId}
          modules={c.modules ?? []}
          notes={c.notes ?? []}
          onChange={() => qc.invalidateQueries({ queryKey: ["teacher-course", courseId] })}
        />
      )}
      {tab === "details" && <DetailsEditor courseId={courseId} initial={c} />}
      {tab === "students" && <StudentsPanel courseId={courseId} />}
    </AppShell>
  );
}

function ContentEditor({
  courseId,
  modules,
  onChange,
}: {
  courseId: string;
  modules: {
    id: string;
    title: string;
    description?: string;
    position?: number;
    is_free?: boolean;
    is_published?: boolean;
    chapters?: {
      id: string;
      title: string;
      position?: number;
      lessons: {
        id: string;
        title: string;
        description?: string;
        type?: string;
        duration_seconds?: number;
        is_free_preview?: boolean;
        is_free?: boolean;
        is_downloadable?: boolean;
        status?: string;
        video_id?: string | null;
      }[];
    }[];
  }[];
  onChange: () => void;
}) {
  const [addingModule, setAddingModule] = useState(false);
  const [moduleTitle, setModuleTitle] = useState("");
  const [draggedModuleId, setDraggedModuleId] = useState<string | null>(null);

  const create = useMutation({
    mutationFn: () =>
      createModule(courseId, {
        title: moduleTitle.trim(),
        position: modules.length + 1,
        is_free: true,
        is_published: true,
      }),
    onSuccess: () => {
      toast.success("Module added");
      setModuleTitle("");
      setAddingModule(false);
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const reorderModules = (ordered: typeof modules) =>
    reorderContent({
      type: "module",
      parent_id: courseId,
      positions: Object.fromEntries(ordered.map((m, i) => [m.id, i + 1])),
    })
      .then(onChange)
      .catch((e) => toast.error((e as Error).message));

  const moveModule = (moduleId: string, direction: "up" | "down") => {
    const ordered = [...modules].sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
    const idx = ordered.findIndex((m) => m.id === moduleId);
    const swap = direction === "up" ? idx - 1 : idx + 1;
    if (idx < 0 || swap < 0 || swap >= ordered.length) return;
    const next = ordered.slice();
    [next[idx], next[swap]] = [next[swap], next[idx]];
    reorderModules(next);
  };

  const orderedModules = modules.slice().sort((a, b) => (a.position ?? 0) - (b.position ?? 0));

  const dropModule = (targetModuleId: string) => {
    if (!draggedModuleId || draggedModuleId === targetModuleId) return;
    const from = orderedModules.findIndex((module) => module.id === draggedModuleId);
    const to = orderedModules.findIndex((module) => module.id === targetModuleId);
    if (from < 0 || to < 0) return;
    const next = orderedModules.slice();
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    setDraggedModuleId(null);
    reorderModules(next);
  };

  return (
    <div>
      <SectionHeading
        title="Curriculum"
        action={
          <button
            onClick={() => setAddingModule(true)}
            className="inline-flex items-center gap-2 px-3 py-2 text-xs font-medium border border-brand/15 hover:bg-brand/[0.03]"
          >
            <Plus className="h-3.5 w-3.5" />
            Module
          </button>
        }
      />

      {addingModule && (
        <div className="border border-brand/15 bg-white/50 p-4 mb-4 flex gap-2">
          <input
            autoFocus
            value={moduleTitle}
            onChange={(e) => setModuleTitle(e.target.value)}
            placeholder="Module title"
            className="flex-1 px-3 py-2 text-sm border border-brand/15 bg-white/50"
          />
          <button
            onClick={() => create.mutate()}
            disabled={!moduleTitle.trim() || create.isPending}
            className="px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
          >
            Add
          </button>
          <button
            onClick={() => setAddingModule(false)}
            className="px-3 py-2 text-xs text-brand/70"
          >
            Cancel
          </button>
        </div>
      )}

      {modules.length === 0 ? (
        <p className="text-sm text-brand/55 border border-dashed border-brand/15 p-8 text-center">
          No modules yet. Add one to start building your curriculum.
        </p>
      ) : (
        <ul className="space-y-3">
          {orderedModules.map((module, index) => (
            <ModuleCard
              key={module.id}
              courseId={courseId}
              module={module}
              index={index}
              total={modules.length}
              onChange={onChange}
              onMove={moveModule}
              draggable
              dragging={draggedModuleId === module.id}
              onDragStart={() => setDraggedModuleId(module.id)}
              onDragEnd={() => setDraggedModuleId(null)}
              onDragOver={(event) => event.preventDefault()}
              onDrop={() => dropModule(module.id)}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

function ModuleCard({
  courseId,
  module,
  index,
  total,
  onChange,
  onMove,
  draggable,
  dragging,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDrop,
}: {
  courseId: string;
  module: {
    id: string;
    title: string;
    description?: string;
    is_free?: boolean;
    is_published?: boolean;
    chapters?: {
      id: string;
      title: string;
      position?: number;
      lessons: {
        id: string;
        title: string;
        description?: string;
        type?: string;
        duration_seconds?: number;
        is_free_preview?: boolean;
        is_free?: boolean;
        is_downloadable?: boolean;
        status?: string;
        video_id?: string | null;
      }[];
    }[];
  };
  index: number;
  total: number;
  onChange: () => void;
  onMove: (moduleId: string, direction: "up" | "down") => void;
  draggable?: boolean;
  dragging?: boolean;
  onDragStart?: () => void;
  onDragEnd?: () => void;
  onDragOver?: (event: DragEvent<HTMLLIElement>) => void;
  onDrop?: () => void;
}) {
  const [open, setOpen] = useState(true);
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState({
    title: module.title,
    description: module.description ?? "",
    is_free: module.is_free ?? true,
    is_published: module.is_published ?? true,
  });
  const [chapterTitle, setChapterTitle] = useState("");

  useEffect(() => {
    setDraft({
      title: module.title,
      description: module.description ?? "",
      is_free: module.is_free ?? true,
      is_published: module.is_published ?? true,
    });
  }, [module]);

  const rename = useMutation({
    mutationFn: () =>
      updateModule(module.id, {
        title: draft.title.trim(),
        description: draft.description,
        is_free: draft.is_free,
        is_published: draft.is_published,
        position: index + 1,
      }),
    onSuccess: () => {
      toast.success("Updated");
      setEditing(false);
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: () => deleteModule(module.id),
    onSuccess: () => {
      toast.success("Module deleted");
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const addChapter = useMutation({
    mutationFn: () =>
      createChapter(module.id, {
        title: chapterTitle.trim(),
        position: (module.chapters?.length ?? 0) + 1,
      }),
    onSuccess: () => {
      toast.success("Chapter added");
      setChapterTitle("");
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const chapters = (module.chapters ?? [])
    .slice()
    .sort((a, b) => (a.position ?? 0) - (b.position ?? 0));

  return (
    <li
      draggable={draggable}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      onDragOver={onDragOver}
      onDrop={onDrop}
      className={`border border-brand/15 bg-white/40 ${dragging ? "opacity-50" : ""}`}
    >
      <div className="flex items-center gap-3 px-4 py-3">
        <button onClick={() => setOpen((v) => !v)} className="text-brand/55 hover:text-brand">
          <ChevronDown className={`h-4 w-4 transition-transform ${open ? "" : "-rotate-90"}`} />
        </button>
        {editing ? (
          <div className="flex-1 grid gap-2">
            <input
              value={draft.title}
              onChange={(e) => setDraft((current) => ({ ...current, title: e.target.value }))}
              className="px-2 py-1 text-sm border border-brand/15 bg-white"
              autoFocus
            />
            <textarea
              value={draft.description}
              onChange={(e) => setDraft((current) => ({ ...current, description: e.target.value }))}
              rows={2}
              placeholder="Module description"
              className="px-2 py-1 text-xs border border-brand/15 bg-white resize-y"
            />
            <div className="flex flex-wrap items-center gap-4 text-xs text-brand/70">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={draft.is_free}
                  onChange={(e) =>
                    setDraft((current) => ({ ...current, is_free: e.target.checked }))
                  }
                />
                Free access
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={draft.is_published}
                  onChange={(e) =>
                    setDraft((current) => ({ ...current, is_published: e.target.checked }))
                  }
                />
                Published
              </label>
              <button
                onClick={() => rename.mutate()}
                disabled={!draft.title.trim() || rename.isPending}
                className="px-3 py-1 bg-brand text-white text-xs disabled:opacity-50"
              >
                Save
              </button>
              <button onClick={() => setEditing(false)} className="text-xs text-brand/55">
                Cancel
              </button>
            </div>
          </div>
        ) : (
          <>
            <button
              onClick={() => setEditing(true)}
              className="flex-1 text-left font-serif text-lg hover:text-accent"
            >
              {module.title}
            </button>
            <span className="text-xs text-brand/45">{chapters.length} chapters</span>
            <span className="text-[11px] uppercase tracking-wider text-brand/40">
              {module.is_free ? "Free" : "Paid"}
            </span>
            {!module.is_published && (
              <span className="text-[11px] uppercase tracking-wider text-brand/40">Hidden</span>
            )}
            <button
              onClick={() => onMove(module.id, "up")}
              disabled={index === 0}
              className="p-1.5 text-brand/45 hover:text-brand disabled:opacity-30"
              aria-label="Move module up"
            >
              <ChevronUpIcon />
            </button>
            <button
              onClick={() => onMove(module.id, "down")}
              disabled={index >= total - 1}
              className="p-1.5 text-brand/45 hover:text-brand disabled:opacity-30"
              aria-label="Move module down"
            >
              <ChevronDownIcon />
            </button>
            <button
              onClick={() => {
                if (confirm("Delete this module and its chapters?")) remove.mutate();
              }}
              className="p-1.5 text-brand/45 hover:text-destructive"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </>
        )}
      </div>

      {open && (
        <div className="border-t border-brand/10 px-4 py-3 space-y-4">
          {chapters.length === 0 ? (
            <p className="text-xs text-brand/50 py-2">No chapters yet.</p>
          ) : (
            <ul className="space-y-3">
              {chapters.map((chapter, chapterIndex) => (
                <ChapterCard
                  key={chapter.id}
                  courseId={courseId}
                  moduleId={module.id}
                  chapter={chapter}
                  index={chapterIndex}
                  total={chapters.length}
                  onChange={onChange}
                />
              ))}
            </ul>
          )}

          <div className="flex gap-2">
            <input
              autoFocus={false}
              value={chapterTitle}
              onChange={(e) => setChapterTitle(e.target.value)}
              placeholder="Chapter title"
              className="flex-1 px-3 py-2 text-sm border border-brand/15 bg-white"
            />
            <button
              onClick={() => addChapter.mutate()}
              disabled={!chapterTitle.trim() || addChapter.isPending}
              className="px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
            >
              Add chapter
            </button>
          </div>
        </div>
      )}
    </li>
  );
}

function ChapterCard({
  courseId,
  moduleId,
  chapter,
  index,
  total,
  onChange,
}: {
  courseId: string;
  moduleId: string;
  chapter: {
    id: string;
    title: string;
    position?: number;
    lessons: {
      id: string;
      title: string;
      description?: string;
      type?: string;
      duration_seconds?: number;
      is_free_preview?: boolean;
      is_free?: boolean;
      is_downloadable?: boolean;
      status?: string;
      video_id?: string | null;
    }[];
  };
  index: number;
  total: number;
  onChange: () => void;
}) {
  const [open, setOpen] = useState(true);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(chapter.title);
  const [lessonTitle, setLessonTitle] = useState("");
  const [lessonType, setLessonType] = useState<"video" | "text" | "attachment">("video");
  const [lessonStatus, setLessonStatus] = useState<"draft" | "published">("draft");
  const [lessonDuration, setLessonDuration] = useState("60");
  const [lessonPreview, setLessonPreview] = useState(false);
  const [lessonFree, setLessonFree] = useState(true);
  const [lessonDownloadable, setLessonDownloadable] = useState(false);
  const [lessonVideo, setLessonVideo] = useState<File | null>(null);
  const [lessonFileKey, setLessonFileKey] = useState(0);
  const [isDetectingLessonDuration, setIsDetectingLessonDuration] = useState(false);
  const [draggedLessonId, setDraggedLessonId] = useState<string | null>(null);

  useEffect(() => {
    setTitle(chapter.title);
  }, [chapter.title]);

  const rename = useMutation({
    mutationFn: () => updateChapter(chapter.id, { title: title.trim(), position: index + 1 }),
    onSuccess: () => {
      toast.success("Updated");
      setEditing(false);
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: () => deleteChapter(chapter.id),
    onSuccess: () => {
      toast.success("Chapter deleted");
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const create = useMutation({
    mutationFn: async () => {
      let videoId: string | undefined;
      if (lessonVideo) {
        const uploaded = await uploadLessonVideo(courseId, lessonVideo);
        videoId = uploaded.video_id;
      }
      return createLesson(chapter.id, {
        title: lessonTitle.trim(),
        type: lessonType,
        video_id: videoId,
        duration_seconds: durationMinutesToSeconds(lessonDuration),
        is_free_preview: lessonPreview,
        is_free: lessonFree,
        is_downloadable: lessonDownloadable,
        status: lessonStatus,
      });
    },
    onSuccess: () => {
      toast.success("Lesson added");
      setLessonTitle("");
      setLessonVideo(null);
      setLessonFileKey((key) => key + 1);
      setLessonFree(true);
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const reorderLessons = (ordered: typeof chapter.lessons) =>
    reorderContent({
      type: "lesson",
      parent_id: chapter.id,
      positions: Object.fromEntries(ordered.map((lesson, i) => [lesson.id, i + 1])),
    })
      .then(onChange)
      .catch((e) => toast.error((e as Error).message));

  const moveLesson = (lessonId: string, direction: "up" | "down") => {
    const ordered = [...chapter.lessons].sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
    const idx = ordered.findIndex((lesson) => lesson.id === lessonId);
    const swap = direction === "up" ? idx - 1 : idx + 1;
    if (idx < 0 || swap < 0 || swap >= ordered.length) return;
    const next = ordered.slice();
    [next[idx], next[swap]] = [next[swap], next[idx]];
    reorderLessons(next);
  };

  const lessons = chapter.lessons.slice().sort((a, b) => (a.position ?? 0) - (b.position ?? 0));

  const dropLesson = (targetLessonId: string) => {
    if (!draggedLessonId || draggedLessonId === targetLessonId) return;
    const from = lessons.findIndex((lesson) => lesson.id === draggedLessonId);
    const to = lessons.findIndex((lesson) => lesson.id === targetLessonId);
    if (from < 0 || to < 0) return;
    const next = lessons.slice();
    const [moved] = next.splice(from, 1);
    next.splice(to, 0, moved);
    setDraggedLessonId(null);
    reorderLessons(next);
  };

  const handleLessonVideoChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    setLessonVideo(file);
    if (!file) return;

    setIsDetectingLessonDuration(true);
    try {
      const seconds = await getMediaDurationSeconds(file);
      setLessonDuration(secondsToMinutesInput(seconds));
    } catch {
      toast.error("Could not read media duration. Enter it manually.");
    } finally {
      setIsDetectingLessonDuration(false);
    }
  };

  return (
    <li className="border border-brand/10 bg-white/50">
      <div className="flex items-center gap-3 px-4 py-3">
        <button onClick={() => setOpen((v) => !v)} className="text-brand/55 hover:text-brand">
          <ChevronRight className={`h-4 w-4 transition-transform ${open ? "rotate-90" : ""}`} />
        </button>
        {editing ? (
          <>
            <input
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              className="flex-1 px-2 py-1 text-sm border border-brand/15 bg-white"
              autoFocus
            />
            <button
              onClick={() => rename.mutate()}
              className="px-3 py-1 bg-brand text-white text-xs"
            >
              Save
            </button>
            <button onClick={() => setEditing(false)} className="text-xs text-brand/55">
              Cancel
            </button>
          </>
        ) : (
          <>
            <button
              onClick={() => setEditing(true)}
              className="flex-1 text-left text-sm font-medium hover:text-accent"
            >
              {chapter.title}
            </button>
            <span className="text-xs text-brand/45">{lessons.length} lessons</span>
            <button
              onClick={() => remove.mutate()}
              className="p-1.5 text-brand/45 hover:text-destructive"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </>
        )}
      </div>

      {open && (
        <div className="border-t border-brand/10 px-4 py-3 space-y-4">
          <ul className="space-y-3">
            {lessons.map((lesson, lessonIndex) => (
              <LessonRow
                key={lesson.id}
                courseId={courseId}
                lesson={lesson}
                index={lessonIndex}
                total={lessons.length}
                onChange={onChange}
                onMove={moveLesson}
                draggable
                dragging={draggedLessonId === lesson.id}
                onDragStart={() => setDraggedLessonId(lesson.id)}
                onDragEnd={() => setDraggedLessonId(null)}
                onDragOver={(event) => event.preventDefault()}
                onDrop={() => dropLesson(lesson.id)}
              />
            ))}
          </ul>

          <div className="border border-dashed border-brand/15 bg-white/60 p-4 space-y-3">
            <p className="text-xs uppercase tracking-wider text-brand/45">Add lesson</p>
            <div className="grid md:grid-cols-2 gap-3">
              <Input
                value={lessonTitle}
                onChange={(e) => setLessonTitle(e.target.value)}
                placeholder="Lesson title"
              />
              <select
                value={lessonType}
                onChange={(e) => setLessonType(e.target.value as "video" | "text" | "attachment")}
                className="h-9 border border-brand/15 bg-white px-3 text-sm"
              >
                <option value="video">Video</option>
                <option value="text">Text</option>
                <option value="attachment">Attachment</option>
              </select>
            </div>
            <div className="grid md:grid-cols-3 gap-3">
              <Input
                type="number"
                min={0}
                step="0.01"
                value={lessonDuration}
                onChange={(e) => setLessonDuration(e.target.value)}
                placeholder={
                  isDetectingLessonDuration ? "Detecting duration..." : "Duration minutes"
                }
              />
              <select
                value={lessonStatus}
                onChange={(e) => setLessonStatus(e.target.value as "draft" | "published")}
                className="h-9 border border-brand/15 bg-white px-3 text-sm"
              >
                <option value="draft">Draft</option>
                <option value="published">Published</option>
              </select>
              <label className="flex items-center gap-2 text-sm text-brand/70">
                <input
                  type="checkbox"
                  checked={lessonPreview}
                  onChange={(e) => setLessonPreview(e.target.checked)}
                />
                Free preview
              </label>
            </div>
            <div className="grid md:grid-cols-2 gap-3">
              <label className="flex items-center gap-2 text-sm text-brand/70">
                <input
                  type="checkbox"
                  checked={lessonFree}
                  onChange={(e) => setLessonFree(e.target.checked)}
                />
                Free access
              </label>
              <label className="flex items-center gap-2 text-sm text-brand/70">
                <input
                  type="checkbox"
                  checked={lessonDownloadable}
                  onChange={(e) => setLessonDownloadable(e.target.checked)}
                />
                Downloadable
              </label>
              <Input
                key={lessonFileKey}
                type="file"
                accept="video/mp4,video/webm,video/quicktime"
                disabled={lessonType !== "video"}
                onChange={(e) => void handleLessonVideoChange(e)}
              />
            </div>
            {lessonVideo && (
              <div className="flex items-center justify-between gap-3 border border-brand/10 bg-white px-3 py-2 text-xs text-brand/60">
                <span className="truncate">{lessonVideo.name}</span>
                <button
                  type="button"
                  onClick={() => {
                    setLessonVideo(null);
                    setLessonFileKey((key) => key + 1);
                    setIsDetectingLessonDuration(false);
                  }}
                  className="text-destructive hover:underline"
                >
                  Remove
                </button>
              </div>
            )}
            <div className="flex justify-end">
              <button
                onClick={() => create.mutate()}
                disabled={!lessonTitle.trim() || create.isPending}
                className="inline-flex items-center gap-2 px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
              >
                <Upload className="h-3.5 w-3.5" />
                {create.isPending ? "Saving…" : "Create lesson"}
              </button>
            </div>
          </div>
        </div>
      )}
    </li>
  );
}

function LessonRow({
  courseId,
  lesson,
  index,
  total,
  onChange,
  onMove,
  draggable,
  dragging,
  onDragStart,
  onDragEnd,
  onDragOver,
  onDrop,
}: {
  courseId: string;
  lesson: {
    id: string;
    title: string;
    description?: string;
    type?: string;
    duration_seconds?: number;
    is_free_preview?: boolean;
    is_free?: boolean;
    is_downloadable?: boolean;
    status?: string;
    video_id?: string | null;
  };
  index: number;
  total: number;
  onChange: () => void;
  onMove: (lessonId: string, direction: "up" | "down") => void;
  draggable?: boolean;
  dragging?: boolean;
  onDragStart?: () => void;
  onDragEnd?: () => void;
  onDragOver?: (event: DragEvent<HTMLLIElement>) => void;
  onDrop?: () => void;
}) {
  const navigate = useNavigate();
  const [editing, setEditing] = useState(false);
  const [addingNote, setAddingNote] = useState(false);
  const [draft, setDraft] = useState({
    title: lesson.title,
    type: (lesson.type as "video" | "text" | "attachment") ?? "video",
    description: lesson.description ?? "",
    duration_minutes: lesson.duration_seconds ? secondsToMinutesInput(lesson.duration_seconds) : "",
    is_free_preview: lesson.is_free_preview ?? false,
    is_free: lesson.is_free ?? true,
    is_downloadable: lesson.is_downloadable ?? false,
    status: (lesson.status as "draft" | "published") ?? "draft",
  });
  const [videoFile, setVideoFile] = useState<File | null>(null);
  const [videoFileKey, setVideoFileKey] = useState(0);
  const [isDetectingVideoDuration, setIsDetectingVideoDuration] = useState(false);
  const [quickNote, setQuickNote] = useState({
    title: "",
    content: "",
    is_free: true,
    is_published: false,
  });

  useEffect(() => {
    setDraft({
      title: lesson.title,
      type: (lesson.type as "video" | "text" | "attachment") ?? "video",
      description: lesson.description ?? "",
      duration_minutes: lesson.duration_seconds
        ? secondsToMinutesInput(lesson.duration_seconds)
        : "",
      is_free_preview: lesson.is_free_preview ?? false,
      is_free: lesson.is_free ?? true,
      is_downloadable: lesson.is_downloadable ?? false,
      status: (lesson.status as "draft" | "published") ?? "draft",
    });
  }, [lesson]);

  const save = useMutation({
    mutationFn: async () => {
      let videoId: string | undefined;
      if (videoFile) {
        const uploaded = await uploadLessonVideo(courseId, videoFile);
        videoId = uploaded.video_id;
      }
      return updateLesson(lesson.id, {
        title: draft.title.trim(),
        description: draft.description,
        type: draft.type,
        video_id: videoId,
        duration_seconds: durationMinutesToSeconds(draft.duration_minutes),
        is_free_preview: draft.is_free_preview,
        is_free: draft.is_free,
        is_downloadable: draft.is_downloadable,
        status: draft.status,
      });
    },
    onSuccess: () => {
      toast.success("Lesson saved");
      setEditing(false);
      setVideoFile(null);
      setVideoFileKey((key) => key + 1);
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const createNote = useMutation({
    mutationFn: () =>
      createCourseNote(courseId, {
        lesson_id: lesson.id,
        title: quickNote.title.trim(),
        content: quickNote.content.trim(),
        is_free: quickNote.is_free,
        is_published: quickNote.is_published,
      }),
    onSuccess: () => {
      toast.success("Note added");
      setAddingNote(false);
      setQuickNote({ title: "", content: "", is_free: true, is_published: false });
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const createLessonQuiz = useMutation({
    mutationFn: () =>
      createQuiz({
        course_id: courseId,
        lesson_id: lesson.id,
        title: `${lesson.title} quiz`,
        passing_score: 70,
        time_limit_minutes: 30,
        attempts_allowed: 1,
        is_free: lesson.is_free ?? true,
        is_published: false,
      }),
    onSuccess: (quiz) => {
      toast.success("Quiz draft created");
      navigate({ to: "/teacher/quiz-builder/$quizId/edit", params: { quizId: quiz.id } });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: () => deleteLesson(lesson.id),
    onSuccess: () => {
      toast.success("Lesson deleted");
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const handleVideoFileChange = async (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0] ?? null;
    setVideoFile(file);
    if (!file) return;

    setIsDetectingVideoDuration(true);
    try {
      const seconds = await getMediaDurationSeconds(file);
      setDraft((current) => ({ ...current, duration_minutes: secondsToMinutesInput(seconds) }));
    } catch {
      toast.error("Could not read media duration. Enter it manually.");
    } finally {
      setIsDetectingVideoDuration(false);
    }
  };

  return (
    <li
      draggable={draggable}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      onDragOver={onDragOver}
      onDrop={onDrop}
      className={`border border-brand/10 bg-white/60 p-4 ${dragging ? "opacity-50" : ""}`}
    >
      {editing ? (
        <div className="space-y-3">
          <div className="grid md:grid-cols-2 gap-3">
            <Input
              value={draft.title}
              onChange={(e) => setDraft((current) => ({ ...current, title: e.target.value }))}
              placeholder="Lesson title"
            />
            <select
              value={draft.type}
              onChange={(e) =>
                setDraft((current) => ({
                  ...current,
                  type: e.target.value as "video" | "text" | "attachment",
                }))
              }
              className="h-9 border border-brand/15 bg-white px-3 text-sm"
            >
              <option value="video">Video</option>
              <option value="text">Text</option>
              <option value="attachment">Attachment</option>
            </select>
          </div>
          <textarea
            value={draft.description}
            onChange={(e) => setDraft((current) => ({ ...current, description: e.target.value }))}
            rows={2}
            placeholder="Lesson description"
            className="w-full px-3 py-2 text-sm border border-brand/15 bg-white resize-y"
          />

          <div className="grid md:grid-cols-3 gap-3">
            <Input
              type="number"
              min={0}
              step="0.01"
              value={draft.duration_minutes}
              onChange={(e) =>
                setDraft((current) => ({
                  ...current,
                  duration_minutes: e.target.value,
                }))
              }
              placeholder={isDetectingVideoDuration ? "Detecting duration..." : "Duration minutes"}
            />
            <select
              value={draft.status}
              onChange={(e) =>
                setDraft((current) => ({
                  ...current,
                  status: e.target.value as "draft" | "published",
                }))
              }
              className="h-9 border border-brand/15 bg-white px-3 text-sm"
            >
              <option value="draft">Draft</option>
              <option value="published">Published</option>
            </select>
            <Input
              key={videoFileKey}
              type="file"
              accept="video/mp4,video/webm,video/quicktime"
              disabled={draft.type !== "video"}
              onChange={(e) => void handleVideoFileChange(e)}
            />
          </div>
          {videoFile && (
            <div className="flex items-center justify-between gap-3 border border-brand/10 bg-white px-3 py-2 text-xs text-brand/60">
              <span className="truncate">{videoFile.name}</span>
              <button
                type="button"
                onClick={() => {
                  setVideoFile(null);
                  setVideoFileKey((key) => key + 1);
                  setIsDetectingVideoDuration(false);
                }}
                className="text-destructive hover:underline"
              >
                Remove
              </button>
            </div>
          )}

          <div className="flex flex-wrap gap-4 text-xs text-brand/70">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={draft.is_free_preview}
                onChange={(e) =>
                  setDraft((current) => ({ ...current, is_free_preview: e.target.checked }))
                }
              />
              Free preview
            </label>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={draft.is_free}
                onChange={(e) => setDraft((current) => ({ ...current, is_free: e.target.checked }))}
              />
              Free access
            </label>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={draft.is_downloadable}
                onChange={(e) =>
                  setDraft((current) => ({ ...current, is_downloadable: e.target.checked }))
                }
              />
              Downloadable
            </label>
          </div>

          <div className="flex justify-between gap-2">
            <div className="flex gap-2">
              <button
                onClick={() => onMove(lesson.id, "up")}
                disabled={index === 0}
                className="px-3 py-2 text-xs border border-brand/15 disabled:opacity-30"
              >
                Up
              </button>
              <button
                onClick={() => onMove(lesson.id, "down")}
                disabled={index >= total - 1}
                className="px-3 py-2 text-xs border border-brand/15 disabled:opacity-30"
              >
                Down
              </button>
            </div>
            <div className="flex gap-2">
              <button onClick={() => setEditing(false)} className="px-3 py-2 text-xs text-brand/70">
                Cancel
              </button>
              <button
                onClick={() => save.mutate()}
                disabled={save.isPending}
                className="inline-flex items-center gap-2 px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
              >
                <Save className="h-3.5 w-3.5" />
                {save.isPending ? "Saving…" : "Save"}
              </button>
            </div>
          </div>
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <button onClick={() => setEditing(true)} className="text-left flex-1">
              <p className="font-medium text-sm">{lesson.title}</p>
              <p className="mt-1 text-[11px] text-brand/45 uppercase tracking-wider">
                {lesson.type ?? "video"}
                {lesson.is_free_preview ? " · preview" : ""}
                {lesson.is_free === false ? " · paid" : " · free"}
                {lesson.is_downloadable ? " · downloadable" : ""}
                {typeof lesson.duration_seconds === "number"
                  ? ` · ${Math.round(lesson.duration_seconds / 60)}m`
                  : ""}
              </p>
            </button>
            <button
              onClick={() => {
                if (confirm("Delete lesson?")) remove.mutate();
              }}
              className="inline-flex items-center gap-1.5 px-2 py-1 text-xs text-brand/50 hover:text-destructive"
            >
              <Trash2 className="h-3.5 w-3.5" />
              Remove
            </button>
          </div>
          <div className="flex flex-wrap items-center gap-2 border-t border-brand/10 pt-3">
            <button
              onClick={() => setAddingNote((value) => !value)}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs border border-brand/15 hover:bg-brand/[0.03]"
            >
              <FileText className="h-3.5 w-3.5" />
              Add note
            </button>
            <button
              onClick={() => createLessonQuiz.mutate()}
              disabled={createLessonQuiz.isPending}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs border border-brand/15 hover:bg-brand/[0.03] disabled:opacity-50"
            >
              <Plus className="h-3.5 w-3.5" />
              {createLessonQuiz.isPending ? "Creating quiz..." : "Add quiz"}
            </button>
          </div>
          {addingNote && (
            <div className="border border-dashed border-brand/15 bg-white p-3 space-y-3">
              <Input
                value={quickNote.title}
                onChange={(e) => setQuickNote((current) => ({ ...current, title: e.target.value }))}
                placeholder="Note title"
              />
              <textarea
                value={quickNote.content}
                onChange={(e) =>
                  setQuickNote((current) => ({ ...current, content: e.target.value }))
                }
                rows={3}
                placeholder="Note content"
                className="w-full px-3 py-2 text-sm border border-brand/15 bg-white resize-y"
              />
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex gap-4 text-xs text-brand/70">
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={quickNote.is_free}
                      onChange={(e) =>
                        setQuickNote((current) => ({ ...current, is_free: e.target.checked }))
                      }
                    />
                    Free access
                  </label>
                  <label className="flex items-center gap-2">
                    <input
                      type="checkbox"
                      checked={quickNote.is_published}
                      onChange={(e) =>
                        setQuickNote((current) => ({
                          ...current,
                          is_published: e.target.checked,
                        }))
                      }
                    />
                    Published
                  </label>
                </div>
                <div className="flex gap-2">
                  <button
                    onClick={() => setAddingNote(false)}
                    className="px-3 py-2 text-xs text-brand/70"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={() => createNote.mutate()}
                    disabled={
                      createNote.isPending || !quickNote.title.trim() || !quickNote.content.trim()
                    }
                    className="px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
                  >
                    {createNote.isPending ? "Saving..." : "Save note"}
                  </button>
                </div>
              </div>
            </div>
          )}
        </div>
      )}
    </li>
  );
}

function ChevronUpIcon() {
  return <ChevronRight className="h-3.5 w-3.5 -rotate-90" />;
}

function ChevronDownIcon() {
  return <ChevronRight className="h-3.5 w-3.5 rotate-90" />;
}

function NotesEditor({
  courseId,
  modules,
  notes,
  onChange,
}: {
  courseId: string;
  modules: {
    id: string;
    title: string;
    chapters?: {
      id: string;
      title: string;
      lessons: { id: string; title: string }[];
    }[];
  }[];
  notes: {
    id: string;
    module_id?: string | null;
    lesson_id?: string | null;
    title: string;
    content: string;
    file_url?: string;
    is_free: boolean;
    is_published: boolean;
  }[];
  onChange: () => void;
}) {
  const [form, setForm] = useState({
    title: "",
    content: "",
    file_url: "",
    module_id: "",
    lesson_id: "",
    is_free: true,
    is_published: false,
  });

  const create = useMutation({
    mutationFn: () =>
      createCourseNote(courseId, {
        title: form.title.trim(),
        content: form.content.trim(),
        file_url: form.file_url.trim() || undefined,
        module_id: form.module_id || undefined,
        lesson_id: form.lesson_id || undefined,
        is_free: form.is_free,
        is_published: form.is_published,
      }),
    onSuccess: () => {
      toast.success("Note added");
      setForm({
        title: "",
        content: "",
        file_url: "",
        module_id: "",
        lesson_id: "",
        is_free: true,
        is_published: false,
      });
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const upload = useMutation({
    mutationFn: (file: File) => uploadLessonFile(file),
    onSuccess: (result) => {
      setForm((current) => ({ ...current, file_url: result.presigned_url }));
      toast.success("Attachment uploaded");
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const lessonOptions = modules.flatMap((module) =>
    (module.chapters ?? []).flatMap((chapter) =>
      (chapter.lessons ?? []).map((lesson) => ({
        id: lesson.id,
        label: `${module.title} / ${chapter.title} / ${lesson.title}`,
      })),
    ),
  );

  return (
    <div className="space-y-6">
      <SectionHeading title="Notes" />
      <div className="border border-brand/15 bg-white/50 p-4 space-y-3">
        <div className="grid md:grid-cols-2 gap-3">
          <Input
            value={form.title}
            onChange={(e) => setForm((current) => ({ ...current, title: e.target.value }))}
            placeholder="Note title"
          />
          <Input
            value={form.file_url}
            onChange={(e) => setForm((current) => ({ ...current, file_url: e.target.value }))}
            placeholder="Optional PDF/file URL"
          />
        </div>
        <label className="flex items-center justify-between gap-3 border border-dashed border-brand/15 bg-white px-3 py-2 text-sm text-brand/60">
          <span>
            {upload.isPending ? "Uploading attachment..." : "Upload PDF, image, or text attachment"}
          </span>
          <input
            type="file"
            accept="application/pdf,image/png,image/jpeg,image/webp,text/plain"
            className="max-w-[220px] text-xs"
            disabled={upload.isPending}
            onChange={(event) => {
              const file = event.target.files?.[0];
              if (file) upload.mutate(file);
              event.currentTarget.value = "";
            }}
          />
        </label>
        <textarea
          value={form.content}
          onChange={(e) => setForm((current) => ({ ...current, content: e.target.value }))}
          rows={5}
          placeholder="Rich text or markdown note content"
          className="w-full px-3 py-2 text-sm border border-brand/15 bg-white resize-y"
        />
        <div className="grid md:grid-cols-2 gap-3">
          <select
            value={form.module_id}
            onChange={(e) =>
              setForm((current) => ({ ...current, module_id: e.target.value, lesson_id: "" }))
            }
            className="h-9 border border-brand/15 bg-white px-3 text-sm"
          >
            <option value="">Course-level note</option>
            {modules.map((module) => (
              <option key={module.id} value={module.id}>
                {module.title}
              </option>
            ))}
          </select>
          <select
            value={form.lesson_id}
            onChange={(e) => setForm((current) => ({ ...current, lesson_id: e.target.value }))}
            className="h-9 border border-brand/15 bg-white px-3 text-sm"
          >
            <option value="">No lesson target</option>
            {lessonOptions.map((lesson) => (
              <option key={lesson.id} value={lesson.id}>
                {lesson.label}
              </option>
            ))}
          </select>
        </div>
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div className="flex flex-wrap gap-4 text-sm text-brand/70">
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={form.is_free}
                onChange={(e) => setForm((current) => ({ ...current, is_free: e.target.checked }))}
              />
              Free access
            </label>
            <label className="flex items-center gap-2">
              <input
                type="checkbox"
                checked={form.is_published}
                onChange={(e) =>
                  setForm((current) => ({ ...current, is_published: e.target.checked }))
                }
              />
              Published
            </label>
          </div>
          <button
            onClick={() => create.mutate()}
            disabled={
              create.isPending ||
              !form.title.trim() ||
              (!form.content.trim() && !form.file_url.trim())
            }
            className="inline-flex items-center gap-2 px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
          >
            <FileText className="h-3.5 w-3.5" />
            {create.isPending ? "Saving…" : "Add note"}
          </button>
        </div>
      </div>

      {notes.length === 0 ? (
        <p className="text-sm text-brand/55 border border-dashed border-brand/15 p-8 text-center">
          No notes yet.
        </p>
      ) : (
        <ul className="space-y-3">
          {notes.map((note) => (
            <NoteRow key={note.id} note={note} onChange={onChange} />
          ))}
        </ul>
      )}
    </div>
  );
}

function NoteRow({
  note,
  onChange,
}: {
  note: {
    id: string;
    module_id?: string | null;
    lesson_id?: string | null;
    title: string;
    content: string;
    file_url?: string;
    is_free: boolean;
    is_published: boolean;
  };
  onChange: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState({
    title: note.title,
    content: note.content,
    file_url: note.file_url ?? "",
    is_free: note.is_free,
    is_published: note.is_published,
  });

  useEffect(() => {
    setDraft({
      title: note.title,
      content: note.content,
      file_url: note.file_url ?? "",
      is_free: note.is_free,
      is_published: note.is_published,
    });
  }, [note]);

  const save = useMutation({
    mutationFn: () =>
      updateCourseNote(note.id, {
        title: draft.title.trim(),
        content: draft.content.trim(),
        file_url: draft.file_url.trim() || undefined,
        module_id: note.module_id ?? undefined,
        lesson_id: note.lesson_id ?? undefined,
        is_free: draft.is_free,
        is_published: draft.is_published,
      }),
    onSuccess: () => {
      toast.success("Note saved");
      setEditing(false);
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const remove = useMutation({
    mutationFn: () => deleteCourseNote(note.id),
    onSuccess: () => {
      toast.success("Note deleted");
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const upload = useMutation({
    mutationFn: (file: File) => uploadLessonFile(file),
    onSuccess: (result) => {
      setDraft((current) => ({ ...current, file_url: result.presigned_url }));
      toast.success("Attachment uploaded");
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <li className="border border-brand/15 bg-white/50 p-4">
      {editing ? (
        <div className="space-y-3">
          <div className="grid md:grid-cols-2 gap-3">
            <Input
              value={draft.title}
              onChange={(e) => setDraft((current) => ({ ...current, title: e.target.value }))}
            />
            <Input
              value={draft.file_url}
              onChange={(e) => setDraft((current) => ({ ...current, file_url: e.target.value }))}
              placeholder="File URL"
            />
          </div>
          <label className="flex items-center justify-between gap-3 border border-dashed border-brand/15 bg-white px-3 py-2 text-sm text-brand/60">
            <span>{upload.isPending ? "Uploading attachment..." : "Replace attachment"}</span>
            <input
              type="file"
              accept="application/pdf,image/png,image/jpeg,image/webp,text/plain"
              className="max-w-[220px] text-xs"
              disabled={upload.isPending}
              onChange={(event) => {
                const file = event.target.files?.[0];
                if (file) upload.mutate(file);
                event.currentTarget.value = "";
              }}
            />
          </label>
          <textarea
            value={draft.content}
            onChange={(e) => setDraft((current) => ({ ...current, content: e.target.value }))}
            rows={4}
            className="w-full px-3 py-2 text-sm border border-brand/15 bg-white resize-y"
          />
          <div className="flex items-center justify-between gap-3">
            <div className="flex gap-4 text-xs text-brand/70">
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={draft.is_free}
                  onChange={(e) =>
                    setDraft((current) => ({ ...current, is_free: e.target.checked }))
                  }
                />
                Free access
              </label>
              <label className="flex items-center gap-2">
                <input
                  type="checkbox"
                  checked={draft.is_published}
                  onChange={(e) =>
                    setDraft((current) => ({ ...current, is_published: e.target.checked }))
                  }
                />
                Published
              </label>
            </div>
            <div className="flex gap-2">
              <button onClick={() => setEditing(false)} className="px-3 py-2 text-xs text-brand/70">
                Cancel
              </button>
              <button
                onClick={() => save.mutate()}
                disabled={save.isPending || !draft.title.trim()}
                className="px-4 py-2 bg-brand text-white text-xs disabled:opacity-50"
              >
                {save.isPending ? "Saving…" : "Save"}
              </button>
            </div>
          </div>
        </div>
      ) : (
        <div className="flex items-start justify-between gap-3">
          <button onClick={() => setEditing(true)} className="text-left flex-1">
            <p className="font-medium text-sm">{note.title}</p>
            <p className="mt-1 text-xs text-brand/55 line-clamp-2">
              {note.content || note.file_url}
            </p>
            <p className="mt-2 text-[11px] uppercase tracking-wider text-brand/40">
              {note.is_free ? "Free" : "Paid"} · {note.is_published ? "Published" : "Draft"}
            </p>
          </button>
          <button
            onClick={() => {
              if (confirm("Delete note?")) remove.mutate();
            }}
            className="p-1.5 text-brand/45 hover:text-destructive"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
        </div>
      )}
    </li>
  );
}

function DetailsEditor({
  courseId,
  initial,
}: {
  courseId: string;
  initial: {
    title: string;
    subtitle?: string;
    description?: string;
    cover_url?: string | null;
    category?: { id: string; name: string } | null;
    level?: "beginner" | "intermediate" | "advanced";
    price?: number;
    price_type?: "free" | "paid";
    visibility?: "public" | "unlisted" | "private";
    learning_outcomes?: string;
    requirements?: string[];
    prerequisites?: string;
    target_audience?: string;
    estimated_duration_minutes?: number;
  };
}) {
  const qc = useQueryClient();
  const [form, setForm] = useState({
    title: initial.title ?? "",
    subtitle: initial.subtitle ?? "",
    description: initial.description ?? "",
    cover_url: initial.cover_url ?? "",
    subject: initial.category?.id ?? initial.category?.name ?? "",
    level: initial.level ?? "beginner",
    price_type: initial.price && initial.price > 0 ? "paid" : (initial.price_type ?? "free"),
    price_cents: Math.round((initial.price ?? 0) * 100),
    visibility: initial.visibility ?? "public",
    learning_outcomes: initial.learning_outcomes ?? "",
    requirements:
      initial.prerequisites ??
      (Array.isArray(initial.requirements) ? initial.requirements.join("\n") : ""),
    target_audience: initial.target_audience ?? "",
    estimated_duration_minutes: initial.estimated_duration_minutes ?? 0,
  });

  useEffect(() => {
    setForm({
      title: initial.title ?? "",
      subtitle: initial.subtitle ?? "",
      description: initial.description ?? "",
      cover_url: initial.cover_url ?? "",
      subject: initial.category?.id ?? initial.category?.name ?? "",
      level: initial.level ?? "beginner",
      price_type: initial.price && initial.price > 0 ? "paid" : (initial.price_type ?? "free"),
      price_cents: Math.round((initial.price ?? 0) * 100),
      visibility: initial.visibility ?? "public",
      learning_outcomes: initial.learning_outcomes ?? "",
      requirements:
        initial.prerequisites ??
        (Array.isArray(initial.requirements) ? initial.requirements.join("\n") : ""),
      target_audience: initial.target_audience ?? "",
      estimated_duration_minutes: initial.estimated_duration_minutes ?? 0,
    });
  }, [initial]);

  const save = useMutation({
    mutationFn: () =>
      updateCourse(courseId, {
        ...form,
        cover_url: form.cover_url || undefined,
        price_cents: form.price_type === "paid" ? form.price_cents : 0,
        prerequisites: form.requirements,
        estimated_duration_minutes: Number(form.estimated_duration_minutes) || 0,
      }),
    onSuccess: () => {
      toast.success("Saved");
      qc.invalidateQueries({ queryKey: ["teacher-course", courseId] });
      qc.invalidateQueries({ queryKey: ["taught-courses"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const Field = ({ label, children }: { label: string; children: React.ReactNode }) => (
    <label className="block mb-5">
      <span className="eyebrow text-brand/45 mb-2 block">{label}</span>
      {children}
    </label>
  );

  return (
    <div className="max-w-2xl">
      <Field label="Title">
        <input
          value={form.title}
          onChange={(e) => setForm({ ...form, title: e.target.value })}
          className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50"
        />
      </Field>
      <Field label="Summary">
        <input
          value={form.subtitle}
          onChange={(e) => setForm({ ...form, subtitle: e.target.value })}
          className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50"
        />
      </Field>
      <Field label="Description">
        <textarea
          value={form.description}
          onChange={(e) => setForm({ ...form, description: e.target.value })}
          rows={6}
          className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50 resize-y"
        />
      </Field>
      <Field label="Cover image URL">
        <input
          value={form.cover_url}
          onChange={(e) => setForm({ ...form, cover_url: e.target.value })}
          placeholder="https://…"
          className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50"
        />
      </Field>
      <div className="grid gap-4 sm:grid-cols-2">
        <Field label="Category / subject">
          <input
            value={form.subject}
            onChange={(e) => setForm({ ...form, subject: e.target.value })}
            className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50"
          />
        </Field>
        <Field label="Level">
          <select
            value={form.level}
            onChange={(e) =>
              setForm({
                ...form,
                level: e.target.value as "beginner" | "intermediate" | "advanced",
              })
            }
            className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50"
          >
            <option value="beginner">Beginner</option>
            <option value="intermediate">Intermediate</option>
            <option value="advanced">Advanced</option>
          </select>
        </Field>
        <Field label="Access">
          <select
            value={form.price_type}
            onChange={(e) => setForm({ ...form, price_type: e.target.value as "free" | "paid" })}
            className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50"
          >
            <option value="free">Free</option>
            <option value="paid">Paid / admin approved</option>
          </select>
        </Field>
        <Field label="Price label">
          <input
            type="number"
            min={0}
            value={Math.round(form.price_cents / 100)}
            onChange={(e) =>
              setForm({ ...form, price_cents: Math.max(0, Number(e.target.value) || 0) * 100 })
            }
            disabled={form.price_type === "free"}
            className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50 disabled:opacity-50"
          />
        </Field>
        <Field label="Visibility">
          <select
            value={form.visibility}
            onChange={(e) =>
              setForm({
                ...form,
                visibility: e.target.value as "public" | "unlisted" | "private",
              })
            }
            className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50"
          >
            <option value="public">Public</option>
            <option value="unlisted">Unlisted</option>
            <option value="private">Private</option>
          </select>
        </Field>
        <Field label="Estimated duration">
          <input
            type="number"
            min={0}
            value={form.estimated_duration_minutes}
            onChange={(e) =>
              setForm({
                ...form,
                estimated_duration_minutes: Math.max(0, Number(e.target.value) || 0),
              })
            }
            className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50"
          />
        </Field>
      </div>
      <Field label="Learning outcomes">
        <textarea
          value={form.learning_outcomes}
          onChange={(e) => setForm({ ...form, learning_outcomes: e.target.value })}
          rows={4}
          placeholder="One outcome per line"
          className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50 resize-y"
        />
      </Field>
      <Field label="Requirements">
        <textarea
          value={form.requirements}
          onChange={(e) => setForm({ ...form, requirements: e.target.value })}
          rows={4}
          placeholder="One requirement per line"
          className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50 resize-y"
        />
      </Field>
      <Field label="Target audience">
        <textarea
          value={form.target_audience}
          onChange={(e) => setForm({ ...form, target_audience: e.target.value })}
          rows={3}
          className="w-full px-3 py-2.5 text-sm border border-brand/15 bg-white/50 resize-y"
        />
      </Field>

      <button
        onClick={() => save.mutate()}
        disabled={save.isPending}
        className="px-5 py-2.5 bg-brand text-white text-sm font-medium disabled:opacity-50"
      >
        {save.isPending ? "Saving…" : "Save changes"}
      </button>
    </div>
  );
}

function StudentsPanel({ courseId }: { courseId: string }) {
  const students = useQuery({
    queryKey: ["course-students", courseId],
    queryFn: () => listCourseStudents(courseId, { limit: 100 }),
  });

  if (students.isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 5 }).map((_, i) => (
          <div key={i} className="h-14 border border-brand/10 bg-white/30 animate-pulse" />
        ))}
      </div>
    );
  }

  if (!students.data || students.data.data.length === 0) {
    return (
      <p className="text-sm text-brand/55 border border-dashed border-brand/15 p-8 text-center">
        <Users className="h-6 w-6 mx-auto mb-3 text-brand/30" />
        No students enrolled yet.
      </p>
    );
  }

  return (
    <ul className="divide-y divide-brand/10 border-y border-brand/10">
      {students.data.data.map((s) => (
        <li key={s.id} className="flex items-center gap-4 py-4 px-2">
          <div className="h-9 w-9 grid place-items-center bg-brand/10 text-brand text-xs font-medium">
            {s.full_name.charAt(0)}
          </div>
          <div className="flex-1 min-w-0">
            <p className="font-medium text-brand truncate">{s.full_name}</p>
            <p className="text-xs text-brand/50 truncate">{s.email}</p>
          </div>
          <div className="text-right shrink-0">
            <p className="text-sm font-medium">{s.progress_percent}%</p>
            <p className="text-[11px] text-brand/45">
              {s.last_active_at ? `Active ${new Date(s.last_active_at).toLocaleDateString()}` : "—"}
            </p>
          </div>
        </li>
      ))}
    </ul>
  );
}

function getMediaDurationSeconds(file: File) {
  return new Promise<number>((resolve, reject) => {
    const media = document.createElement("video");
    const objectUrl = URL.createObjectURL(file);
    const cleanup = () => {
      media.removeAttribute("src");
      media.load();
      URL.revokeObjectURL(objectUrl);
    };

    media.preload = "metadata";
    media.onloadedmetadata = () => {
      const duration = media.duration;
      cleanup();
      if (Number.isFinite(duration) && duration > 0) {
        resolve(duration);
      } else {
        reject(new Error("Invalid media duration"));
      }
    };
    media.onerror = () => {
      cleanup();
      reject(new Error("Could not load media metadata"));
    };
    media.src = objectUrl;
  });
}

function secondsToMinutesInput(seconds: number) {
  const minutes = seconds / 60;
  if (Number.isInteger(minutes)) return String(minutes);
  return minutes.toFixed(2).replace(/0+$/, "").replace(/\.$/, "");
}

function durationMinutesToSeconds(minutes: string | number) {
  return Math.max(0, Math.round((Number(minutes) || 0) * 60));
}

function VideoUploader({
  lessonId,
  onUploaded,
}: {
  lessonId: string;
  onUploaded: (publicUrl: string) => void;
}) {
  const [progress, setProgress] = useState<number | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function handleFile(file: File) {
    setError(null);
    setProgress(0);
    try {
      const ticket = await requestLessonVideoUpload(lessonId, {
        filename: file.name,
        content_type: file.type || "application/octet-stream",
        size_bytes: file.size,
      });
      await uploadLessonVideoFile(ticket, file, setProgress);
      await finalizeLessonVideo(lessonId, {
        public_url: ticket.public_url,
        asset_id: ticket.asset_id,
      });
      onUploaded(ticket.public_url);
      toast.success("Video uploaded");
      setProgress(null);
    } catch (e) {
      setError((e as Error).message);
      setProgress(null);
    }
  }

  return (
    <div className="border border-dashed border-brand/20 bg-white/50 p-3">
      <label className="flex items-center gap-3 text-xs text-brand/70 cursor-pointer">
        <span className="px-3 py-1.5 bg-brand text-white">Upload file</span>
        <span className="text-brand/55">MP4 / MOV / WebM up to ~2&nbsp;GB</span>
        <input
          type="file"
          accept="video/*"
          className="hidden"
          disabled={progress !== null}
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) handleFile(f);
            e.target.value = "";
          }}
        />
      </label>
      {progress !== null && (
        <div className="mt-3">
          <div className="h-1.5 bg-brand/10 overflow-hidden">
            <div className="h-full bg-accent transition-all" style={{ width: `${progress}%` }} />
          </div>
          <p className="text-[11px] text-brand/55 mt-1">Uploading… {progress}%</p>
        </div>
      )}
      {error && <p className="mt-2 text-[11px] text-destructive">{error}</p>}
    </div>
  );
}
