import assert from 'node:assert/strict'
import test from 'node:test'

import {
  applyStreamingTailFade,
  closeDanglingStreamingEmphasis,
  createChatMarkdownRenderer,
  markStandaloneStrongParagraphs,
  normalizeFullwidthMarkdownImageParentheses,
  normalizeLegacyImageContextMarkup,
  preprocessMathDelimiters,
  renderChatMarkdown,
  repairFlankingEmphasis,
  replaceIncompleteImageWithPlaceholder,
  stripTrailingStreamingHorizontalRule,
  stripTrailingStreamingListMarker,
} from './chatMarkdownRenderer.ts'
import {
  collapseStandaloneCitationParagraphs,
  joinCitationTagsToPreviousLine,
  resolveCitationChunkId,
  stripIncompleteCitationTag,
} from './citationMarkdown.ts'

const SAMPLE_DOC = 'example-report.docx'
const SAMPLE_CHUNK_A = '00000001-0000-4000-8000-000000000001'
const SAMPLE_CHUNK_B = '00000002-0000-4000-8000-000000000002'
const SAMPLE_CHUNK_C = '00000003-0000-4000-8000-000000000003'
const SAMPLE_CHUNK_PRESERVE = 'aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee'

/** Remove the streaming tail-fade wrapper so structural assertions stay focused. */
function stripFadeTail(html: string): string {
  return html.replace(/<span class="stream-fade-tail">([\s\S]*?)<\/span>/g, '$1')
}

test('preprocessMathDelimiters converts escaped math delimiters for marked-katex', () => {
  assert.equal(
    preprocessMathDelimiters('inline \\(a+b\\) and block \\[x^2\\]'),
    'inline $a+b$ and block $$x^2$$',
  )
})

test('replaceIncompleteImageWithPlaceholder hides an unfinished streaming image', () => {
  assert.equal(
    replaceIncompleteImageWithPlaceholder('before ![chart](local://bucket/path'),
    'before <span class="streaming-image-loading"><span class="streaming-image-loading__skeleton"></span></span>',
  )
})

test('normalizeFullwidthMarkdownImageParentheses repairs localized image delimiters', () => {
  const ref = 'resource://yB7V7wE1gls7h9WonCDq5Q'
  assert.equal(
    normalizeFullwidthMarkdownImageParentheses(`![]（${ref}）`),
    `![](${ref})`,
  )
  assert.equal(
    normalizeFullwidthMarkdownImageParentheses(`![Flow chart]（${ref}）`),
    `![Flow chart](${ref})`,
  )
  assert.equal(
    normalizeFullwidthMarkdownImageParentheses(`Ordinary prose（a note）and ![Example]（unknown://value）`),
    `Ordinary prose（a note）and ![Example]（unknown://value）`,
  )
})

test('normalizeFullwidthMarkdownImageParentheses preserves literal examples in code', () => {
  const code = '`![]（resource://example）`\n```md\n![]（resource://example）\n```'
  assert.equal(normalizeFullwidthMarkdownImageParentheses(code), code)
})

test('renderChatMarkdown safely renders an image with fullwidth parentheses', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, text }) => `<img src="${href}" alt="${text}">`,
    isValidImageUrl: (href) => href.startsWith('resource://'),
  })
  const html = renderChatMarkdown('![]（resource://yB7V7wE1gls7h9WonCDq5Q）', {
    renderer,
    escapeMarkdown: (text) => text,
    sanitizeHtml: (value) => value,
    streaming: false,
  })

  assert.match(html, /<img src="resource:\/\/yB7V7wE1gls7h9WonCDq5Q" alt="">/)
  assert.doesNotMatch(html, /（|）/)
})

test('renderChatMarkdown skips an image with an empty destination', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, text }) => `<img src="${href}" alt="${text}">`,
    invalidImageHtml: () => '<p>invalid</p>',
    isValidImageUrl: (href) => Boolean(href),
  })
  const html = renderChatMarkdown('![Sample file]()', {
    renderer,
    escapeMarkdown: (text) => text,
    sanitizeHtml: (value) => value,
    streaming: false,
  })

  assert.doesNotMatch(html, /<img/)
  assert.doesNotMatch(html, /invalid/)
})

test('renderChatMarkdown hides an unfinished fullwidth-parenthesis image while streaming', () => {
  const renderer = createChatMarkdownRenderer()
  const html = renderChatMarkdown('before ![Flow chart]（resource://yB7V7wE1gls7h9WonCDq5Q', {
    renderer,
    escapeMarkdown: (text) => text,
    sanitizeHtml: (value) => value,
    streaming: true,
  })

  assert.match(html, /streaming-image-loading/)
  assert.doesNotMatch(html, /resource:\/\/|（/)
})

test('normalizeLegacyImageContextMarkup converts copied image XML to Markdown', () => {
  const input = [
    'before',
    '<image url="resource://AbCdEfGhIjKlMnOpQrStUv">',
    '<image_caption>Target speaker extraction flow chart [test]</image_caption>',
    '<image_ocr>Target speaker extraction</image_ocr>',
    '</image>',
    'after',
  ].join('\n')

  const output = normalizeLegacyImageContextMarkup(input)
  assert.ok(
    output.includes('![Target speaker extraction flow chart \\[test\\]](resource://AbCdEfGhIjKlMnOpQrStUv)'),
  )
  assert.doesNotMatch(output, /<image|image_caption|image_ocr/)
  assert.match(output, /before[\s\S]*after/)
})

test('normalizeLegacyImageContextMarkup keeps original Markdown when present', () => {
  const input = [
    '<images>',
    '<image url="resource://AbCdEfGhIjKlMnOpQrStUv">',
    '<image_original>![Original](resource://AbCdEfGhIjKlMnOpQrStUv)</image_original>',
    '<image_caption>description</image_caption>',
    '</image>',
    '</images>',
  ].join('\n')

  assert.equal(
    normalizeLegacyImageContextMarkup(input).trim(),
    '![Original](resource://AbCdEfGhIjKlMnOpQrStUv)',
  )
})

test('normalizeLegacyImageContextMarkup hides an unfinished XML block while streaming', () => {
  const prefix = 'Main test-phase flow\n\n'
  for (const partial of [
    '<ima',
    '<image url="resource://AbCdEfGhIjKlMnOpQrStUv">',
    '<image url="resource://AbCdEfGhIjKlMnOpQrStUv">\n<image_caption>Flow chart',
  ]) {
    const output = normalizeLegacyImageContextMarkup(prefix + partial, true)
    assert.equal(output, prefix + '<span class="streaming-image-loading"><span class="streaming-image-loading__skeleton"></span></span>')
    assert.doesNotMatch(output, /resource:\/\/|image_caption|<ima/)
  }
})

test('normalizeLegacyImageContextMarkup preserves literal image XML in code', () => {
  const code = '```xml\n<image url="resource://example">\n</image>\n```'
  assert.equal(normalizeLegacyImageContextMarkup(code, true), code)
  assert.equal(normalizeLegacyImageContextMarkup(code, false), code)
  assert.equal(normalizeLegacyImageContextMarkup('ordinary <input', true), 'ordinary <input')
})

test('renderChatMarkdown renders leaked legacy image XML through the safe image renderer', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, text }) => `<img src="${href}" alt="${text}">`,
    isValidImageUrl: (href) => href.startsWith('resource://'),
  })
  const html = renderChatMarkdown(
    '<image url="resource://AbCdEfGhIjKlMnOpQrStUv"><image_caption>Flow chart</image_caption></image>',
    {
      renderer,
      escapeMarkdown: (text) => text,
      sanitizeHtml: (value) => value,
      streaming: false,
    },
  )

  assert.match(html, /<img src="resource:\/\/AbCdEfGhIjKlMnOpQrStUv" alt="Flow chart">/)
  assert.doesNotMatch(html, /image_caption|&lt;image/)
})

test('stripIncompleteCitationTag hides only an unfinished streaming citation tail', () => {
  const prefix = 'Source '
  const complete = '<kb doc="2.jpg" chunk_id="3c67efd5-f2ff-4e26-9032-9e44e6861178" />'

  for (const partial of ['<', '<k', '<kb', '<kb ', '<kb doc="2.jpg"', '<w', '<we', '<web url="https://example.com"']) {
    assert.equal(stripIncompleteCitationTag(prefix + partial), prefix)
  }

  assert.equal(stripIncompleteCitationTag(prefix + complete), prefix + complete)
  assert.equal(stripIncompleteCitationTag('Value < 5'), 'Value < 5')
})

test('stripTrailingStreamingHorizontalRule hides an ambiguous trailing rule only mid-stream', () => {
  for (const rule of ['---', '* * *', '___']) {
    assert.equal(stripTrailingStreamingHorizontalRule(`- item\n\n${rule}`), '- item\n\n')
  }
  assert.equal(stripTrailingStreamingHorizontalRule('- item\n\n---\nnext'), '- item\n\n---\nnext')

  const renderer = createChatMarkdownRenderer()
  const options = {
    renderer,
    escapeMarkdown: (text: string) => text,
    sanitizeHtml: (html: string) => html,
  }
  const source = '- item\n\n---'
  assert.doesNotMatch(renderChatMarkdown(source, { ...options, streaming: true }), /<hr>/)
  assert.match(renderChatMarkdown(source, { ...options, streaming: false }), /<hr>/)
})

test('closeDanglingStreamingEmphasis closes unfinished inline markers without false positives', () => {
  assert.equal(closeDanglingStreamingEmphasis('**प्लेटफ़ॉर्म पता: example.com'), '**प्लेटफ़ॉर्म पता: example.com**')
  assert.equal(closeDanglingStreamingEmphasis('पूर्व पाठ *तिरछा'), 'पूर्व पाठ *तिरछा*')
  assert.equal(closeDanglingStreamingEmphasis('पूर्व पाठ ~~हटाया'), 'पूर्व पाठ ~~हटाया~~')
  assert.equal(closeDanglingStreamingEmphasis('***मोटा और तिरछा'), '***मोटा और तिरछा***')
  assert.equal(closeDanglingStreamingEmphasis('चलाएँ `npm run'), 'चलाएँ `npm run`')

  // A trailing marker run with no content after it is ambiguous (e.g. the start
  // of the next `**` in a bold list) and is hidden until content arrives, rather
  // than rendered as a literal `*`/`**` that flickers a frame later.
  assert.equal(closeDanglingStreamingEmphasis('4. **अंधेरी प्रयोगशाला**\n5. *'), '4. **अंधेरी प्रयोगशाला**\n5. ')
  assert.equal(closeDanglingStreamingEmphasis('5. **'), '5. ')
  assert.equal(closeDanglingStreamingEmphasis('3. **तरंग तैराकी*'), '3. **तरंग तैराकी**')

  // No dangling markers / structural markers must be left untouched.
  assert.equal(closeDanglingStreamingEmphasis('मुख्य पाठ **मोटा** अंत'), 'मुख्य पाठ **मोटा** अंत')
  assert.equal(closeDanglingStreamingEmphasis('* सूची मद एक'), '* सूची मद एक')
  assert.equal(closeDanglingStreamingEmphasis('सामान्य पाठ बिना किसी चिह्न के'), 'सामान्य पाठ बिना किसी चिह्न के')
  // Markers inside an open fenced code block are literal, not emphasis.
  assert.equal(
    closeDanglingStreamingEmphasis('```js\nconst x = **y'),
    '```js\nconst x = **y',
  )
})

test('renderChatMarkdown renders an unfinished bold line as bold immediately while streaming', () => {
  const renderer = createChatMarkdownRenderer()
  const options = {
    renderer,
    escapeMarkdown: (text: string) => text,
    sanitizeHtml: (html: string) => html,
  }
  const partial = '**प्लेटफ़ॉर्म पता: rag.example.co.in'
  // Streaming: optimistically bold so no raw `**` and no late layout jump.
  assert.match(
    stripFadeTail(renderChatMarkdown(partial, { ...options, streaming: true })),
    /<p class="md-strong-title"><strong>प्लेटफ़ॉर्म पता: rag\.example\.co\.in<\/strong><\/p>/,
  )
  // Completed: a genuinely unterminated marker stays literal (we never invent
  // content for the final, authoritative render).
  assert.match(
    renderChatMarkdown(partial, { ...options, streaming: false }),
    /<p>\*\*प्लेटफ़ॉर्म पता/,
  )
})

test('repairFlankingEmphasis bolds punctuation-adjacent emphasis CommonMark would drop', () => {
  // Closing delimiter sits between punctuation and a letter -> CommonMark rejects
  // it; we convert exactly that blocked pattern to explicit HTML.
  assert.equal(repairFlankingEmphasis('**XBRL (भाषा)**एक मानक'), '<strong>XBRL (भाषा)</strong>एक मानक')
  assert.equal(repairFlankingEmphasis('**a)**b'), '<strong>a)</strong>b')
  assert.equal(repairFlankingEmphasis('~~हटाया)~~पाठ'), '<del>हटाया)</del>पाठ')
  assert.equal(repairFlankingEmphasis('*तिरछा)*पाठ'), '<em>तिरछा)</em>पाठ')

  // Opening delimiter sits between a letter/number and punctuation -> CommonMark
  // refuses to open emphasis even though the closer is valid; we bold it too.
  assert.equal(
    repairFlankingEmphasis('ज्ञानकोश**"xxx" पुस्तिका**संकलन'),
    'ज्ञानकोश<strong>"xxx" पुस्तिका</strong>संकलन',
  )
  assert.equal(repairFlankingEmphasis('पुस्तक**"नाम"**'), 'पुस्तक<strong>"नाम"</strong>')
  assert.equal(repairFlankingEmphasis('देखकर~~"हटाया" पाठ~~बाद'), 'देखकर<del>"हटाया" पाठ</del>बाद')

  // Cases marked already handles, literal markers, and code must be untouched.
  assert.equal(repairFlankingEmphasis('**पाठ**बाद'), '**पाठ**बाद')
  assert.equal(repairFlankingEmphasis('**पाठ)** बाद'), '**पाठ)** बाद')
  assert.equal(repairFlankingEmphasis('मुख्य पाठ **मोटा** अंत'), 'मुख्य पाठ **मोटा** अंत')
  assert.equal(repairFlankingEmphasis('`**a)**b`'), '`**a)**b`')
  assert.equal(repairFlankingEmphasis('2 ** 3 ** 4'), '2 ** 3 ** 4')
  assert.equal(repairFlankingEmphasis('कीमत 3 * 5 * 7 रुपये'), 'कीमत 3 * 5 * 7 रुपये')
  // Exponent / glob markers stay literal: the opener is followed by a number or
  // path content, not an emphasis-opening punctuation run.
  assert.equal(repairFlankingEmphasis('x**2 + y**2'), 'x**2 + y**2')
  assert.equal(repairFlankingEmphasis('2**3**4'), '2**3**4')
})

test('renderChatMarkdown bolds punctuation-adjacent emphasis both mid-stream and when complete', () => {
  const renderer = createChatMarkdownRenderer()
  const options = {
    renderer,
    escapeMarkdown: (text: string) => text,
    sanitizeHtml: (html: string) => html,
  }
  const text = '**XBRL (भाषा)**एक मानक है'
  for (const streaming of [true, false]) {
    assert.match(
      stripFadeTail(renderChatMarkdown(text, { ...options, streaming })),
      /<strong>XBRL \(भाषा\)<\/strong>एक मानक है/,
    )
  }
})

test('stripTrailingStreamingListMarker hides a content-less trailing list/underline marker', () => {
  assert.equal(stripTrailingStreamingListMarker('1. **AAAAA**\n   - '), '1. **AAAAA**\n')
  assert.equal(stripTrailingStreamingListMarker('पाठ\n1. '), 'पाठ\n')
  assert.equal(stripTrailingStreamingListMarker('शीर्षक\n=='), 'शीर्षक\n')
  // A lone trailing number is the start of an ordered marker (no `.` yet).
  assert.equal(stripTrailingStreamingListMarker('पाठ\n1'), 'पाठ\n')
  // A marker with content after it is a real list item and must stay.
  assert.equal(stripTrailingStreamingListMarker('- मद'), '- मद')
  assert.equal(stripTrailingStreamingListMarker('1. सामग्री'), '1. सामग्री')
  // An in-sentence number must not be touched.
  assert.equal(stripTrailingStreamingListMarker('मान है 1'), 'मान है 1')
})

test('renderChatMarkdown does not flash a setext heading when a nested bullet dash streams in', () => {
  const renderer = createChatMarkdownRenderer()
  const options = {
    renderer,
    escapeMarkdown: (text: string) => text,
    sanitizeHtml: (html: string) => html,
    streaming: true,
  }
  // A must stay a plain bold item through the whole lead-up to B: the bare dash
  // must not flash <h2>, and the next bullet's `*`/`**` (before its text) must
  // not flash an empty nested <li> under A.
  for (const partial of ['1. **AAAAA**\n   - ', '1. **AAAAA**\n   - *', '1. **AAAAA**\n   - **']) {
    const html = stripFadeTail(renderChatMarkdown(partial, options))
    assert.doesNotMatch(html, /<h2/)
    assert.doesNotMatch(html, /<ul>/)
    assert.match(html, /<ol>\s*<li><strong>AAAAA<\/strong><\/li>\s*<\/ol>/)
  }
})

test('renderChatMarkdown streams a bold ordered list without literal-marker flicker', () => {
  const renderer = createChatMarkdownRenderer()
  const options = {
    renderer,
    escapeMarkdown: (text: string) => text,
    sanitizeHtml: (html: string) => html,
    streaming: true,
  }
  // Next item's marker has arrived but its content/closing ** has not.
  const html = stripFadeTail(renderChatMarkdown('1. **तेज़ तैराकी**\n2. *', options))
  assert.doesNotMatch(html, /<li>\*+<\/li>/)
  assert.match(html, /<li><strong>तेज़ तैराकी<\/strong><\/li>/)
})

test('markStandaloneStrongParagraphs tags only paragraphs that are entirely one bold run', () => {
  // A model emitting **Section heading** as a pseudo-heading: should be tagged.
  assert.equal(
    markStandaloneStrongParagraphs('<p><strong>अनुभाग शीर्षक:</strong></p>'),
    '<p class="md-strong-title"><strong>अनुभाग शीर्षक:</strong></p>',
  )

  // Mid-stream body paragraph with one completed bold run plus body text must
  // NOT be tagged (this is the streaming spacing-jump regression).
  for (const html of [
    '<p>मुख्य पाठ <strong>मोटा भाग</strong> और **</p>',
    '<p>मुख्य पाठ <strong>A</strong> और <strong>B</strong> अंत</p>',
    '<p><strong>मोटा</strong> आगे का पाठ</p>',
  ]) {
    assert.equal(markStandaloneStrongParagraphs(html), html)
  }

  // A bold list item (loose list wraps content in <p>) is list text, not a
  // subtitle: it must keep the list's own margins, not gain a 1.25em top margin.
  assert.equal(
    markStandaloneStrongParagraphs('<ol><li><p><strong>AAA</strong></p></li></ol>'),
    '<ol><li><p><strong>AAA</strong></p></li></ol>',
  )
})

test('renderChatMarkdown does not give a text-heavy paragraph the strong-title margin mid-stream', () => {
  const renderer = createChatMarkdownRenderer()
  const options = {
    renderer,
    escapeMarkdown: (text: string) => text,
    sanitizeHtml: (html: string) => html,
    streaming: true,
  }
  const head = '#### **सारांश और सुझाव**\n\n'
  // One bold run closed inside an otherwise text-heavy body paragraph.
  const midStream = head + 'कुल मिलाकर, यह **"कमी पूरी करना" (मुख्य कलपुर्जे)** और **'
  const html = renderChatMarkdown(midStream, options)
  assert.doesNotMatch(html, /<p class="md-strong-title">कुल मिलाकर/)
  assert.match(html, /<h4><strong>सारांश और सुझाव<\/strong><\/h4>/)
})

test('renderChatMarkdown preserves citations, math, and sanitized output through one shared pipeline', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, title, text }) =>
      `<img src="${href}" alt="${text}" title="${title || ''}" class="markdown-image">`,
    isValidImageUrl: (href) => href.startsWith('https://'),
  })

  const html = renderChatMarkdown(
    [
      'See <kb doc="sample-product-guide.pdf" chunk_id="chunk-1" kb_id="kb-1"/>',
      '',
      'Formula: \\(E=mc^2\\)',
      '',
      '| A | B |',
      '| --- | --- |',
      '| 1 | 2 |',
      '',
      '![ok](https://example.com/a.png "Diagram")',
      '',
      '![bad](javascript:alert(1))',
    ].join('\n'),
    {
      renderer,
      escapeMarkdown: (text) => text,
      sanitizeHtml: (html) => html.replace(/javascript:alert\(1\)/g, ''),
    },
  )

  assert.match(html, /class="citation citation-kb"/)
  assert.match(html, /data-chunk-id="chunk-1"/)
  assert.match(html, /katex/)
  assert.match(html, /<div class="chat-markdown-table"><table>/)
  assert.match(html, /<img src="https:\/\/example\.com\/a\.png"/)
  assert.doesNotMatch(html, /javascript:alert/)
})

test('resolveCitationChunkId maps context index to retrieval chunk UUID', () => {
  const refs = [
    { id: 'uuid-chunk-1', knowledge_title: 'Doc A', chunk_type: 'faq' },
    { id: 'uuid-chunk-2', knowledge_title: 'FAQ TEST - FAQ', chunk_type: 'faq' },
  ]

  assert.equal(
    resolveCitationChunkId('2', { doc: 'FAQ TEST - FAQ' }, refs),
    'uuid-chunk-2',
  )
  assert.equal(
    resolveCitationChunkId('FAQ-2', { doc: 'FAQ TEST - FAQ' }, refs),
    'uuid-chunk-2',
  )
  assert.equal(
    resolveCitationChunkId('uuid-chunk-1', { doc: 'Doc A' }, refs),
    'uuid-chunk-1',
  )
})

test('renderChatMarkdown preserves chunk UUIDs when escapeMarkdown strips UUIDs from prose', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, text }) => `<img src="${href}" alt="${text}">`,
    isValidImageUrl: () => true,
  })
  const chunkId = SAMPLE_CHUNK_PRESERVE
  const stripUuids = (text: string) => text.replace(
    /[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}/gi,
    '',
  )

  const html = renderChatMarkdown(
    `Sample text <kb doc="sample-topic.pdf" chunk_id="${chunkId}" />`,
    {
      renderer,
      escapeMarkdown: stripUuids,
      sanitizeHtml: (html) => html,
    },
  )

  assert.match(html, /class="citation citation-kb"/)
  assert.match(html, new RegExp(`data-chunk-id="${chunkId}"`))
})

test('renderChatMarkdown resolves indexed chunk_id when knowledge references are provided', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, text }) => `<img src="${href}" alt="${text}">`,
    isValidImageUrl: () => true,
  })

  const html = renderChatMarkdown(
    'See <kb doc="Sample FAQ" chunk_id="2" />',
    {
      renderer,
      escapeMarkdown: (text) => text,
      sanitizeHtml: (html) => html,
      knowledgeReferences: [
        { id: 'resolved-chunk-id', knowledge_title: 'Sample FAQ', chunk_type: 'faq' },
        { id: 'other-chunk', knowledge_title: 'Other FAQ', chunk_type: 'faq' },
      ],
    },
  )

  assert.match(html, /data-chunk-id="resolved-chunk-id"/)
  assert.doesNotMatch(html, /data-chunk-id="2"/)
})

test('joinCitationTagsToPreviousLine removes blank lines before citation tags', () => {
  const input = 'Setup is complete.\n\n<kb doc="faq.pdf" chunk_id="1" />'
  assert.equal(joinCitationTagsToPreviousLine(input), 'Setup is complete. <kb doc="faq.pdf" chunk_id="1" />')
})

test('joinCitationTagsToPreviousLine inlines consecutive citation tags across single newlines', () => {
  const tag1 = `<kb doc="${SAMPLE_DOC}" chunk_id="${SAMPLE_CHUNK_A}" />`
  const tag2 = `<kb doc="${SAMPLE_DOC}" chunk_id="${SAMPLE_CHUNK_B}" />`
  const tag3 = `<kb doc="${SAMPLE_DOC}" chunk_id="${SAMPLE_CHUNK_C}" />`
  const input = `${tag1}\n${tag2}\n${tag3}`
  assert.equal(joinCitationTagsToPreviousLine(input), `${tag1} ${tag2} ${tag3}`)
})

test('renderChatMarkdown inlines consecutive citation tags across newlines', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, text }) => `<img src="${href}" alt="${text}">`,
    isValidImageUrl: () => true,
  })
  const html = renderChatMarkdown(
    [
      `<kb doc="${SAMPLE_DOC}" chunk_id="${SAMPLE_CHUNK_A}" />`,
      `<kb doc="${SAMPLE_DOC}" chunk_id="${SAMPLE_CHUNK_B}" />`,
      `<kb doc="${SAMPLE_DOC}" chunk_id="${SAMPLE_CHUNK_C}" />`,
    ].join('\n'),
    {
      renderer,
      escapeMarkdown: (text) => text,
      sanitizeHtml: (html) => html,
    },
  )

  assert.equal((html.match(/citation-kb/g) || []).length, 3)
  assert.doesNotMatch(html, /<\/p>\s*<p>\s*<span class="citation citation-kb"/)
})

test('joinCitationTagsToPreviousLine appends an indented citation to the preceding list item', () => {
  const tag = '<kb doc="Reading Star national youth reading showcase.pdf" chunk_id="chunk-1" />'
  const input = [
    '#### 5️⃣ Reading Star centres of excellence',
    '- Every school with a group winner, runner-up or top-ten finisher receives a **"Reading Star centre"** plaque',
    '',
    `  ${tag}`,
  ].join('\n')
  assert.equal(
    joinCitationTagsToPreviousLine(input),
    [
      '#### 5️⃣ Reading Star centres of excellence',
      `- Every school with a group winner, runner-up or top-ten finisher receives a **"Reading Star centre"** plaque ${tag}`,
    ].join('\n'),
  )
})

test('renderChatMarkdown renders a citation after a list item inline in that item', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, text }) => `<img src="${href}" alt="${text}">`,
    isValidImageUrl: () => true,
  })
  const tag = '<kb doc="Reading Star national youth reading showcase.pdf" chunk_id="chunk-1" />'
  const html = renderChatMarkdown(`- Centre plaque\n\n  ${tag}`, {
    renderer,
    escapeMarkdown: (text) => text,
    sanitizeHtml: (value) => value,
  })

  assert.match(html, /<li>Centre plaque <span class="citation citation-kb"/)
  assert.doesNotMatch(html, /<\/ul>\s*<p>\s*<span class="citation citation-kb"/)
})

test('joinCitationTagsToPreviousLine does not merge citations onto fenced code closing delimiter', () => {
  const tag = '<kb doc="guide.pdf" chunk_id="1" />'
  const input = '```bash\nunzip setup.zip\n```\n\n' + tag
  assert.equal(joinCitationTagsToPreviousLine(input), '```bash\nunzip setup.zip\n```\n\n' + tag)
})

test('joinCitationTagsToPreviousLine does not merge citations onto an unlabeled closing fence on a single newline', () => {
  const tag = '<kb doc="guide.pdf" chunk_id="1" />'
  const input = '```\nAPR = principal\n```\n' + tag
  assert.equal(joinCitationTagsToPreviousLine(input), '```\nAPR = principal\n```\n' + tag)
})

test('applyStreamingTailFade wraps the trailing text run', () => {
  const out = applyStreamingTailFade('<p>Great, that narrows it down a lot. Two more quick</p>')
  assert.match(out, /<span class="stream-fade-tail">[^<]*Two more quick<\/span><\/p>$/)
})

test('applyStreamingTailFade skips whitespace-only runs and fades the last list item', () => {
  const out = applyStreamingTailFade('<ol>\n<li>पहली मद</li>\n<li>दूसरी मद बन रही है</li>\n</ol>')
  assert.match(out, /<li><span class="stream-fade-tail">दूसरी मद बन रही है<\/span><\/li>/)
})

test('applyStreamingTailFade is a no-op for empty content', () => {
  assert.equal(applyStreamingTailFade(''), '')
  assert.equal(applyStreamingTailFade('<p></p>'), '<p></p>')
})

test('renderChatMarkdown adds the tail fade only while streaming', () => {
  const renderer = createChatMarkdownRenderer()
  const opts = {
    renderer,
    escapeMarkdown: (text: string) => text,
    sanitizeHtml: (html: string) => html,
  }
  const streamed = renderChatMarkdown('उत्तर अभी तैयार हो रहा है', { ...opts, streaming: true })
  assert.match(streamed, /stream-fade-tail/)
  const settled = renderChatMarkdown('उत्तर अभी तैयार हो रहा है', { ...opts, streaming: false })
  assert.doesNotMatch(settled, /stream-fade-tail/)
})

test('renderChatMarkdown keeps an unlabeled fenced code block closed when a citation immediately follows', () => {
  const renderer = createChatMarkdownRenderer()
  const tag = '<kb doc="guide.pdf" chunk_id="1" />'
  const html = renderChatMarkdown(
    ['```', 'APR = principal', '```', tag, '', '### Why it matters'].join('\n'),
    {
      renderer,
      escapeMarkdown: (text) => text,
      sanitizeHtml: (html) => html,
    },
  )

  assert.doesNotMatch(html, /### Why it matters/)
  assert.match(html, /<h3>Why it matters<\/h3>/)
  assert.equal((html.match(/<pre>/g) || []).length, 1)
})

test('renderChatMarkdown keeps fenced code blocks closed when citations follow', () => {
  const renderer = createChatMarkdownRenderer({
    imageRenderer: ({ href, text }) => `<img src="${href}" alt="${text}">`,
    isValidImageUrl: () => true,
  })
  const tag = '<kb doc="guide.pdf" chunk_id="1" />'
  const html = renderChatMarkdown(
    ['```bash', 'unzip setup.zip', '```', '', tag, '', '#### Next step'].join('\n'),
    {
      renderer,
      escapeMarkdown: (text) => text,
      sanitizeHtml: (html) => html,
    },
  )

  assert.doesNotMatch(html, /#### Next step/)
  assert.match(html, /<h4>Next step<\/h4>/)
  assert.equal((html.match(/<pre>/g) || []).length, 1)
})

test('collapseStandaloneCitationParagraphs merges citations across empty paragraphs', () => {
  const html = '<p>Steps:</p><p></p><p><span class="citation citation-kb" data-chunk-id="x">doc</span></p>'
  const out = collapseStandaloneCitationParagraphs(html)
  assert.match(out, /Steps:.*citation-kb/s)
  assert.doesNotMatch(out, /<p><\/p>/)
})
