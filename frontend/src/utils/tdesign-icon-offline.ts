/**
 * Offline icon guard.
 *
 * Background (issue #867 / #897):
 *   On mount, tdesign-icons-vue-next's Icon / IconFont components use
 *   `checkScriptAndLoad` / `checkLinkAndLoad` to inject a <script> / <link> into the
 *   document pointing at `https://tdesign.gtimg.com/icon/<version>/fonts/index.(js|css)`.
 *   In an environment without internet access that request fails and no icon renders.
 *
 * Approach: before the Vue app mounts, insert placeholder nodes that match tdesign's own
 *   lookup rules, so the de-duplication check inside `checkScriptAndLoad` /
 *   `checkLinkAndLoad` hits and returns early instead of appending the real CDN node:
 *
 *       `.t-svg-js-stylesheet--unique-class[src="<url>"]`
 *       `.t-iconfont-stylesheet--unique-class[href="<url>"]`
 */

const SVG_SCRIPT_CLASS = "t-svg-js-stylesheet--unique-class";
const ICONFONT_LINK_CLASS = "t-iconfont-stylesheet--unique-class";

const BLOCKED_ICON_VERSIONS = ["0.4.0", "0.4.1", "0.4.2", "0.4.3", "0.4.4"];

const BLOCKED_SCRIPT_URLS = BLOCKED_ICON_VERSIONS.map(
  (version) => `https://tdesign.gtimg.com/icon/${version}/fonts/index.js`,
);

const BLOCKED_LINK_URLS = BLOCKED_ICON_VERSIONS.map(
  (version) => `https://tdesign.gtimg.com/icon/${version}/fonts/index.css`,
);

let installed = false;

export function installTDesignIconOfflineGuard(): void {
  if (installed || typeof document === "undefined") return;
  installed = true;

  const body = document.body;
  if (!body) {
    document.addEventListener(
      "DOMContentLoaded",
      () => installTDesignIconOfflineGuard(),
      { once: true },
    );
    installed = false;
    return;
  }

  BLOCKED_SCRIPT_URLS.forEach((src) => {
    const exists = document.querySelector(
      `script.${SVG_SCRIPT_CLASS}[src="${src}"]`,
    );
    if (exists) return;
    const stub = document.createElement("script");
    stub.setAttribute("class", SVG_SCRIPT_CLASS);
    stub.setAttribute("src", src);
    stub.setAttribute("type", "text/no-load");
    stub.setAttribute("data-enterpriserag-blocked-cdn", "tdesign-icons");
    body.appendChild(stub);
  });

  BLOCKED_LINK_URLS.forEach((href) => {
    const exists = document.querySelector(
      `link.${ICONFONT_LINK_CLASS}[href="${href}"]`,
    );
    if (exists) return;
    const stub = document.createElement("link");
    stub.setAttribute("class", ICONFONT_LINK_CLASS);
    stub.setAttribute("href", href);
    stub.setAttribute("rel", "preload-blocked");
    stub.setAttribute("data-enterpriserag-blocked-cdn", "tdesign-icons");
    document.head.appendChild(stub);
  });
}
