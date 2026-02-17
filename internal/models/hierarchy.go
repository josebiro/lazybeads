package models

import "strings"

// ParentID returns the parent issue ID derived from the dot-notation hierarchy.
// "bb-abc.1" → "bb-abc", "bb-abc.1.2" → "bb-abc.1", "bb-abc" → ""
func ParentID(id string) string {
	lastDot := strings.LastIndex(id, ".")
	if lastDot == -1 {
		return ""
	}
	return id[:lastDot]
}

// Depth returns the nesting level based on dot separators.
// "bb-abc" → 0, "bb-abc.1" → 1, "bb-abc.1.2" → 2
func Depth(id string) int {
	return strings.Count(id, ".")
}

// IsDirectChildOf returns true if childID is an immediate child of parentID.
// "bb-abc.1" is a direct child of "bb-abc".
// "bb-abc.1.2" is NOT a direct child of "bb-abc" (it's a grandchild).
func IsDirectChildOf(childID, parentID string) bool {
	if !strings.HasPrefix(childID, parentID+".") {
		return false
	}
	remainder := childID[len(parentID)+1:]
	return !strings.Contains(remainder, ".")
}
