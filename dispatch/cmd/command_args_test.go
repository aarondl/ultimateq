package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

var (
	rgxCreator = strings.NewReplacer(
		`(`, `\(`, `)`, `\)`, `]`, `\]`, `[`,
		`\[`, `\`, `\\`, `/`, `\/`, `%v`, `.*`,
		`*`, `\*`,
	)
)

func TestArgs(t *testing.T) {
	t.Parallel()

	helper := func(args ...string) error {
		c := &Command{
			Args: args,
		}

		return c.parseArgs()
	}

	chkStr := func(msg, pattern string) error {
		pattern = `^` + rgxCreator.Replace(pattern) + `$`
		match, err := regexp.MatchString(pattern, msg)
		if err != nil {
			return fmt.Errorf("Error making pattern: \n\t%s\n\t%s", msg, pattern)
		}
		if !match {
			return fmt.Errorf("Unexpected: \n\t%s\n\t%s", msg, pattern)
		}
		return nil
	}

	chkErr := func(err error, pattern string) error {
		if err == nil {
			return fmt.Errorf("Error was nil but expected: %s", pattern)
		}
		return chkStr(err.Error(), pattern)
	}

	var err error

	err = helper("!!!")
	err = chkErr(err, errFmtArgumentForm)
	if err != nil {
		t.Error(err)
	}

	err = helper("~#badarg")
	err = chkErr(err, errFmtArgumentForm)
	if err != nil {
		t.Error(err)
	}

	err = helper("#*badarg")
	err = chkErr(err, errFmtArgumentForm)
	if err != nil {
		t.Error(err)
	}

	err = helper("[opt]", "req")
	err = chkErr(err, errFmtArgumentOrderReq)
	if err != nil {
		t.Error(err)
	}

	err = helper("req...", "[opt]")
	err = chkErr(err, errFmtArgumentOrderOpt)
	if err != nil {
		t.Error(err)
	}

	err = helper("name", "[name]")
	err = chkErr(err, errFmtArgumentDupName)
	if err != nil {
		t.Error(err)
	}

	err = helper("vrgs...", "vrgs2...")
	err = chkErr(err, errFmtArgumentDupVargs)
	if err != nil {
		t.Error(err)
	}

	err = helper("[opt]", "#chan1")
	err = chkErr(err, errFmtArgumentOrderChan)
	if err != nil {
		t.Error(err)
	}

	err = helper("vargs...", "#chan1")
	err = chkErr(err, errFmtArgumentOrderChan)
	if err != nil {
		t.Error(err)
	}

	err = helper("req", "#chan1")
	err = chkErr(err, errFmtArgumentOrderChan)
	if err != nil {
		t.Error(err)
	}

	err = helper("#chan1", "#chan2")
	err = chkErr(err, errFmtArgumentDupChan)
	if err != nil {
		t.Error(err)
	}
}
