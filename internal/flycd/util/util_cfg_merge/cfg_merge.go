package util_cfg_merge

func Merge(base map[string]any, overlay map[string]any) (map[string]any, error) {
	for k, v := range overlay {
		if v == nil {
			delete(base, k)
		} else {
			base[k] = v
		}
	}
	return base, nil
}
