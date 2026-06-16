import { useQuery } from "@tanstack/react-query";
import { createFileRoute, Link } from "@tanstack/react-router";
import { CheckCircle2, ShieldCheck, XCircle } from "lucide-react";

import { verifyCertificate } from "@/lib/api/certificates";

export const Route = createFileRoute("/verify/$certId")({
  component: CertificateVerifyPage,
});

function CertificateVerifyPage() {
  const { certId } = Route.useParams();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["certificate-verification", certId],
    queryFn: () => verifyCertificate(certId),
  });

  const valid = data?.valid === true;

  return (
    <div className="min-h-screen bg-surface text-brand font-sans">
      <header className="border-b border-brand/10">
        <div className="max-w-5xl mx-auto px-6 lg:px-10 py-6 flex items-center justify-between">
          <Link to="/" className="font-serif italic text-2xl text-accent">
            Inspire LMS
          </Link>
          <Link to="/courses" className="text-sm text-brand/65 hover:text-brand">
            Browse courses
          </Link>
        </div>
      </header>

      <main className="max-w-5xl mx-auto px-6 lg:px-10 py-14">
        <p className="eyebrow text-accent mb-4">Certificate verification</p>
        <h1 className="font-serif text-4xl lg:text-6xl text-balance">
          Verify an Inspire certificate.
        </h1>

        <section className="mt-10 border border-brand/10 bg-white/60 p-8">
          {isLoading ? (
            <div className="h-40 bg-brand/5 animate-pulse" />
          ) : isError || !data ? (
            <div className="flex gap-4">
              <XCircle className="h-8 w-8 text-destructive shrink-0" />
              <div>
                <h2 className="font-serif text-2xl">Verification unavailable</h2>
                <p className="mt-2 text-sm text-brand/60">
                  {(error as Error)?.message ?? "This certificate could not be verified."}
                </p>
              </div>
            </div>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-4">
                {valid ? (
                  <CheckCircle2 className="h-10 w-10 text-emerald-600" />
                ) : (
                  <XCircle className="h-10 w-10 text-destructive" />
                )}
                <div>
                  <p className="eyebrow text-brand/45">Status</p>
                  <h2 className="font-serif text-3xl">
                    {valid ? "Certificate is valid" : "Certificate is not valid"}
                  </h2>
                </div>
              </div>

              <dl className="mt-8 grid sm:grid-cols-2 gap-5">
                <Meta label="Verification ID" value={data.verification_id} />
                <Meta label="Student" value={data.student_name ?? "Not available"} />
                <Meta label="Course" value={data.course_title ?? "Not available"} />
                <Meta label="Instructor" value={data.instructor_name ?? "Not available"} />
                <Meta
                  label="Completed"
                  value={
                    data.completion_date
                      ? new Date(data.completion_date).toLocaleDateString()
                      : "Not available"
                  }
                />
                <Meta
                  label="Issued"
                  value={data.issued_at ? new Date(data.issued_at).toLocaleDateString() : "—"}
                />
              </dl>
            </>
          )}
        </section>

        <div className="mt-8 flex items-start gap-3 text-sm text-brand/60 max-w-2xl">
          <ShieldCheck className="h-5 w-5 text-accent shrink-0" />
          <p>
            This page checks the public verification record stored by Inspire. It does not expose
            private student account information.
          </p>
        </div>
      </main>
    </div>
  );
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="border border-brand/10 bg-white/50 p-4">
      <dt className="eyebrow text-brand/45">{label}</dt>
      <dd className="mt-1 text-brand">{value}</dd>
    </div>
  );
}
