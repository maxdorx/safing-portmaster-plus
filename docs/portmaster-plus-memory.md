# RAM investigation: installed official Portmaster 2.2.8

Read-only measurements on 2026-11-08, while the original installed firewall remained active:

- Windows working set: approximately 596 MB; private committed memory: approximately 644 MB; peak working set: approximately 654 MB.
- Installed version verified by `portmaster-core.exe version`: official 2.2.8, commit 5c82035444a6a383b8b7011ff6834a52f2d238a4.
- Local `/api/v1/debug/heap` snapshot: approximately 159.54 MB sampled live Go heap.
- Largest cumulative allocation sites: connection creation 63.11 MB, connection detail collection 32.01 MB, GeoIP 17.50 MB. Allocation-site samples are estimates and do not fully explain the Windows working set.

The difference between sampled Go heap and Windows memory counters does not establish a leak. Retained heap, stacks, database allocations, and other runtime/OS allocations can contribute. SPN was not a major contributor in this snapshot. The custom build has not been deployed, so these are baseline measurements, not measured savings from the cleanup.

The 2.2.8 source includes a bandwidth-table cleanup fix missing from the fork's previous 2.2.1 base. Its regression test passes. The official installed 2.2.8 already contains that fix.

The custom build additionally streams uncompressed artifact hashing instead of loading each executable into one large byte buffer. This bounds temporary allocations during startup indexing; it does not impose a RAM limit or establish lower steady-state usage.

Profiles and analyzer output are stored locally under `dist/diagnostics/` and are not committed. No service restart, forced garbage collection, or firewall setting changes were performed for these measurements.
