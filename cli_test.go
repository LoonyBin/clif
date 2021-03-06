package clif

import (
	"bytes"
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"io"
	"testing"
)

type testCliAlias interface {
	Hello() int
}

type testCliInject struct {
	Foo int
}

func (this *testCliInject) Hello() int {
	return this.Foo
}

func TestCliRun(t *testing.T) {
	Convey("Run cli command", t, func() {
		called := 0
		var handledErr error
		Die = func(msg string, args ...interface{}) {
			panic(fmt.Sprintf(msg, args...))
		}
		Exit = func(s int) {
			panic(fmt.Sprintf("Exit %d", s))
		}
		namedActual := make(map[string]interface{})

		c := New("foo", "1.0.0", "").
			New("bar", "", func(c *Cli, o *Command) error {
			called = 1
			return nil
		}).
			New("zoing", "", func(x *testCliInject) error {
			called = x.Foo
			return nil
		}).
			New("zoing2", "", func(x testCliAlias) error {
			called = x.Hello()
			return nil
		}).
			New("oops", "", func(x io.Writer) error {
			panic("Should never be called")
			return nil
		}).
			New("errme", "", func() error {
			return fmt.Errorf("I error!")
		}).
			New("named", "", func(named NamedParameters) {
			namedActual = map[string]interface{}(named)
		}).
			New("named2", "", func(x testCliAlias, named NamedParameters, y *testCliInject) {
			namedActual = map[string]interface{}(named)
		}).
			Register(&testCliInject{
			Foo: 100,
		}).
			RegisterAs("clif.testCliAlias", &testCliInject{
			Foo: 200,
		})

		cmdInvalid := NewCommand("bla", "Dont use me", func() {})
		argInvalid := NewArgument("something", "..", "", false, false)
		argInvalid.SetParse(func(name, value string) (string, error) {
			return "", fmt.Errorf("Never works!")
		})
		cmdInvalid.AddArgument(argInvalid)
		c.Add(cmdInvalid)

		Convey("Run existing method", func() {
			c.RunWith([]string{"bar"})
			So(handledErr, ShouldBeNil)
			So(called, ShouldEqual, 1)
		})
		Convey("Run existing method with injection", func() {
			c.RunWith([]string{"zoing"})
			So(handledErr, ShouldBeNil)
			So(called, ShouldEqual, 100)
		})
		Convey("Run existing method with interface injection", func() {
			c.RunWith([]string{"zoing2"})
			So(handledErr, ShouldBeNil)
			So(called, ShouldEqual, 200)
		})
		Convey("Run existing method with named parameters", func() {
			c.RegisterNamed("foo", "bar")
			c.RegisterNamed("baz", 213)
			c.RunWith([]string{"named"})
			So(namedActual, ShouldResemble, map[string]interface{}{"foo": "bar", "baz": 213})
		})
		Convey("Run existing method with named parameters on arbitrary position", func() {
			c.RegisterNamed("foo", "bar")
			c.RegisterNamed("baz", 213)
			c.RunWith([]string{"named2"})
			So(namedActual, ShouldResemble, map[string]interface{}{"foo": "bar", "baz": 213})
		})
		Convey("Run not existing method", func() {
			So(func() {
				c.RunWith([]string{"baz"})
			}, ShouldPanicWith, "Command \"baz\" unknown")
		})
		Convey("Run without args describes and exits", func() {
			buf := bytes.NewBuffer(nil)
			out := NewOutput(buf, NewDefaultFormatter(map[string]string{}))
			c.SetOutput(out)
			c.RunWith([]string{})
			So(buf.String(), ShouldEqual, DescribeCli(c))
		})
		Convey("Run method with not registered arg fails", func() {
			So(func() {
				c.RunWith([]string{"oops"})
			}, ShouldPanicWith, `Callback parameter of type io.Writer for command "oops" was not found in registry`)
		})
		Convey("Run method with invalid arg fails", func() {
			So(func() {
				c.RunWith([]string{"bla", "bla"})
			}, ShouldPanicWith, "Parse error: Parameter \"something\" invalid: Never works!")
		})
		Convey("Run method with resulting error returns it", func() {
			So(func() {
				c.RunWith([]string{"errme"})
			}, ShouldPanicWith, "Failure in execution: I error!")
		})
	})
}

func TestCliConstruction(t *testing.T) {
	Convey("Create new Cli with commands", t, func() {
		app := New("My App", "1.0.0", "Testing app")
		cb := func() {}

		Convey("Two default commands exist", func() {
			So(len(app.Commands), ShouldEqual, 2)
			Convey("One is \"help\"", func() {
				_, ok := app.Commands["help"]
				So(ok, ShouldBeTrue)
				Convey("Other is \"list\"", func() {
					_, ok := app.Commands["list"]
					So(ok, ShouldBeTrue)
				})
			})
		})

		Convey("Command constructur adds new command", func() {
			app.New("foo", "For fooing", cb)
			So(len(app.Commands), ShouldEqual, 3)
			So(app.Commands["foo"], ShouldNotBeNil)
		})

		Convey("Adding can be used variadic", func() {
			app.New("foo", "For fooing", cb)
			cmds := []*Command{
				NewCommand("foo", "For fooing", cb),
				NewCommand("bar", "For baring", cb),
			}
			app.Add(cmds...)
			So(len(app.Commands), ShouldEqual, 4)
			So(app.Commands["foo"], ShouldNotBeNil)
			So(app.Commands["bar"], ShouldNotBeNil)
		})
	})
}

func TestCliDefaultCommand(t *testing.T) {
	Convey("Change default command of cli", t, func() {
		x := 0
		app := New("My App", "1.0.0", "Testing app").
			SetDefaultCommand("other").
			New("other", "Something else", func() { x += 1 })
		So(app.DefaultCommand, ShouldEqual, "other")
		Convey("Calling default command", func() {
			app.RunWith(nil)
			So(x, ShouldEqual, 1)
		})
	})
}

func TestCliDefaultOptions(t *testing.T) {
	Convey("Adding default options to cli", t, func() {
		app := New("My App", "1.0.0", "Testing app")
		So(len(app.DefaultOptions), ShouldEqual, 0)

		Convey("Using default option creator adds option", func() {
			app.NewDefaultOption("foo", "f", "fooing", "", false, false)
			So(len(app.DefaultOptions), ShouldEqual, 1)
		})

		Convey("Adding default option .. adds them", func() {
			app.AddDefaultOptions(
				NewOption("foo", "f", "fooing", "", false, false),
				NewOption("bar", "b", "baring", "", false, false),
			)
			So(len(app.DefaultOptions), ShouldEqual, 2)
		})

		Convey("Cli default options are not added to command on command create", func() {
			app.NewDefaultOption("foo", "f", "fooing", "", false, false)
			cmd := NewCommand("bla", "bla", func() {})
			app.Add(cmd)
			So(len(cmd.Options), ShouldEqual, len(DefaultOptions))

			Convey("Default options are added in run", func() {
				app.RunWith([]string{"bla"})
				So(len(cmd.Options), ShouldEqual, len(DefaultOptions)+1)
			})
		})
	})
}

func TestCliHeralds(t *testing.T) {
	Convey("Command heralds are add late, in run", t, func() {
		app := New("My App", "1.0.0", "Testing app")
		So(len(app.Commands), ShouldEqual, 2)
		So(len(app.Heralds), ShouldEqual, 0)

		Convey("Heralding command does not add it to list", func() {
			x := 0
			app.Herald(func(c *Cli) *Command {
				return NewCommand("foo", "fooing", func(){ x = 2 })
			})
			So(len(app.Commands), ShouldEqual, 2)
			So(len(app.Heralds), ShouldEqual, 1)

			Convey("Running adds heralded commands", func() {
				app.RunWith([]string{"foo"})
				So(x, ShouldEqual, 2)
				So(len(app.Commands), ShouldEqual, 3)
				So(len(app.Heralds), ShouldEqual, 0)
			})
		})

	})
}
