import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { PhoneOff } from "lucide-react";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { endLiveSession, hostJoinSession } from "@/lib/api/live";

export const Route = createFileRoute("/_authenticated/teacher/live/$sessionId/room")({
  component: Page,
});

function Page() {
  const { sessionId } = Route.useParams();
  const navigate = useNavigate();
  const qc = useQueryClient();

  const { data, isError, error } = useQuery({
    queryKey: ["teacher-live-join", sessionId],
    queryFn: () => hostJoinSession(sessionId),
  });

  const end = useMutation({
    mutationFn: () => endLiveSession(sessionId),
    onSuccess: () => {
      toast.success("Session ended");
      qc.invalidateQueries({ queryKey: ["teacher-live"] });
      navigate({ to: "/teacher/live/$sessionId", params: { sessionId } });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <AppShell eyebrow="Live room" title="Host session">
      {isError && <p className="text-sm text-destructive mb-4">{(error as Error)?.message}</p>}

      <div className="max-w-2xl border border-brand/10 bg-white/60 p-8 space-y-6">
        <div>
          <p className="eyebrow text-brand/45">Room token</p>
          <p className="mt-2 text-sm text-brand/65">
            The backend currently returns a room token for host entry.
          </p>
          <div className="mt-4 p-4 border border-dashed border-brand/15 bg-brand/[0.02] break-all text-sm">
            {(data as any)?.room_token ?? (data as any)?.join_url ?? "Loading…"}
          </div>
        </div>

        <button
          onClick={() => {
            if (confirm("End session for everyone?")) end.mutate();
          }}
          disabled={end.isPending}
          className="inline-flex items-center gap-2 px-4 py-3 bg-destructive text-white text-xs hover:bg-destructive/90 disabled:opacity-60"
        >
          <PhoneOff className="h-4 w-4" />
          {end.isPending ? "Ending…" : "End session"}
        </button>
      </div>
    </AppShell>
  );
}
