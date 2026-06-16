import { createFileRoute, Link } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Award, Download } from "lucide-react";
import { toast } from "sonner";

import { AppShell, EmptyState } from "@/components/layout/app-shell";
import { getCertificate, downloadCertificatePdf } from "@/lib/api/certificates";
import { useAuth } from "@/context/auth-context";

export const Route = createFileRoute("/_authenticated/student/certificates/$courseId")({
  component: Page,
});

function Page() {
  const { courseId } = Route.useParams();
  const { user } = useAuth();
  const { data, isLoading } = useQuery({
    queryKey: ["certificate", courseId],
    queryFn: () => getCertificate(courseId),
  });

  const handleDownload = async () => {
    try {
      const r = await downloadCertificatePdf(courseId);
      window.open(r.url, "_blank");
    } catch (e) {
      toast.error((e as Error).message);
    }
  };

  return (
    <AppShell eyebrow="Certificate" title={data?.course_title ?? "Certificate"}>
      <Link
        to="/student/certificates"
        className="inline-block text-xs text-brand/55 hover:text-brand mb-6"
      >
        ← All certificates
      </Link>

      {isLoading && <div className="h-80 border border-brand/10 bg-white/30 animate-pulse" />}
      {!isLoading && !data && <EmptyState title="Certificate not found" />}

      {data && (
        <>
          {/* Visual certificate */}
          <div className="relative mx-auto max-w-3xl border-[12px] border-brand/15 bg-gradient-to-br from-white to-brand/[0.02] p-12 text-center shadow-sm">
            <div className="absolute inset-3 border border-brand/10 pointer-events-none" />
            <Award className="h-12 w-12 mx-auto text-accent" />
            <p className="eyebrow text-brand/45 mt-6">Certificate of Completion</p>
            <p className="mt-6 text-sm text-brand/65">This is to certify that</p>
            <h2 className="font-serif text-4xl lg:text-5xl mt-3 text-balance">
              {user?.full_name ?? "Scholar"}
            </h2>
            <p className="mt-6 text-sm text-brand/65">has successfully completed</p>
            <h3 className="font-serif text-2xl mt-2 text-balance">{data.course_title}</h3>
            <div className="mt-10 grid sm:grid-cols-3 gap-6 text-left max-w-xl mx-auto">
              <div>
                <p className="eyebrow text-brand/45">Issued</p>
                <p className="text-sm mt-1">{new Date(data.issued_at).toLocaleDateString()}</p>
              </div>
              <div>
                <p className="eyebrow text-brand/45">Verification</p>
                <p className="text-sm mt-1 font-mono break-all">{data.verification_code ?? "—"}</p>
              </div>
              <div>
                <p className="eyebrow text-brand/45">ID</p>
                <p className="text-sm mt-1 font-mono break-all">
                  {(data.id ?? data.verification_code ?? data.course_id).slice(0, 12)}…
                </p>
              </div>
            </div>
          </div>

          <div className="mt-8 flex flex-wrap gap-3 justify-center">
            <button
              onClick={handleDownload}
              className="inline-flex items-center gap-2 bg-brand text-white px-6 py-3 text-sm font-medium"
            >
              <Download className="h-4 w-4" />
              Download PDF
            </button>
            {data.verification_code && (
              <a
                href={`/verify/${data.verification_code}`}
                className="border border-brand/15 px-6 py-3 text-sm hover:bg-brand/[0.03]"
              >
                Verification page
              </a>
            )}
          </div>
        </>
      )}
    </AppShell>
  );
}
