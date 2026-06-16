import { createFileRoute, Link } from "@tanstack/react-router";
import { Award } from "lucide-react";

import { DataPage } from "@/components/layout/data-page";
import { listCertificates, type Certificate } from "@/lib/api/certificates";

export const Route = createFileRoute("/_authenticated/student/certificates/")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Certificates"
      title="Your certificates"
      queryKey={["certificates"]}
      queryFn={listCertificates}
      empty={{ title: "No certificates yet", description: "Complete a course to earn one." }}
    >
      {(data) => (
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {data.items.map((c: Certificate) => (
            <Link
              key={c.id}
              to="/student/certificates/$courseId"
              params={{ courseId: c.course_id }}
              className="block border border-brand/10 bg-white/50 hover:bg-white p-6 transition-colors"
            >
              <Award className="h-6 w-6 text-accent" />
              <p className="mt-4 font-serif text-lg">{c.course_title}</p>
              <p className="mt-2 text-xs text-brand/55">
                Issued {new Date(c.issued_at).toLocaleDateString()}
              </p>
            </Link>
          ))}
        </div>
      )}
    </DataPage>
  );
}
