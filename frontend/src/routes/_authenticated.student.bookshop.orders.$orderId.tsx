import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { AppShell } from "@/components/layout/app-shell";
import { DetailGrid } from "@/components/layout/data-page";
import { getMyOrder } from "@/lib/api/orders";

export const Route = createFileRoute("/_authenticated/student/bookshop/orders/$orderId")({
  component: Page,
});

function Page() {
  const { orderId } = Route.useParams();
  const { data } = useQuery({
    queryKey: ["my-order", orderId],
    queryFn: () => getMyOrder(orderId),
  });

  return (
    <AppShell eyebrow="Order" title={`Order ${orderId.slice(0, 8)}`}>
      {data && (
        <>
          <DetailGrid
            items={[
              { label: "Status", value: data.status },
              { label: "Placed", value: new Date(data.created_at).toLocaleString() },
              { label: "Total", value: `${data.currency} ${data.total.toFixed(2)}` },
              { label: "Ship to", value: data.shipping_address ?? "—" },
            ]}
          />
          <h2 className="mt-10 font-serif text-2xl">Items</h2>
          <ul className="mt-4 border border-brand/10 bg-white/40 divide-y divide-brand/10">
            {data.items.map((i) => (
              <li key={i.id} className="p-4 flex justify-between text-sm">
                <span>
                  {i.title} <span className="text-brand/55">× {i.quantity}</span>
                </span>
                <span>
                  {data.currency} {i.unit_price.toFixed(2)}
                </span>
              </li>
            ))}
          </ul>
        </>
      )}
    </AppShell>
  );
}
