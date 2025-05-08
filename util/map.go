package util

import "maps"

// MergeMaps merges src into dst.
// If overwrite is true, values in src will replace those in dst.
func MergeMaps[K comparable](dst map[K]any, src map[K]any, overwrite bool) map[K]any {
	if dst == nil {
		dst = make(map[K]any)
	}
	if src == nil {
		return dst
	}
	dst = maps.Clone(dst)

	for k, v := range src {
		if _, exists := dst[k]; !exists || overwrite {
			dst[k] = v
		}
	}
	return dst
}

// MergeMapsDeep merges src into dst recursively.
// If overwrite is true, values in src will replace those in dst.
func MergeMapsDeep[K comparable](dst map[K]any, src map[K]any, overwrite bool) map[K]any {
	if dst == nil {
		dst = make(map[K]any)
	}
	if src == nil {
		return dst
	}
	dst = maps.Clone(dst)

	for k, v := range src {
		if innerSrcMap, ok := v.(map[K]any); ok {
			if innerDstMap, ok := dst[k].(map[K]any); ok {
				dst[k] = MergeMapsDeep(innerDstMap, innerSrcMap, overwrite)
			} else if overwrite {
				dst[k] = innerSrcMap
			}
		} else if _, exists := dst[k]; !exists || overwrite {
			dst[k] = v
		}
	}
	return dst
}
