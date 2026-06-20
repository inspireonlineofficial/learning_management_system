import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { MessageSquare, Search } from "lucide-react";
import { useState } from "react";

import { BrandLogo } from "@/components/layout/brand-logo";
import { useAuth } from "@/context/auth-context";
import { listForumThreads, type ForumThreadSummary } from "@/lib/api/forum";

type Sort = "recent" | "popular" | "unanswered";

export const Route = createFileRoute("/forum/")({
  component: PublicForumPage,
});

function PublicForumPage() {
  const { isAuthenticated } = useAuth();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<Sort>("recent");
  const threads = useQuery({
    queryKey: ["public-forum-threads", { search, sort }],
    queryFn: () => listForumThreads({ search, sort, limit: 50 }),
  });

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <header className="border-b border-brand/10">
        <div className="max-w-6xl mx-auto px-6 lg:px-10 py-6 flex items-center justify-between">
          <BrandLogo imageClassName="max-h-14 max-w-[220px]" />
          <nav className="flex items-center gap-5 text-sm">
            <Link to="/courses" className="text-brand/65 hover:text-brand">
              Courses
            </Link>
            <Link to="/bookshop" className="text-brand/65 hover:text-brand">
              Bookshop
            </Link>
            <Link
              to={isAuthenticated ? "/student/forum" : "/login"}
              className="bg-brand text-white px-4 py-2 text-xs"
            >
              {isAuthenticated ? "New post" : "Sign in"}
            </Link>
          </nav>
        </div>
      </header>

      <main className="max-w-6xl mx-auto px-6 lg:px-10 py-12">
        <p className="eyebrow text-accent mb-4">Community forum</p>
        <h1 className="font-serif text-4xl lg:text-6xl text-balance">
          Read science questions and answers from the Inspire community.
        </h1>

        <div className="mt-8 flex flex-wrap items-center gap-3">
          <div className="relative flex-1 min-w-[240px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-brand/40" />
            <input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search discussions"
              className="w-full border border-brand/15 bg-white/50 pl-10 pr-4 py-3 text-sm"
            />
          </div>
          {(["recent", "popular", "unanswered"] as Sort[]).map((item) => (
            <button
              key={item}
              onClick={() => setSort(item)}
              className={`px-4 py-3 text-xs capitalize ${
                sort === item ? "bg-brand text-white" : "border border-brand/15 text-brand/70"
              }`}
            >
              {item}
            </button>
          ))}
        </div>

        <section className="mt-10">
          {threads.isLoading ? (
            <div className="space-y-3">
              {Array.from({ length: 4 }).map((_, i) => (
                <div key={i} className="h-24 border border-brand/10 bg-white/30 animate-pulse" />
              ))}
            </div>
          ) : threads.isError ? (
            <p className="border border-destructive/20 bg-destructive/5 p-6 text-sm text-destructive">
              {(threads.error as Error)?.message ?? "Could not load forum posts."}
            </p>
          ) : !threads.data || threads.data.data.length === 0 ? (
            <div className="border border-dashed border-brand/15 px-8 py-16 text-center">
              <MessageSquare className="h-8 w-8 mx-auto text-brand/30" />
              <h2 className="mt-4 font-serif text-2xl">No discussions found</h2>
            </div>
          ) : (
            <ul className="divide-y divide-brand/10 border-y border-brand/10">
              {threads.data.data.map((thread) => (
                <ThreadRow key={thread.id} thread={thread} />
              ))}
            </ul>
          )}
        </section>
      </main>
    </div>
  );
}

function ThreadRow({ thread }: { thread: ForumThreadSummary }) {
  return (
    <li>
      <Link
        to="/forum/$threadId"
        params={{ threadId: thread.id }}
        className="block px-2 py-5 hover:bg-brand/[0.02]"
      >
        <p className="font-serif text-xl">{thread.title}</p>
        {thread.excerpt && (
          <p className="mt-1 text-sm text-brand/60 line-clamp-2">{thread.excerpt}</p>
        )}
        <p className="mt-3 text-xs text-brand/50">
          {thread.author.full_name} · {new Date(thread.created_at).toLocaleDateString()} ·{" "}
          {thread.reply_count} repl{thread.reply_count === 1 ? "y" : "ies"}
        </p>
      </Link>
    </li>
  );
}
