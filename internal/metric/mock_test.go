// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metric

import (
	"fmt"
	"io"
	"os"
	"testing"
	"unsafe"
)

// FieldOffset is a preprocessor representation of a struct field alignment.
type FieldOffset struct {
	// Name of the field.
	Name string

	// Offset of the field in bytes.
	//
	// To compute this at compile time use unsafe.Offsetof.
	Offset uintptr
}

// Aligned8Byte returns if all fields are aligned modulo 8-bytes.
//
// Error messaging is printed to out for any fileds determined misaligned.
func Aligned8Byte(fields []FieldOffset, out io.Writer) bool {
	misaligned := make([]FieldOffset, 0)
	for _, f := range fields {
		if f.Offset%8 != 0 {
			misaligned = append(misaligned, f)
		}
	}

	if len(misaligned) == 0 {
		return true
	}

	fmt.Fprintln(out, "struct fields not aligned for 64-bit atomic operations:")
	for _, f := range misaligned {
		fmt.Fprintf(out, "  %s: %d-byte offset\n", f.Name, f.Offset)
	}

	return false
}

// Ensure struct alignment prior to running tests.
func TestMain(m *testing.M) {
	fields := []FieldOffset{
		{
			Name:   "Batch.Measurments",
			Offset: unsafe.Offsetof(Batch{}.Measurements),
		},
		{
			Name:   "Measurement.Number",
			Offset: unsafe.Offsetof(Measurement{}.Number),
		},
	}
	if !Aligned8Byte(fields, os.Stderr) {
		os.Exit(1)
	}

	os.Exit(m.Run())
}
