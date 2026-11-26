import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, MessageSquare } from "lucide-react";

import { PublicFooter } from "@/components/layout/public-footer";
import { PublicHeader } from "@/components/layout/public-header";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import { getForumThread, listForumPosts } from "@/lib/api/forum";

export const Route = createFileRoute("/forum/$threadId")({
  component: PublicForumThreadPage,
});

function PublicForumThreadPage() {
  const { threadId } = Route.useParams();
  const thread = useQuery({
    queryKey: ["public-forum-thread", threadId],
    queryFn: () => getForumThread(threadId),
  });
  const posts = useQuery({
    queryKey: ["public-forum-posts", threadId],
    queryFn: () => listForumPosts(threadId, { limit: 100 }),
  });

  return (
    <div className="min-h-screen overflow-x-hidden bg-canvas text-navy-light font-sans antialiased flex flex-col">
      <PublicHeader active="forum" />

      <main className="flex-1 max-w-4xl w-full mx-auto px-4 sm:px-6 lg:px-8 py-8 sm:py-12">
        <Link
          to="/forum"
          className="inline-flex items-center gap-2 min-h-[44px] text-sm font-semibold text-navy hover:text-bronze underline underline-offset-4 decoration-gold decoration-2 mb-6"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" />
          Back to forum
        </Link>

        {thread.isLoading ? (
          <div className="h-48 rounded-2xl border border-borderui bg-white animate-pulse" />
        ) : thread.isError || !thread.data ? (
          <QueryErrorPanel
            error={thread.error}
            title="Thread not found"
            onRetry={() => thread.refetch()}
          />
        ) : (
          <>
            <article className="rounded-2xl border border-borderui bg-white p-6 sm:p-8 shadow-sm">
              {thread.data.category_name && (
                <p className="text-xs font-bold tracking-[0.16em] uppercase text-bronze mb-2.5">
                  {thread.data.category_name}
                </p>
              )}
              <h1 className="font-serif text-3xl sm:text-4xl font-bold text-navy text-balance">
                {thread.data.title}
              </h1>
              <p className="mt-3 text-xs font-medium text-slatebody">
                <span className="text-navy font-semibold">{thread.data.author.full_name}</span>{" "}
                &bull; {new Date(thread.data.created_at).toLocaleString()}
              </p>
              <div
                className="mt-6 pt-6 border-t border-borderui/70 prose prose-sm max-w-none text-navy-light whitespace-pre-wrap leading-relaxed"
                dangerouslySetInnerHTML={
                  thread.data.body_html ? { __html: thread.data.body_html } : undefined
                }
              >
                {thread.data.body_html ? undefined : thread.data.body}
              </div>
            </article>

            <section className="mt-10">
              <h2 className="font-serif text-2xl font-bold text-navy mb-5">
                {posts.data?.meta.total ?? 0} repl
                {(posts.data?.meta.total ?? 0) === 1 ? "y" : "ies"}
              </h2>
              {posts.isLoading ? (
                <div className="space-y-3">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <div
                      key={i}
                      className="h-24 rounded-xl border border-borderui bg-white animate-pulse"
                    />
                  ))}
                </div>
              ) : (posts.data?.data.length ?? 0) === 0 ? (
                <div className="rounded-xl border border-dashed border-borderui bg-white p-8 text-center text-sm text-slatebody">
                  <MessageSquare className="h-7 w-7 mx-auto text-navy/30 mb-3" aria-hidden="true" />
                  No replies yet.
                </div>
              ) : (
                <ul className="space-y-4">
                  {posts.data?.data.map((post) => (
                    <li
                      key={post.id}
                      className="rounded-xl border border-borderui bg-white p-5 sm:p-6 shadow-sm"
                    >
                      <p className="text-sm font-semibold text-navy">{post.author.full_name}</p>
                      <p className="mt-0.5 text-xs text-slatebody">
                        {new Date(post.created_at).toLocaleString()}
                      </p>
                      <div
                        className="mt-4 prose prose-sm max-w-none text-navy-light whitespace-pre-wrap leading-relaxed"
                        dangerouslySetInnerHTML={
                          post.body_html ? { __html: post.body_html } : undefined
                        }
                      >
                        {post.body_html ? undefined : post.body}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </>
        )}
      </main>

      <PublicFooter />
    </div>
  );
}
