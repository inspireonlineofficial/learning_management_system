import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import type { ReactNode } from "react";

import {
  safeHref,
  type AuthPageCopy,
  type FaqItem,
  type FooterColumn,
  type HomepageConfig,
  type PageHeroCopy,
  type SiteConfig,
  type SiteLink,
} from "@/lib/api/homepage";

type Props = {
  form: HomepageConfig;
  setForm: (next: HomepageConfig) => void;
};

function move<T>(list: T[], from: number, to: number): T[] {
  if (to < 0 || to >= list.length) return list;
  const next = [...list];
  const [item] = next.splice(from, 1);
  next.splice(to, 0, item);
  return next;
}

const inputClass =
  "mt-1.5 w-full p-2.5 bg-white border border-brand/15 text-sm focus:border-brand/40 focus:outline-none";

function Card({
  title,
  hint,
  right,
  children,
}: {
  title: string;
  hint?: string;
  right?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="border border-brand/10 bg-white/40 p-6 space-y-4">
      <div className="flex items-start justify-between gap-4 pb-3 border-b border-brand/10">
        <div>
          <h3 className="font-serif text-xl">{title}</h3>
          {hint && <p className="text-xs text-brand/55 mt-1">{hint}</p>}
        </div>
        {right}
      </div>
      <div className="space-y-4">{children}</div>
    </section>
  );
}

function Text({
  label,
  value,
  onChange,
  placeholder,
  multiline,
}: {
  label: string;
  value: string;
  onChange: (v: string) => void;
  placeholder?: string;
  multiline?: boolean;
}) {
  return (
    <label className="block">
      <span className="eyebrow text-brand/55">{label}</span>
      {multiline ? (
        <textarea
          rows={3}
          value={value}
          placeholder={placeholder}
          onChange={(e) => onChange(e.target.value)}
          className={inputClass}
        />
      ) : (
        <input
          type="text"
          value={value}
          placeholder={placeholder}
          onChange={(e) => onChange(e.target.value)}
          className={inputClass}
        />
      )}
    </label>
  );
}

function Toggle({
  label,
  checked,
  onChange,
}: {
  label: string;
  checked: boolean;
  onChange: (v: boolean) => void;
}) {
  return (
    <label className="inline-flex items-center gap-2 text-xs text-brand/70 cursor-pointer">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 accent-brand"
      />
      <span>{label}</span>
    </label>
  );
}

function RowButtons({
  idx,
  length,
  onMove,
  onRemove,
}: {
  idx: number;
  length: number;
  onMove: (to: number) => void;
  onRemove: () => void;
}) {
  const btn = "p-2 border border-brand/15 text-brand/70 hover:bg-brand/[0.03] disabled:opacity-30";
  return (
    <div className="flex items-center gap-1 pb-0.5">
      <button
        type="button"
        disabled={idx === 0}
        onClick={() => onMove(idx - 1)}
        aria-label="Move up"
        className={btn}
      >
        <ArrowUp className="h-3.5 w-3.5" />
      </button>
      <button
        type="button"
        disabled={idx === length - 1}
        onClick={() => onMove(idx + 1)}
        aria-label="Move down"
        className={btn}
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

function AddButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="inline-flex items-center gap-1.5 border border-brand/15 px-3 py-2 text-xs font-medium text-brand/75 hover:bg-brand/[0.03]"
    >
      <Plus className="h-3.5 w-3.5" />
      {label}
    </button>
  );
}

export function LinkListEditor({
  links,
  onChange,
  addLabel = "Add link",
}: {
  links: SiteLink[];
  onChange: (next: SiteLink[]) => void;
  addLabel?: string;
}) {
  return (
    <div className="space-y-3">
      {links.map((link, idx) => {
        const invalid = link.href.trim() !== "" && !safeHref(link.href);
        return (
          <div key={idx} className="grid sm:grid-cols-[1fr_1.4fr_auto] gap-3 items-end">
            <Text
              label="Label"
              value={link.label}
              onChange={(label) => {
                const next = [...links];
                next[idx] = { ...link, label };
                onChange(next);
              }}
            />
            <div>
              <Text
                label="Link (/path, /#anchor, https://, mailto:, tel:)"
                value={link.href}
                placeholder="/courses"
                onChange={(href) => {
                  const next = [...links];
                  next[idx] = { ...link, href };
                  onChange(next);
                }}
              />
              {invalid && (
                <p className="mt-1 text-xs text-destructive">
                  Not a supported link — it will be hidden on the site.
                </p>
              )}
            </div>
            <RowButtons
              idx={idx}
              length={links.length}
              onMove={(to) => onChange(move(links, idx, to))}
              onRemove={() => onChange(links.filter((_, i) => i !== idx))}
            />
          </div>
        );
      })}
      <AddButton label={addLabel} onClick={() => onChange([...links, { label: "", href: "" }])} />
    </div>
  );
}

export function SiteIdentitySection({ form, setForm }: Props) {
  const site = form.site;
  const set = (patch: Partial<SiteConfig>) => setForm({ ...form, site: { ...site, ...patch } });
  return (
    <Card
      title="Brand & SEO"
      hint="Shown in the header, footer, browser tab and search-engine results across the whole public site."
    >
      <div className="grid sm:grid-cols-2 gap-4">
        <Text
          label="Brand name"
          value={form.footer.brand_name}
          onChange={(brand_name) => setForm({ ...form, footer: { ...form.footer, brand_name } })}
        />
        <Text
          label="Tagline (footer copyright line)"
          value={form.footer.tagline}
          onChange={(tagline) => setForm({ ...form, footer: { ...form.footer, tagline } })}
        />
        <Text
          label="Logo image URL (blank = default logo)"
          value={site.logo_url}
          placeholder="https://… or /logo.png"
          onChange={(logo_url) => set({ logo_url })}
        />
        <Text
          label="Favicon URL (blank = default)"
          value={site.favicon_url}
          placeholder="https://…/favicon.png"
          onChange={(favicon_url) => set({ favicon_url })}
        />
      </div>
      {site.logo_url && safeHref(site.logo_url) && (
        <div className="inline-flex border border-brand/10 bg-white p-3">
          <img
            src={site.logo_url}
            alt="Logo preview"
            className="max-h-12 max-w-[200px] object-contain"
          />
        </div>
      )}
      <Text
        label="Browser / SEO title"
        value={site.seo_title}
        onChange={(seo_title) => set({ seo_title })}
      />
      <Text
        label="SEO description"
        multiline
        value={site.seo_description}
        onChange={(seo_description) => set({ seo_description })}
      />
    </Card>
  );
}

export function NavigationSection({ form, setForm }: Props) {
  return (
    <Card
      title="Header navigation"
      hint="Links in the top menu (desktop and mobile), in this order."
    >
      <LinkListEditor
        links={form.site.nav_links}
        onChange={(nav_links) => setForm({ ...form, site: { ...form.site, nav_links } })}
      />
    </Card>
  );
}

export function FooterSection({ form, setForm }: Props) {
  const site = form.site;
  const set = (patch: Partial<SiteConfig>) => setForm({ ...form, site: { ...site, ...patch } });
  const setColumns = (footer_columns: FooterColumn[]) => set({ footer_columns });

  return (
    <Card title="Footer & contact" hint="Empty fields are hidden on the site.">
      <Text
        label="About text"
        multiline
        value={site.about_text}
        onChange={(about_text) => set({ about_text })}
      />

      <div className="grid sm:grid-cols-[auto_1fr] gap-4 items-end">
        <Toggle
          label="Show course tracks column"
          checked={site.show_tracks_column}
          onChange={(show_tracks_column) => set({ show_tracks_column })}
        />
        <Text
          label="Tracks column heading"
          value={site.tracks_column_title}
          onChange={(tracks_column_title) => set({ tracks_column_title })}
        />
      </div>

      <div className="space-y-4">
        <p className="eyebrow text-brand/55">Link columns</p>
        {site.footer_columns.map((col, idx) => (
          <div key={idx} className="border border-brand/10 bg-white/60 p-4 space-y-3">
            <div className="grid sm:grid-cols-[1fr_auto] gap-3 items-end">
              <Text
                label="Column heading"
                value={col.title}
                onChange={(title) => {
                  const next = [...site.footer_columns];
                  next[idx] = { ...col, title };
                  setColumns(next);
                }}
              />
              <RowButtons
                idx={idx}
                length={site.footer_columns.length}
                onMove={(to) => setColumns(move(site.footer_columns, idx, to))}
                onRemove={() => setColumns(site.footer_columns.filter((_, i) => i !== idx))}
              />
            </div>
            <LinkListEditor
              links={col.links}
              onChange={(links) => {
                const next = [...site.footer_columns];
                next[idx] = { ...col, links };
                setColumns(next);
              }}
            />
          </div>
        ))}
        <AddButton
          label="Add column"
          onClick={() => setColumns([...site.footer_columns, { title: "", links: [] }])}
        />
      </div>

      <div className="grid sm:grid-cols-2 gap-4 pt-2 border-t border-brand/10">
        <Text
          label="Contact column heading"
          value={site.contact_title}
          onChange={(contact_title) => set({ contact_title })}
        />
        <Text
          label="Support email"
          value={site.support_email}
          onChange={(support_email) => set({ support_email })}
        />
        <Text
          label="Phone number"
          value={site.phone}
          placeholder="+880 1…"
          onChange={(phone) => set({ phone })}
        />
        <Text
          label="Phone label (hours etc.)"
          value={site.phone_label}
          onChange={(phone_label) => set({ phone_label })}
        />
      </div>
      <Text
        label="Address"
        multiline
        value={site.address}
        onChange={(address) => set({ address })}
      />

      <div className="space-y-2">
        <p className="eyebrow text-brand/55">Social links</p>
        <LinkListEditor
          links={site.social_links}
          addLabel="Add social link"
          onChange={(social_links) => set({ social_links })}
        />
      </div>
      <div className="space-y-2">
        <p className="eyebrow text-brand/55">Legal links (Terms, Privacy, Refund policy…)</p>
        <LinkListEditor
          links={site.legal_links}
          addLabel="Add legal link"
          onChange={(legal_links) => set({ legal_links })}
        />
      </div>
      <Text
        label="Bottom note"
        value={site.bottom_note}
        onChange={(bottom_note) => set({ bottom_note })}
      />
    </Card>
  );
}

export function FaqSection({ form, setForm }: Props) {
  const faq = form.faq;
  const setItems = (items: FaqItem[]) => setForm({ ...form, faq: { ...faq, items } });
  return (
    <Card
      title="FAQ"
      right={
        <Toggle
          label={faq.enabled ? "Visible" : "Hidden"}
          checked={faq.enabled}
          onChange={(enabled) => setForm({ ...form, faq: { ...faq, enabled } })}
        />
      }
    >
      <div className={faq.enabled ? "space-y-4" : "space-y-4 opacity-50"}>
        <div className="grid sm:grid-cols-2 gap-4">
          <Text
            label="Eyebrow"
            value={faq.eyebrow}
            onChange={(eyebrow) => setForm({ ...form, faq: { ...faq, eyebrow } })}
          />
          <Text
            label="Heading"
            value={faq.title}
            onChange={(title) => setForm({ ...form, faq: { ...faq, title } })}
          />
        </div>
        {faq.items.map((item, idx) => (
          <div key={idx} className="border border-brand/10 bg-white/60 p-4 space-y-3">
            <div className="grid sm:grid-cols-[1fr_auto] gap-3 items-end">
              <Text
                label={`Question ${idx + 1}`}
                value={item.question}
                onChange={(question) => {
                  const next = [...faq.items];
                  next[idx] = { ...item, question };
                  setItems(next);
                }}
              />
              <RowButtons
                idx={idx}
                length={faq.items.length}
                onMove={(to) => setItems(move(faq.items, idx, to))}
                onRemove={() => setItems(faq.items.filter((_, i) => i !== idx))}
              />
            </div>
            <Text
              label="Answer"
              multiline
              value={item.answer}
              onChange={(answer) => {
                const next = [...faq.items];
                next[idx] = { ...item, answer };
                setItems(next);
              }}
            />
          </div>
        ))}
        <AddButton
          label="Add question"
          onClick={() => setItems([...faq.items, { question: "", answer: "" }])}
        />
      </div>
    </Card>
  );
}

export function PageCopySection({ form, setForm }: Props) {
  const pages = form.pages;
  const setHero = (key: "courses" | "bookshop" | "forum", patch: Partial<PageHeroCopy>) =>
    setForm({ ...form, pages: { ...pages, [key]: { ...pages[key], ...patch } } });
  const setAuth = (key: "login" | "register", patch: Partial<AuthPageCopy>) =>
    setForm({ ...form, pages: { ...pages, [key]: { ...pages[key], ...patch } } });

  const heroes: Array<["courses" | "bookshop" | "forum", string]> = [
    ["courses", "Courses page"],
    ["bookshop", "Bookshop page"],
    ["forum", "Forum page"],
  ];
  const auth: Array<["login" | "register", string]> = [
    ["login", "Sign-in page"],
    ["register", "Registration page"],
  ];

  return (
    <Card
      title="Page headers"
      hint="Top-of-page text for the public catalog, bookshop, forum and sign-in pages."
    >
      {heroes.map(([key, label]) => (
        <div key={key} className="border border-brand/10 bg-white/60 p-4 space-y-3">
          <p className="font-medium text-sm">{label}</p>
          <div className="grid sm:grid-cols-2 gap-3">
            <Text
              label="Eyebrow"
              value={pages[key].eyebrow}
              onChange={(eyebrow) => setHero(key, { eyebrow })}
            />
            <Text
              label="Heading"
              value={pages[key].title}
              onChange={(title) => setHero(key, { title })}
            />
          </div>
          <Text
            label="Intro text"
            multiline
            value={pages[key].body}
            onChange={(body) => setHero(key, { body })}
          />
        </div>
      ))}
      {auth.map(([key, label]) => (
        <div key={key} className="border border-brand/10 bg-white/60 p-4 space-y-3">
          <p className="font-medium text-sm">{label}</p>
          <div className="grid sm:grid-cols-3 gap-3">
            <Text
              label="Eyebrow"
              value={pages[key].eyebrow}
              onChange={(eyebrow) => setAuth(key, { eyebrow })}
            />
            <Text
              label="Heading"
              value={pages[key].heading}
              onChange={(heading) => setAuth(key, { heading })}
            />
            <Text
              label="Subheading"
              value={pages[key].subheading}
              onChange={(subheading) => setAuth(key, { subheading })}
            />
          </div>
        </div>
      ))}
    </Card>
  );
}
