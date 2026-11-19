package base

import "testing"

func TestParseInfoOutput(t *testing.T) {
	for _, output := range []string{"", " \r\n\t\n", "log message\nnot JSON\n"} {
		if _, err := ParseInfoOutput[InfoBase]([]byte(output)); err == nil {
			t.Fatalf("accepted invalid info output %q", output)
		}
	}
	for _, output := range []string{
		`{"status":"ready"}`,
		"log message\r\n{\"status\":\"ready\"}\r\n \t\n",
	} {
		info, err := ParseInfoOutput[InfoBase]([]byte(output))
		if err != nil || info.Status != "ready" {
			t.Fatalf("info output %q: got %+v, error %v", output, info, err)
		}
	}
}
