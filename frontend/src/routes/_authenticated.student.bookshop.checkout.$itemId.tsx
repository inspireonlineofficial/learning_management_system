import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { useMutation } from "@tanstack/react-query";
import { toast } from "sonner";

import { AppShell } from "@/components/layout/app-shell";
import { createPurchaseRequest } from "@/lib/api/approvals";

export const Route = createFileRoute("/_authenticated/student/bookshop/checkout/$itemId")({
  component: Page,
});

function Page() {
  const { itemId } = Route.useParams();
  const navigate = useNavigate();
  const mut = useMutation({
    mutationFn: () => createPurchaseRequest({ item_type: "book", item_id: itemId }),
    onSuccess: () => {
      toast.success("Request submitted for approval");
      navigate({ to: "/student/bookshop/requests" });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  return (
    <AppShell eyebrow="Checkout" title="Confirm purchase">
      <p className="max-w-xl text-sm text-brand/65">
        Purchases of paid items require approval. Submit your request and you'll be notified when
        it's processed.
      </p>
      <button
        onClick={() => mut.mutate()}
        disabled={mut.isPending}
        className="mt-6 bg-brand text-white px-8 py-3 text-sm disabled:opacity-50"
      >
        {mut.isPending ? "Submitting…" : "Submit request"}
      </button>
    </AppShell>
  );
}
