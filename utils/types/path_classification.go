package types

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// PathClassification defines the classification of a path
type PathClassification int

// ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

// PathClassification constants
//   - PathClassificationNone: The path does not exist in the courses table
//   - PathClassificationAncestor: The path is an ancestor of a course path (parent, grandparent, etc.)
//   - PathClassificationCourse: The path is an exact match to a course path
//   - PathClassificationDescendant: The path is a descendant of a course path (child, grandchild, etc.)
const (
	PathClassificationNone PathClassification = iota
	PathClassificationAncestor
	PathClassificationCourse     PathClassification = 2
	PathClassificationDescendant PathClassification = 3
)
