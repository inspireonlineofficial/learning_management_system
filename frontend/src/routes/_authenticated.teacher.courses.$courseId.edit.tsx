import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import {
  ArrowLeft,
  ChevronDown,
  ChevronRight,
  Plus,
  Save,
  Trash2,
  Upload,
  Users,
} from "lucide-react";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { AppShell, SectionHeading } from "@/components/layout/app-shell";
import { Input } from "@/components/ui/input";
import {
  createChapter,
  createLesson,
  createModule,
  deleteChapter,
  deleteLesson,
  deleteModule,
  getTeacherCourse,
  listCourseStudents,
  reorderContent,
  uploadLessonVideo,
  updateCourse,
  updateChapter,
  updateLesson,
  updateModule,
} from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/edit")({
  component: EditCoursePage,
});

function EditCoursePage() {
  const { courseId } = Route.useParams();
  return <CourseEditor courseId={courseId} />;
}

export function CourseEditor({ courseId }: { courseId: string }) {
  const qc = useQueryClient();
  const [tab, setTab] = useState<"content" | "details" | "students">("content");

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
        {(["content", "details", "students"] as const).map((t) => (
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
    position?: number;
    chapters?: {
      id: string;
      title: string;
      position?: number;
      lessons: {
        id: string;
        title: string;
        type?: string;
        duration_seconds?: number;
        is_free_preview?: boolean;
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

  const create = useMutation({
    mutationFn: () =>
      createModule(courseId, { title: moduleTitle.trim(), position: modules.length + 1 }),
    onSuccess: () => {
      toast.success("Module added");
      setModuleTitle("");
      setAddingModule(false);
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const moveModule = (moduleId: string, direction: "up" | "down") => {
    const ordered = [...modules].sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
    const idx = ordered.findIndex((m) => m.id === moduleId);
    const swap = direction === "up" ? idx - 1 : idx + 1;
    if (idx < 0 || swap < 0 || swap >= ordered.length) return;
    const next = ordered.slice();
    [next[idx], next[swap]] = [next[swap], next[idx]];
    reorderContent({
      type: "module",
      parent_id: courseId,
      positions: Object.fromEntries(next.map((m, i) => [m.id, i + 1])),
    })
      .then(onChange)
      .catch((e) => toast.error((e as Error).message));
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
          {modules
            .slice()
            .sort((a, b) => (a.position ?? 0) - (b.position ?? 0))
            .map((module, index) => (
              <ModuleCard
                key={module.id}
                courseId={courseId}
                module={module}
                index={index}
                total={modules.length}
                onChange={onChange}
                onMove={moveModule}
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
}: {
  courseId: string;
  module: {
    id: string;
    title: string;
    chapters?: {
      id: string;
      title: string;
      position?: number;
      lessons: {
        id: string;
        title: string;
        type?: string;
        duration_seconds?: number;
        is_free_preview?: boolean;
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
}) {
  const [open, setOpen] = useState(true);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(module.title);
  const [chapterTitle, setChapterTitle] = useState("");

  useEffect(() => {
    setTitle(module.title);
  }, [module.title]);

  const rename = useMutation({
    mutationFn: () => updateModule(module.id, { title: title.trim(), position: index + 1 }),
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
    <li className="border border-brand/15 bg-white/40">
      <div className="flex items-center gap-3 px-4 py-3">
        <button onClick={() => setOpen((v) => !v)} className="text-brand/55 hover:text-brand">
          <ChevronDown className={`h-4 w-4 transition-transform ${open ? "" : "-rotate-90"}`} />
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
              className="flex-1 text-left font-serif text-lg hover:text-accent"
            >
              {module.title}
            </button>
            <span className="text-xs text-brand/45">{chapters.length} chapters</span>
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
      type?: string;
      duration_seconds?: number;
      is_free_preview?: boolean;
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
  const [lessonDownloadable, setLessonDownloadable] = useState(false);
  const [lessonVideo, setLessonVideo] = useState<File | null>(null);

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
        duration_seconds: Math.max(0, Number(lessonDuration) || 0) * 60,
        is_free_preview: lessonPreview,
        is_downloadable: lessonDownloadable,
        status: lessonStatus,
      });
    },
    onSuccess: () => {
      toast.success("Lesson added");
      setLessonTitle("");
      setLessonVideo(null);
      onChange();
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const moveLesson = (lessonId: string, direction: "up" | "down") => {
    const ordered = [...chapter.lessons].sort((a, b) => (a.position ?? 0) - (b.position ?? 0));
    const idx = ordered.findIndex((lesson) => lesson.id === lessonId);
    const swap = direction === "up" ? idx - 1 : idx + 1;
    if (idx < 0 || swap < 0 || swap >= ordered.length) return;
    const next = ordered.slice();
    [next[idx], next[swap]] = [next[swap], next[idx]];
    reorderContent({
      type: "lesson",
      parent_id: chapter.id,
      positions: Object.fromEntries(next.map((lesson, i) => [lesson.id, i + 1])),
    })
      .then(onChange)
      .catch((e) => toast.error((e as Error).message));
  };

  const lessons = chapter.lessons.slice().sort((a, b) => (a.position ?? 0) - (b.position ?? 0));

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
                value={lessonDuration}
                onChange={(e) => setLessonDuration(e.target.value)}
                placeholder="Duration minutes"
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
                  checked={lessonDownloadable}
                  onChange={(e) => setLessonDownloadable(e.target.checked)}
                />
                Downloadable
              </label>
              <Input
                type="file"
                accept="video/mp4,video/webm,video/quicktime"
                onChange={(e) => setLessonVideo(e.target.files?.[0] ?? null)}
              />
            </div>
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
}: {
  courseId: string;
  lesson: {
    id: string;
    title: string;
    type?: string;
    duration_seconds?: number;
    is_free_preview?: boolean;
    is_downloadable?: boolean;
    status?: string;
    video_id?: string | null;
  };
  index: number;
  total: number;
  onChange: () => void;
  onMove: (lessonId: string, direction: "up" | "down") => void;
}) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState({
    title: lesson.title,
    type: (lesson.type as "video" | "text" | "attachment") ?? "video",
    duration_minutes: lesson.duration_seconds ? Math.round(lesson.duration_seconds / 60) : 0,
    is_free_preview: lesson.is_free_preview ?? false,
    is_downloadable: lesson.is_downloadable ?? false,
    status: (lesson.status as "draft" | "published") ?? "draft",
  });
  const [videoFile, setVideoFile] = useState<File | null>(null);

  useEffect(() => {
    setDraft({
      title: lesson.title,
      type: (lesson.type as "video" | "text" | "attachment") ?? "video",
      duration_minutes: lesson.duration_seconds ? Math.round(lesson.duration_seconds / 60) : 0,
      is_free_preview: lesson.is_free_preview ?? false,
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
        type: draft.type,
        video_id: videoId,
        duration_seconds: Math.max(0, Number(draft.duration_minutes) || 0) * 60,
        is_free_preview: draft.is_free_preview,
        is_downloadable: draft.is_downloadable,
        status: draft.status,
      });
    },
    onSuccess: () => {
      toast.success("Lesson saved");
      setEditing(false);
      setVideoFile(null);
      onChange();
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

  return (
    <li className="border border-brand/10 bg-white/60 p-4">
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

          <div className="grid md:grid-cols-3 gap-3">
            <Input
              type="number"
              min={0}
              value={draft.duration_minutes}
              onChange={(e) =>
                setDraft((current) => ({
                  ...current,
                  duration_minutes: Number(e.target.value) || 0,
                }))
              }
              placeholder="Duration minutes"
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
              type="file"
              accept="video/mp4,video/webm,video/quicktime"
              onChange={(e) => setVideoFile(e.target.files?.[0] ?? null)}
            />
          </div>

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
        <div className="flex items-center justify-between gap-3">
          <button onClick={() => setEditing(true)} className="text-left flex-1">
            <p className="font-medium text-sm">{lesson.title}</p>
            <p className="mt-1 text-[11px] text-brand/45 uppercase tracking-wider">
              {lesson.type ?? "video"}
              {lesson.is_free_preview ? " · preview" : ""}
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
            className="p-1.5 text-brand/45 hover:text-destructive"
          >
            <Trash2 className="h-3.5 w-3.5" />
          </button>
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

function DetailsEditor({
  courseId,
  initial,
}: {
  courseId: string;
  initial: { title: string; subtitle?: string; description?: string; cover_url?: string | null };
}) {
  const qc = useQueryClient();
  const [form, setForm] = useState({
    title: initial.title ?? "",
    subtitle: initial.subtitle ?? "",
    description: initial.description ?? "",
    cover_url: initial.cover_url ?? "",
  });

  useEffect(() => {
    setForm({
      title: initial.title ?? "",
      subtitle: initial.subtitle ?? "",
      description: initial.description ?? "",
      cover_url: initial.cover_url ?? "",
    });
  }, [initial]);

  const save = useMutation({
    mutationFn: () => updateCourse(courseId, { ...form, cover_url: form.cover_url || undefined }),
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
