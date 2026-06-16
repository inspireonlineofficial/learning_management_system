import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";

import { AppShell } from "@/components/layout/app-shell";
import { getTeacherCourse } from "@/lib/api/teacher";

export const Route = createFileRoute("/_authenticated/teacher/courses/$courseId/preview")({
  component: Page,
});

function Page() {
  const { courseId } = Route.useParams();
  const { data } = useQuery({
    queryKey: ["teacher-course", courseId],
    queryFn: () => getTeacherCourse(courseId),
  });
  return (
    <AppShell eyebrow="Preview" title={data?.title ?? "Course preview"}>
      <p className="max-w-xl text-sm text-brand/65">{(data as any)?.subtitle}</p>
      <div className="mt-8 grid lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <h3 className="font-serif text-xl">Curriculum</h3>
          <ul className="mt-4 space-y-2">
            {(data as any)?.modules?.map((m: any) => (
              <li key={m.id} className="border border-brand/10 bg-white/40 p-4">
                <p className="font-serif">{m.title}</p>
                <ul className="mt-2 text-xs text-brand/55 space-y-1">
                  {m.lessons?.map((l: any) => (
                    <li key={l.id}>· {l.title}</li>
                  ))}
                </ul>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </AppShell>
  );
}
