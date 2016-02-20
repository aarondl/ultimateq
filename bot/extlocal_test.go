package bot

import (
	"errors"
	"testing"

	"github.com/inconshreveable/log15"
)

type testFakeExt struct {
	b *Bot

	handlerID int64
}

func (f *testFakeExt) Init(b *Bot) error {
	f.b = b
	return nil
}

func (f *testFakeExt) Deinit(b *Bot) error {
	f.b = nil
	return nil
}

func TestRegister(t *testing.T) {
	defer func() {
		extensions = make(map[string]Extension)
	}()

	fake := &testFakeExt{}
	RegisterExtension("fakeext", fake)

	if got := extensions["fakeext"]; got != fake {
		t.Errorf("Fake extension expected: (%T)#%v (%T)#%v", got, got, fake, fake)
	}
}

var errTestExtensionFailure = errors.New("ext error")

type testFakeExtErr struct {
}

func (t *testFakeExtErr) Init(*Bot) error {
	return errTestExtensionFailure
}
func (t *testFakeExtErr) Deinit(*Bot) error {
	return errTestExtensionFailure
}

func TestBot_Extensions(t *testing.T) {
	defer func() {
		extensions = make(map[string]Extension)
	}()

	fake := &testFakeExt{}
	RegisterExtension("fakeext", fake)

	b := &Bot{}
	b.Logger = log15.New()
	b.Logger.SetHandler(log15.DiscardHandler())
	if err := b.initLocalExtensions(); err != nil {
		t.Error(err)
	}

	if fake.b != b {
		t.Error("Expected bot to be passed to extension")
	}

	if err := b.deinitLocalExtensions(); err != nil {
		t.Error(err)
	}

	if fake.b != nil {
		t.Error("Expected bot to be erased by destructor")
	}
}

func TestBot_ExtensionsErrors(t *testing.T) {
	defer func() {
		extensions = make(map[string]Extension)
	}()

	fake := &testFakeExtErr{}
	RegisterExtension("fakeext", fake)

	b := &Bot{}
	b.Logger = log15.New()
	b.Logger.SetHandler(log15.DiscardHandler())

	if err := b.initLocalExtensions(); err == nil {
		t.Error("Expected an error about failure to init")
	}

	if err := b.deinitLocalExtensions(); err == nil {
		t.Error("Expected an error about failure to deinit")
	}
}
