/** Settings outside the local firewall product are never shown, even in search results. */
export function isLocalSetting(setting: { Key: string }): boolean {
  return !setting.Key.startsWith('spn/')
    && setting.Key !== 'core/releaseChannel'
    && setting.Key !== 'core/automaticUpdates';
}
