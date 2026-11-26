import { useQuery } from "@tanstack/react-query";

import { DEFAULT_HOMEPAGE_CONFIG, getPublicHomepage } from "@/lib/api/homepage";

/**
 * Public site configuration (brand, navigation, footer, page copy) edited in
 * Admin → System → Homepage. Shares the ["homepage", "public"] cache with the
 * homepage and bookshop routes, so it is fetched once per visit and refreshed
 * as soon as an admin saves.
 */
export function usePublicSiteConfig() {
  const { data } = useQuery({
    queryKey: ["homepage", "public"],
    queryFn: getPublicHomepage,
    staleTime: 5 * 60 * 1000,
  });
  return data?.homepage_config ?? DEFAULT_HOMEPAGE_CONFIG;
}
