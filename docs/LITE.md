# Differences Between EnterpriseRag Lite and the Standard Edition

Lite is aimed at people who want to get started locally in a hurry with the simplest possible deployment; the Standard Edition is aimed at multi-workspace collaboration and the full set of enterprise capabilities. The main differences are listed below.


| Dimension | Lite | Standard Edition |
| --------- | ----------------------------------------------- | --------------------------------- |
| **Shared workspaces** | No shared workspaces (member invitations, sharing knowledge bases and agents across members, and so on) | Shared workspaces plus collaboration features such as workspace-isolated retrieval |
| **Workspaces and accounts** | Single workspace; works out of the box, **no registration required** | Multiple workspaces; registration, login and organisation management are usually required |
| **Document parsing** | Only the built-in **Simple** parsing engine; other parsing capabilities can be connected through options such as **Cloud** | Several parsing engines can be configured (including high-accuracy ones), integrated with the complete document processing pipeline |
| **Deployment shape** | **Single application, zero dependencies** (no separate database, message queue or other external service stack) | Typically a multi-service deployment such as Docker Compose, with more dependencies and components |
| **Data ownership** | Data is stored and processed **entirely on your own machine** | With a private deployment the data can also stay local; it depends on how you deploy |
| **Network exposure** | **Local access only by default**; configurable, so **you decide whether to expose it to the public internet** | Bind addresses and gateways are chosen according to your deployment and security policy |


If you do not need multi-team collaboration, a complex parsing pipeline or a multi-service architecture, Lite is the better fit for an individual or a small team trying things out locally with zero dependencies. If you need shared workspaces, multiple workspaces and the full matrix of parsing engines, use the Standard Edition.
