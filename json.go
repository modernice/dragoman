package dragoman

import (
	"encoding/json"
	"fmt"
)

// JSONPath represents a sequence of keys that specify a unique path through a
// JSON object hierarchy, similar to an address for locating a specific value
// within a nested JSON structure. It is used to traverse and extract data from
// complex JSON documents.
type JSONPath []string

// JSONDiff identifies the differences between two JSON objects or two raw JSON
// byte representations. It returns a slice of JSONPaths that represent the
// hierarchical structure of keys where differences exist, and an error if any
// occur during the process. The function is generic and can accept either raw
// bytes or maps as inputs for comparison.
func JSONDiff[TInput []byte | map[string]any](source, target TInput) ([]JSONPath, error) {
	var sourceMap, targetMap map[string]any

	switch source := any(source).(type) {
	case []byte:
		if err := json.Unmarshal(source, &sourceMap); err != nil {
			return nil, fmt.Errorf("unmarshal source: %w", err)
		}

		if err := json.Unmarshal(any(target).([]byte), &targetMap); err != nil {
			return nil, fmt.Errorf("unmarshal target: %w", err)
		}
	case map[string]any:
		sourceMap = source
		targetMap = any(target).(map[string]any)
	}

	return jsonDiffPaths(sourceMap, targetMap)
}

func jsonDiffPaths(source, target map[string]any) (paths []JSONPath, _ error) {
	for k, v := range source {
		switch v := v.(type) {
		case map[string]any:
			targetValue, ok := target[k]
			if ok {
				targetMap, ok := targetValue.(map[string]any)
				if !ok {
					return paths, fmt.Errorf("target value at %q is not a map", k)
				}

				subPaths, err := jsonDiffPaths(v, targetMap)
				if err != nil {
					return paths, err
				}

				subPaths = mapSlice(subPaths, func(p JSONPath) JSONPath {
					return append(JSONPath{k}, p...)
				})

				paths = append(paths, subPaths...)
			} else {
				subKeys := allKeys(v)
				subKeys = mapSlice(subKeys, func(p JSONPath) JSONPath {
					return append(JSONPath{k}, p...)
				})

				paths = append(paths, subKeys...)
			}
		default:
			if _, ok := target[k]; !ok {
				paths = append(paths, JSONPath{k})
			}
		}
	}
	return
}

// JSONExtract extracts values from a JSON document according to specified paths
// and returns them as a map. It supports both raw JSON bytes and already-parsed
// maps as input. If any path does not exist or leads to an unexpected type, an
// error is returned alongside the partial output.
func JSONExtract[TData []byte | map[string]any](data TData, paths []JSONPath) (map[string]any, error) {
	var dataMap map[string]any
	switch data := any(data).(type) {
	case []byte:
		if err := json.Unmarshal(data, &dataMap); err != nil {
			return nil, fmt.Errorf("unmarshal data: %w", err)
		}
	case map[string]any:
		dataMap = data
	}

	out := make(map[string]any)
	for _, path := range paths {
		if err := jsonExtract(dataMap, path, out); err != nil {
			return out, err
		}
	}
	return out, nil
}

func jsonExtract(data map[string]any, path JSONPath, out map[string]any) error {
	if len(path) == 0 {
		return nil
	}

	key := path[0]
	value, ok := data[key]
	if !ok {
		return fmt.Errorf("key %q not found", key)
	}

	if len(path) == 1 {
		out[key] = value
		return nil
	}

	subPath := path[1:]
	subMap, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("value at %q is not a map", key)
	}

	if _, ok := out[key]; !ok {
		outSubMap := make(map[string]any)
		out[key] = outSubMap
	}

	outSubMap := out[key].(map[string]any)

	return jsonExtract(subMap, subPath, outSubMap)
}

// JSONMerge combines the contents of two JSON object maps, where 'from' is
// merged into 'into'. If there are matching keys, the values from 'from' will
// overwrite those in 'into'. For nested maps, merging is performed recursively.
// This function modifies the 'into' map directly and does not return a new map.
func JSONMerge(into map[string]any, from map[string]any) {
	for k, v := range from {
		switch v := v.(type) {
		case map[string]any:
			intoValue, ok := into[k]
			if ok {
				intoMap, ok := intoValue.(map[string]any)
				if !ok {
					intoMap = make(map[string]any)
					into[k] = intoMap
				}
				JSONMerge(intoMap, v)
			} else {
				into[k] = v
			}
		default:
			into[k] = v
		}
	}
}

func mapSlice[V, O any](s []V, fn func(V) O) []O {
	out := make([]O, len(s))
	for i, v := range s {
		out[i] = fn(v)
	}
	return out
}

func allKeys(m map[string]any) []JSONPath {
	var keys []JSONPath
	for k, v := range m {
		switch v := v.(type) {
		case map[string]any:
			subKeys := allKeys(v)
			subKeys = mapSlice(subKeys, func(p JSONPath) JSONPath {
				return append(JSONPath{k}, p...)
			})
			keys = append(keys, subKeys...)
		default:
			keys = append(keys, JSONPath{k})
		}
	}
	return keys
}
