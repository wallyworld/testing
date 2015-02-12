// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package testing

import (
	jc "github.com/juju/testing/checkers"
	gc "gopkg.in/check.v1"
)

// TODO(ericsnow) Drop FakeCall.Receiver? In the cases that it matters
// it can be set as the first arg...  Still, it does make sense to treat
// the receiver specially.

// FakeCall records the name of a called function and the passed args.
type FakeCall struct {
	// Receiver is the fake for which the function was called. It is
	// not required, particularly if the function is not a method.
	Receiver interface{}

	// Funcname is the name of the function that was called.
	FuncName string

	// Args is the set of arguments passed to the function. They are
	// in the same order as the function's parameters
	Args []interface{}
}

// Fake is used in testing to stand in for some other value, to record
// all calls to faked methods/functions, and to allow users to set the
// values that are returned from those calls. Fake is intended to be
// embedded in another struct that will define the methods to track:
//
//    type fakeConn struct {
//        *testing.Fake
//        Response []byte
//    }
//
//    func newFakeConn() *fakeConn {
//        return &fakeConn{
//            Fake: &testing.Fake{},
//        }
//    }
//
//    // Send implements Connection.
//    func (fc *fakeConn) Send(request string) []byte {
//        fc.MethodCall(fc, "Send", request)
//        return fc.Response, fc.NextErr()
//    }
//
// As demonstrated in the example, embed a pointer to testing.Fake. This
// allows a single testing.Fake to be shared between multiple fakes.
//
// Error return values are set through Fake.Errors. Set it to the errors
// you want returned (or use the convenience method `SetErrors`). The
// `NextErr` method returns the errors from Fake.Errors in sequence,
// falling back to `DefaultError` when the sequence is exhausted. Thus
// each fake method should call `NextErr` to get its error return value.
//
// To validate calls made to the fake in a test, check Fake.Calls or
// call the CheckCalls method:
//
//    c.Check(s.fake.Calls, jc.DeepEquals, []FakeCall{{
//        Receiver: s.fake,
//        FuncName: "Send",
//        Args: []interface{}{
//            expected,
//        },
//    }})
//
//    // We don't care about the receiver here.
//    s.fake.CheckCalls(c, []FakeCall{{
//        FuncName: "Send",
//        Args: []interface{}{
//            expected,
//        },
//    }})
//
// Not only is Fake useful for building a interface implementation to
// use in testing (e.g. a network API client), it is also useful in
// regular function patching situations:
//
//    type myFake struct {
//        *testing.Fake
//    }
//
//    func (f *myFake) SomeFunc(arg interface{}) error {
//        f.AddCall("SomeFunc", arg)
//        return f.NextErr()
//    }
//
//    s.PatchValue(&somefunc, s.myFake.SomeFunc)
//
// This allows for easily monitoring the args passed to the patched
// func, as well as controlling the return value from the func in a
// clean manner (by simply setting the correct fake field).
type Fake struct {
	// Calls is the list of calls that have been registered on the fake
	// (i.e. made on the fake's methods), in the order that they were
	// made.
	Calls []FakeCall

	// Errors holds the list of error return values to use for
	// successive calls to methods that return an error. Each call
	// pops the next error off the list. An empty list (the default)
	// implies a nil error. nil may be precede actual errors in the
	// list, which means that the first calls will succeed, followed
	// by the failure. All this is facilitated through the Err method.
	Errors []error

	// DefaultError is the default error (when Errors is empty). The
	// typical Fake usage will leave this nil (i.e. no error).
	DefaultError error
}

// TODO(ericsnow) Add something similar to NextErr for all return values
// using reflection?

// NextErr returns the error that should be returned on the nth call to
// any method on the fake. It should be called for the error return in
// all faked methods.
func (f *Fake) NextErr() error {
	if len(f.Errors) == 0 {
		return f.DefaultError
	}
	err := f.Errors[0]
	f.Errors = f.Errors[1:]
	return err
}

// AddCall records a faked function call for later inspection using the
// CheckCalls method. In the case of methods the receiver is not
// recorded. However, the receiver is not significant for most testing.
// All faked functions should call AddCall (or perhaps MethodCall in
// the case of methods).
func (f *Fake) AddCall(funcName string, args ...interface{}) int {
	f.Calls = append(f.Calls, FakeCall{
		FuncName: funcName,
		Args:     args,
	})
	return len(f.Calls) - 1
}

// TODO(ericsnow) Drop MethodCall?

// MethodCall records a faked method call for later inspection using
// the CheckCalls method. Unlike AddCall, MethodCall tracks the
// receiver of the called method.
func (f *Fake) MethodCall(receiver interface{}, funcName string, args ...interface{}) int {
	index := f.AddCall(funcName, args...)
	f.Calls[index].Receiver = receiver
	return index
}

// SetErrors sets the sequence of error returns for the fake. Each call
// to Err (thus each fake method call) pops an error off the front. So
// frontloading nil here will allow calls to pass, followed by a
// failure.
func (f *Fake) SetErrors(errors ...error) {
	f.Errors = errors
}

// CheckCalls verifies that the history of calls on the fake's methods
// matches the expected calls. This includes checking the receiver. If
// the receiver is not significant then the faked method should not set
// it.
func (f *Fake) CheckCalls(c *gc.C, expected []FakeCall) {
	c.Check(f.Calls, jc.DeepEquals, expected)
}

// CheckCall checks the recorded call at the given index against the
// provided values. If the index is out of bounds then the check fails.
// The receiver of the call is significant for a test then it can be
// checked separately:
//
//     c.Check(myfake.Calls[index].Receiver, gc.Equals, expected)
func (f *Fake) CheckCall(c *gc.C, index int, funcName string, args ...interface{}) {
	if !c.Check(index, jc.LessThan, len(f.Calls)) {
		return
	}
	call := f.Calls[index]
	expected := FakeCall{
		Receiver: call.Receiver,
		FuncName: funcName,
		Args:     args,
	}
	c.Check(call, jc.DeepEquals, expected)
}

// CheckCallNames verifies that the in-order list of called method names
// matches the expected calls.
func (f *Fake) CheckCallNames(c *gc.C, expected ...string) {
	var funcNames []string
	for _, call := range f.Calls {
		funcNames = append(funcNames, call.FuncName)
	}
	c.Check(funcNames, jc.DeepEquals, expected)
}
