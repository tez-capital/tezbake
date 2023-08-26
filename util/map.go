package util

func MergeMaps[K comparable](a map[K]interface{}, b map[K]interface{}, overwrite bool) map[K]interface{} {
	result := make(map[K]interface{}, len(a))
	keyMap := make(map[K]bool)
	for k, v := range a {
		result[k] = v
		keyMap[k] = true
	}
	for k, v := range b {
		if _, found := keyMap[k]; found {
			if !overwrite {
				continue
			}
		}
		result[k] = v
	}
	return result
}

// func MergeMapsDeep(a map[interface{}]interface{}, b map[interface{}]interface{}, overwrite bool) map[interface{}]interface{} {
// 	out := make(map[interface{}]interface{}, len(a))
// 	keyMap := make(map[interface{}]bool)
// 	for k, v := range a {
// 		out[k] = v
// 		keyMap[k] = true
// 	}
// 	for k, v := range b {
// 		if v, ok := v.(map[interface{}]interface{}); ok {
// 			if bv, ok := out[k]; ok {
// 				if bv, ok := bv.(map[interface{}]interface{}); ok {
// 					out[k] = MergeMaps(bv, v, overwrite)
// 					continue
// 				}
// 			}
// 		}
// 		if _, found := keyMap[k]; found {
// 			if !overwrite {
// 				continue
// 			}
// 		}
// 		out[k] = v
// 	}
// 	return out
// }

// func MergeStringKeyedMaps(a map[string]interface{}, b map[string]interface{}, overwrite bool, level uint) map[string]interface{} {
// 	out := make(map[string]interface{}, len(a))
// 	keyMap := make(map[interface{}]bool)
// 	for k, v := range a {
// 		out[k] = v
// 		keyMap[k] = true
// 	}
// 	for k, v := range b {
// 		if v, ok := v.(map[interface{}]interface{}); ok {
// 			if bv, ok := out[k]; ok {
// 				if bv, ok := bv.(map[interface{}]interface{}); ok {
// 					out[k] = MergeMaps(bv, v, overwrite)
// 					continue
// 				}
// 			}
// 		}
// 		if v, ok := v.(map[string]interface{}); ok {
// 			if bv, ok := out[k]; ok {
// 				if bv, ok := bv.(map[string]interface{}); ok {
// 					out[k] = MergeStringKeyedMaps(bv, v, overwrite, level-1)
// 					continue
// 				}
// 			}
// 		}
// 		if _, found := keyMap[k]; found {
// 			if !overwrite {
// 				continue
// 			}
// 		}
// 		out[k] = v
// 	}
// 	return out
// }
