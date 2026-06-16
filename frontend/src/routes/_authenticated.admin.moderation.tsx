import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { DataPage } from "@/components/layout/data-page";
import { decideModeration, listModerationQueue, type ModerationItem } from "@/lib/api/moderation";
import {
  listPendingPosts,
  approvePendingPost,
  rejectPendingPost,
  type PendingPost,
} from "@/lib/api/forum";

export const Route = createFileRoute("/_authenticated/admin/moderation")({
  component: Page,
});

type KindFilter = "all" | ModerationItem["kind"];
const KINDS: { value: KindFilter; label: string }[] = [
  { value: "all", label: "All" },
  { value: "post", label: "Posts" },
  { value: "comment", label: "Comments" },
  { value: "review", label: "Reviews" },
  { value: "message", label: "Messages" },
];

function Page() {
  const qc = useQueryClient();
  const [activeTab, setActiveTab] = useState<"flagged" | "pending_posts">("flagged");
  const [kind, setKind] = useState<KindFilter>("all");

  // State for flagged content moderation action
  const [decision, setDecision] = useState<{
    item: ModerationItem;
    action: "approve" | "remove";
  } | null>(null);
  const [note, setNote] = useState("");

  // State for pending posts rejection reason dialog
  const [rejectingPost, setRejectingPost] = useState<PendingPost | null>(null);
  const [rejectionReason, setRejectionReason] = useState("");

  // Flagged Content Action Mutation
  const flagMut = useMutation({
    mutationFn: (args: { id: string; action: "approve" | "remove"; note?: string }) =>
      decideModeration(args.id, args.action, args.note),
    onSuccess: () => {
      toast.success("Decision recorded");
      qc.invalidateQueries({ queryKey: ["moderation"] });
      setDecision(null);
      setNote("");
    },
    onError: (e: Error) => toast.error(e.message),
  });

  // Pending Posts queries
  const pendingPostsQuery = useQuery({
    queryKey: ["pending-posts"],
    queryFn: () => listPendingPosts(),
    enabled: activeTab === "pending_posts",
  });

  // Pending Posts Approval Mutation
  const approveMutation = useMutation({
    mutationFn: (id: string) => approvePendingPost(id),
    onSuccess: () => {
      toast.success("Post approved and published");
      qc.invalidateQueries({ queryKey: ["pending-posts"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  // Pending Posts Rejection Mutation
  const rejectMutation = useMutation({
    mutationFn: (args: { id: string; reason: string }) => rejectPendingPost(args.id, args.reason),
    onSuccess: () => {
      toast.success("Post rejected");
      qc.invalidateQueries({ queryKey: ["pending-posts"] });
      setRejectingPost(null);
      setRejectionReason("");
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <AppShell eyebrow="Moderation" title="Content Governance">
      <div className="flex gap-6 border-b border-brand/10 mb-6">
        <button
          onClick={() => setActiveTab("flagged")}
          className={`pb-3 text-sm font-medium border-b-2 transition-all ${
            activeTab === "flagged"
              ? "border-brand text-brand font-semibold"
              : "border-transparent text-brand/65 hover:text-brand"
          }`}
        >
          Flagged Content
        </button>
        <button
          onClick={() => setActiveTab("pending_posts")}
          className={`pb-3 text-sm font-medium border-b-2 transition-all ${
            activeTab === "pending_posts"
              ? "border-brand text-brand font-semibold"
              : "border-transparent text-brand/65 hover:text-brand"
          }`}
        >
          Pending Forum Posts
        </button>
      </div>

      {activeTab === "flagged" ? (
        <DataPage
          eyebrow="Flagged content review"
          title="Reports Queue"
          queryKey={["moderation"]}
          queryFn={listModerationQueue}
          empty={{ title: "Nothing flagged" }}
          toolbar={
            <div className="flex flex-wrap gap-2">
              {KINDS.map((k) => (
                <button
                  key={k.value}
                  onClick={() => setKind(k.value)}
                  className={`px-4 py-1.5 text-xs font-medium border transition-colors ${
                    kind === k.value
                      ? "bg-brand text-white border-brand"
                      : "border-brand/15 text-brand/65 hover:text-brand hover:bg-brand/[0.03]"
                  }`}
                >
                  {k.label}
                </button>
              ))}
            </div>
          }
        >
          {(data: { items: ModerationItem[] }) => {
            const filtered = data.items.filter(
              (it) => it.status === "pending" && (kind === "all" || it.kind === kind),
            );
            if (filtered.length === 0) {
              return <p className="text-sm text-brand/55">Queue is clear.</p>;
            }
            return (
              <ul className="space-y-3">
                {filtered.map((it) => (
                  <li key={it.id} className="border border-brand/10 bg-white/40 p-5">
                    <div className="flex flex-col sm:flex-row sm:justify-between sm:items-start gap-4">
                      <div className="min-w-0 flex-1">
                        <p className="eyebrow text-brand/55">
                          {it.kind} · {it.reason}
                        </p>
                        <p className="mt-2 text-sm whitespace-pre-line border-l-2 border-brand/15 pl-3 text-brand/80">
                          {it.content}
                        </p>
                        <p className="mt-2 text-xs text-brand/45">
                          Reported by {it.reported_by.full_name} ·{" "}
                          {new Date(it.created_at).toLocaleString()}
                          {it.target_url && (
                            <>
                              {" · "}
                              <a
                                href={it.target_url}
                                target="_blank"
                                rel="noreferrer"
                                className="text-accent hover:underline underline-offset-4"
                              >
                                View context
                              </a>
                            </>
                          )}
                        </p>
                      </div>
                      <div className="flex gap-2 flex-shrink-0">
                        <button
                          onClick={() => setDecision({ item: it, action: "approve" })}
                          className="text-xs border border-brand/15 px-3 py-1.5 hover:bg-brand/[0.03] transition-colors"
                        >
                          Keep
                        </button>
                        <button
                          onClick={() => setDecision({ item: it, action: "remove" })}
                          className="text-xs bg-destructive text-white px-3 py-1.5 hover:bg-destructive/90 transition-colors"
                        >
                          Remove
                        </button>
                      </div>
                    </div>
                  </li>
                ))}
              </ul>
            );
          }}
        </DataPage>
      ) : (
        <div className="space-y-6">
          {pendingPostsQuery.isLoading && (
            <p className="text-sm text-brand/55">Loading pending posts…</p>
          )}

          {pendingPostsQuery.isError && (
            <div className="border border-destructive/20 bg-destructive/5 p-6 text-sm">
              <p className="font-medium text-destructive">Couldn't load pending posts</p>
              <p className="mt-1 text-brand/60">{(pendingPostsQuery.error as Error)?.message}</p>
            </div>
          )}

          {!pendingPostsQuery.isLoading &&
            (!pendingPostsQuery.data?.data || pendingPostsQuery.data.data.length === 0) && (
              <div className="border border-dashed border-brand/15 px-8 py-16 text-center bg-white/40">
                <h3 className="font-serif text-2xl">All clear</h3>
                <p className="mt-2 text-sm text-brand/55">
                  No forum posts are awaiting pre-publication approval.
                </p>
              </div>
            )}

          {pendingPostsQuery.data?.data && pendingPostsQuery.data.data.length > 0 && (
            <ul className="space-y-3">
              {pendingPostsQuery.data.data.map((post) => (
                <li key={post.id} className="border border-brand/10 bg-white/40 p-5">
                  <div className="flex flex-col sm:flex-row sm:justify-between sm:items-start gap-4">
                    <div className="min-w-0 flex-1">
                      <h3 className="font-serif text-lg font-medium text-brand">{post.title}</h3>
                      <div
                        className="mt-2 text-sm whitespace-pre-line text-brand/85 max-h-40 overflow-y-auto pl-3 border-l-2 border-brand/15"
                        dangerouslySetInnerHTML={{ __html: post.body_html || post.body_markdown }}
                      />
                      <p className="mt-3 text-xs text-brand/45">
                        Submitted at {new Date(post.created_at).toLocaleString()}
                      </p>
                    </div>
                    <div className="flex gap-2 flex-shrink-0">
                      <button
                        onClick={() => approveMutation.mutate(post.id)}
                        disabled={approveMutation.isPending || rejectMutation.isPending}
                        className="text-xs bg-brand text-white px-3 py-1.5 hover:bg-brand/90 transition-colors"
                      >
                        {approveMutation.isPending ? "Approving…" : "Approve"}
                      </button>
                      <button
                        onClick={() => setRejectingPost(post)}
                        disabled={approveMutation.isPending || rejectMutation.isPending}
                        className="text-xs border border-destructive/40 text-destructive px-3 py-1.5 hover:bg-destructive/5 transition-colors"
                      >
                        Reject
                      </button>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      )}

      {/* Flagged Content Decision Dialog */}
      {decision && (
        <div
          className="fixed inset-0 z-50 bg-black/40 grid place-items-center p-4"
          onClick={() => {
            setDecision(null);
            setNote("");
          }}
        >
          <div
            className="bg-white border border-brand/10 max-w-md w-full p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <p className="eyebrow text-brand/55">
              {decision.action === "approve" ? "Keep content" : "Remove content"}
            </p>
            <p className="mt-3 text-sm whitespace-pre-line border-l-2 border-brand/15 pl-3 text-brand/80 max-h-32 overflow-y-auto">
              {decision.item.content}
            </p>
            <label className="block mt-5">
              <span className="eyebrow text-brand/55">Moderator note</span>
              <textarea
                value={note}
                onChange={(e) => setNote(e.target.value)}
                rows={3}
                maxLength={500}
                placeholder="Reason logged in the audit trail"
                className="mt-1 w-full p-3 border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
              />
            </label>
            <div className="mt-5 flex justify-end gap-2">
              <button
                onClick={() => {
                  setDecision(null);
                  setNote("");
                }}
                className="px-4 py-2 text-sm border border-brand/15"
              >
                Cancel
              </button>
              <button
                disabled={flagMut.isPending}
                onClick={() =>
                  flagMut.mutate({
                    id: decision.item.id,
                    action: decision.action,
                    note: note.trim() || undefined,
                  })
                }
                className={`px-5 py-2 text-sm text-white disabled:opacity-60 ${
                  decision.action === "remove" ? "bg-destructive" : "bg-brand"
                }`}
              >
                {flagMut.isPending ? "Saving…" : decision.action === "remove" ? "Remove" : "Keep"}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Reject Pending Post Dialog */}
      {rejectingPost && (
        <div
          className="fixed inset-0 z-50 bg-black/40 grid place-items-center p-4"
          onClick={() => {
            setRejectingPost(null);
            setRejectionReason("");
          }}
        >
          <div
            className="bg-white border border-brand/10 max-w-md w-full p-6"
            onClick={(e) => e.stopPropagation()}
          >
            <p className="eyebrow text-brand/55 text-destructive">Reject Forum Post</p>
            <p className="mt-3 font-serif text-lg text-brand font-medium truncate">
              {rejectingPost.title}
            </p>
            <label className="block mt-5">
              <span className="eyebrow text-brand/55">Rejection reason (required)</span>
              <textarea
                value={rejectionReason}
                onChange={(e) => setRejectionReason(e.target.value)}
                rows={4}
                maxLength={1000}
                placeholder="Why is this post being rejected?"
                className="mt-1 w-full p-3 border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
              />
            </label>
            <div className="mt-5 flex justify-end gap-2">
              <button
                onClick={() => {
                  setRejectingPost(null);
                  setRejectionReason("");
                }}
                className="px-4 py-2 text-sm border border-brand/15"
              >
                Cancel
              </button>
              <button
                disabled={rejectMutation.isPending || !rejectionReason.trim()}
                onClick={() =>
                  rejectMutation.mutate({
                    id: rejectingPost.id,
                    reason: rejectionReason.trim(),
                  })
                }
                className="px-5 py-2 text-sm text-white bg-destructive disabled:opacity-60"
              >
                {rejectMutation.isPending ? "Rejecting…" : "Reject"}
              </button>
            </div>
          </div>
        </div>
      )}
    </AppShell>
  );
}
