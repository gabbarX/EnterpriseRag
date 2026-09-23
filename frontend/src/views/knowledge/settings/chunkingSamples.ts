// Curated sample texts for the chunking debug drawer. Each preset is sized
// to ≈2000–4000 characters and shaped to exercise a distinct tier of the
// chunker, so users can quickly see how their config behaves on realistic
// content without preparing their own sample.

export interface ChunkingSample {
  id: string
  label: string
  text: string
}

const MARKDOWN_SAMPLE = `# Leave and Expense Policy

This policy applies to all full-time employees of the company across our
Bengaluru, Pune and Gurugram offices. It is reviewed every financial year and
supersedes all earlier circulars on leave and reimbursements.

## Scope and effective date

The policy is effective from 1 April and runs to 31 March, in line with the
Indian financial year. Employees who join mid-year accrue leave on a pro-rata
basis from their date of joining.

## Leave entitlement

### Earned leave

Every confirmed employee accrues **1.75 days** of earned leave per completed
month, i.e. 21 days a year. Up to 45 days may be carried forward; anything
above that lapses on 31 March and is not encashed.

### Casual and sick leave

| Leave type | Days per year | Carry forward | Medical certificate |
|------------|---------------|---------------|---------------------|
| Casual leave | 7 | No | Not required |
| Sick leave | 12 | Up to 30 days | Required beyond 2 consecutive days |
| Bereavement leave | 5 | No | Not required |
| Maternity leave | 182 | Not applicable | As per the Maternity Benefit Act |
| Paternity leave | 15 | No | Not required |

### Public holidays

Each office publishes its own list of 12 gazetted holidays, of which 2 are
restricted holidays that an employee may choose. Regional festivals such as
Ugadi, Pongal, Onam, Bihu and Durga Puja are covered by the restricted list so
that teams across locations can pick what is relevant to them.

## Applying for leave

- Raise the request on the HR portal at least **3 working days** in advance.
- Leave of more than 5 consecutive working days needs the skip-level manager's
  approval as well.
- Sick leave may be regularised within 2 working days of returning.
- Leave taken without an approved request is treated as loss of pay.

## Expense reimbursement

### Daily allowance while travelling

| City tier | Lodging cap (per night) | Meals (per day) | Local travel (per day) |
|-----------|-------------------------|-----------------|------------------------|
| Metro | ₹6,500 | ₹1,500 | ₹1,200 |
| Tier 2 | ₹4,000 | ₹1,000 | ₹800 |
| Tier 3 and below | ₹2,500 | ₹750 | ₹600 |

Metro covers Mumbai, Delhi NCR, Bengaluru, Hyderabad, Chennai and Kolkata.

### What needs a bill

Any single claim above **₹500** needs a GST-compliant invoice carrying the
vendor's GSTIN. Claims below that threshold may be filed with a self-declared
note, subject to a ceiling of ₹3,000 per month.

### Filing a claim

\`\`\`text
Portal  ->  Expenses  ->  New claim
  1. Pick the cost centre and the project code
  2. Attach the invoice as a PDF or a clear photograph
  3. Enter the amount in INR, exclusive of GST, and the GST separately
  4. Submit before the 5th of the following month
\`\`\`

Claims filed after the 5th roll into the next payroll cycle. Anything older
than 90 days is rejected by the system and needs a finance exception.

## Approvals and escalation

\`\`\`
┌──────────────────────────┐
│  Employee                │  raises the request
├──────────────────────────┤
│  Reporting manager       │  approves within 2 working days
├──────────────────────────┤
│  Finance (claims > ₹25k) │  verifies invoices and GST credit
├──────────────────────────┤
│  HR business partner     │  handles exceptions and appeals
└──────────────────────────┘
\`\`\`

If an approval is pending for more than 5 working days, the request escalates
automatically to the next level. Employees may appeal a rejection once, in
writing, within 15 days.

## Further reading

- Travel booking guidelines: \`docs/travel-desk.md\`
- Payroll calendar and cut-off dates: \`docs/payroll.md\`
- Code of conduct: \`docs/code-of-conduct.md\``

const FAQ_SAMPLE = `# Goods and Services Tax (GST) — frequently asked questions

This note collects the questions our finance helpdesk answers most often. It
is guidance for internal use and is not a substitute for advice from a
chartered accountant.

## Registration

### Q1: When must a business register for GST?
Registration is mandatory once aggregate turnover in a financial year crosses
₹40 lakh for goods or ₹20 lakh for services. The threshold is ₹20 lakh and
₹10 lakh respectively for the special category states. Inter-state suppliers
and e-commerce operators must register from the first rupee.

### Q2: What is a GSTIN made of?
A GSTIN is 15 characters: the first two are the state code, the next ten are
the PAN of the entity, the thirteenth is the entity number for that PAN in the
state, the fourteenth is the letter Z, and the last is a checksum.

### Q3: Do we need a separate registration per state?
Yes. GST is state-wise, so a presence in Karnataka and Maharashtra needs two
registrations against the same PAN, and stock transfers between them are
treated as supplies.

### Q4: What is the composition scheme?
Small taxpayers below ₹1.5 crore turnover may pay tax at a flat rate on
turnover instead of the normal rates, but they cannot claim input tax credit
and cannot make inter-state supplies.

## Invoicing

### Q5: What must a tax invoice carry?
The supplier's and recipient's name, address and GSTIN, an invoice number and
date, the HSN or SAC code, taxable value, the rate and amount of CGST, SGST,
IGST or cess, place of supply, and a signature or digital signature.

### Q6: When is e-invoicing mandatory?
For businesses with aggregate turnover above ₹5 crore, B2B invoices must be
reported to the Invoice Registration Portal, which returns an IRN and a signed
QR code. An invoice without a valid IRN is not a valid document for credit.

### Q7: What about an e-way bill?
An e-way bill is required for the movement of goods worth more than ₹50,000.
It is valid for one day per 200 km for regular cargo and must be generated
before the movement begins.

## Input tax credit

### Q8: When can input tax credit be claimed?
All four conditions must hold: you hold a tax invoice, you have received the
goods or services, the supplier has paid the tax and reported the invoice, and
you have filed the relevant return.

### Q9: Why does GSTR-2B not show a supplier's invoice?
Work through it in this order:

1. Confirm the supplier has filed GSTR-1 for the period
2. Check that they used your correct GSTIN, not a sister entity's
3. Check the invoice date falls inside the return period you are looking at
4. Ask for the IRN if e-invoicing applies to them

### Q10: What is blocked credit?
Credit is not available on motor vehicles for personal use, food and beverages,
club memberships, health insurance beyond statutory obligations, works contract
services for immovable property, and goods lost, stolen or given as free
samples.

## Returns and payment

### Q11: Which returns do we file and when?
GSTR-1 for outward supplies by the 11th of the following month, GSTR-3B for the
summary and payment by the 20th, and the annual GSTR-9 by 31 December of the
next financial year.

### Q12: What is the reverse charge mechanism?
For notified supplies, and for purchases from unregistered dealers in some
cases, the recipient pays the tax directly instead of the supplier. Legal
services from an advocate and goods transport agency services are the common
examples.

### Q13: What interest and late fee apply?
Interest runs at 18% a year on tax paid late, and 24% where credit was wrongly
availed. The late fee is ₹50 a day, or ₹20 a day for a nil return, capped per
return.

## Common errors

### Q14: We charged IGST instead of CGST plus SGST
Place of supply was determined wrongly. Issue a credit note against the
original invoice and raise a fresh one with the correct heads before the
September return of the following financial year.

### Q15: A supplier raised an invoice against the wrong GSTIN
Neither party can amend the counterparty's GSTIN after filing; the supplier
must issue a credit note and a fresh invoice. Track this monthly, because the
credit is lost once the amendment window closes.`

const CHAPTER_SAMPLE = `Chapter 1: Introduction

1.1 Purpose of this manual

This manual describes the installation, configuration and day-to-day operation
of the platform for operations engineers, site reliability engineers and
solution architects. Readers are expected to be comfortable with Docker,
Kubernetes and PostgreSQL.

1.2 Terminology

- Gateway: handles ingress traffic, routing and protocol translation
- Service tier: carries the domain logic and the orchestration
- Storage tier: encapsulates persistence and caching
- Inference tier: the isolation layer that talks to external language models

1.3 Document version

This is version 1.4, matching the v0.5.x release series. The manual is updated
alongside the code; the change log is in Appendix A.

Chapter 2: System architecture

2.1 Overall design

The platform follows a microservice architecture, but the boundaries are drawn
conservatively — a service is split out only where it genuinely needs to scale
on its own. There are four long-running services today: the gateway, the
application backend, the document reader and the vector indexer.

2.2 Module breakdown

2.2.1 Users and permissions

Users, organisations, shared spaces, roles and permission rules all live in the
user service. The role-based model supports inheritance and overrides, and
external identity providers are wired in through an adapter layer.

2.2.2 Content and indexing

The content side owns the document lifecycle: upload, parse, chunk, embed,
store, retrieve and cite. A chunk is the finest retrieval unit and carries the
source document's metadata along with its position.

2.2.3 Reasoning and orchestration

The agent orchestrator runs the reason-act-observe loop. Tool calls are
dispatched either through the Model Context Protocol or through the built-in
registry. Every model call goes through a single proxy layer, which makes
switching providers, rate limiting and charge-back straightforward.

2.3 Data flow

An uploaded document goes through parse, clean, chunk, embed and store. A query
goes through rewrite, multi-route recall, fusion, rerank, context assembly and
generation.

Chapter 3: Deployment guide

3.1 Sizing

Start with 8 vCPU, 16 GB of memory and 100 GB of SSD. For production, plan for
16 vCPU, 32 GB and 500 GB. A GPU is needed only when inference runs locally.

3.2 Container orchestration

3.2.1 Single-node Docker Compose

Suitable for a proof of concept and for smaller teams — fewer than 100 people
and under a million chunks. One command brings up every service:

docker compose up -d

3.2.2 Kubernetes with Helm

Suitable for scaled deployments. The Helm chart lives in the helm directory and
contains the stateful sets, the configuration secrets and the ingress
templates. It can point at an existing PostgreSQL or Redis cluster.

3.3 Configuration practices

- Keep external dependencies in a configuration store rather than hard-coding
- Manage model API keys as secrets, isolated per environment
- Log to stdout and stderr and let the orchestrator collect, never to local disk

Chapter 4: Routine operations

4.1 Upgrades

Blue-green and rolling upgrades are both supported. Always read the change log
for schema changes first, and run migrations before replacing the application
image.

4.2 Backup and restore

Take a full PostgreSQL backup daily with continuous write-ahead log archiving.
Object storage relies on the provider's own versioning. Snapshot the vector
store on a schedule and ship the snapshots to object storage.

4.3 Troubleshooting

Work through gateway, then service, then storage, then model. Each tier exposes
a health endpoint and a trace entry point; together they locate roughly 80% of
incidents within five minutes.`

const PLAIN_SAMPLE = `Retrieval quality depends on several things, and the most direct of them is how
well the chunking strategy matches the embedding model. Chunks that are too
coarse mix several topics into one passage and dilute the relevance score;
chunks that are too fine lose their context, so no single passage can answer a
question that spans a section. A reasonable starting point is to keep the chunk
between 50% and 80% of the embedding model's recommended window, which keeps
the meaning intact while leaving some headroom.

Overlap matters as much as size. With no overlap at all, a question that
straddles a boundary usually recalls only half a sentence. Ten to twenty
percent is normally enough to cover the boundary cases. Beyond about thirty
percent, neighbouring chunks become largely redundant, which raises index cost
without improving recall. A simple rule of thumb: start with one third of your
average answer length and tune from there against a small evaluation set.

Separators should follow the document's real structure. Plain text does well
with a blank line as the strong separator. Markdown is better served by
splitting on heading levels first and only then on paragraphs inside a section.
Content that mixes prose with code and tables needs something more careful —
extract the code structure separately, detect table boundaries, or pre-separate
those two kinds of content before the text splitter sees them.

Script matters too, and it is easy to get wrong. If your corpus mixes English
with Hindi, Marathi or Nepali, the separator list must carry both the full stop
and the danda; if it carries only one of them, splitting works for one script
and silently fails for the other. The same trap applies to counting: a
Devanagari syllable is several Unicode code points, matras and the virama among
them, and three bytes each, so a splitter that measures in bytes will cut
roughly three times too early and can slice a cluster in half. Measure in runes
and test with a non-Latin sample before you trust the numbers.

The choice of embedding model is routinely underrated. Among open models the
BGE family is a dependable baseline, and BGE-M3 covers dense, sparse and
multi-vector retrieval at once. Among commercial options the general-purpose
OpenAI embeddings are strong and reasonably priced. For long documents, watch
the maximum input length: exceeding it means the input is either truncated or
embedded in pieces and averaged, and neither works well.

Finally, do not skip post-processing. A reranker lifts end-to-end quality by
roughly 5% to 15% in almost every setting, at the cost of another 100 to 300
milliseconds per query. If your retrieval path includes keyword recall, a
reranker as a second filter is close to mandatory, because literal BM25 matches
will otherwise pollute the context window.`

export const CHUNKING_SAMPLES: ChunkingSample[] = [
  { id: "markdown", label: "Markdown policy", text: MARKDOWN_SAMPLE },
  { id: "faq", label: "FAQ", text: FAQ_SAMPLE },
  { id: "chapter", label: "Chaptered manual", text: CHAPTER_SAMPLE },
  { id: "plain", label: "Plain prose", text: PLAIN_SAMPLE },
];

export const DEFAULT_SAMPLE_ID = "markdown";
