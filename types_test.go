package main

import "testing"

// TestCellValueString tests CellValue string representation
func TestCellValueString(t *testing.T) {
	tests := []struct {
		cell     CellValue
		expected string
	}{
		{CellValue{Value: "hello", IsNull: false}, "hello"},
		{CellValue{Value: "", IsNull: true}, "<NULL>"},
		{CellValue{Value: "", IsNull: false}, ""},
	}

	for _, tc := range tests {
		t.Run(tc.expected, func(t *testing.T) {
			result := tc.cell.String()
			if result != tc.expected {
				t.Errorf("CellValue.String() = %q, want %q", result, tc.expected)
			}
		})
	}
}

// TestColumnTypeHelpers tests ColumnType helper methods
func TestColumnTypeHelpers(t *testing.T) {
	if !ColTypeNumeric.IsNumeric() {
		t.Error("ColTypeNumeric.IsNumeric() should be true")
	}
	if ColTypeText.IsNumeric() {
		t.Error("ColTypeText.IsNumeric() should be false")
	}
	if !ColTypeBoolean.IsBoolean() {
		t.Error("ColTypeBoolean.IsBoolean() should be true")
	}
	if !ColTypeText.IsText() {
		t.Error("ColTypeText.IsText() should be true")
	}
	if !ColTypeDatetime.IsText() {
		t.Error("ColTypeDatetime.IsText() should be true")
	}
}
