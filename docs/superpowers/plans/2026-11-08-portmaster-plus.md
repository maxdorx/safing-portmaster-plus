# Portmaster Plus 2.2.8 implementation record

Approved behavior: integrate upstream v2.2.8; retain local history and bandwidth without an account; remove SPN runtime/UI, accounts, subscriptions, promotions, and news; preserve local firewall behavior and existing visual style. Keep application updates manual and intelligence updates automatic. Preserve executable/service/API identifiers and legal attribution.

Implementation is split across integrated service/account gating, updater policy, Angular interface, and native tray/packaging. Shared SPN source/types remain where local runtime consumers require them. The active installed firewall is not changed.

Rulings:
- Initial implementation used the existing checkout and a pending upstream merge. The user subsequently installed the artifacts and explicitly requested committing, pushing, and publishing the Windows release.
- Disable upstream replacement at the updater layer, rather than depending on a toggle or default. Read locally built application artifacts through a locally generated index.
- Exclude optional news/promotional artifacts only after signature validation; leave the signed index file unchanged.
- Preserve stored credentials, but never use them to contact account infrastructure from the integrated service.

Verification results and artifact inventory are recorded in docs/portmaster-plus-validation.md after integration.
