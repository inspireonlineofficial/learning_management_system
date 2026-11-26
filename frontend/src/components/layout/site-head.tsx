import { useEffect } from "react";

import { safeHref } from "@/lib/api/homepage";
import { usePublicSiteConfig } from "@/hooks/use-site-config";

function setMeta(attr: "name" | "property", key: string, content: string) {
  let el = document.head.querySelector<HTMLMetaElement>(`meta[${attr}="${key}"]`);
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute(attr, key);
    document.head.appendChild(el);
  }
  el.setAttribute("content", content);
}

/**
 * Applies the admin-configured SEO title, description and favicon. Pages that
 * set their own document.title afterwards still win.
 */
export function SiteHead() {
  const { site, footer } = usePublicSiteConfig();

  useEffect(() => {
    if (site.seo_title) {
      document.title = site.seo_title;
      setMeta("property", "og:title", site.seo_title);
    }
    if (site.seo_description) {
      setMeta("name", "description", site.seo_description);
      setMeta("property", "og:description", site.seo_description);
    }
    if (footer.brand_name) setMeta("name", "author", footer.brand_name);
    const favicon = site.favicon_url ? safeHref(site.favicon_url) : null;
    if (favicon) {
      let link = document.head.querySelector<HTMLLinkElement>('link[rel="icon"]');
      if (!link) {
        link = document.createElement("link");
        link.rel = "icon";
        document.head.appendChild(link);
      }
      link.href = favicon;
    }
  }, [site.seo_title, site.seo_description, site.favicon_url, footer.brand_name]);

  return null;
}
