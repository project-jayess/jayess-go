package runtime

import "strconv"

func arrayIndexKey(index int) string {
	return strconv.Itoa(index)
}

func arrayIndexFromKey(key string) (int, bool) {
	if key == "" {
		return 0, false
	}
	if key != "0" && key[0] == '0' {
		return 0, false
	}
	index, err := strconv.Atoi(key)
	if err != nil || index < 0 {
		return 0, false
	}
	if strconv.Itoa(index) != key {
		return 0, false
	}
	return index, true
}
