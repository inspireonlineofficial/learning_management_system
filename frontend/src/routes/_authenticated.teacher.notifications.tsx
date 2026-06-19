import { createFileRoute } from "@tanstack/react-router";

import { AppShell } from "@/components/layout/app-shell";
import { NotificationInbox } from "@/components/notifications/notification-inbox";

export const Route = createFileRoute("/_authenticated/teacher/notifications")({
  component: Page,
});

function Page() {
  return (
    <AppShell eyebrow="Notifications" title="Notifications">
      <NotificationInbox />
    </AppShell>
  );
}
