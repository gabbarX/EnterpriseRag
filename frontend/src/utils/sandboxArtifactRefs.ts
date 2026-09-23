/**
 * Wires references to sandbox-generated files in an answer body into the artifact
 * download path.
 *
 * The model references files it generated in the sandbox with Markdown image syntax
 * (the prompt requires `![caption](sandbox:filename)`). Before persisting, the server
 * rewrites that into the file's stable handle `resource://<handle>` - the same
 * reference form used by knowledge-base images and chat attachments. Both spellings
 * point at the same `Message.Artifacts`:
 *
 *   - `resource://<handle>` - the authoritative persisted form, which is what a
 *     historical session reads back;
 *   - `sandbox:<filename>`  - the model's raw spelling, used while this turn is still
 *     streaming and has not been rewritten yet.
 *
 * Because the handle form is identical to that of knowledge-base images, a single
 * answer routinely mixes retrieved knowledge-base images with skill-generated ones.
 * So when a handle does not match this message's artifact list we MUST return null and
 * fall back to the default protected-image rendering, rather than showing "file
 * unavailable".
 *
 * Image artifacts render inline (fetched with auth, then swapped for a blob URL); every
 * other type (HTML charts, CSV, documents) renders as a card that opens in the artifacts
 * tab of the sandbox side panel - inlining a 1MB self-contained HTML iframe in the answer
 * body would be both slow and unsafe.
 *
 * Artifacts the user deleted stay in the list as tombstones (with deleted_at) and render
 * as a greyed-out, non-clickable card. Callers must NOT filter tombstones out before
 * passing the list in: without them the handle no longer matches this answer's artifacts,
 * the reference is treated as a knowledge-base image, and the user ends up with a broken
 * image that can never load.
 */
import { escapeHTML } from './security.ts';
import { renderArtifactFileIcon } from './artifactFileIcon';

export interface ArtifactRefMeta {
  index: number;
  file_name: string;
  file_type?: string;
  handle?: string;
  url?: string;
  /**
   * When the user deleted this file. Tombstone entries MUST stay in the list handed to
   * the renderer: the index is the download address, so dropping one shifts every later
   * file, and a handle only identifies the artifact as belonging to this answer while it
   * still matches - otherwise it falls through to protected-image rendering and shows a
   * broken image.
   */
  deleted_at?: string | null;
}

export interface ArtifactRefContext {
  sessionId: string;
  messageId: string;
}

export interface ArtifactRefLabels {
  previewHint: string;
  missingHint: string;
  deletedHint: string;
}

const RESOURCE_HANDLE_RE = /^resource:\/\/([A-Za-z0-9_-]{22})$/;
const SANDBOX_NAME_RE = /^sandbox:(?:\/\/)?(.+)$/i;
const IMAGE_EXTENSIONS = new Set(['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'avif']);
const TRANSPARENT_PIXEL =
  'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==';

type ArtifactRef = { kind: 'handle'; handle: string } | { kind: 'name'; name: string };

function parseArtifactRef(href: string): ArtifactRef | null {
  const trimmed = (href || '').trim();
  if (!trimmed) return null;

  const handleMatch = trimmed.match(RESOURCE_HANDLE_RE);
  if (handleMatch) {
    return { kind: 'handle', handle: handleMatch[1] };
  }

  const nameMatch = trimmed.match(SANDBOX_NAME_RE);
  if (!nameMatch) return null;
  let name = nameMatch[1].trim();
  try {
    name = decodeURIComponent(name);
  } catch {
  }
  name = name.split('/').pop() || '';
  return name ? { kind: 'name', name } : null;
}

/**
 * Whether this link target could be a sandbox artifact reference.
 *
 * The handle form is identical to a knowledge-base image, so true here only means "worth
 * trying the artifact resolver once"; it does not mean this message really has the file.
 */
export function isArtifactRefHref(href: string): boolean {
  return parseArtifactRef(href) !== null;
}

function artifactHandle(artifact: ArtifactRefMeta): string {
  const raw = (artifact.handle || artifact.url || '').trim();
  return raw.match(RESOURCE_HANDLE_RE)?.[1] || '';
}

const CODE_SPAN_OR_FENCE_RE = /(```[\s\S]*?```|~~~[\s\S]*?~~~|`[^`\n]*`)/g;

/**
 * Scans a Markdown link destination and returns the raw text inside the parentheses plus
 * the index of the closing paren.
 *
 * Bracket matching rather than a regex, because skill-generated filenames often contain
 * parentheses and spaces (e.g. `Holdings (00700) volume_838ccc.html`). marked truncates
 * the destination at the first space, so the reference would never match an artifact;
 * scanning by depth finds the paren that actually closes the link.
 */
function scanLinkDestination(text: string, openIndex: number): { inner: string; end: number } | null {
  let depth = 1;
  for (let i = openIndex + 1; i < text.length; i += 1) {
    const ch = text[i];
    if (ch === '\n') return null;
    if (ch === '(') depth += 1;
    else if (ch === ')') {
      depth -= 1;
      if (depth === 0) return { inner: text.slice(openIndex + 1, i), end: i };
    }
  }
  return null;
}

function splitDestinationTitle(inner: string): { destination: string; title: string } {
  const match = inner.match(/^([\s\S]*?)(\s+(?:"[^"]*"|'[^']*'))$/);
  if (!match) return { destination: inner.trim(), title: '' };
  return { destination: match[1].trim(), title: match[2] };
}

function hasLinkLabelBefore(text: string, closeBracketIndex: number): boolean {
  for (let i = closeBracketIndex - 1; i >= 0; i -= 1) {
    const ch = text[i];
    if (ch === '\n') return false;
    if (ch === '[') return true;
  }
  return false;
}

function normalizeSegment(segment: string): string {
  if (!segment.includes('](')) return segment;

  let out = '';
  let cursor = 0;
  while (cursor < segment.length) {
    const relative = segment.slice(cursor).indexOf('](');
    if (relative < 0) break;
    const closeBracket = cursor + relative;
    const open = closeBracket + 1;

    if (!hasLinkLabelBefore(segment, closeBracket)) {
      out += segment.slice(cursor, open + 1);
      cursor = open + 1;
      continue;
    }
    const scanned = scanLinkDestination(segment, open);
    if (!scanned) {
      out += segment.slice(cursor, open + 1);
      cursor = open + 1;
      continue;
    }

    const { destination, title } = splitDestinationTitle(scanned.inner);
    const ref = parseArtifactRef(destination);
    if (!ref || ref.kind !== 'name') {
      out += segment.slice(cursor, scanned.end + 1);
      cursor = scanned.end + 1;
      continue;
    }

    // Percent-encode so marked sees a single whitespace-free token;
    // parseArtifactRef decodes it again on the way out.
    out += `${segment.slice(cursor, open + 1)}sandbox:${encodeURIComponent(ref.name)}${title})`;
    cursor = scanned.end + 1;
  }
  return out + segment.slice(cursor);
}

/**
 * Before marked parses, collapse the target of a `sandbox:` reference into a single
 * space-free token.
 *
 * Without this, marked splits a filename containing spaces at the first space: the target
 * keeps only the first half and the rest leaks into the body text - exactly the "card name
 * truncated, tail turns into bare text" symptom. Examples inside code blocks are left alone.
 */
export function normalizeSandboxArtifactRefs(markdown: string): string {
  if (!markdown || !markdown.includes('](')) return markdown;
  if (!/\]\(\s*sandbox:/i.test(markdown)) return markdown;

  const parts = markdown.split(CODE_SPAN_OR_FENCE_RE);
  for (let i = 0; i < parts.length; i += 2) {
    parts[i] = normalizeSegment(parts[i]);
  }
  return parts.join('');
}

export function resolveArtifactRef(
  href: string,
  artifacts: ArtifactRefMeta[] | undefined | null,
): ArtifactRefMeta | null {
  const ref = parseArtifactRef(href);
  if (!ref || !artifacts?.length) return null;

  if (ref.kind === 'handle') {
    return artifacts.find((item) => artifactHandle(item) === ref.handle) || null;
  }
  return artifacts.find((item) => (item.file_name || '').trim() === ref.name) || null;
}

function fileExtension(fileName: string): string {
  const base = (fileName || '').trim().toLowerCase();
  const dot = base.lastIndexOf('.');
  return dot > 0 ? base.slice(dot + 1) : '';
}

/**
 * Whether to render inline as an image. SVG is deliberately excluded: it is executable
 * content, so it takes the card + sandboxed preview path.
 */
function rendersAsImage(artifact: ArtifactRefMeta): boolean {
  const ext = fileExtension(artifact.file_name);
  if (ext) return IMAGE_EXTENSIONS.has(ext);
  const type = (artifact.file_type || '').toLowerCase();
  return type.startsWith('image/') && !type.includes('svg');
}

type ArtifactBlobState = { blobByKey: Map<string, string>; inflight: Map<string, Promise<string | null>> };

const artifactBlobState: ArtifactBlobState = (() => {
  const fresh = (): ArtifactBlobState => ({ blobByKey: new Map(), inflight: new Map() });
  if (typeof window === 'undefined') return fresh();
  const scope = window as typeof window & { __enterpriseragArtifactBlobCacheV1__?: ArtifactBlobState };
  scope.__enterpriseragArtifactBlobCacheV1__ ||= fresh();
  return scope.__enterpriseragArtifactBlobCacheV1__;
})();

function blobCacheKey(ctx: ArtifactRefContext, index: number): string {
  return `${ctx.sessionId}\u0000${ctx.messageId}\u0000${index}`;
}

const STREAMING_PLACEHOLDER =
  '<span class="streaming-image-loading"><span class="streaming-image-loading__skeleton"></span></span>';

/**
 * The three card states:
 *   - `ready`   - normal, click to open the preview;
 *   - `pending` - the turn finished but the reference matches no artifact (the model
 *                 referenced a file that does not exist);
 *   - `deleted` - the file really was generated, but the user deleted it and the bytes
 *                 are gone.
 * Neither of the last two is clickable, but they must stay distinguishable: one never
 * existed, the other you deleted yourself.
 */
type ArtifactCardVariant = 'ready' | 'pending' | 'deleted';

function renderCard(
  fileName: string,
  hint: string,
  index: number | null,
  variant: ArtifactCardVariant,
): string {
  const safeName = escapeHTML(fileName);
  const safeHint = escapeHTML(hint);
  const interactive = variant === 'ready' && index !== null
    ? ` data-artifact-index="${index}" role="button" tabindex="0"`
    : '';
  const state = variant === 'ready' ? '' : ` artifact-ref-card--${variant}`;
  return (
    `<span class="artifact-ref-card${state}"${interactive} title="${safeName}">`
    + `<span class="artifact-ref-card__icon" aria-hidden="true">${renderArtifactFileIcon(fileName)}</span>`
    + '<span class="artifact-ref-card__text">'
    + `<span class="artifact-ref-card__name">${safeName}</span>`
    + `<span class="artifact-ref-card__hint">${safeHint}</span>`
    + '</span></span>'
  );
}

function renderImage(
  artifact: ArtifactRefMeta,
  alt: string,
  ctx: ArtifactRefContext | null,
): string {
  const safeAlt = escapeHTML(alt || artifact.file_name || '');
  const cached = ctx ? artifactBlobState.blobByKey.get(blobCacheKey(ctx, artifact.index)) : undefined;
  const src = cached || TRANSPARENT_PIXEL;
  const loading = cached ? '' : ' data-img-loading="1"';
  return (
    `<img class="markdown-image artifact-ref-image" src="${src}" alt="${safeAlt}"`
    + ` data-artifact-index="${artifact.index}"${loading}>`
  );
}

/**
 * Renders one Markdown image/link target.
 *
 * Returning null means this is not a sandbox artifact reference and the caller should fall
 * back to default rendering (plain images, `resource://` protected images and external
 * links are all unaffected). Returning an empty string means the target was empty and the
 * caller should not draw a broken image.
 */
export function renderArtifactReference(args: {
  href: string;
  alt?: string;
  artifacts?: ArtifactRefMeta[] | null;
  labels: ArtifactRefLabels;
  context?: ArtifactRefContext | null;
  streaming?: boolean;
}): string | null {
  const href = (args.href || '').trim();
  if (!href) return '';
  const ref = parseArtifactRef(href);
  if (!ref) return null;

  const artifact = resolveArtifactRef(href, args.artifacts);
  if (!artifact) {
    if (ref.kind === 'handle') return null;
    // The artifact list only arrives with the complete event at the end of the turn, so
    // during streaming it can never resolve. Show a skeleton rather than a card: that
    // avoids flashing half a filename, and avoids leaving a "generating" state inside an
    // answer that has in fact already finished.
    if (args.streaming) return STREAMING_PLACEHOLDER;
    const fallbackName = ref.name || (args.alt || '').trim();
    if (!fallbackName) return '';
    return renderCard(fallbackName, args.labels.missingHint, null, 'pending');
  }

  if (artifact.deleted_at) {
    const name = artifact.file_name || (args.alt || '').trim();
    if (!name) return '';
    return renderCard(name, args.labels.deletedHint, null, 'deleted');
  }

  if (rendersAsImage(artifact)) {
    return renderImage(artifact, args.alt || '', args.context ?? null);
  }
  return renderCard(artifact.file_name, args.labels.previewHint, artifact.index, 'ready');
}

async function loadArtifactBlobURL(ctx: ArtifactRefContext, index: number): Promise<string | null> {
  const key = blobCacheKey(ctx, index);
  const cached = artifactBlobState.blobByKey.get(key);
  if (cached) return cached;

  let task = artifactBlobState.inflight.get(key);
  if (!task) {
    task = (async () => {
      try {
        // Imported lazily so parsing/rendering stays free of the axios
        // transport — those parts run in plain Node during unit tests.
        const { downloadArtifact } = await import('@/api/chat');
        const blob = await downloadArtifact(ctx.sessionId, ctx.messageId, index);
        const blobURL = URL.createObjectURL(blob);
        artifactBlobState.blobByKey.set(key, blobURL);
        return blobURL;
      } catch (error) {
        console.warn('[sandboxArtifactRefs] artifact image load failed:', error);
        return null;
      } finally {
        artifactBlobState.inflight.delete(key);
      }
    })();
    artifactBlobState.inflight.set(key, task);
  }
  return task;
}

/**
 * Swaps artifact images in the body for auth-fetched blobs.
 *
 * Idempotent in the same way as hydrateProtectedFileImages: elements already swapped carry
 * an authHydrated marker, and concurrent requests for the same file share one Promise.
 */
export async function hydrateArtifactImages(
  root: ParentNode | null | undefined,
  ctx: ArtifactRefContext | null | undefined,
): Promise<void> {
  if (!root || !ctx?.sessionId || !ctx?.messageId) return;

  const images = Array.from(
    root.querySelectorAll<HTMLImageElement>('img.artifact-ref-image[data-artifact-index]'),
  ).filter((img) => img.dataset.authHydrated !== '1');
  if (!images.length) return;

  await Promise.all(images.map(async (img) => {
    const index = Number(img.getAttribute('data-artifact-index'));
    if (!Number.isInteger(index) || index < 0) return;
    img.dataset.authHydrated = '1';

    const blobURL = await loadArtifactBlobURL(ctx, index);
    if (!blobURL) {
      img.dataset.authHydrated = '0';
      return;
    }
    img.src = blobURL;
    img.removeAttribute('data-img-loading');
  }));
}

/**
 * Pulls the index of the activated artifact card out of a click/keyboard event; null means
 * the event had nothing to do with a card.
 */
export function artifactIndexFromEventTarget(target: EventTarget | null): number | null {
  if (!(target instanceof Element)) return null;
  const card = target.closest('.artifact-ref-card[data-artifact-index]');
  if (!card) return null;
  const index = Number(card.getAttribute('data-artifact-index'));
  return Number.isInteger(index) && index >= 0 ? index : null;
}
