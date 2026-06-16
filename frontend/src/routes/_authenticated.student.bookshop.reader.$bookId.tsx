import { useMutation, useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, ChevronLeft, ChevronRight, List } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { useAuth } from "@/context/auth-context";
import { readBook, setBookmark } from "@/lib/api/bookshop";

export const Route = createFileRoute("/_authenticated/student/bookshop/reader/$bookId")({
  component: ReaderPage,
});

function ReaderPage() {
  const { bookId } = Route.useParams();
  const { user } = useAuth();
  void user;

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["read-book", bookId],
    queryFn: () => readBook(bookId),
  });

  const chapters = useMemo(
    () => (data?.chapters ?? []).slice().sort((a, b) => a.position - b.position),
    [data],
  );

  const [chapterIdx, setChapterIdx] = useState(0);
  const [tocOpen, setTocOpen] = useState(false);

  useEffect(() => {
    if (!data) return;
    const bookmarked = data.bookmark?.chapter_id;
    if (!bookmarked) return;
    const idx = chapters.findIndex((c) => c.id === bookmarked);
    if (idx >= 0) setChapterIdx(idx);
  }, [data, chapters]);

  const bookmark = useMutation({
    mutationFn: (chapterId: string) => setBookmark(bookId, chapterId),
  });

  const current = chapters[chapterIdx];

  useEffect(() => {
    if (current) bookmark.mutate(current.id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [current?.id]);

  if (isLoading) {
    return (
      <div className="min-h-screen bg-surface px-6 py-16">
        <div className="max-w-3xl mx-auto h-96 bg-brand/5 animate-pulse" />
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="min-h-screen bg-surface px-6 py-16 text-center text-sm text-brand/60">
        {(error as Error)?.message ?? "Book not available"}
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <header className="sticky top-0 z-30 bg-surface/95 backdrop-blur border-b border-brand/10">
        <div className="max-w-5xl mx-auto px-4 lg:px-6 py-3 flex items-center justify-between gap-4">
          <Link
            to="/student/bookshop/library"
            className="flex items-center gap-2 text-xs text-brand/55 hover:text-brand"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Library
          </Link>
          <p className="font-serif text-sm truncate max-w-[50%]">{data.book.title}</p>
          <button
            onClick={() => setTocOpen((v) => !v)}
            className="flex items-center gap-1.5 text-xs text-brand/70 hover:text-brand"
          >
            <List className="h-3.5 w-3.5" />
            Contents
          </button>
        </div>
      </header>

      <div className="flex">
        {tocOpen && (
          <aside className="fixed lg:static inset-0 lg:inset-auto z-20 bg-surface lg:bg-transparent lg:w-72 border-r border-brand/10 overflow-y-auto">
            <div className="p-4 lg:p-6">
              <div className="flex items-center justify-between mb-4">
                <p className="eyebrow text-brand/45">Chapters</p>
                <button
                  onClick={() => setTocOpen(false)}
                  className="text-xs text-brand/55 hover:text-brand lg:hidden"
                >
                  Close
                </button>
              </div>
              <ol className="space-y-1">
                {chapters.map((c, i) => (
                  <li key={c.id}>
                    <button
                      onClick={() => {
                        setChapterIdx(i);
                        setTocOpen(false);
                      }}
                      className={`w-full text-left text-sm px-3 py-2 transition-colors ${
                        i === chapterIdx
                          ? "bg-brand text-white"
                          : "text-brand/70 hover:bg-brand/[0.03]"
                      }`}
                    >
                      <span className="text-xs opacity-60 mr-2">
                        {String(i + 1).padStart(2, "0")}
                      </span>
                      {c.title}
                    </button>
                  </li>
                ))}
              </ol>
            </div>
          </aside>
        )}

        <main className="flex-1">
          <article className="max-w-2xl mx-auto px-6 lg:px-10 py-12 lg:py-20">
            {current ? (
              <>
                <p className="eyebrow text-accent mb-4">
                  Chapter {chapterIdx + 1} of {chapters.length}
                </p>
                <h1 className="font-serif text-3xl lg:text-4xl text-balance mb-10">
                  {current.title}
                </h1>
                <div
                  className="prose prose-lg max-w-none text-brand/85 leading-loose font-serif"
                  dangerouslySetInnerHTML={{ __html: current.body_html }}
                />

                <nav className="mt-16 pt-8 border-t border-brand/10 flex items-center justify-between">
                  <button
                    onClick={() => setChapterIdx((i) => Math.max(0, i - 1))}
                    disabled={chapterIdx === 0}
                    className="inline-flex items-center gap-2 text-sm text-brand/70 hover:text-brand disabled:opacity-30"
                  >
                    <ChevronLeft className="h-4 w-4" />
                    Previous
                  </button>
                  <button
                    onClick={() => setChapterIdx((i) => Math.min(chapters.length - 1, i + 1))}
                    disabled={chapterIdx >= chapters.length - 1}
                    className="inline-flex items-center gap-2 text-sm text-brand/70 hover:text-brand disabled:opacity-30"
                  >
                    Next
                    <ChevronRight className="h-4 w-4" />
                  </button>
                </nav>
              </>
            ) : (
              <p className="text-brand/55">This book has no chapters available.</p>
            )}
          </article>
        </main>
      </div>
    </div>
  );
}
