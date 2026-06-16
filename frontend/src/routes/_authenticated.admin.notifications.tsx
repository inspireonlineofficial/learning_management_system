import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Bell, Send } from "lucide-react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import {
  broadcastNotification,
  listBroadcasts,
  listNotificationTemplates,
  updateNotificationTemplate,
  type BroadcastAudience,
  type BroadcastInput,
  type NotificationTemplate,
} from "@/lib/api/notifications";

export const Route = createFileRoute("/_authenticated/admin/notifications")({
  component: Page,
});

const TITLE_MAX = 80;
const BODY_MAX = 500;

function Page() {
  const qc = useQueryClient();
  const [form, setForm] = useState<BroadcastInput>({
    audience: "all",
    title: "",
    body: "",
  });
  const [schedule, setSchedule] = useState(false);

  const history = useQuery({ queryKey: ["broadcasts"], queryFn: listBroadcasts });
  const templates = useQuery({
    queryKey: ["notification-templates"],
    queryFn: listNotificationTemplates,
  });
  const [editing, setEditing] = useState<NotificationTemplate | null>(null);
  const [templateDraft, setTemplateDraft] = useState({
    subject_template: "",
    body_template: "",
  });

  const mut = useMutation({
    mutationFn: () => broadcastNotification(form),
    onSuccess: (r) => {
      toast.success(
        r.scheduled
          ? "Broadcast scheduled"
          : `Sent to ${r.sent_count} user${r.sent_count === 1 ? "" : "s"}`,
      );
      setForm({ audience: form.audience, title: "", body: "" });
      setSchedule(false);
      qc.invalidateQueries({ queryKey: ["broadcasts"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const saveTemplate = useMutation({
    mutationFn: () => {
      if (!editing) throw new Error("No template selected");
      return updateNotificationTemplate(editing.id, {
        subject_template: templateDraft.subject_template.trim() || null,
        body_template: templateDraft.body_template,
      });
    },
    onSuccess: () => {
      toast.success("Template updated");
      setEditing(null);
      qc.invalidateQueries({ queryKey: ["notification-templates"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  const canSend =
    form.title.trim().length > 0 &&
    form.body.trim().length > 0 &&
    (form.audience !== "course" || !!form.course_id) &&
    (form.audience !== "user" || (form.user_ids?.length ?? 0) > 0) &&
    (!schedule || !!form.scheduled_for);

  return (
    <AppShell eyebrow="Notifications" title="Notification center">
      <div className="grid gap-8 lg:grid-cols-[1fr_360px]">
        <div className="space-y-5 max-w-xl">
          <label className="block">
            <span className="text-xs eyebrow text-brand/45">Audience</span>
            <select
              value={form.audience}
              onChange={(e) => setForm({ ...form, audience: e.target.value as BroadcastAudience })}
              className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
            >
              <option value="all">Everyone</option>
              <option value="students">All students</option>
              <option value="teachers">All teachers</option>
              <option value="course">Enrolled in a course</option>
              <option value="user">Specific users</option>
            </select>
          </label>

          {form.audience === "course" && (
            <label className="block">
              <span className="text-xs eyebrow text-brand/45">Course ID</span>
              <input
                className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
                value={form.course_id ?? ""}
                onChange={(e) => setForm({ ...form, course_id: e.target.value })}
                placeholder="course_…"
              />
            </label>
          )}

          {form.audience === "user" && (
            <label className="block">
              <span className="text-xs eyebrow text-brand/45">User IDs (comma-separated)</span>
              <textarea
                rows={2}
                className="mt-1 w-full border border-brand/15 bg-white p-3 text-sm font-mono"
                value={form.user_ids?.join(", ") ?? ""}
                onChange={(e) =>
                  setForm({
                    ...form,
                    user_ids: e.target.value
                      .split(",")
                      .map((s) => s.trim())
                      .filter(Boolean),
                  })
                }
              />
            </label>
          )}

          <label className="block">
            <div className="flex justify-between items-baseline">
              <span className="text-xs eyebrow text-brand/45">Title</span>
              <span className="text-xs text-brand/40">
                {form.title.length}/{TITLE_MAX}
              </span>
            </div>
            <input
              maxLength={TITLE_MAX}
              className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              value={form.title}
              onChange={(e) => setForm({ ...form, title: e.target.value })}
            />
          </label>

          <label className="block">
            <div className="flex justify-between items-baseline">
              <span className="text-xs eyebrow text-brand/45">Body</span>
              <span className="text-xs text-brand/40">
                {form.body.length}/{BODY_MAX}
              </span>
            </div>
            <textarea
              rows={5}
              maxLength={BODY_MAX}
              className="mt-1 w-full border border-brand/15 bg-white p-3 text-sm"
              value={form.body}
              onChange={(e) => setForm({ ...form, body: e.target.value })}
            />
          </label>

          <label className="block">
            <span className="text-xs eyebrow text-brand/45">Action URL (optional)</span>
            <input
              className="mt-1 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              placeholder="/student/courses/abc or https://…"
              value={form.action_url ?? ""}
              onChange={(e) => setForm({ ...form, action_url: e.target.value })}
            />
          </label>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={schedule}
              onChange={(e) => {
                setSchedule(e.target.checked);
                if (!e.target.checked) setForm({ ...form, scheduled_for: undefined });
              }}
            />
            Schedule for later
          </label>
          {schedule && (
            <input
              type="datetime-local"
              className="border border-brand/15 bg-white px-3 py-2 text-sm"
              value={form.scheduled_for ?? ""}
              onChange={(e) => setForm({ ...form, scheduled_for: e.target.value })}
            />
          )}

          <button
            onClick={() => mut.mutate()}
            disabled={mut.isPending || !canSend}
            className="inline-flex items-center gap-2 bg-brand text-white px-6 py-2 text-sm disabled:opacity-50"
          >
            <Send className="h-4 w-4" />
            {mut.isPending ? "Sending…" : schedule ? "Schedule broadcast" : "Send broadcast"}
          </button>
        </div>

        <aside className="space-y-6">
          <div>
            <p className="text-xs eyebrow text-brand/45 mb-2">Preview</p>
            <div className="border border-brand/10 bg-white/60 p-4 rounded-sm">
              <div className="flex items-start gap-3">
                <Bell className="h-4 w-4 text-brand/60 mt-0.5" />
                <div className="min-w-0">
                  <p className="text-sm font-medium">{form.title || "Notification title"}</p>
                  <p className="mt-1 text-xs text-brand/70 whitespace-pre-line">
                    {form.body || "Notification body preview…"}
                  </p>
                  {form.action_url && (
                    <p className="mt-2 text-xs text-accent underline underline-offset-2 truncate">
                      {form.action_url}
                    </p>
                  )}
                </div>
              </div>
            </div>
          </div>

          <div>
            <p className="text-xs eyebrow text-brand/45 mb-2">Recent broadcasts</p>
            <div className="space-y-2 max-h-[400px] overflow-y-auto">
              {history.isLoading ? (
                <p className="text-xs text-brand/50">Loading…</p>
              ) : history.data?.items.length === 0 ? (
                <p className="text-xs text-brand/50">No broadcasts yet.</p>
              ) : (
                history.data?.items.map((b) => (
                  <div key={b.id} className="border border-brand/10 bg-white/40 p-3 text-xs">
                    <div className="flex justify-between gap-2">
                      <p className="font-medium truncate">{b.title}</p>
                      <span
                        className={`shrink-0 px-1.5 py-0.5 text-[10px] uppercase tracking-wide ${
                          b.status === "sent"
                            ? "bg-emerald-100 text-emerald-700"
                            : b.status === "scheduled"
                              ? "bg-amber-100 text-amber-700"
                              : "bg-destructive/10 text-destructive"
                        }`}
                      >
                        {b.status}
                      </span>
                    </div>
                    <p className="mt-1 text-brand/55">
                      {b.audience} · {b.sent_count} recipients ·{" "}
                      {new Date(b.created_at).toLocaleDateString()}
                    </p>
                  </div>
                ))
              )}
            </div>
          </div>
        </aside>
      </div>

      <section className="mt-14">
        <div className="flex items-center justify-between gap-4 border-b border-brand/10 pb-3">
          <div>
            <p className="eyebrow text-brand/45">Templates</p>
            <h2 className="font-serif text-2xl">System notification templates</h2>
          </div>
        </div>

        {templates.isLoading ? (
          <div className="mt-6 grid gap-3">
            {Array.from({ length: 4 }).map((_, index) => (
              <div key={index} className="h-20 border border-brand/10 bg-white/30 animate-pulse" />
            ))}
          </div>
        ) : templates.isError ? (
          <div className="mt-6 border border-destructive/20 bg-destructive/5 p-6 text-sm">
            <p className="font-medium text-destructive">Couldn't load templates</p>
            <p className="mt-1 text-brand/60">{(templates.error as Error).message}</p>
          </div>
        ) : (
          <div className="mt-6 grid gap-3">
            {templates.data?.items.map((template) => (
              <div
                key={template.id}
                className="border border-brand/10 bg-white/40 p-4 flex flex-col lg:flex-row lg:items-start lg:justify-between gap-4"
              >
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-medium text-brand">{template.type.replaceAll("_", " ")}</p>
                    <span className="eyebrow text-[10px] text-brand/45">{template.channel}</span>
                  </div>
                  {template.subject_template && (
                    <p className="mt-1 text-sm text-brand/75 truncate">
                      {template.subject_template}
                    </p>
                  )}
                  <p className="mt-1 text-xs text-brand/55 line-clamp-2">
                    {template.body_template}
                  </p>
                  {template.allowed_variables.length > 0 && (
                    <p className="mt-2 text-[11px] text-brand/45">
                      Variables: {template.allowed_variables.map((v) => `{{${v}}}`).join(", ")}
                    </p>
                  )}
                </div>
                <button
                  onClick={() => {
                    setEditing(template);
                    setTemplateDraft({
                      subject_template: template.subject_template ?? "",
                      body_template: template.body_template,
                    });
                  }}
                  className="self-start text-xs border border-brand/15 px-3 py-1.5 hover:bg-brand/[0.03]"
                >
                  Edit template
                </button>
              </div>
            ))}
          </div>
        )}
      </section>

      {editing && (
        <div
          className="fixed inset-0 z-50 bg-black/40 grid place-items-center p-4"
          onClick={() => setEditing(null)}
        >
          <div
            className="bg-white border border-brand/10 max-w-2xl w-full p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <p className="eyebrow text-brand/45">Edit template</p>
            <h3 className="font-serif text-2xl">{editing.type.replaceAll("_", " ")}</h3>
            <label className="block mt-5">
              <span className="eyebrow text-brand/45">Subject</span>
              <input
                value={templateDraft.subject_template}
                onChange={(e) =>
                  setTemplateDraft({ ...templateDraft, subject_template: e.target.value })
                }
                className="mt-2 w-full border border-brand/15 bg-white px-3 py-2 text-sm"
              />
            </label>
            <label className="block mt-4">
              <span className="eyebrow text-brand/45">Body</span>
              <textarea
                value={templateDraft.body_template}
                onChange={(e) =>
                  setTemplateDraft({ ...templateDraft, body_template: e.target.value })
                }
                rows={7}
                className="mt-2 w-full border border-brand/15 bg-white p-3 text-sm"
              />
            </label>
            {editing.allowed_variables.length > 0 && (
              <p className="mt-2 text-xs text-brand/55">
                Variables: {editing.allowed_variables.map((v) => `{{${v}}}`).join(", ")}
              </p>
            )}
            <div className="mt-4 border border-brand/10 bg-brand/[0.02] p-4">
              <p className="eyebrow text-brand/45">Preview</p>
              <p className="mt-2 text-sm font-medium">
                {renderPreview(templateDraft.subject_template) || "No subject"}
              </p>
              <p className="mt-1 text-sm text-brand/70 whitespace-pre-line">
                {renderPreview(templateDraft.body_template)}
              </p>
            </div>
            <div className="mt-5 flex justify-end gap-2">
              <button
                onClick={() => setEditing(null)}
                className="px-4 py-2 text-sm border border-brand/15"
              >
                Cancel
              </button>
              <button
                onClick={() => saveTemplate.mutate()}
                disabled={saveTemplate.isPending || !templateDraft.body_template.trim()}
                className="px-5 py-2 text-sm bg-brand text-white disabled:opacity-50"
              >
                {saveTemplate.isPending ? "Saving..." : "Save template"}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}

function renderPreview(template: string) {
  return template
    .replaceAll("{{student_name}}", "Aisha Rahman")
    .replaceAll("{{teacher_name}}", "Dr. Karim")
    .replaceAll("{{course_title}}", "College Physics")
    .replaceAll("{{assignment_title}}", "Lab report")
    .replaceAll("{{due_at}}", "Friday 5:00 PM");
}
