package analyzer

import (
	"fmt"
	"strings"
)

// Analyzer is responsible for analyzing and comparing Helm values
type Analyzer struct {
	UpstreamValues   map[string]interface{}
	DownstreamValues map[string]interface{}
}

// NewAnalyzer creates a new Analyzer with the given upstream and downstream values
func NewAnalyzer(upstream, downstream map[string]interface{}) *Analyzer {
	return &Analyzer{
		UpstreamValues:   upstream,
		DownstreamValues: downstream,
	}
}

// Analyze compares upstream and downstream values to detect various types of value differences
func (a *Analyzer) Analyze() ValueStatus {
	valueStatus := ValueStatus{
		Redundant:   make(map[string]interface{}),
		Unsupported: make(map[string]interface{}),
		Optimized:   make(map[string]interface{}),
	}

	// First, create a deep copy of the downstream values for optimized output
	for k, v := range a.DownstreamValues {
		valueStatus.Optimized[k] = deepCopy(v)
	}

	// Process the values
	a.detectValuesStatus("", a.UpstreamValues, a.DownstreamValues, &valueStatus)

	// Special handling for service section which often has complex structures
	a.handleServiceValues(&valueStatus)

	return valueStatus
}

// CreateGeneratedValues creates a map of values that exist in downstream but not in upstream
func (a *Analyzer) CreateGeneratedValues() map[string]interface{} {
	generatedValues := make(map[string]interface{})
	for k, v := range a.DownstreamValues {
		if _, exists := a.UpstreamValues[k]; !exists {
			generatedValues[k] = v
		}
	}
	return generatedValues
}

// handleServiceValues gives special attention to service configurations
func (a *Analyzer) handleServiceValues(status *ValueStatus) {
	upService, hasUpService := a.UpstreamValues["service"].(map[string]interface{})
	downService, hasDownService := a.DownstreamValues["service"].(map[string]interface{})

	if !hasUpService || !hasDownService {
		return
	}

	// Extract service from optimized values for modification
	optimizedService, hasOptimizedService := status.Optimized["service"].(map[string]interface{})
	if !hasOptimizedService {
		return
	}

	for key, downVal := range downService {
		upVal, exists := upService[key]
		if !exists {
			// This key is unsupported
			if _, hasUnsupportedService := status.Unsupported["service"].(map[string]interface{}); !hasUnsupportedService {
				status.Unsupported["service"] = make(map[string]interface{})
			}
			status.Unsupported["service"].(map[string]interface{})[key] = downVal
		} else if equalValues(downVal, upVal) {
			// This key is redundant
			if _, hasRedundantService := status.Redundant["service"].(map[string]interface{}); !hasRedundantService {
				status.Redundant["service"] = make(map[string]interface{})
			}
			status.Redundant["service"].(map[string]interface{})[key] = downVal

			// Remove from optimized
			delete(optimizedService, key)
		}
	}

	// Update optimized service section
	if len(optimizedService) > 0 {
		status.Optimized["service"] = optimizedService
	} else {
		delete(status.Optimized, "service")
	}
}

// detectValuesStatus recursively compares upstream and downstream values
func (a *Analyzer) detectValuesStatus(path string, upstream, downstream map[string]interface{}, status *ValueStatus) {
	// First pass: identify unsupported keys
	for key, downVal := range downstream {
		currentPath := joinPath(path, key)

		// Check if the key exists in upstream
		_, exists := upstream[key]
		if !exists {
			// Key in downstream doesn't exist in upstream, it's unsupported
			setNestedValue(status.Unsupported, currentPath, downVal)
		}
	}

	// Second pass: identify redundant keys and process nested maps
	for key, downVal := range downstream {
		currentPath := joinPath(path, key)

		// Skip if already identified as unsupported
		if isPathInMap(status.Unsupported, currentPath) {
			continue
		}

		// Check if the key exists in upstream
		upVal, exists := upstream[key]
		if !exists {
			continue // Already handled in first pass
		}

		// If both are maps, recurse
		downMap, downIsMap := downVal.(map[string]interface{})
		upMap, upIsMap := upVal.(map[string]interface{})

		if downIsMap && upIsMap {
			// Recursively process nested maps
			a.detectValuesStatus(currentPath, upMap, downMap, status)
		} else if equalValues(downVal, upVal) {
			// Values are the same, this is redundant
			setNestedValue(status.Redundant, currentPath, downVal)

			// Remove redundant value from optimized map
			removeNestedValue(status.Optimized, currentPath)
		}
	}
}

// CountNestedKeys counts all nested keys in a map (including nested maps)
func CountNestedKeys(m map[string]interface{}) int {
	count := 0
	for _, v := range m {
		count++
		if nestedMap, isMap := v.(map[string]interface{}); isMap {
			count += CountNestedKeys(nestedMap)
		}
	}
	return count
}

// deepCopy creates a deep copy of an interface{}
func deepCopy(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	// Handle maps (nested structures)
	if srcMap, isMap := src.(map[string]interface{}); isMap {
		dstMap := make(map[string]interface{})
		for k, v := range srcMap {
			dstMap[k] = deepCopy(v)
		}
		return dstMap
	}

	// Handle slices/arrays
	if srcSlice, isSlice := src.([]interface{}); isSlice {
		dstSlice := make([]interface{}, len(srcSlice))
		for i, v := range srcSlice {
			dstSlice[i] = deepCopy(v)
		}
		return dstSlice
	}

	// For basic types (ints, strings, bools, etc.), just return as is
	return src
}

// isPathInMap checks if a given path exists in a nested map
func isPathInMap(m map[string]interface{}, path string) bool {
	parts := strings.Split(path, ".")
	current := m

	for i := 0; i < len(parts); i++ {
		if i == len(parts)-1 {
			_, exists := current[parts[i]]
			return exists
		}

		next, exists := current[parts[i]]
		if !exists {
			return false
		}

		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return false
		}

		current = nextMap
	}

	return false
}

// joinPath creates a dot-notation path
func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

// equalValues checks if two values are equal
func equalValues(a, b interface{}) bool {
	// Handle nil cases
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Type-specific comparisons
	switch aTyped := a.(type) {
	case map[string]interface{}:
		// Compare maps
		bMap, ok := b.(map[string]interface{})
		if !ok || len(aTyped) != len(bMap) {
			return false
		}

		for k, v := range aTyped {
			bVal, exists := bMap[k]
			if !exists || !equalValues(v, bVal) {
				return false
			}
		}
		return true

	case []interface{}:
		// Compare slices
		bSlice, ok := b.([]interface{})
		if !ok || len(aTyped) != len(bSlice) {
			return false
		}

		for i, v := range aTyped {
			if !equalValues(v, bSlice[i]) {
				return false
			}
		}
		return true

	default:
		// For basic types, use string representation comparison
		aStr := fmt.Sprintf("%v", a)
		bStr := fmt.Sprintf("%v", b)
		return aStr == bStr
	}
}

// setNestedValue sets a value in a nested map based on dot notation path
func setNestedValue(m map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	lastIndex := len(parts) - 1

	// Navigate to the correct nested level
	current := m
	for i := 0; i < lastIndex; i++ {
		part := parts[i]

		// Create nested map if it doesn't exist
		if _, exists := current[part]; !exists {
			current[part] = make(map[string]interface{})
		}

		// Cast to map for next iteration
		current = current[part].(map[string]interface{})
	}

	// Set the value at the final level
	current[parts[lastIndex]] = value
}

// removeNestedValue removes a value from a nested map based on dot notation path
func removeNestedValue(m map[string]interface{}, path string) {
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return
	}

	// If there's only one level, just delete it from the map
	if len(parts) == 1 {
		delete(m, parts[0])
		return
	}

	// For nested paths, we need to traverse the structure
	current := m
	var parent map[string]interface{}
	var lastKey string

	// Navigate through the nested levels until the second-to-last element
	for i := 0; i < len(parts)-1; i++ {
		key := parts[i]

		// If this level doesn't exist, nothing to remove
		nextLevel, exists := current[key]
		if !exists {
			return
		}

		// Cast to map to continue traversal
		var ok bool
		nextMap, ok := nextLevel.(map[string]interface{})
		if !ok {
			// If it's not a map, we can't go deeper
			return
		}

		// Save parent for later cleanup
		parent = current
		lastKey = key
		current = nextMap
	}

	// Delete the leaf key
	lastPart := parts[len(parts)-1]
	delete(current, lastPart)

	// Clean up empty parent maps
	if len(current) == 0 && parent != nil {
		delete(parent, lastKey)
	}
}
