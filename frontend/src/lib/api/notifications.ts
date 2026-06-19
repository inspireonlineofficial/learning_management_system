import { apiRequest } from "./client";

export type Notification = {
  id: string;
  type: string;
  title: string;
  body?: string;
  created_at: string;
  is_read?: boolean;
  read_at?: string | null;
  action_url?: string;
};
type NotificationListApiResponse = {
  data?: Notification[];
  items?: Notification[];
  unread_count?: number;
};
export const listNotifications = () =>
  apiRequest<NotificationListApiResponse>("/v1/notifications", { auth: true }).then((response) => ({
    items: (response.items ?? response.data ?? []).map((notification) => ({
      ...notification,
      read_at: notification.read_at ?? (notification.is_read ? notification.created_at : null),
    })),
    unread_count: response.unread_count ?? 0,
  }));
export const markRead = (ids: string[]) =>
  Promise.all(
    ids.map((id) =>
      apiRequest<{ ok: true }>(`/v1/notifications/${encodeURIComponent(id)}/read`, {
        method: "PATCH",
        auth: true,
      }),
    ),
  ).then(() => ({ ok: true }));
export const markAllRead = () =>
  apiRequest<{ ok: true }>("/v1/notifications/read-all", { method: "PATCH", auth: true });

export type BroadcastAudience = "all" | "students" | "teachers";
export type BroadcastInput = {
  audience: BroadcastAudience;
  title: string;
  body: string;
};
export const broadcastNotification = (input: BroadcastInput) =>
  apiRequest<{ id?: string; sent_count?: number; recipient_count?: number; scheduled?: boolean }>(
    "/v1/admin/notifications/broadcast",
    {
      method: "POST",
      auth: true,
      body: {
        title: input.title,
        body: input.body,
        target_role:
          input.audience === "students"
            ? "student"
            : input.audience === "teachers"
              ? "teacher"
              : undefined,
      },
    },
  ).then((result) => ({
    id: result.id ?? "broadcast",
    sent_count: result.sent_count ?? result.recipient_count ?? 0,
    scheduled: result.scheduled,
  }));

export type NotificationTemplate = {
  id: string;
  type: string;
  channel: "in_app" | "email" | "both";
  subject_template?: string | null;
  body_template: string;
  allowed_variables: string[];
  updated_at: string;
};

export const listNotificationTemplates = () =>
  apiRequest<NotificationTemplate[]>("/v1/admin/notifications/templates", { auth: true }).then(
    (templates) => ({ items: templates }),
  );

export const updateNotificationTemplate = (
  templateId: string,
  input: { subject_template?: string | null; body_template: string },
) =>
  apiRequest<NotificationTemplate>(
    `/v1/admin/notifications/templates/${encodeURIComponent(templateId)}`,
    {
      method: "PATCH",
      auth: true,
      body: input,
    },
  );

export type BroadcastHistoryItem = {
  id: string;
  audience: BroadcastAudience;
  title: string;
  body: string;
  sent_count: number;
  created_at: string;
  scheduled_for?: string | null;
  status: "sent" | "scheduled" | "failed";
};
export const listBroadcasts = () =>
  apiRequest<{ items?: BroadcastHistoryItem[]; data?: BroadcastHistoryItem[] }>(
    "/v1/admin/notifications/broadcasts",
    { auth: true },
  ).then((response) => ({ items: response.items ?? response.data ?? [] }));
