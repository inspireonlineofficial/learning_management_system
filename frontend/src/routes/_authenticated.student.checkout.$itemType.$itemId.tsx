import { createFileRoute, Link, Navigate } from "@tanstack/react-router";
import { AlertTriangle } from "lucide-react";

import { AppShell, EmptyState } from "@/components/layout/app-shell";

export const Route = createFileRoute("/_authenticated/student/checkout/$itemType/$itemId")({
  component: Page,
});

function Page() {
  const { itemType, itemId } = Route.useParams();

  if (itemType === "course") {
    return <Navigate to="/student/checkout/$courseId" params={{ courseId: itemId }} replace />;
  }
  if (itemType === "book") {
    return <Navigate to="/student/bookshop/checkout/$itemId" params={{ itemId }} replace />;
  }
  return (
    <AppShell eyebrow="Checkout" title="Unsupported approval request">
      <EmptyState
        icon={AlertTriangle}
        title="This checkout link is not supported"
        description="Approval requests are available for courses and books. Choose a valid item and submit a new request."
        action={
          <div className="flex flex-wrap justify-center gap-3">
            <Link to="/courses" className="bg-brand px-5 py-2.5 text-sm font-medium text-white">
              Browse courses
            </Link>
            <Link
              to="/student/bookshop"
              className="border border-brand/15 px-5 py-2.5 text-sm font-medium"
            >
              Browse books
            </Link>
          </div>
        }
      />
    </AppShell>
  );
}
