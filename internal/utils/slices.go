package utils

func RemoveDuplicateInt64(intSlice []int64) []int64 {
	allKeys := make(map[int64]struct{})
	list := []int64{}
	for _, item := range intSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = struct{}{}
			list = append(list, item)
		}
	}
	return list
}

func Chunks[T any](items []T, chunkSize int) (chunks [][]T) {
	for chunkSize < len(items) {
		items, chunks = items[chunkSize:], append(chunks, items[0:chunkSize:chunkSize])
	}
	return append(chunks, items)
}
