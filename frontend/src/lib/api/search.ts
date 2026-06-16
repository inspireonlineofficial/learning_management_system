import { apiRequest } from "./client";

export type SearchResult = {
  type: "course" | "lesson" | "book" | "thread" | "user";
  id: string;
  title: string;
  snippet?: string;
  url: string;
};
export const search = (q: string, scope?: string) =>
  apiRequest<{ items: SearchResult[] }>("/v1/search", { auth: true, query: { q, scope } });
