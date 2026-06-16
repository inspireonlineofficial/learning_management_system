import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, CheckCircle2, Flag, Heart, Lock, Pin } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import {
  flagForumPost,
  flagForumThread,
  getForumThread,
  listForumPosts,
  replyToThread,
  togglePostLike,
  type ForumPost,
} from "@/lib/api/forum";

export const Route = createFileRoute("/_authenticated/student/forum/$threadId")({
  component: ThreadPage,
});

type ReportTarget = { kind: "post" | "thread"; id: string } | null;

function ThreadPage() {
  const { threadId } = Route.useParams();
  const qc = useQueryClient();

  const thread = useQuery({
    queryKey: ["forum-thread", threadId],
    queryFn: () => getForumThread(threadId),
  });

  const posts = useQuery({
    queryKey: ["forum-posts", threadId],
    queryFn: () => listForumPosts(threadId, { limit: 100 }),
  });

  const [reply, setReply] = useState("");
  const [report, setReport] = useState<ReportTarget>(null);

  const submit = useMutation({
    mutationFn: () => replyToThread(threadId, reply.trim()),
    onSuccess: () => {
      setReply("");
      toast.success("Reply posted");
      qc.invalidateQueries({ queryKey: ["forum-posts", threadId] });
      qc.invalidateQueries({ queryKey: ["forum-thread", threadId] });
      qc.invalidateQueries({ queryKey: ["forum-threads"] });
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not post reply"),
  });

  if (thread.isLoading) {
    return (
      <AppShell>
        <div className="h-12 w-2/3 bg-brand/10 animate-pulse mb-4" />
        <div className="h-40 bg-brand/5 animate-pulse" />
      </AppShell>
    );
  }

  if (thread.isError || !thread.data) {
    return (
      <AppShell title="Thread unavailable">
        <p className="text-sm text-brand/60">{(thread.error as Error)?.message ?? "Not found"}</p>
      </AppShell>
    );
  }

  const t = thread.data;

  return (
    <AppShell>
      <Link
        to="/student/forum"
        className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-6"
      >
        <ArrowLeft className="h-3.5 w-3.5" />
        Back to community
      </Link>

      <article className="mb-10">
        <div className="flex items-center gap-2 mb-3">
          {t.is_pinned && <Pin className="h-3.5 w-3.5 text-accent" />}
          {t.is_resolved && (
            <span className="flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium text-emerald-700 bg-emerald-50 border border-emerald-200">
              <CheckCircle2 className="h-3 w-3" />
              Resolved
            </span>
          )}
          {t.is_locked && (
            <span className="flex items-center gap-1 px-2 py-0.5 text-[11px] font-medium text-brand/55 border border-brand/15">
              <Lock className="h-3 w-3" />
              Locked
            </span>
          )}
          {t.category_name && <span className="eyebrow text-brand/45">{t.category_name}</span>}
        </div>
        <h1 className="font-serif text-3xl lg:text-4xl text-balance mb-4">{t.title}</h1>
        <p className="text-xs text-brand/55 mb-6">
          {t.author.full_name} · {new Date(t.created_at).toLocaleString()}
        </p>
        <div
          className="prose prose-sm max-w-none text-brand/80 whitespace-pre-wrap"
          dangerouslySetInnerHTML={t.body_html ? { __html: t.body_html } : undefined}
        >
          {t.body_html ? undefined : t.body}
        </div>
        <button
          onClick={() => setReport({ kind: "thread", id: threadId })}
          className="mt-4 inline-flex items-center gap-1.5 text-xs text-brand/55 hover:text-destructive"
        >
          <Flag className="h-3.5 w-3.5" /> Report thread
        </button>
      </article>

      <section>
        <h2 className="font-serif text-xl mb-4">
          {posts.data?.meta.total ?? 0} repl{(posts.data?.meta.total ?? 0) === 1 ? "y" : "ies"}
        </h2>

        {posts.isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="h-24 border border-brand/10 bg-white/30 animate-pulse" />
            ))}
          </div>
        ) : (
          <ul className="space-y-4 mb-8">
            {(posts.data?.data ?? []).map((p) => (
              <PostItem
                key={p.id}
                post={p}
                threadId={threadId}
                onReport={() => setReport({ kind: "post", id: p.id })}
              />
            ))}
          </ul>
        )}

        {t.is_locked ? (
          <p className="text-xs text-brand/55 border border-brand/10 bg-brand/[0.02] p-4">
            This thread is locked.
          </p>
        ) : (
          <div className="border border-brand/15 bg-white/50 p-5">
            <p className="eyebrow text-accent mb-3">Reply</p>
            <textarea
              value={reply}
              onChange={(e) => setReply(e.target.value)}
              placeholder="Add your reply…"
              rows={4}
              className="w-full px-3 py-2 text-sm border border-brand/15 bg-white/50 focus:outline-none focus:border-brand/40 resize-y"
            />
            <div className="flex justify-end mt-3">
              <button
                onClick={() => submit.mutate()}
                disabled={submit.isPending || !reply.trim()}
                className="px-5 py-2 bg-brand text-white text-xs font-medium disabled:opacity-50"
              >
                {submit.isPending ? "Posting…" : "Post reply"}
              </button>
            </div>
          </div>
        )}
      </section>

      {report && (
        <ReportDialog
          target={report}
          onClose={() => setReport(null)}
          onSubmitted={() => setReport(null)}
        />
      )}
    </AppShell>
  );
}

function PostItem({
  post,
  threadId,
  onReport,
}: {
  post: ForumPost;
  threadId: string;
  onReport: () => void;
}) {
  const qc = useQueryClient();
  const like = useMutation({
    mutationFn: () => togglePostLike(post.id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["forum-posts", threadId] });
    },
  });
  return (
    <li
      className={`border p-5 ${
        post.is_answer ? "border-emerald-300 bg-emerald-50/40" : "border-brand/10 bg-white/40"
      }`}
    >
      <div className="flex items-center justify-between mb-3">
        <div>
          <p className="text-sm font-medium text-brand">{post.author.full_name}</p>
          <p className="text-xs text-brand/50">
            {new Date(post.created_at).toLocaleString()}
            {post.author.role && ` · ${post.author.role}`}
          </p>
        </div>
        {post.is_answer && (
          <span className="flex items-center gap-1 px-2 py-1 text-[11px] font-medium text-emerald-700 bg-emerald-100 border border-emerald-300">
            <CheckCircle2 className="h-3 w-3" />
            Answer
          </span>
        )}
      </div>
      <div
        className="prose prose-sm max-w-none text-brand/80 whitespace-pre-wrap"
        dangerouslySetInnerHTML={post.body_html ? { __html: post.body_html } : undefined}
      >
        {post.body_html ? undefined : post.body}
      </div>
      <div className="mt-4 flex items-center gap-4">
        <button
          onClick={() => like.mutate()}
          disabled={like.isPending}
          className={`inline-flex items-center gap-1.5 text-xs ${
            post.has_liked ? "text-destructive" : "text-brand/55 hover:text-brand"
          }`}
        >
          <Heart className={`h-3.5 w-3.5 ${post.has_liked ? "fill-current" : ""}`} />
          {post.like_count ?? 0}
        </button>
        <button
          onClick={onReport}
          className="inline-flex items-center gap-1.5 text-xs text-brand/55 hover:text-destructive"
        >
          <Flag className="h-3.5 w-3.5" /> Report
        </button>
      </div>
    </li>
  );
}

const REPORT_REASONS = [
  "Spam",
  "Harassment or abuse",
  "Off-topic",
  "Inappropriate content",
  "Other",
];

function ReportDialog({
  target,
  onClose,
  onSubmitted,
}: {
  target: { kind: "post" | "thread"; id: string };
  onClose: () => void;
  onSubmitted: () => void;
}) {
  const [reason, setReason] = useState(REPORT_REASONS[0]);
  const [details, setDetails] = useState("");
  const submit = useMutation({
    mutationFn: () =>
      target.kind === "post"
        ? flagForumPost(target.id, reason, details || undefined)
        : flagForumThread(target.id, reason, details || undefined),
    onSuccess: () => {
      toast.success("Report submitted. Moderators will review it.");
      onSubmitted();
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not submit report"),
  });
  return (
    <div
      className="fixed inset-0 z-50 bg-brand/30 backdrop-blur-sm grid place-items-center p-4"
      onClick={onClose}
    >
      <div
        className="bg-surface border border-brand/15 w-full max-w-md p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <p className="eyebrow text-accent mb-2">Report {target.kind}</p>
        <h3 className="font-serif text-xl mb-4">Help us keep the community safe</h3>
        <label className="block text-xs text-brand/60 mb-1">Reason</label>
        <select
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          className="w-full px-3 py-2 text-sm border border-brand/15 bg-white/60 mb-4"
        >
          {REPORT_REASONS.map((r) => (
            <option key={r}>{r}</option>
          ))}
        </select>
        <label className="block text-xs text-brand/60 mb-1">Additional details (optional)</label>
        <textarea
          value={details}
          onChange={(e) => setDetails(e.target.value)}
          rows={3}
          className="w-full px-3 py-2 text-sm border border-brand/15 bg-white/60 resize-y mb-4"
        />
        <div className="flex justify-end gap-2">
          <button onClick={onClose} className="px-4 py-2 text-xs text-brand/70 hover:text-brand">
            Cancel
          </button>
          <button
            onClick={() => submit.mutate()}
            disabled={submit.isPending}
            className="px-5 py-2 bg-brand text-white text-xs font-medium disabled:opacity-50"
          >
            {submit.isPending ? "Sending…" : "Submit report"}
          </button>
        </div>
      </div>
    </div>
  );
}
