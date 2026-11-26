import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { ArrowDown, ArrowUp, Plus, RotateCcw, Save, Trash2 } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { toast } from "sonner";

import {
  FaqSection,
  FooterSection,
  NavigationSection,
  PageCopySection,
  SiteIdentitySection,
} from "@/components/admin/site-config-sections";
import { QueryErrorPanel } from "@/components/layout/query-error-panel";
import {
  DEFAULT_HOMEPAGE_CONFIG,
  getAdminHomepageConfig,
  updateAdminHomepageConfig,
  type HomepageConfig,
} from "@/lib/api/homepage";

function moveItem<T>(list: T[], from: number, to: number): T[] {
  if (to < 0 || to >= list.length) return list;
  const copy = [...list];
  const [item] = copy.splice(from, 1);
  copy.splice(to, 0, item);
  return copy;
}

export function HomepageConfigEditor() {
  const qc = useQueryClient();
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["admin-homepage-config"],
    queryFn: getAdminHomepageConfig,
  });

  const [form, setForm] = useState<HomepageConfig | null>(null);
  const [subjectsText, setSubjectsText] = useState("");

  useEffect(() => {
    if (data) {
      setForm(data);
      setSubjectsText(data.bookshop_subjects.join(", "));
    }
  }, [data]);

  const save = useMutation({
    mutationFn: (payload: HomepageConfig) => updateAdminHomepageConfig(payload),
    onSuccess: (saved) => {
      setForm(saved);
      setSubjectsText(saved.bookshop_subjects.join(", "));
      toast.success("Website content saved");
      qc.invalidateQueries({ queryKey: ["admin-homepage-config"] });
      qc.invalidateQueries({ queryKey: ["homepage", "public"] });
    },
    onError: (e: Error) => toast.error(e.message),
  });

  if (isLoading) return <div className="h-64 border border-brand/10 bg-white/30 animate-pulse" />;
  if (isError)
    return <QueryErrorPanel error={error} variant="compact" message={(error as Error)?.message} />;
  if (!form) return null;

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const subjects = subjectsText
      .split(",")
      .map((s) => s.trim())
      .filter(Boolean);
    save.mutate({
      ...form,
      bookshop_subjects: subjects,
    });
  };

  const handleReset = () => {
    if (!confirm("Reset all public website content to the defaults? Remember to save.")) return;
    setForm(DEFAULT_HOMEPAGE_CONFIG);
    setSubjectsText(DEFAULT_HOMEPAGE_CONFIG.bookshop_subjects.join(", "));
  };

  return (
    <form onSubmit={handleSubmit} className="max-w-4xl space-y-8">
      <div className="flex flex-wrap items-center justify-between gap-4 border border-brand/10 bg-white/50 p-4">
        <div>
          <p className="text-sm font-medium">Public website content</p>
          <p className="text-xs text-brand/60 mt-0.5">
            Brand, logo, SEO, navigation, footer, contact details, FAQ, page headers and every
            homepage section.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <Link
            to="/admin/slides"
            className="border border-brand/15 px-3 py-2 text-xs font-medium text-brand/75 hover:bg-brand/[0.03]"
          >
            Manage hero slides
          </Link>
          <button
            type="button"
            onClick={handleReset}
            className="inline-flex items-center gap-1.5 border border-brand/15 px-3 py-2 text-xs font-medium text-brand/75 hover:bg-brand/[0.03]"
          >
            <RotateCcw className="h-3.5 w-3.5" />
            Reset defaults
          </button>
          <button
            type="submit"
            disabled={save.isPending}
            className="inline-flex items-center gap-1.5 bg-brand px-4 py-2 text-xs font-medium text-white hover:bg-brand/90 disabled:opacity-60"
          >
            <Save className="h-3.5 w-3.5" />
            {save.isPending ? "Saving…" : "Save homepage"}
          </button>
        </div>
      </div>

      <SiteIdentitySection form={form} setForm={setForm} />
      <NavigationSection form={form} setForm={setForm} />
      <FooterSection form={form} setForm={setForm} />
      <PageCopySection form={form} setForm={setForm} />

      {/* 1. Course catalog / tracks */}
      <SectionCard
        title="Course catalog (tracks)"
        enabled={form.course_catalog.enabled}
        onToggle={(enabled) =>
          setForm({ ...form, course_catalog: { ...form.course_catalog, enabled } })
        }
      >
        <div className="grid sm:grid-cols-3 gap-4">
          <Field
            label="Eyebrow"
            value={form.course_catalog.eyebrow}
            onChange={(eyebrow) =>
              setForm({ ...form, course_catalog: { ...form.course_catalog, eyebrow } })
            }
          />
          <Field
            label="Heading"
            value={form.course_catalog.title}
            onChange={(title) =>
              setForm({ ...form, course_catalog: { ...form.course_catalog, title } })
            }
          />
          <Field
            label="Link label"
            value={form.course_catalog.action_label}
            onChange={(action_label) =>
              setForm({ ...form, course_catalog: { ...form.course_catalog, action_label } })
            }
          />
        </div>

        <div className="pt-4 border-t border-brand/10 space-y-3">
          <div className="flex items-center justify-between">
            <span className="eyebrow text-brand/55">
              Tracks ({form.course_catalog.tracks.length})
            </span>
            <button
              type="button"
              onClick={() =>
                setForm({
                  ...form,
                  course_catalog: {
                    ...form.course_catalog,
                    tracks: [...form.course_catalog.tracks, { title: "", detail: "", query: "" }],
                  },
                })
              }
              className="inline-flex items-center gap-1 border border-brand/15 px-3 py-1.5 text-xs hover:bg-brand/[0.03]"
            >
              <Plus className="h-3.5 w-3.5" /> Add track
            </button>
          </div>
          {form.course_catalog.tracks.map((track, idx) => (
            <div
              key={idx}
              className="grid sm:grid-cols-[1fr_1.4fr_0.9fr_auto] gap-3 items-end border border-brand/10 bg-white/60 p-3"
            >
              <Field
                label="Title"
                value={track.title}
                onChange={(title) => {
                  const tracks = [...form.course_catalog.tracks];
                  tracks[idx] = { ...track, title };
                  setForm({ ...form, course_catalog: { ...form.course_catalog, tracks } });
                }}
              />
              <Field
                label="Description"
                value={track.detail}
                onChange={(detail) => {
                  const tracks = [...form.course_catalog.tracks];
                  tracks[idx] = { ...track, detail };
                  setForm({ ...form, course_catalog: { ...form.course_catalog, tracks } });
                }}
              />
              <Field
                label="Search query (optional)"
                placeholder={track.title || "Defaults to title"}
                value={track.query ?? ""}
                onChange={(query) => {
                  const tracks = [...form.course_catalog.tracks];
                  tracks[idx] = { ...track, query: query || undefined };
                  setForm({ ...form, course_catalog: { ...form.course_catalog, tracks } });
                }}
              />
              <RowControls
                canUp={idx > 0}
                canDown={idx < form.course_catalog.tracks.length - 1}
                onUp={() =>
                  setForm({
                    ...form,
                    course_catalog: {
                      ...form.course_catalog,
                      tracks: moveItem(form.course_catalog.tracks, idx, idx - 1),
                    },
                  })
                }
                onDown={() =>
                  setForm({
                    ...form,
                    course_catalog: {
                      ...form.course_catalog,
                      tracks: moveItem(form.course_catalog.tracks, idx, idx + 1),
                    },
                  })
                }
                onRemove={() =>
                  setForm({
                    ...form,
                    course_catalog: {
                      ...form.course_catalog,
                      tracks: form.course_catalog.tracks.filter((_, i) => i !== idx),
                    },
                  })
                }
              />
            </div>
          ))}
        </div>
      </SectionCard>

      {/* 2. Featured courses */}
      <SectionCard
        title="Featured courses"
        enabled={form.featured_courses.enabled}
        onToggle={(enabled) =>
          setForm({ ...form, featured_courses: { ...form.featured_courses, enabled } })
        }
      >
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <Field
            label="Eyebrow"
            value={form.featured_courses.eyebrow}
            onChange={(eyebrow) =>
              setForm({ ...form, featured_courses: { ...form.featured_courses, eyebrow } })
            }
          />
          <Field
            label="Heading"
            value={form.featured_courses.title}
            onChange={(title) =>
              setForm({ ...form, featured_courses: { ...form.featured_courses, title } })
            }
          />
          <Field
            label="Link label"
            value={form.featured_courses.action_label}
            onChange={(action_label) =>
              setForm({ ...form, featured_courses: { ...form.featured_courses, action_label } })
            }
          />
          <NumberInput
            label="Max courses"
            value={form.featured_courses.max_items}
            onChange={(max_items) =>
              setForm({ ...form, featured_courses: { ...form.featured_courses, max_items } })
            }
          />
        </div>
        <div className="grid sm:grid-cols-2 gap-4">
          <Field
            label="Empty state title"
            value={form.featured_courses.empty_title}
            onChange={(empty_title) =>
              setForm({ ...form, featured_courses: { ...form.featured_courses, empty_title } })
            }
          />
          <Field
            label="Empty state button"
            value={form.featured_courses.empty_action_label}
            onChange={(empty_action_label) =>
              setForm({
                ...form,
                featured_courses: { ...form.featured_courses, empty_action_label },
              })
            }
          />
        </div>
        <TextAreaField
          label="Empty state description"
          value={form.featured_courses.empty_body}
          onChange={(empty_body) =>
            setForm({ ...form, featured_courses: { ...form.featured_courses, empty_body } })
          }
        />
      </SectionCard>

      {/* 3. Bookshop catalog */}
      <SectionCard
        title="Bookshop catalog (study materials)"
        enabled={form.bookshop_catalog.enabled}
        onToggle={(enabled) =>
          setForm({ ...form, bookshop_catalog: { ...form.bookshop_catalog, enabled } })
        }
      >
        <div className="grid sm:grid-cols-3 gap-4">
          <Field
            label="Eyebrow"
            value={form.bookshop_catalog.eyebrow}
            onChange={(eyebrow) =>
              setForm({ ...form, bookshop_catalog: { ...form.bookshop_catalog, eyebrow } })
            }
          />
          <Field
            label="Heading"
            value={form.bookshop_catalog.title}
            onChange={(title) =>
              setForm({ ...form, bookshop_catalog: { ...form.bookshop_catalog, title } })
            }
          />
          <Field
            label="Link label"
            value={form.bookshop_catalog.action_label}
            onChange={(action_label) =>
              setForm({ ...form, bookshop_catalog: { ...form.bookshop_catalog, action_label } })
            }
          />
        </div>

        <div className="pt-4 border-t border-brand/10 space-y-3">
          <div className="flex items-center justify-between">
            <span className="eyebrow text-brand/55">
              Categories ({form.bookshop_catalog.categories.length})
            </span>
            <button
              type="button"
              onClick={() =>
                setForm({
                  ...form,
                  bookshop_catalog: {
                    ...form.bookshop_catalog,
                    categories: [...form.bookshop_catalog.categories, { label: "", query: "" }],
                  },
                })
              }
              className="inline-flex items-center gap-1 border border-brand/15 px-3 py-1.5 text-xs hover:bg-brand/[0.03]"
            >
              <Plus className="h-3.5 w-3.5" /> Add category
            </button>
          </div>
          {form.bookshop_catalog.categories.map((cat, idx) => (
            <div
              key={idx}
              className="grid sm:grid-cols-[1fr_1fr_auto] gap-3 items-end border border-brand/10 bg-white/60 p-3"
            >
              <Field
                label="Label"
                value={cat.label}
                onChange={(label) => {
                  const categories = [...form.bookshop_catalog.categories];
                  categories[idx] = { ...cat, label };
                  setForm({ ...form, bookshop_catalog: { ...form.bookshop_catalog, categories } });
                }}
              />
              <Field
                label="Search query (optional)"
                placeholder={cat.label || "Defaults to label"}
                value={cat.query ?? ""}
                onChange={(query) => {
                  const categories = [...form.bookshop_catalog.categories];
                  categories[idx] = { ...cat, query: query || undefined };
                  setForm({ ...form, bookshop_catalog: { ...form.bookshop_catalog, categories } });
                }}
              />
              <RowControls
                canUp={idx > 0}
                canDown={idx < form.bookshop_catalog.categories.length - 1}
                onUp={() =>
                  setForm({
                    ...form,
                    bookshop_catalog: {
                      ...form.bookshop_catalog,
                      categories: moveItem(form.bookshop_catalog.categories, idx, idx - 1),
                    },
                  })
                }
                onDown={() =>
                  setForm({
                    ...form,
                    bookshop_catalog: {
                      ...form.bookshop_catalog,
                      categories: moveItem(form.bookshop_catalog.categories, idx, idx + 1),
                    },
                  })
                }
                onRemove={() =>
                  setForm({
                    ...form,
                    bookshop_catalog: {
                      ...form.bookshop_catalog,
                      categories: form.bookshop_catalog.categories.filter((_, i) => i !== idx),
                    },
                  })
                }
              />
            </div>
          ))}
        </div>

        <div className="pt-4 border-t border-brand/10">
          <Field
            label="Bookshop page subject filter chips (comma-separated)"
            value={subjectsText}
            onChange={setSubjectsText}
            placeholder="Physics, Chemistry, Biology, Mathematics, ICT, Exam Prep"
          />
        </div>
      </SectionCard>

      {/* 4. Featured books */}
      <SectionCard
        title="Featured books"
        enabled={form.featured_books.enabled}
        onToggle={(enabled) =>
          setForm({ ...form, featured_books: { ...form.featured_books, enabled } })
        }
      >
        <div className="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
          <Field
            label="Eyebrow"
            value={form.featured_books.eyebrow}
            onChange={(eyebrow) =>
              setForm({ ...form, featured_books: { ...form.featured_books, eyebrow } })
            }
          />
          <Field
            label="Default heading"
            value={form.featured_books.title}
            onChange={(title) =>
              setForm({ ...form, featured_books: { ...form.featured_books, title } })
            }
          />
          <Field
            label="Free/offer heading"
            value={form.featured_books.offer_title}
            onChange={(offer_title) =>
              setForm({ ...form, featured_books: { ...form.featured_books, offer_title } })
            }
          />
          <NumberInput
            label="Max books"
            value={form.featured_books.max_items}
            onChange={(max_items) =>
              setForm({ ...form, featured_books: { ...form.featured_books, max_items } })
            }
          />
        </div>
        <div className="grid sm:grid-cols-3 gap-4">
          <Field
            label="Link label"
            value={form.featured_books.action_label}
            onChange={(action_label) =>
              setForm({ ...form, featured_books: { ...form.featured_books, action_label } })
            }
          />
          <Field
            label="Empty state title"
            value={form.featured_books.empty_title}
            onChange={(empty_title) =>
              setForm({ ...form, featured_books: { ...form.featured_books, empty_title } })
            }
          />
          <Field
            label="Empty state button"
            value={form.featured_books.empty_action_label}
            onChange={(empty_action_label) =>
              setForm({ ...form, featured_books: { ...form.featured_books, empty_action_label } })
            }
          />
        </div>
        <TextAreaField
          label="Empty state description"
          value={form.featured_books.empty_body}
          onChange={(empty_body) =>
            setForm({ ...form, featured_books: { ...form.featured_books, empty_body } })
          }
        />
      </SectionCard>

      {/* 5. Outcomes & stats */}
      <SectionCard
        title="Outcomes & stats"
        enabled={form.outcomes.enabled}
        onToggle={(enabled) => setForm({ ...form, outcomes: { ...form.outcomes, enabled } })}
      >
        <div className="grid sm:grid-cols-2 gap-4">
          <Field
            label="Eyebrow"
            value={form.outcomes.eyebrow}
            onChange={(eyebrow) => setForm({ ...form, outcomes: { ...form.outcomes, eyebrow } })}
          />
          <Field
            label="Heading"
            value={form.outcomes.title}
            onChange={(title) => setForm({ ...form, outcomes: { ...form.outcomes, title } })}
          />
        </div>
        <TextAreaField
          label="Description"
          value={form.outcomes.body}
          onChange={(body) => setForm({ ...form, outcomes: { ...form.outcomes, body } })}
        />
        <div className="pt-4 border-t border-brand/10 space-y-3">
          <div className="flex items-center justify-between">
            <span className="eyebrow text-brand/55">Stats ({form.outcomes.stats.length})</span>
            <button
              type="button"
              onClick={() =>
                setForm({
                  ...form,
                  outcomes: {
                    ...form.outcomes,
                    stats: [...form.outcomes.stats, { label: "", value: "" }],
                  },
                })
              }
              className="inline-flex items-center gap-1 border border-brand/15 px-3 py-1.5 text-xs hover:bg-brand/[0.03]"
            >
              <Plus className="h-3.5 w-3.5" /> Add stat
            </button>
          </div>
          {form.outcomes.stats.map((stat, idx) => (
            <div
              key={idx}
              className="grid sm:grid-cols-[1fr_1fr_auto] gap-3 items-end border border-brand/10 bg-white/60 p-3"
            >
              <Field
                label="Label"
                value={stat.label}
                onChange={(label) => {
                  const stats = [...form.outcomes.stats];
                  stats[idx] = { ...stat, label };
                  setForm({ ...form, outcomes: { ...form.outcomes, stats } });
                }}
              />
              <Field
                label="Value"
                value={stat.value}
                onChange={(value) => {
                  const stats = [...form.outcomes.stats];
                  stats[idx] = { ...stat, value };
                  setForm({ ...form, outcomes: { ...form.outcomes, stats } });
                }}
              />
              <RowControls
                canUp={idx > 0}
                canDown={idx < form.outcomes.stats.length - 1}
                onUp={() =>
                  setForm({
                    ...form,
                    outcomes: {
                      ...form.outcomes,
                      stats: moveItem(form.outcomes.stats, idx, idx - 1),
                    },
                  })
                }
                onDown={() =>
                  setForm({
                    ...form,
                    outcomes: {
                      ...form.outcomes,
                      stats: moveItem(form.outcomes.stats, idx, idx + 1),
                    },
                  })
                }
                onRemove={() =>
                  setForm({
                    ...form,
                    outcomes: {
                      ...form.outcomes,
                      stats: form.outcomes.stats.filter((_, i) => i !== idx),
                    },
                  })
                }
              />
            </div>
          ))}
        </div>
      </SectionCard>

      {/* 6. Info panels */}
      <SectionCard
        title="Info panels"
        enabled={form.info_panels.enabled}
        onToggle={(enabled) => setForm({ ...form, info_panels: { ...form.info_panels, enabled } })}
      >
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <span className="eyebrow text-brand/55">Panels ({form.info_panels.items.length})</span>
            <button
              type="button"
              onClick={() =>
                setForm({
                  ...form,
                  info_panels: {
                    ...form.info_panels,
                    items: [...form.info_panels.items, { eyebrow: "", title: "", body: "" }],
                  },
                })
              }
              className="inline-flex items-center gap-1 border border-brand/15 px-3 py-1.5 text-xs hover:bg-brand/[0.03]"
            >
              <Plus className="h-3.5 w-3.5" /> Add panel
            </button>
          </div>
          {form.info_panels.items.map((panel, idx) => (
            <div key={idx} className="space-y-3 border border-brand/10 bg-white/60 p-4">
              <div className="grid sm:grid-cols-[1fr_1.5fr_auto] gap-3 items-end">
                <Field
                  label="Eyebrow"
                  value={panel.eyebrow}
                  onChange={(eyebrow) => {
                    const items = [...form.info_panels.items];
                    items[idx] = { ...panel, eyebrow };
                    setForm({ ...form, info_panels: { ...form.info_panels, items } });
                  }}
                />
                <Field
                  label="Title"
                  value={panel.title}
                  onChange={(title) => {
                    const items = [...form.info_panels.items];
                    items[idx] = { ...panel, title };
                    setForm({ ...form, info_panels: { ...form.info_panels, items } });
                  }}
                />
                <RowControls
                  canUp={idx > 0}
                  canDown={idx < form.info_panels.items.length - 1}
                  onUp={() =>
                    setForm({
                      ...form,
                      info_panels: {
                        ...form.info_panels,
                        items: moveItem(form.info_panels.items, idx, idx - 1),
                      },
                    })
                  }
                  onDown={() =>
                    setForm({
                      ...form,
                      info_panels: {
                        ...form.info_panels,
                        items: moveItem(form.info_panels.items, idx, idx + 1),
                      },
                    })
                  }
                  onRemove={() =>
                    setForm({
                      ...form,
                      info_panels: {
                        ...form.info_panels,
                        items: form.info_panels.items.filter((_, i) => i !== idx),
                      },
                    })
                  }
                />
              </div>
              <TextAreaField
                label="Body"
                value={panel.body}
                onChange={(body) => {
                  const items = [...form.info_panels.items];
                  items[idx] = { ...panel, body };
                  setForm({ ...form, info_panels: { ...form.info_panels, items } });
                }}
              />
            </div>
          ))}
        </div>
      </SectionCard>

      <FaqSection form={form} setForm={setForm} />

      <div className="flex justify-end">
        <button
          type="submit"
          disabled={save.isPending}
          className="inline-flex items-center gap-2 bg-brand text-white px-6 py-3 text-sm hover:bg-brand/90 disabled:opacity-60"
        >
          <Save className="h-4 w-4" />
          {save.isPending ? "Saving…" : "Save homepage configuration"}
        </button>
      </div>
    </form>
  );
}

function SectionCard({
  title,
  enabled,
  onToggle,
  children,
}: {
  title: string;
  enabled: boolean;
  onToggle: (v: boolean) => void;
  children: ReactNode;
}) {
  return (
    <section className="border border-brand/10 bg-white/40 p-6 space-y-4">
      <div className="flex items-center justify-between gap-4 pb-3 border-b border-brand/10">
        <h3 className="font-serif text-xl">{title}</h3>
        <label className="inline-flex items-center gap-2 text-xs text-brand/70 cursor-pointer">
          <input
            type="checkbox"
            checked={enabled}
            onChange={(e) => onToggle(e.target.checked)}
            className="h-4 w-4 accent-brand"
          />
          <span>{enabled ? "Visible" : "Hidden"}</span>
        </label>
      </div>
      <div className={enabled ? "space-y-4" : "space-y-4 opacity-50"}>{children}</div>
    </section>
  );
}

function Field({
  label,
  value,
  onChange,
  placeholder,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/55">{label}</span>
      <input
        type="text"
        value={value}
        placeholder={placeholder}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1.5 w-full p-2.5 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
      />
    </label>
  );
}

function TextAreaField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/55">{label}</span>
      <textarea
        rows={2}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        className="mt-1.5 w-full p-2.5 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
      />
    </label>
  );
}

function NumberInput({
  label,
  value,
  onChange,
}: {
  label: string;
  value: number;
  onChange: (v: number) => void;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/55">{label}</span>
      <input
        type="number"
        min={1}
        max={24}
        value={Number.isFinite(value) ? value : 6}
        onChange={(e) => onChange(Math.max(1, Math.min(24, Number(e.target.value) || 6)))}
        className="mt-1.5 w-full p-2.5 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none"
      />
    </label>
  );
}

function RowControls({
  canUp,
  canDown,
  onUp,
  onDown,
  onRemove,
}: {
  canUp: boolean;
  canDown: boolean;
  onUp: () => void;
  onDown: () => void;
  onRemove: () => void;
}) {
  return (
    <div className="flex items-center gap-1 pb-0.5">
      <button
        type="button"
        disabled={!canUp}
        onClick={onUp}
        aria-label="Move up"
        className="p-2 border border-brand/15 text-brand/70 hover:bg-brand/[0.03] disabled:opacity-30"
      >
        <ArrowUp className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        disabled={!canDown}
        onClick={onDown}
        aria-label="Move down"
        className="p-2 border border-brand/15 text-brand/70 hover:bg-brand/[0.03] disabled:opacity-30"
      >
        <ArrowDown className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        onClick={onRemove}
        aria-label="Remove"
        className="p-2 border border-brand/15 text-destructive hover:bg-destructive/5"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}
