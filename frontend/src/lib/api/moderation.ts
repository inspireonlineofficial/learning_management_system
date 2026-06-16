import { apiRequest } from "./client";

export type ModerationItem = {
  id: string;
  kind: "post" | "comment" | "review" | "message";
  content: string;
  reported_by: { id: string; full_name: string };
  reason: string;
  target_url?: string;
  created_at: string;
  status: "pending" | "approved" | "removed";
};
export const listModerationQueue = () =>
  apiRequest<{ items?: ModerationItem[]; data?: ModerationItem[] }>("/v1/admin/moderation", {
    auth: true,
  }).then((result) => ({ items: result.items ?? result.data ?? [] }));
export const decideModeration = (id: string, action: "approve" | "remove", note?: string) =>
  apiRequest<{ ok: true }>(`/v1/admin/moderation/${encodeURIComponent(id)}/action`, {
    method: "POST",
    auth: true,
    body: { action, reason: note },
  });
