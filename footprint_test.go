package hyperex

import (
	"testing"
	"unsafe"
)

// Footprint budgets for hot-path structs (amd64/arm64).
// Handle Sizeof only — heap content (maps, slices, string data) is separate.
//
// Justification:
//   Value       — tagged union (kind+bits+ptr); documented as 32 B
//   Engine      — config/builtins/optimizer/typeChecker + RWMutex + Stats atomics
//   Program     — source + bytecode + engine refs + expect flags
//   vm          — pooled executor frame (stack/env/limits + Run context)
//   instruction — Op + 2 ints + Name string header
//   bytecode    — four slice headers (instructions/constants/names/params)
const (
	maxValueBytes       = 32
	maxEngineBytes      = 96
	maxProgramBytes     = 48
	maxVMBytes          = 144
	maxInstructionBytes = 40
	maxBytecodeBytes    = 96
)

func TestFootprint(t *testing.T) {
	entries := []struct {
		name string
		size uintptr
		max  uintptr
	}{
		{"Value", unsafe.Sizeof(Value{}), maxValueBytes},
		{"Engine", unsafe.Sizeof(Engine{}), maxEngineBytes},
		{"Program", unsafe.Sizeof(Program{}), maxProgramBytes},
		{"execVM", unsafe.Sizeof(execVM{}), maxVMBytes},
		{"instruction", unsafe.Sizeof(instruction{}), maxInstructionBytes},
		{"bytecode", unsafe.Sizeof(bytecode{}), maxBytecodeBytes},
	}
	for _, e := range entries {
		t.Run(e.name, func(t *testing.T) {
			if e.size > e.max {
				t.Fatalf("%s: unsafe.Sizeof = %d bytes > max %d (footprint regression)",
					e.name, e.size, e.max)
			}
			t.Logf("%s: %d bytes (max %d)", e.name, e.size, e.max)
		})
	}
}
