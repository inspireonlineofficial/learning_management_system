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

type BackendModerationItem = Partial<ModerationItem> & {
  flag_id?: string;
  target_type?: string;
  target_id?: string;
  content_preview?: string;
  note?: string | null;
  reporter?: { id?: string; full_name?: string };
};

export const listModerationQueue = () =>
  apiRequest<{ items?: BackendModerationItem[]; data?: BackendModerationItem[] }>(
    "/v1/admin/moderation",
    {
      auth: true,
    },
  ).then((result) => ({ items: (result.items ?? result.data ?? []).map(normalizeModerationItem) }));

function normalizeModerationItem(item: BackendModerationItem): ModerationItem {
  const kind = item.kind ?? (item.target_type as ModerationItem["kind"]) ?? "post";
  return {
    id: item.id ?? item.flag_id ?? item.target_id ?? "moderation-item",
    kind,
    content: item.content ?? item.content_preview ?? "",
    reported_by: item.reported_by ?? {
      id: item.reporter?.id ?? "unknown",
      full_name: item.reporter?.full_name ?? "Reporter",
    },
    reason: item.reason ?? item.note ?? "Flagged for review",
    target_url:
      item.target_url ??
      (item.target_id ? (kind === "post" ? `/forum/${item.target_id}` : undefined) : undefined),
    created_at: item.created_at ?? new Date().toISOString(),
    status: item.status ?? "pending",
  };
}

export const decideModeration = (id: string, action: "approve" | "remove", note?: string) =>
  apiRequest<{ ok: true }>(`/v1/admin/moderation/${encodeURIComponent(id)}/action`, {
    method: "POST",
    auth: true,
    body: { action, reason: note },
  });
