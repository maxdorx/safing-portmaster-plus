package updates

import "slices"

// filterIndex is called only after signature validation; the signed file remains intact.
func (cfg *Config) filterIndex(index *Index) {
	if index == nil {
		return
	}
	index.Artifacts = slices.DeleteFunc(index.Artifacts, func(a *Artifact) bool { return slices.Contains(cfg.ExcludeArtifacts, a.Filename) })
}
