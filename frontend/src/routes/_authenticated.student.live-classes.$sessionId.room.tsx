import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { AppShell } from "@/components/layout/app-shell";
import { joinSession } from "@/lib/api/live";

export const Route = createFileRoute("/_authenticated/student/live-classes/$sessionId/room")({
  component: Page,
});

function Page() {
  const { sessionId } = Route.useParams();
  const { data: token } = useQuery({
    queryKey: ["live-join", sessionId],
    queryFn: () => joinSession(sessionId),
  });

  return (
    <AppShell eyebrow="Live room" title="In session">
      <div className="border border-brand/10 bg-white/50 p-8 max-w-2xl">
        <p className="text-sm text-brand/60">
          The backend returns a room token for live sessions. This screen is a simple handoff point
          until the full realtime classroom is connected.
        </p>
        <div className="mt-6 p-4 border border-dashed border-brand/15 bg-brand/[0.02] break-all text-sm">
          Token: {(token as any)?.room_token ?? (token as any)?.join_url ?? "Loading…"}
        </div>
      </div>
    </AppShell>
  );
}
