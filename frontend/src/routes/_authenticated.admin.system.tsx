import { createFileRoute } from "@tanstack/react-router";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { toast } from "sonner";

import { AppShell, StatCard } from "@/components/layout/app-shell";
import { apiRequest } from "@/lib/api/client";
import {
  getPlatformSettings,
  getPointsConfig,
  updatePlatformSettings,
  updatePointsConfig,
  type PlatformSettings,
  type PointsConfig,
} from "@/lib/api/settings";

type SysHealth = {
  uptime_seconds: number;
  queue_depth: number;
  worker_count: number;
  db_status: "ok" | "degraded" | "down";
  cache_hit_rate: number;
};

export const Route = createFileRoute("/_authenticated/admin/system")({
  component: Page,
});

type Tab = "health" | "general" | "points";

function Page() {
  const [tab, setTab] = useState<Tab>("health");

  return (
    <AppShell eyebrow="System" title="System & settings">
      <nav className="flex flex-wrap gap-2 mb-8">
        {(
          [
            ["health", "Health"],
            ["general", "General"],
            ["points", "Points & levels"],
          ] as [Tab, string][]
        ).map(([t, label]) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`px-5 py-2 text-sm font-medium ${
              tab === t
                ? "bg-brand text-white"
                : "border border-brand/15 text-brand/70 hover:bg-brand/[0.03]"
            }`}
          >
            {label}
          </button>
        ))}
      </nav>

      {tab === "health" && <HealthTab />}
      {tab === "general" && <GeneralTab />}
      {tab === "points" && <PointsTab />}
    </AppShell>
  );
}

function HealthTab() {
  const { data } = useQuery({
    queryKey: ["system-health"],
    queryFn: () =>
      apiRequest<Partial<SysHealth> & { status?: string }>("/v1/admin/system/health", {
        auth: true,
      }).then(
        (health) =>
          ({
            uptime_seconds: health.uptime_seconds ?? 0,
            queue_depth: health.queue_depth ?? 0,
            worker_count: health.worker_count ?? 0,
            db_status: health.db_status ?? (health.status === "ok" ? "ok" : "degraded"),
            cache_hit_rate: health.cache_hit_rate ?? 0,
          }) satisfies SysHealth,
      ),
    refetchInterval: 30000,
  });
  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <StatCard label="DB" value={data?.db_status ?? "—"} />
      <StatCard label="Queue" value={data?.queue_depth ?? "—"} hint="depth" />
      <StatCard label="Workers" value={data?.worker_count ?? "—"} />
      <StatCard
        label="Cache"
        value={data ? `${Math.round(data.cache_hit_rate * 100)}%` : "—"}
        hint="hit rate"
      />
    </div>
  );
}

function GeneralTab() {
  const qc = useQueryClient();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["platform-settings"],
    queryFn: getPlatformSettings,
  });
  const [form, setForm] = useState<PlatformSettings | null>(null);

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  const save = useMutation({
    mutationFn: (input: Partial<PlatformSettings>) => updatePlatformSettings(input),
    onSuccess: () => {
      toast.success("Settings saved");
      qc.invalidateQueries({ queryKey: ["platform-settings"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (isLoading) return <div className="h-48 border border-brand/10 bg-white/30 animate-pulse" />;
  if (isError) return <p className="text-sm text-destructive">{(error as Error)?.message}</p>;
  if (!form) return null;

  const set = <K extends keyof PlatformSettings>(k: K, v: PlatformSettings[K]) =>
    setForm({ ...form, [k]: v });

  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        save.mutate(form);
      }}
      className="max-w-2xl space-y-5"
    >
      <div className="grid sm:grid-cols-2 gap-4">
        <TextField label="Site name" value={form.site_name} onChange={(v) => set("site_name", v)} />
        <TextField
          label="Support email"
          type="email"
          value={form.support_email}
          onChange={(v) => set("support_email", v)}
        />
        <TextField
          label="Default currency (ISO)"
          value={form.default_currency}
          onChange={(v) => set("default_currency", v.toUpperCase().slice(0, 3))}
        />
        <TextField
          label="Default language"
          value={form.default_language}
          onChange={(v) => set("default_language", v)}
        />
        <TextField
          label="Default timezone"
          value={form.default_timezone}
          onChange={(v) => set("default_timezone", v)}
        />
      </div>

      <div className="space-y-2 pt-3 border-t border-brand/10">
        <SwitchRow
          label="Allow self sign-up"
          hint="If off, accounts must be created by an admin."
          checked={form.allow_self_signup}
          onChange={(v) => set("allow_self_signup", v)}
        />
        <SwitchRow
          label="Require email verification"
          checked={form.require_email_verification}
          onChange={(v) => set("require_email_verification", v)}
        />
        <SwitchRow
          label="Teacher applications open"
          hint="Lets users apply to become teachers."
          checked={form.teacher_application_open}
          onChange={(v) => set("teacher_application_open", v)}
        />
        <SwitchRow
          label="Maintenance mode"
          hint="Shows a maintenance page to everyone except admins."
          checked={form.maintenance_mode}
          onChange={(v) => set("maintenance_mode", v)}
        />
      </div>

      <button
        type="submit"
        disabled={save.isPending}
        className="bg-brand text-white px-6 py-3 text-sm hover:bg-brand/90 disabled:opacity-60"
      >
        {save.isPending ? "Saving…" : "Save settings"}
      </button>
    </form>
  );
}

function PointsTab() {
  const qc = useQueryClient();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["points-config"],
    queryFn: getPointsConfig,
  });
  const [form, setForm] = useState<PointsConfig | null>(null);
  const [levelsText, setLevelsText] = useState("");

  useEffect(() => {
    if (data) {
      setForm(data);
      setLevelsText(data.level_thresholds.join(", "));
    }
  }, [data]);

  const save = useMutation({
    mutationFn: (input: PointsConfig) => updatePointsConfig(input),
    onSuccess: () => {
      toast.success("Points config saved");
      qc.invalidateQueries({ queryKey: ["points-config"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (isLoading) return <div className="h-48 border border-brand/10 bg-white/30 animate-pulse" />;
  if (isError) return <p className="text-sm text-destructive">{(error as Error)?.message}</p>;
  if (!form) return null;

  const set = <K extends keyof PointsConfig>(k: K, v: PointsConfig[K]) =>
    setForm({ ...form, [k]: v });

  const numericFields: { key: keyof PointsConfig; label: string; hint?: string }[] = [
    { key: "lesson_completed", label: "Lesson completed" },
    { key: "quiz_passed", label: "Quiz passed" },
    { key: "quiz_perfect_bonus", label: "Quiz perfect bonus", hint: "Added on 100% score" },
    { key: "assignment_submitted", label: "Assignment submitted" },
    { key: "assignment_graded_bonus", label: "Assignment graded bonus" },
    { key: "live_class_attended", label: "Live class attended" },
    { key: "daily_streak_bonus", label: "Daily streak bonus" },
    { key: "forum_post_created", label: "Forum post created" },
    { key: "forum_helpful_vote", label: "Forum helpful vote" },
  ];

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    const thresholds = levelsText
      .split(",")
      .map((s) => Number(s.trim()))
      .filter((n) => Number.isFinite(n) && n >= 0)
      .sort((a, b) => a - b);
    if (thresholds.length === 0) {
      toast.error("Add at least one level threshold.");
      return;
    }
    save.mutate({ ...form, level_thresholds: thresholds });
  };

  return (
    <form onSubmit={submit} className="max-w-3xl space-y-6">
      <section>
        <h3 className="font-serif text-xl mb-1">Points awarded per action</h3>
        <p className="text-xs text-brand/55 mb-4">
          Tune how generous each event is. Changes apply to new events only.
        </p>
        <div className="grid sm:grid-cols-2 gap-4">
          {numericFields.map((f) => (
            <NumberField
              key={f.key as string}
              label={f.label}
              hint={f.hint}
              value={form[f.key] as number}
              onChange={(v) => set(f.key, v as PointsConfig[typeof f.key])}
            />
          ))}
        </div>
      </section>

      <section className="pt-4 border-t border-brand/10">
        <h3 className="font-serif text-xl mb-1">Level thresholds</h3>
        <p className="text-xs text-brand/55 mb-3">
          Comma-separated cumulative points for each level. Example:{" "}
          <code>0, 100, 300, 700, 1500</code>.
        </p>
        <textarea
          value={levelsText}
          onChange={(e) => setLevelsText(e.target.value)}
          rows={3}
          className="w-full p-3 bg-white border border-brand/15 text-sm font-mono focus:border-brand/40 focus:outline-none"
        />
        <div className="mt-3 flex flex-wrap gap-2">
          {levelsText
            .split(",")
            .map((s) => s.trim())
            .filter(Boolean)
            .map((v, i) => (
              <span
                key={i}
                className="px-2.5 py-1 text-xs border border-brand/15 bg-white text-brand/70"
              >
                Level {i + 1} · {v}
              </span>
            ))}
        </div>
      </section>

      <button
        type="submit"
        disabled={save.isPending}
        className="bg-brand text-white px-6 py-3 text-sm hover:bg-brand/90 disabled:opacity-60"
      >
        {save.isPending ? "Saving…" : "Save points config"}
      </button>
    </form>
  );
}

function TextField({
  label,
  value,
  onChange,
  type = "text",
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  type?: string;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/55">{label}</span>
      <input
        type={type}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1.5 w-full p-3 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
      />
    </label>
  );
}

function NumberField({
  label,
  hint,
  value,
  onChange,
}: {
  label: string;
  hint?: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/55">{label}</span>
      <input
        type="number"
        min={0}
        value={Number.isFinite(value) ? value : 0}
        onChange={(e) => onChange(Number(e.target.value))}
        className="mt-1.5 w-full p-3 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
      />
      {hint && <p className="mt-1 text-[11px] text-brand/45">{hint}</p>}
    </label>
  );
}

function SwitchRow({
  label,
  hint,
  checked,
  onChange,
}: {
  label: string;
  hint?: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="flex items-center justify-between gap-4 py-3 cursor-pointer">
      <div className="min-w-0">
        <p className="text-sm">{label}</p>
        {hint && <p className="text-xs text-brand/55 mt-0.5">{hint}</p>}
      </div>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 accent-brand flex-shrink-0"
      />
    </label>
  );
}
