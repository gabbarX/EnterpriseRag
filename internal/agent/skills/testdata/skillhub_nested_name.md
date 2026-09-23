---
name: समय
  version: 1.2.6
  description: |
    समय प्रबंधन सहायक — दैनिक योजना, अनुस्मारक, साप्ताहिक समीक्षा, अवकाश कैलेंडर,
    यात्रा योजना, बैठक नोट्स और कार्य-सूची का एक ही जगह संकलन। केवल सामान्य मार्गदर्शन के लिए; यह चिकित्सा, कानूनी,
    मानसिक स्वास्थ्य या वित्तीय सलाह का विकल्प नहीं है; बड़े निर्णयों के लिए किसी पेशेवर से परामर्श करें।
    [डेटा और गोपनीयता सूचना] यह skill उपयोगकर्ता द्वारा दी गई जन्मतिथि, शहर,
    नाम, वैकल्पिक पारिवारिक जानकारी, विषय-रुचि रिकॉर्ड (डिफ़ॉल्ट रूप से बंद) और पुश-कार्य लॉग स्थानीय फ़ाइल सिस्टम में रखता है।
    कोई भी डेटा अपलोड नहीं होता, केवल स्थानीय data/ निर्देशिका में रहता है; उपयोगकर्ता profile.js से कभी भी देख, बदल या मिटा सकता है।
    दैनिक पुश opt-in है और डिफ़ॉल्ट रूप से बंद रहता है।
metadata:
  displayName: "समय"
  author:
    - "senior-engineer-enoyao"
    - "senior-product-ops-rekyhe"
  version: 1.2.0
  keywords: दैनिक योजना, अनुस्मारक, साप्ताहिक समीक्षा, अवकाश कैलेंडर, छुट्टी, बैठक नोट्स, कार्य-सूची, यात्रा योजना, त्योहार कैलेंडर, समय प्रबंधन, planner, reminders, weekly review, leave calendar, meeting notes, todo list, travel planning, festival calendar, time management
  # Trigger keywords tightened in 1.1.8: over-broad terms ("plan / schedule / time / productivity")
  # were removed so the skill does not activate unless a concrete feature or topic is named.
  openclaw:
    emoji: "📅"
    skillKey: "university-applications"
    runtime:
      node: ">=18"
      python3: true
    install:
      - kind: node
        package: iztro
    env: []  # This skill no longer reads any environment variable; the old OPENCLAW_KNOWLEDGE_DIR was removed to avoid file-system enumeration.
    security:
      network:
        default: none
        optional:
          - feature: "planner HTML LLM summary (user-initiated, in-browser, OFF by default)"
            allowed-endpoints:
              - "https://api.openai.com"
              - "https://api.anthropic.com"
              - "https://api.deepseek.com"
            custom-endpoint: "only after explicit in-UI consent dialog; HTTPS-only, no IP/localhost auto-trust"
            data-sent: "only the schedule entry and the user's typed question; no profile, no env vars, no file paths"
            credential: "user-provided LLM API key, entered at runtime, stored in browser localStorage only"
          - feature: "planner HTML Google Fonts (commented out by default)"
            endpoint: "https://fonts.googleapis.com"
            data-sent: "none beyond standard font request"
            credential: none
      credentials:
        bundled: none
        required: none
        user-optional:
          - "LLM API key for the planner/index.html summary feature (scope it to a separate limited key, never reused)"
      push-mechanism: openclaw-ipc
      push-optin: true
      push-default-state: disabled
      data-retention:
        location: "local filesystem under data/profiles/ and data/push-log.json"
        remote-upload: none
        user-controls:
          - "view:   node scripts/profile.js show <userId>"
          - "list:   node scripts/profile.js list"
          - "edit:   node scripts/profile.js save <userId> <field> <value>"
          - "delete: node scripts/profile.js delete <userId>"
          - "disable push: node scripts/push-toggle.js off <userId>"
      notes: |
        All bundled scripts perform local computation only — no fetch, axios,
        https.request, curl, wget, or any outbound network calls from the Node/Python
        side. Push delivery is handled entirely by the OpenClaw runtime via stdout/IPC
        protocol, and is OPT-IN (disabled until the user runs push-toggle.js on).
        The 'channels' field in user profiles (e.g. telegram) is a routing hint for
        the OpenClaw runtime, not a direct API integration. This skill does not hold
        or require any third-party API tokens (Telegram Bot Token, SMTP credentials,
        webhook URLs, etc.). The local-only release helper script is excluded
        from the published bundle via .clawhubignore and is not part of the
        installed skill surface.
        EXCEPTION — OPTIONAL LLM NETWORK USE: the browser-only file planner/index.html
        exposes an optional "LLM summary" button. If and only if the user clicks
        it and fills in their own API key + endpoint, the browser (not the skill
        process) will POST the schedule entry and question to that user-configured endpoint.
        No key is bundled, hardcoded, or transmitted anywhere else. Users are advised
        to supply a scoped/limited API key rather than a primary account key.
        User profile data (birth details, optional family members, interaction log)
        is stored only on the local filesystem and can be viewed, edited, or deleted
        at any time via scripts/profile.js (see data-retention.user-controls above).
---

# body
