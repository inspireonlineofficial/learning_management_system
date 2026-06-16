import { apiRequest } from "./client";

export type ForumCategory = {
  id: string;
  name: string;
  slug: string;
  description?: string;
  thread_count?: number;
};

export type ForumAuthor = {
  id: string;
  full_name: string;
  avatar_url?: string | null;
  role?: string;
};

export type ForumThreadSummary = {
  id: string;
  title: string;
  excerpt?: string;
  category_id?: string;
  category_name?: string;
  course_id?: string | null;
  course_title?: string | null;
  author: ForumAuthor;
  created_at: string;
  last_reply_at?: string | null;
  reply_count: number;
  view_count?: number;
  is_pinned?: boolean;
  is_locked?: boolean;
  is_resolved?: boolean;
  tags?: string[];
};

export type ForumThreadDetail = ForumThreadSummary & {
  body_html?: string;
  body?: string;
};

export type ForumPost = {
  id: string;
  thread_id: string;
  author: ForumAuthor;
  body_html?: string;
  body: string;
  created_at: string;
  updated_at?: string;
  is_answer?: boolean;
  like_count?: number;
  has_liked?: boolean;
};

export type Paginated<T> = {
  data: T[];
  meta: { page: number; limit: number; total: number; total_pages: number };
};

export function listForumCategories() {
  return Promise.resolve({
    data: [{ id: "general", name: "General", slug: "general", description: "Community posts" }],
  });
}

export function listForumThreads(
  params: {
    category_id?: string;
    course_id?: string;
    search?: string;
    sort?: "recent" | "popular" | "unanswered";
    page?: number;
    limit?: number;
  } = {},
) {
  return apiRequest<{
    data?: Array<{
      id: string;
      title: string;
      body_markdown?: string;
      body_html?: string;
      author_id: string;
      author_name?: string;
      created_at: string;
      comment_count?: number;
      upvote_count?: number;
      status?: string;
    }>;
    posts?: Array<{
      id: string;
      title: string;
      body_markdown?: string;
      body_html?: string;
      author_id: string;
      author_name?: string;
      created_at: string;
      comment_count?: number;
      upvote_count?: number;
      status?: string;
    }>;
    meta?: { page: number; limit: number; total: number; total_pages: number };
  }>("/v1/forum/posts", {
    query: params,
  }).then((result) => {
    const posts = result.data ?? result.posts ?? [];
    return {
      data: posts.map((post) => ({
        id: post.id,
        title: post.title,
        excerpt: post.body_markdown ?? post.body_html,
        category_id: "general",
        category_name: "General",
        author: { id: post.author_id, full_name: post.author_name ?? "Member" },
        created_at: post.created_at,
        reply_count: post.comment_count ?? 0,
        view_count: post.upvote_count ?? 0,
        tags: post.status ? [post.status] : [],
      })),
      meta: result.meta ?? { page: 1, limit: posts.length, total: posts.length, total_pages: 1 },
    };
  });
}

export function getForumThread(threadId: string) {
  return listForumThreads().then((result) => {
    const thread = result.data.find((item) => item.id === threadId);
    if (!thread) throw new Error("Forum post not found");
    return { ...thread, body: thread.excerpt, body_html: thread.excerpt };
  });
}

export function listForumPosts(threadId: string, params: { page?: number; limit?: number } = {}) {
  return apiRequest<{
    data?: Array<{
      id: string;
      post_id: string;
      body_markdown?: string;
      body_html?: string;
      author_id: string;
      author_name?: string;
      created_at: string;
    }>;
    comments?: Array<{
      id: string;
      post_id: string;
      body_markdown?: string;
      body_html?: string;
      author_id: string;
      author_name?: string;
      created_at: string;
    }>;
    meta?: { page: number; limit: number; total: number; total_pages: number };
  }>(`/v1/forum/posts/${encodeURIComponent(threadId)}/comments`, {
    query: params,
  }).then((result) => {
    const comments = result.data ?? result.comments ?? [];
    return {
      data: comments.map((comment) => ({
        id: comment.id,
        thread_id: threadId,
        author: { id: comment.author_id, full_name: comment.author_name ?? "Member" },
        body: comment.body_markdown ?? comment.body_html ?? "",
        body_html: comment.body_html,
        created_at: comment.created_at,
      })),
      meta: result.meta ?? {
        page: 1,
        limit: comments.length,
        total: comments.length,
        total_pages: 1,
      },
    };
  });
}

export function createForumThread(input: {
  title: string;
  body: string;
  category_id?: string;
  course_id?: string;
  tags?: string[];
}) {
  return apiRequest<ForumThreadDetail>("/v1/forum/posts", {
    method: "POST",
    auth: true,
    body: { title: input.title, body_markdown: input.body, course_id: input.course_id },
  });
}

export function replyToThread(threadId: string, body: string) {
  return apiRequest<ForumPost>(`/v1/forum/posts/${encodeURIComponent(threadId)}/comments`, {
    method: "POST",
    auth: true,
    body: { body_markdown: body },
  });
}

export function togglePostLike(postId: string) {
  return apiRequest<{ like_count: number; has_liked: boolean }>(
    `/v1/forum/posts/${encodeURIComponent(postId)}/upvote`,
    { method: "POST", auth: true },
  );
}

export function flagForumPost(postId: string, reason: string, details?: string) {
  return apiRequest<{ id: string; status: string }>(
    `/v1/forum/posts/${encodeURIComponent(postId)}/flag`,
    { method: "POST", auth: true, body: { reason, details } },
  );
}

export function flagForumThread(threadId: string, reason: string, details?: string) {
  return apiRequest<{ id: string; status: string }>(
    `/v1/forum/posts/${encodeURIComponent(threadId)}/flag`,
    { method: "POST", auth: true, body: { reason, details } },
  );
}

export type PendingPost = {
  id: string;
  author_id: string;
  title: string;
  body_markdown: string;
  body_html: string;
  status: "pending" | "active" | "rejected" | "removed";
  created_at: string;
  updated_at: string;
};

export function listPendingPosts(params: { page?: number; limit?: number } = {}) {
  return apiRequest<Paginated<PendingPost>>("/v1/admin/forum/posts", {
    auth: true,
    query: { ...params, status: "pending" },
  });
}

export function approvePendingPost(postId: string) {
  return apiRequest<{ id: string; status: string }>(
    `/v1/admin/forum/posts/${encodeURIComponent(postId)}/approve`,
    { method: "POST", auth: true },
  );
}

export function rejectPendingPost(postId: string, reason: string) {
  return apiRequest<{ id: string; status: string }>(
    `/v1/admin/forum/posts/${encodeURIComponent(postId)}/reject`,
    { method: "POST", auth: true, body: { reason } },
  );
}
