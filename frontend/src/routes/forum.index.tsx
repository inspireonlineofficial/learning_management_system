import { usePublicSiteConfig } from "@/hooks/use-site-config";
import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { MessageSquare, Search } from "lucide-react";
import { useState } from "react";

import { EmptyState } from "@/components/layout/app-shell";
import { FilterChip, HeroPanel, PageHero, fieldClass } from "@/components/layout/page-hero";
import { PublicFooter } from "@/components/layout/public-footer";
import { PublicHeader } from "@/components/layout/public-header";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { listForumThreads, type ForumThreadSummary } from "@/lib/api/forum";

type Sort = "recent" | "popular" | "unanswered";

export const Route = createFileRoute("/forum/")({
  component: PublicForumPage,
});

function PublicForumPage() {
  const page = usePublicSiteConfig().pages.forum;
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<Sort>("recent");
  const threads = useQuery({
    queryKey: ["public-forum-threads", { search, sort }],
    queryFn: () => listForumThreads({ search, sort, limit: 50 }),
  });

  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased flex flex-col">
      <PublicHeader active="forum" />

      <PageHero eyebrow={page.eyebrow} title={page.title} body={page.body}>
        <HeroPanel className="flex flex-col sm:flex-row sm:items-center gap-3">
          <div className="relative flex-1 min-w-0">
            <Search
              className="absolute left-3.5 top-1/2 -translate-y-1/2 h-4 w-4 text-slatebody pointer-events-none"
              aria-hidden="true"
            />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search discussions…"
              aria-label="Search discussions"
              className={`${fieldClass} pl-10`}
            />
          </div>
          <div className="flex flex-wrap gap-2" role="group" aria-label="Sort discussions">
            {(["recent", "popular", "unanswered"] as Sort[]).map((item) => (
              <FilterChip key={item} active={sort === item} onClick={() => setSort(item)}>
                <span className="capitalize">{item}</span>
              </FilterChip>
            ))}
          </div>
        </HeroPanel>
      </PageHero>

      <main className="flex-1 page-container py-10 sm:py-12">
        {threads.isLoading ? (
          <div className="space-y-3">
            {Array.from({ length: 4 }).map((_, i) => (
              <div
                key={i}
                className="h-24 rounded-lg border border-borderui bg-white animate-pulse"
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
          <EmptyState
            icon={MessageSquare}
            title="No discussions found"
            description={
              search
                ? "No threads match your search. Try a different keyword."
                : "There are no public discussions yet. Questions posted by students will appear here."
            }
          />
        ) : (
          <ul className="space-y-3">
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
        className="group block rounded-lg border border-borderui bg-white p-5 sm:p-6 hover:border-navy transition-colors focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-navy"
      >
        <p className="font-serif text-lg sm:text-xl font-semibold text-navy group-hover:text-bronze transition-colors">
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
