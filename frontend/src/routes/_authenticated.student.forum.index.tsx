import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { CheckCircle2, MessageSquare, Pin, Plus, Search } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

import { AppShell, EmptyState, SectionHeading } from "@/components/layout/app-shell";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  createForumThread,
  listForumCategories,
  listForumThreads,
  type ForumThreadSummary,
} from "@/lib/api/forum";

type Sort = "recent" | "popular" | "unanswered";

export const Route = createFileRoute("/_authenticated/student/forum/")({
  component: ForumIndex,
});

function ForumIndex() {
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<Sort>("recent");
  const [category, setCategory] = useState<string | undefined>();
  const [composing, setComposing] = useState(false);

  const categories = useQuery({
    queryKey: ["forum-categories"],
    queryFn: () => listForumCategories(),
  });

  const threads = useQuery({
    queryKey: ["forum-threads", { search, sort, category }],
    queryFn: () => listForumThreads({ search, sort, category_id: category, limit: 50 }),
  });

  return (
    <AppShell eyebrow="Community" title="The commons.">
      <div className="grid lg:grid-cols-[240px_1fr] gap-10">
        <aside className="space-y-6">
          <div>
            <p className="eyebrow text-brand/45 mb-3">Categories</p>
            <ul className="space-y-1">
              <li>
                <button
                  onClick={() => setCategory(undefined)}
                  className={`w-full text-left text-sm px-3 py-2 transition-colors ${
                    !category
                      ? "bg-brand text-white"
                      : "text-brand/70 hover:text-brand hover:bg-brand/[0.03]"
                  }`}
                >
                  All discussions
                </button>
              </li>
              {(categories.data?.data ?? []).map((c) => (
                <li key={c.id}>
                  <button
                    onClick={() => setCategory(c.id)}
                    className={`w-full text-left text-sm px-3 py-2 transition-colors flex items-center justify-between ${
                      category === c.id
                        ? "bg-brand text-white"
                        : "text-brand/70 hover:text-brand hover:bg-brand/[0.03]"
                    }`}
                  >
                    <span className="truncate">{c.name}</span>
                    {typeof c.thread_count === "number" && (
                      <span className="text-xs opacity-60">{c.thread_count}</span>
                    )}
                  </button>
                </li>
              ))}
            </ul>
          </div>
        </aside>

        <div>
          <div className="flex flex-wrap items-center gap-3 mb-6">
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-brand/40" />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search discussions…"
                className="w-full pl-10 pr-4 py-2.5 text-sm border border-brand/15 bg-white/40 focus:outline-none focus:border-brand/40"
              />
            </div>
            <div className="flex gap-1">
              {(["recent", "popular", "unanswered"] as Sort[]).map((s) => (
                <button
                  key={s}
                  onClick={() => setSort(s)}
                  className={`px-3 py-2 text-xs font-medium capitalize ${
                    sort === s
                      ? "bg-brand text-white"
                      : "border border-brand/15 text-brand/70 hover:text-brand"
                  }`}
                >
                  {s}
                </button>
              ))}
            </div>
            <button
              onClick={() => setComposing((v) => !v)}
              className="inline-flex items-center gap-2 px-4 py-2.5 bg-accent text-white text-xs font-medium"
            >
              <Plus className="h-3.5 w-3.5" />
              New thread
            </button>
          </div>

          {composing && (
            <NewThreadForm
              categories={categories.data?.data ?? []}
              defaultCategory={category}
              onDone={() => setComposing(false)}
            />
          )}

          {threads.isError ? (
            <QueryErrorPanel
              error={threads.error}
              title="Couldn't load discussions"
              onRetry={() => threads.refetch()}
            />
          ) : threads.isLoading ? (
            <div className="space-y-2">
              {Array.from({ length: 5 }).map((_, i) => (
                <div key={i} className="h-20 border border-brand/10 bg-white/30 animate-pulse" />
              ))}
            </div>
          ) : !threads.data || threads.data.data.length === 0 ? (
            <EmptyState
              icon={MessageSquare}
              title="No discussions yet"
              description="Be the first to start a conversation."
            />
          ) : (
            <>
              <SectionHeading
                title={`${threads.data.meta.total} discussion${threads.data.meta.total === 1 ? "" : "s"}`}
              />
              <ul className="divide-y divide-brand/10 border-y border-brand/10">
                {threads.data.data.map((t) => (
                  <ThreadRow key={t.id} thread={t} />
                ))}
              </ul>
            </>
          )}
        </div>
      </div>
    </AppShell>
  );
}

function ThreadRow({ thread: t }: { thread: ForumThreadSummary }) {
  const last = t.last_reply_at ?? t.created_at;
  return (
    <li>
      <Link
        to="/student/forum/$threadId"
        params={{ threadId: t.id }}
        className="flex items-start gap-4 py-5 px-2 hover:bg-brand/[0.02] transition-colors"
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            {t.is_pinned && <Pin className="h-3 w-3 text-accent" />}
            {t.is_resolved && <CheckCircle2 className="h-3.5 w-3.5 text-emerald-600" />}
            {t.category_name && <span className="eyebrow text-brand/40">{t.category_name}</span>}
          </div>
          <p className="font-serif text-lg leading-snug">{t.title}</p>
          {t.excerpt && <p className="mt-1 text-sm text-brand/60 line-clamp-2">{t.excerpt}</p>}
          <p className="mt-2 text-xs text-brand/50">
            {t.author.full_name} · {new Date(last).toLocaleDateString()}
          </p>
        </div>
        <div className="text-right shrink-0 text-xs text-brand/55">
          <p className="font-medium text-brand">{t.reply_count}</p>
          <p>repl{t.reply_count === 1 ? "y" : "ies"}</p>
        </div>
      </Link>
    </li>
  );
}

function NewThreadForm({
  categories,
  defaultCategory,
  onDone,
}: {
  categories: { id: string; name: string }[];
  defaultCategory?: string;
  onDone: () => void;
}) {
  const qc = useQueryClient();
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [categoryId, setCategoryId] = useState(defaultCategory ?? categories[0]?.id ?? "");

  const create = useMutation({
    mutationFn: () =>
      createForumThread({
        title: title.trim(),
        body: body.trim(),
        category_id: categoryId || undefined,
      }),
    onSuccess: () => {
      toast.success("Thread published");
      qc.invalidateQueries({ queryKey: ["forum-threads"] });
      qc.invalidateQueries({ queryKey: ["forum-categories"] });
      onDone();
    },
    onError: (e: Error) => toast.error(e.message ?? "Could not publish"),
  });

  return (
    <div className="border border-brand/15 bg-white/50 p-6 mb-6">
      <p className="eyebrow text-accent mb-4">New thread</p>
      <input
        value={title}
        onChange={(e) => setTitle(e.target.value)}
        placeholder="Title"
        className="w-full text-lg font-serif px-0 py-2 border-b border-brand/15 bg-transparent focus:outline-none focus:border-brand/40"
      />
      <textarea
        value={body}
        onChange={(e) => setBody(e.target.value)}
        placeholder="What's on your mind?"
        rows={5}
        className="w-full mt-3 px-3 py-2 text-sm border border-brand/15 bg-white/50 focus:outline-none focus:border-brand/40 resize-y"
      />
      <div className="flex items-center justify-between gap-3 mt-4">
        {categories.length > 0 ? (
          <select
            value={categoryId}
            onChange={(e) => setCategoryId(e.target.value)}
            className="px-3 py-2 text-xs border border-brand/15 bg-white/50"
          >
            {categories.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </select>
        ) : (
          <span />
        )}
        <div className="flex gap-2">
          <button onClick={onDone} className="px-4 py-2 text-xs text-brand/70 hover:text-brand">
            Cancel
          </button>
          <button
            onClick={() => create.mutate()}
            disabled={create.isPending || !title.trim() || !body.trim()}
            className="px-5 py-2 bg-brand text-white text-xs font-medium disabled:opacity-50"
          >
            {create.isPending ? "Publishing…" : "Publish"}
          </button>
        </div>
      </div>
    </div>
  );
}
