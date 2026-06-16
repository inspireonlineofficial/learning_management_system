import { createFileRoute } from "@tanstack/react-router";
import { Calendar as CalIcon, Radio } from "lucide-react";

import { DataPage } from "@/components/layout/data-page";
import { getCalendar, type CalendarEvent } from "@/lib/api/calendar";

export const Route = createFileRoute("/_authenticated/student/calendar")({
  component: Page,
});

function Page() {
  const from = new Date();
  from.setDate(from.getDate() - 7);
  const to = new Date();
  to.setDate(to.getDate() + 30);
  const range = { from: from.toISOString(), to: to.toISOString() };

  return (
    <DataPage
      eyebrow="Calendar"
      title="Your schedule"
      queryKey={["calendar", range.from, range.to]}
      queryFn={() => getCalendar(range)}
      empty={{
        title: "Nothing scheduled",
        description: "Upcoming live sessions will appear here when your instructors add them.",
      }}
    >
      {(data) => (
        <ul className="space-y-3">
          {data.items.map((e: CalendarEvent) => (
            <li
              key={e.id}
              className={`border p-4 flex gap-4 ${
                e.is_today ? "border-accent bg-accent/5" : "border-brand/10 bg-white/50"
              }`}
            >
              {e.is_today ? (
                <Radio className="h-5 w-5 text-accent flex-shrink-0 mt-0.5" />
              ) : (
                <CalIcon className="h-5 w-5 text-accent flex-shrink-0 mt-0.5" />
              )}
              <div className="min-w-0 flex-1">
                <p className="font-serif text-base">{e.title}</p>
                {e.course_title && <p className="mt-1 text-xs text-brand/55">{e.course_title}</p>}
                <p className="mt-1 text-xs text-brand/45">
                  {new Date(e.starts_at).toLocaleString()}
                  {e.ends_at && ` → ${new Date(e.ends_at).toLocaleTimeString()}`}
                </p>
              </div>
              <span className="eyebrow text-brand/45 self-start">
                {e.is_today ? "Today" : e.type.replace("_", " ")}
              </span>
            </li>
          ))}
        </ul>
      )}
    </DataPage>
  );
}
