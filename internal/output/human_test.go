package output

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout captures stdout output from a function.
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

// captureStderr captures stderr output from a function.
func captureStderr(f func()) string {
	old := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	f()

	w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestHumanFormatter_Print(t *testing.T) {
	formatter := &HumanFormatter{}

	type TestStruct struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Count int    `json:"count"`
	}

	data := TestStruct{
		Name:  "John Doe",
		Email: "john@example.com",
		Count: 42,
	}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	// Verify output contains expected fields
	if !strings.Contains(output, "name: John Doe") {
		t.Errorf("Print() output should contain 'name: John Doe', got: %s", output)
	}
	if !strings.Contains(output, "email: john@example.com") {
		t.Errorf("Print() output should contain 'email: john@example.com', got: %s", output)
	}
	if !strings.Contains(output, "count: 42") {
		t.Errorf("Print() output should contain 'count: 42', got: %s", output)
	}
}

func TestHumanFormatter_PrintTable(t *testing.T) {
	formatter := &HumanFormatter{}

	headers := []string{"ID", "Name", "Status"}
	rows := [][]string{
		{"1", "Alice", "Active"},
		{"2", "Bob", "Inactive"},
	}

	output := captureStdout(func() {
		formatter.PrintTable(headers, rows)
	})

	// Verify headers are present
	if !strings.Contains(output, "ID") {
		t.Errorf("PrintTable() should contain 'ID' header, got: %s", output)
	}
	if !strings.Contains(output, "Name") {
		t.Errorf("PrintTable() should contain 'Name' header, got: %s", output)
	}
	if !strings.Contains(output, "Status") {
		t.Errorf("PrintTable() should contain 'Status' header, got: %s", output)
	}

	// Verify separators are present (dashes under headers)
	if !strings.Contains(output, "--") {
		t.Errorf("PrintTable() should contain separator dashes, got: %s", output)
	}

	// Verify row data is present
	if !strings.Contains(output, "Alice") {
		t.Errorf("PrintTable() should contain 'Alice', got: %s", output)
	}
	if !strings.Contains(output, "Bob") {
		t.Errorf("PrintTable() should contain 'Bob', got: %s", output)
	}
	if !strings.Contains(output, "Active") {
		t.Errorf("PrintTable() should contain 'Active', got: %s", output)
	}
	if !strings.Contains(output, "Inactive") {
		t.Errorf("PrintTable() should contain 'Inactive', got: %s", output)
	}
}

func TestHumanFormatter_PrintError(t *testing.T) {
	formatter := &HumanFormatter{}
	testErr := errors.New("something went wrong")

	output := captureStderr(func() {
		formatter.PrintError(testErr)
	})

	// Verify error message is present
	if !strings.Contains(output, "Error:") {
		t.Errorf("PrintError() should contain 'Error:', got: %s", output)
	}
	if !strings.Contains(output, "something went wrong") {
		t.Errorf("PrintError() should contain error message, got: %s", output)
	}
}

func TestHumanFormatter_PrintMessage(t *testing.T) {
	formatter := &HumanFormatter{}
	msg := "Operation completed successfully"

	output := captureStdout(func() {
		formatter.PrintMessage(msg)
	})

	// Verify message is present with newline
	expected := msg + "\n"
	if output != expected {
		t.Errorf("PrintMessage() = %q, want %q", output, expected)
	}
}

func TestPrintStruct_Nested(t *testing.T) {
	formatter := &HumanFormatter{}

	type Address struct {
		City    string `json:"city"`
		Country string `json:"country"`
	}

	type Person struct {
		Name    string  `json:"name"`
		Address Address `json:"address"`
	}

	data := Person{
		Name: "Jane Doe",
		Address: Address{
			City:    "New York",
			Country: "USA",
		},
	}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	// Verify nested struct output
	if !strings.Contains(output, "name: Jane Doe") {
		t.Errorf("Print() should contain 'name: Jane Doe', got: %s", output)
	}
	if !strings.Contains(output, "address:") {
		t.Errorf("Print() should contain 'address:', got: %s", output)
	}
	if !strings.Contains(output, "city: New York") {
		t.Errorf("Print() should contain 'city: New York', got: %s", output)
	}
	if !strings.Contains(output, "country: USA") {
		t.Errorf("Print() should contain 'country: USA', got: %s", output)
	}
}

func TestPrintStruct_Empty(t *testing.T) {
	formatter := &HumanFormatter{}

	type EmptyStruct struct{}

	data := EmptyStruct{}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	// Empty struct should produce no output (or just whitespace)
	trimmed := strings.TrimSpace(output)
	if trimmed != "" {
		t.Errorf("Print() for empty struct should produce no output, got: %q", output)
	}
}

func TestHumanFormatter_Print_Nil(t *testing.T) {
	formatter := &HumanFormatter{}

	output := captureStdout(func() {
		err := formatter.Print(nil)
		if err != nil {
			t.Errorf("Print(nil) returned error: %v", err)
		}
	})

	// Nil input should produce no output
	if output != "" {
		t.Errorf("Print(nil) should produce no output, got: %q", output)
	}
}

func TestHumanFormatter_Print_Slice(t *testing.T) {
	formatter := &HumanFormatter{}

	type Item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	data := []Item{
		{ID: 1, Name: "First"},
		{ID: 2, Name: "Second"},
	}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	// Verify slice output has numbered items
	if !strings.Contains(output, "[1]") {
		t.Errorf("Print() for slice should contain '[1]', got: %s", output)
	}
	if !strings.Contains(output, "[2]") {
		t.Errorf("Print() for slice should contain '[2]', got: %s", output)
	}
	if !strings.Contains(output, "First") {
		t.Errorf("Print() for slice should contain 'First', got: %s", output)
	}
	if !strings.Contains(output, "Second") {
		t.Errorf("Print() for slice should contain 'Second', got: %s", output)
	}
}

func TestHumanFormatter_Print_Map(t *testing.T) {
	formatter := &HumanFormatter{}

	data := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	// Verify map output
	if !strings.Contains(output, "key1: value1") {
		t.Errorf("Print() for map should contain 'key1: value1', got: %s", output)
	}
	if !strings.Contains(output, "key2: value2") {
		t.Errorf("Print() for map should contain 'key2: value2', got: %s", output)
	}
}

func TestHumanFormatter_Print_Pointer(t *testing.T) {
	formatter := &HumanFormatter{}

	type TestStruct struct {
		Value string `json:"value"`
	}

	data := &TestStruct{Value: "pointer test"}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	// Verify pointer is dereferenced
	if !strings.Contains(output, "value: pointer test") {
		t.Errorf("Print() for pointer should contain 'value: pointer test', got: %s", output)
	}
}

func TestHumanFormatter_Print_NilPointer(t *testing.T) {
	formatter := &HumanFormatter{}

	type TestStruct struct {
		Value string `json:"value"`
	}

	var data *TestStruct = nil

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print(nil pointer) returned error: %v", err)
		}
	})

	// Nil pointer should produce no output
	if output != "" {
		t.Errorf("Print(nil pointer) should produce no output, got: %q", output)
	}
}

func TestHumanFormatter_Print_PlainValue(t *testing.T) {
	formatter := &HumanFormatter{}

	output := captureStdout(func() {
		err := formatter.Print("hello world")
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	if !strings.Contains(output, "hello world") {
		t.Errorf("Print(string) should contain 'hello world', got: %q", output)
	}
}

func TestHumanFormatter_Print_SliceOfStrings(t *testing.T) {
	formatter := &HumanFormatter{}

	data := []string{"alpha", "beta", "gamma"}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	if !strings.Contains(output, "[1] alpha") {
		t.Errorf("Print() should contain '[1] alpha', got: %s", output)
	}
	if !strings.Contains(output, "[2] beta") {
		t.Errorf("Print() should contain '[2] beta', got: %s", output)
	}
}

func TestHumanFormatter_Print_NestedPointerStruct(t *testing.T) {
	formatter := &HumanFormatter{}

	type Inner struct {
		Value string `json:"value"`
	}

	type Outer struct {
		Inner *Inner `json:"inner"`
	}

	data := Outer{Inner: &Inner{Value: "nested"}}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	if !strings.Contains(output, "inner:") {
		t.Errorf("Print() should contain 'inner:', got: %s", output)
	}
	if !strings.Contains(output, "value: nested") {
		t.Errorf("Print() should contain 'value: nested', got: %s", output)
	}
}

func TestHumanFormatter_Print_SliceOfPointerStructs(t *testing.T) {
	formatter := &HumanFormatter{}

	type Item struct {
		Name string `json:"name"`
	}

	data := []*Item{
		{Name: "first"},
		{Name: "second"},
	}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	if !strings.Contains(output, "[1]") {
		t.Errorf("Print() should contain '[1]', got: %s", output)
	}
	if !strings.Contains(output, "first") {
		t.Errorf("Print() should contain 'first', got: %s", output)
	}
}

func TestHumanFormatter_Print_FieldWithoutJSONTag(t *testing.T) {
	formatter := &HumanFormatter{}

	type MyStruct struct {
		Name string
	}

	data := MyStruct{Name: "test"}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	// Should use field name when no json tag
	if !strings.Contains(output, "Name: test") {
		t.Errorf("Print() should contain 'Name: test', got: %s", output)
	}
}

func TestHumanFormatter_Print_FieldWithDashJSONTag(t *testing.T) {
	formatter := &HumanFormatter{}

	type MyStruct struct {
		Name   string `json:"name"`
		Hidden string `json:"-"`
	}

	data := MyStruct{Name: "visible", Hidden: "hidden"}

	output := captureStdout(func() {
		err := formatter.Print(data)
		if err != nil {
			t.Errorf("Print() returned error: %v", err)
		}
	})

	if !strings.Contains(output, "name: visible") {
		t.Errorf("Print() should contain 'name: visible', got: %s", output)
	}
}
