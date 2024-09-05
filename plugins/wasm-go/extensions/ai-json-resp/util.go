package main

func GetMaxDepth(data interface{}) int {
	type item struct {
		value interface{}
		depth int
	}

	maxDepth := 0
	stack := []item{{value: data, depth: 1}}

	for len(stack) > 0 {
		currentItem := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if currentItem.depth > maxDepth {
			maxDepth = currentItem.depth
		}

		switch v := currentItem.value.(type) {
		case map[string]interface{}:
			for _, value := range v {
				stack = append(stack, item{value: value, depth: currentItem.depth + 1})
			}
		case []interface{}:
			for _, value := range v {
				stack = append(stack, item{value: value, depth: currentItem.depth + 1})
			}
		}
	}

	return maxDepth
}
