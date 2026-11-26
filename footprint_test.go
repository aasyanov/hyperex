package hyperex

import (
	"testing"
	"unsafe"
)

// Footprint budgets for hot-path structs (amd64/arm64).
// Handle Sizeof / Alignof only - heap content (maps, slices, string data) is separate.
//
// Justification:
//
//	Value       - tagged union (kind+bits+ptr); documented as 32 B
//	Engine      - config/builtins/optimizer/typeChecker + RWMutex + Stats atomics
//	Program     - source + bytecode + engine refs + expect flags
//	vm          - pooled executor frame (stack/env/limits + Run context + lexical bindings)
//	instruction - Op + 2 ints + Name string header
//	bytecode    - four slice headers (instructions/constants/names/params)
const (
	maxValueBytes       = 32
	maxEngineBytes      = 96
	maxProgramBytes     = 40
	maxVMBytes          = 168 // +24 B: saved lexical bindings for nested let/destructure
	maxInstructionBytes = 40
	maxBytecodeBytes    = 96
)

func TestFootprint(t *testing.T) {
	entries := []struct {
		name  string
		size  uintptr
		align uintptr
		max   uintptr
	}{
		{"Value", unsafe.Sizeof(Value{}), unsafe.Alignof(Value{}), maxValueBytes},
		{"Engine", unsafe.Sizeof(Engine{}), unsafe.Alignof(Engine{}), maxEngineBytes},
		{"Program", unsafe.Sizeof(Program{}), unsafe.Alignof(Program{}), maxProgramBytes},
		{"execVM", unsafe.Sizeof(execVM{}), unsafe.Alignof(execVM{}), maxVMBytes},
		{"instruction", unsafe.Sizeof(instruction{}), unsafe.Alignof(instruction{}), maxInstructionBytes},
		{"bytecode", unsafe.Sizeof(bytecode{}), unsafe.Alignof(bytecode{}), maxBytecodeBytes},
	}
	for _, e := range entries {
		t.Run(e.name, func(t *testing.T) {
			if e.size > e.max {
				t.Fatalf("%s: unsafe.Sizeof = %d bytes > max %d (footprint regression)",
					e.name, e.size, e.max)
			}
			if e.align == 0 || e.size%e.align != 0 {
				t.Fatalf("%s: Alignof=%d Sizeof=%d (size must be multiple of align)",
					e.name, e.align, e.size)
			}
			t.Logf("%s: %d bytes align %d (max %d)", e.name, e.size, e.align, e.max)
		})
	}
}
