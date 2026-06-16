import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, MessageSquare } from "lucide-react";

import { useAuth } from "@/context/auth-context";
import { getForumThread, listForumPosts } from "@/lib/api/forum";

export const Route = createFileRoute("/forum/$threadId")({
  component: PublicForumThreadPage,
});

function PublicForumThreadPage() {
  const { threadId } = Route.useParams();
  const { isAuthenticated } = useAuth();
  const thread = useQuery({
    queryKey: ["public-forum-thread", threadId],
    queryFn: () => getForumThread(threadId),
  });
  const posts = useQuery({
    queryKey: ["public-forum-posts", threadId],
    queryFn: () => listForumPosts(threadId, { limit: 100 }),
  });

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <header className="border-b border-brand/10">
        <div className="max-w-5xl mx-auto px-6 lg:px-10 py-6 flex items-center justify-between">
          <Link to="/" className="font-serif italic text-2xl text-accent">
            Inspire LMS
          </Link>
          <Link
            to={isAuthenticated ? "/student/forum" : "/login"}
            className="bg-brand text-white px-4 py-2 text-xs"
          >
            {isAuthenticated ? "Reply" : "Sign in to reply"}
          </Link>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 lg:px-10 py-12">
        <Link
          to="/forum"
          className="inline-flex items-center gap-2 text-xs text-brand/55 hover:text-brand mb-8"
        >
          <ArrowLeft className="h-3.5 w-3.5" />
          Back to forum
        </Link>

        {thread.isLoading ? (
          <div className="h-48 border border-brand/10 bg-white/30 animate-pulse" />
        ) : thread.isError || !thread.data ? (
          <p className="text-sm text-brand/60">
            {(thread.error as Error)?.message ?? "Thread not found"}
          </p>
        ) : (
          <>
            <article className="border-b border-brand/10 pb-10">
              {thread.data.category_name && (
                <p className="eyebrow text-accent mb-3">{thread.data.category_name}</p>
              )}
              <h1 className="font-serif text-4xl lg:text-5xl text-balance">{thread.data.title}</h1>
              <p className="mt-4 text-xs text-brand/50">
                {thread.data.author.full_name} · {new Date(thread.data.created_at).toLocaleString()}
              </p>
              <div
                className="mt-6 prose prose-sm max-w-none text-brand/75 whitespace-pre-wrap"
                dangerouslySetInnerHTML={
                  thread.data.body_html ? { __html: thread.data.body_html } : undefined
                }
              >
                {thread.data.body_html ? undefined : thread.data.body}
              </div>
            </article>

            <section className="mt-10">
              <h2 className="font-serif text-2xl mb-5">
                {posts.data?.meta.total ?? 0} repl
                {(posts.data?.meta.total ?? 0) === 1 ? "y" : "ies"}
              </h2>
              {posts.isLoading ? (
                <div className="space-y-3">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <div
                      key={i}
                      className="h-24 border border-brand/10 bg-white/30 animate-pulse"
                    />
                  ))}
                </div>
              ) : (posts.data?.data.length ?? 0) === 0 ? (
                <div className="border border-dashed border-brand/15 p-8 text-center text-sm text-brand/55">
                  <MessageSquare className="h-7 w-7 mx-auto text-brand/30 mb-3" />
                  No replies yet.
                </div>
              ) : (
                <ul className="space-y-4">
                  {posts.data?.data.map((post) => (
                    <li key={post.id} className="border border-brand/10 bg-white/40 p-5">
                      <p className="text-sm font-medium">{post.author.full_name}</p>
                      <p className="mt-1 text-xs text-brand/45">
                        {new Date(post.created_at).toLocaleString()}
                      </p>
                      <div
                        className="mt-4 prose prose-sm max-w-none text-brand/75 whitespace-pre-wrap"
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
    </div>
  );
}
