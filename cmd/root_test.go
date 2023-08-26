package cmd

import (
	"fmt"
	"testing"
)

func TestStart(t *testing.T) {
	// is := is.New(t)

	_, err := ExecuteTest(t, RootCmd, "info")
	fmt.Println("error", err)
	if err != nil {
		t.Fail()
	}
	// is.NoErr(err)
	//t.Skip()
}

func TestInfo(t *testing.T) {
	// is := is.New(t)
	// root := cmd.RootCmd

	// output, err := execute(t, root)

	// is.NoErr(err)
	//t.Skip()
}
