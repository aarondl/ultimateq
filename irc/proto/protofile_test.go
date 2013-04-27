package proto

import (
	"io"
	. "launchpad.net/gocheck"
)

type testFile struct {
	line, index int
	buffer      [][]byte
}

func (t *testFile) LoadData(buffer []string) {
	t.buffer = make([][]byte, len(buffer))
	for i := 0; i < len(buffer); i++ {
		t.buffer[i] = []byte(buffer[i])
	}
}

func (t *testFile) Read(p []byte) (int, error) {
	var n int = 0
	var value []byte
	if t.line < len(t.buffer) {
		splits := (t.line % 3) + 1
		length := len(t.buffer[t.line])
		if splits > length {
			splits = length
		}
		index := t.index + int(float64(length)/float64(splits)+0.5)
		if index > length {
			index = length
		}
		value = t.buffer[t.line][t.index:index]
		t.index = index
		if index == length {
			t.index = 0
			t.line++
		}

		for i, value := range value {
			p[i], n = value, n+1
		}
		if t.line == len(t.buffer) {
			return n, io.EOF
		} else {
			return n, nil
		}
	}
	return 0, io.EOF
}

func (s *testSuite) TestTestFile(c *C) {
	var (
		file = &testFile{}
		buf  = make([]byte, 50)
		n    int
		err  error
	)

	file.LoadData([]string{"JOIN #chan\n", "PING :id\n", "Pi"})

	expect := []string{"JOIN #chan\n", "PING ", ":id\n", "P", "i"}
	for _, v := range expect {
		n, err = file.Read(buf)
		c.Assert(string(buf[:n]), Equals, v)
	}
	c.Assert(err, Equals, io.EOF)
	n, err = file.Read(buf)
	c.Assert(n, Equals, 0)
	c.Assert(err, Equals, io.EOF)
}

func (s *testSuite) TestReadProtoFile(c *C) {
	file := &testFile{}
	file.LoadData([]string{"JOIN #chan\n", "PING :id\n", "PONG :id"})
	parser, errs := ReadProtoFile(file)
	c.Assert(len(errs), Equals, 0)
	c.Assert(parser.handlers["JOIN"], NotNil)
}

func (s *testSuite) TestReadProtoFile_Errors(c *C) {
	file := &testFile{}
	file.LoadData([]string{"JOIN #c*han\n", "PING :id\n", "PONG :id"})
	_, errs := ReadProtoFile(file)
	c.Assert(len(errs), Not(Equals), 0)
	perr, ok := errs[0].(ProtoFileError)
	c.Check(ok, Equals, true)
	c.Assert(perr.Line, Equals, 1)
	c.Assert(perr.Error(), Equals, errIllegalIdentifiers.Error())
}
