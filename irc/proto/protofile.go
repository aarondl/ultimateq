package proto

import (
	"bufio"
	"io"
	"strings"
)

type ProtoFileError struct {
	Msg  string
	Line int
}

func (p ProtoFileError) Error() string {
	return p.Msg
}

func ReadProtoFile(reader io.Reader) (*IrcParser, []error) {
	parser := CreateIrcParser()
	errs := make([]error, 0, 10)

	buf := bufio.NewReader(reader)
	i := 1
	for str, err := buf.ReadString('\n'); str != "" && err == nil; {
		str = strings.TrimRight(str, "\n\r")
		entry := strings.Split(str, " ")
		name := entry[0]
		chain, err := createFragmentChain(entry[1:])
		if err == nil {
			parser.AddIrcHandler(name, chain)
		} else {
			errs = append(errs, ProtoFileError{err.Error(), i})
		}
		str = ""
		i++
	}
	return parser, errs
}
