import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { MessageSquare, Search } from "lucide-react";
import { useState } from "react";

import { PublicFooter } from "@/components/layout/public-footer";
import { PublicHeader } from "@/components/layout/public-header";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { listForumThreads, type ForumThreadSummary } from "@/lib/api/forum";

type Sort = "recent" | "popular" | "unanswered";

export const Route = createFileRoute("/forum/")({
  component: PublicForumPage,
});

function PublicForumPage() {
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<Sort>("recent");
  const threads = useQuery({
    queryKey: ["public-forum-threads", { search, sort }],
    queryFn: () => listForumThreads({ search, sort, limit: 50 }),
  });

  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased flex flex-col">
      <PublicHeader active="forum" />

      <section className="hero-atmosphere text-white">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 sm:py-16 lg:py-20">
          <p className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full bg-white/10 border border-amber-300/35 text-amber-200 text-xs font-semibold tracking-[0.14em] uppercase mb-5">
            <span className="w-2 h-2 rounded-full bg-gold" aria-hidden="true" />
            Community Forum
          </p>
          <h1 className="font-serif text-3xl sm:text-4xl lg:text-5xl font-bold leading-[1.14] text-balance max-w-3xl">
            Read science questions and answers from the Inspire community.
          </h1>

          <div className="mt-8 bg-white rounded-xl p-3 sm:p-4 shadow-xl border border-borderui max-w-3xl flex flex-col sm:flex-row sm:items-center gap-3">
            <div className="relative flex-1 min-w-0">
              <Search
                className="absolute left-3.5 top-1/2 -translate-y-1/2 h-5 w-5 text-slatebody pointer-events-none"
                aria-hidden="true"
              />
              <input
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search discussions…"
                className="w-full min-h-[46px] rounded-lg border border-borderui bg-canvas pl-11 pr-4 py-2.5 text-sm sm:text-base text-navy placeholder:text-slatebody focus:bg-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-navy"
              />
            </div>
            <div className="flex flex-wrap gap-2">
              {(["recent", "popular", "unanswered"] as Sort[]).map((item) => (
                <button
                  key={item}
                  type="button"
                  onClick={() => setSort(item)}
                  aria-pressed={sort === item}
                  className={`inline-flex items-center justify-center min-h-[44px] px-4 rounded-lg text-xs sm:text-sm font-semibold capitalize border transition-colors ${
                    sort === item
                      ? "bg-navy text-white border-navy"
                      : "bg-canvas text-navy-light border-borderui hover:border-navy hover:bg-white"
                  }`}
                >
                  {item}
                </button>
              ))}
            </div>
          </div>
        </div>
      </section>

      <main className="flex-1 max-w-7xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-10 sm:py-12">
        {threads.isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div
                key={i}
                className="h-24 rounded-xl border border-borderui bg-white animate-pulse"
              />
            ))}
          </div>
        ) : threads.isError ? (
          <QueryErrorPanel
            error={threads.error}
            title="Could not load forum posts"
            onRetry={() => threads.refetch()}
          />
        ) : !threads.data || threads.data.data.length === 0 ? (
          <div className="rounded-xl border border-dashed border-borderui bg-white px-6 py-16 text-center">
            <span className="inline-flex h-12 w-12 items-center justify-center rounded-xl bg-stoneband text-navy">
              <MessageSquare className="h-6 w-6" aria-hidden="true" />
            </span>
            <h2 className="mt-4 font-serif text-2xl font-bold text-navy">No discussions found</h2>
            <p className="mt-2 text-sm text-slatebody">
              Try a different keyword or check back as students post new chapter questions.
            </p>
          </div>
        ) : (
          <ul className="space-y-3.5">
            {threads.data.data.map((thread) => (
              <ThreadRow key={thread.id} thread={thread} />
            ))}
          </ul>
        )}
      </main>

      <PublicFooter />
    </div>
  );
}

function ThreadRow({ thread }: { thread: ForumThreadSummary }) {
  return (
    <li>
      <Link
        to="/forum/$threadId"
        params={{ threadId: thread.id }}
        className="group block rounded-xl border border-borderui bg-white p-5 sm:p-6 shadow-sm hover:border-navy hover:shadow-md transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
      >
        <p className="font-serif text-xl font-bold text-navy group-hover:text-bronze transition-colors">
          {thread.title}
        </p>
        {thread.excerpt && (
          <p className="mt-1.5 text-sm text-slatebody line-clamp-2 leading-relaxed">
            {thread.excerpt}
          </p>
        )}
        <p className="mt-3 text-xs font-medium text-slatebody">
          <span className="text-navy font-semibold">{thread.author.full_name}</span> &bull;{" "}
          {new Date(thread.created_at).toLocaleDateString()} &bull; {thread.reply_count} repl
          {thread.reply_count === 1 ? "y" : "ies"}
        </p>
      </Link>
    </li>
  );
}
