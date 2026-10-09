package lenses

import "net/url"

// DeepLinkGlobal is the console path for the global lens (I05/I09).
func DeepLinkGlobal() string {
	return "/ui/observe?lens=global"
}

// DeepLinkAgent is the console path for an agent lens run.
func DeepLinkAgent(runID string) string {
	return "/ui/observe?lens=agent&run=" + url.QueryEscape(runID)
}

// DeepLinkMemory is the console path for a memory lineage id.
func DeepLinkMemory(memoryID string) string {
	return "/ui/observe?lens=memory&id=" + url.QueryEscape(memoryID)
}
