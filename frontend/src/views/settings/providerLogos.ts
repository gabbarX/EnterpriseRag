// Logo lookup table for the left side of settings cards.
//
// Assets are split into two kinds:
//   color/ — vendor official multi-colour SVGs, rendered directly through
//            <img> so the brand colours are preserved.
//   mono/  — single-colour SVGs (mostly from simple-icons / vendor marks),
//            tinted with mask-image to the card's own brand colour, so they
//            follow the low-saturation brand shades defined by rules such as
//            .store-card--<id>.
//
// Callers pass (category, id) and get back { mode, url }; undefined is
// returned when nothing matches, and the card then falls back to its
// initial-letter monogram.

const colorModules = import.meta.glob('@/assets/img/providers/color/*/*.svg', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>;

const monoModules = import.meta.glob('@/assets/img/providers/mono/*/*.svg', {
  eager: true,
  query: '?url',
  import: 'default',
}) as Record<string, string>;

export type ProviderCategory = 'vectorstore' | 'storage' | 'websearch' | 'parser' | 'sandbox';

export type LogoMatch = {
  mode: 'color' | 'mono';
  url: string;
};

const buildLookup = (modules: Record<string, string>, segment: string) => {
  const map: Partial<Record<ProviderCategory, Record<string, string>>> = {};
  const re = new RegExp(`providers/${segment}/([^/]+)/([^/]+)\\.svg$`);
  for (const [path, url] of Object.entries(modules)) {
    const match = path.match(re);
    if (!match) continue;
    const [, category, id] = match;
    const bucket = (map[category as ProviderCategory] ||= {});
    bucket[id.toLowerCase()] = url;
  }
  return map;
};

const colorLookup = buildLookup(colorModules, 'color');
const monoLookup = buildLookup(monoModules, 'mono');

export function providerLogo(
  category: ProviderCategory,
  id: string | undefined | null,
): LogoMatch | undefined {
  if (!id) return undefined;
  const key = id.toLowerCase();
  const colorUrl = colorLookup[category]?.[key];
  if (colorUrl) return { mode: 'color', url: colorUrl };
  const monoUrl = monoLookup[category]?.[key];
  if (monoUrl) return { mode: 'mono', url: monoUrl };
  return undefined;
}
