import { createFileRoute, Link } from "@tanstack/react-router";

import { DataPage, ListTable } from "@/components/layout/data-page";
import { listMyCourseAccessRequests, type CourseAccessRequest } from "@/lib/api/access-requests";

export const Route = createFileRoute("/_authenticated/student/access-requests")({
  component: Page,
});

function Page() {
  return (
    <DataPage
      eyebrow="Access Requests"
      title="Course access requests"
      queryKey={["my-course-access-requests"]}
      queryFn={() => listMyCourseAccessRequests()}
      empty={{ title: "No access requests yet" }}
    >
      {(data: { data: CourseAccessRequest[] }) => (
        <ListTable
          rows={data.data}
          columns={[
            {
              key: "course",
              label: "Course",
              render: (request) => (
                <Link
                  to="/student/courses/$courseId"
                  params={{ courseId: request.item_id }}
                  className="font-medium text-brand hover:text-accent"
                >
                  {request.item_title}
                </Link>
              ),
            },
            { key: "teacher", label: "Teacher", render: (request) => request.item_subtitle || "-" },
            {
              key: "status",
              label: "Status",
              render: (request) => (
                <span className="eyebrow text-brand/55">{request.status.replaceAll("_", " ")}</span>
              ),
            },
            {
              key: "requested",
              label: "Requested",
              render: (request) => new Date(request.created_at).toLocaleDateString(),
            },
            {
              key: "reviewed",
              label: "Reviewed",
              render: (request) =>
                request.reviewed_at ? new Date(request.reviewed_at).toLocaleDateString() : "-",
            },
          ]}
        />
      )}
    </DataPage>
  );
}
