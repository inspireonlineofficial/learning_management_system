import { apiRequest } from "./client";

export type AuditLog = {
  id: string;
  actor?: { id: string; full_name: string };
  actor_id?: string;
  actor_name?: string;
  action: string;
  target?: string;
  target_type?: string | null;
  target_id?: string | null;
  ip?: string;
  ip_address?: string | null;
  created_at: string;
  metadata?: Record<string, unknown>;
};
export const listAuditLogs = (query?: {
  actor?: string;
  actor_id?: string;
  action?: string;
  target_type?: string;
  target_id?: string;
  from?: string;
  to?: string;
  from_date?: string;
  to_date?: string;
  limit?: number;
  page?: number;
}) =>
  apiRequest<{ items?: AuditLog[]; data?: AuditLog[] }>("/v1/admin/audit-logs", {
    auth: true,
    query: {
      ...query,
      actor_id: query?.actor_id ?? query?.actor,
      from_date: query?.from_date ?? query?.from,
      to_date: query?.to_date ?? query?.to,
    },
  }).then((result) => ({ items: result.items ?? result.data ?? [] }));
