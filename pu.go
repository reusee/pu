package pu

import (
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

type Context struct {
	Input  []byte
	Offset int
	Proc   Proc
	Err    error
}

type Proc func(Context) Context

func (c Context) Read() (rune, int) {
	return utf8.DecodeRune(c.Input[c.Offset:])
}

func (c Context) IsEnd() bool {
	return c.Offset >= len(c.Input)
}

func (c Context) Cont(cont Proc) Context {
	c.Proc = cont
	return c
}

func SkipSpaces(cont Proc) Proc {
	return func(ctx Context) Context {
		if ctx.IsEnd() {
			return ctx.Cont(cont)
		}
		ru, l := ctx.Read()
		if unicode.IsSpace(ru) {
			ctx.Offset += l
			return ctx.Cont(
				SkipSpaces(cont),
			)
		}
		return ctx.Cont(cont)
	}
}

func Expect(str string, cont Proc) Proc {
	return func(ctx Context) Context {
		if len(str) == 0 {
			return ctx.Cont(cont)
		}
		if ctx.IsEnd() {
			ctx.Err = fmt.Errorf("expecting %s, but input is end", str)
			return ctx
		}
		r1, l1 := utf8.DecodeRuneInString(str)
		r2, l2 := ctx.Read()
		if r1 != r2 {
			ctx.Err = fmt.Errorf("expecting %s", str)
			return ctx
		}
		ctx.Offset += l2
		return ctx.Cont(
			Expect(str[l1:], cont),
		)
	}
}

func ReadTo(predict func(rune) bool, w io.Writer, cont Proc) Proc {
	return func(ctx Context) Context {
		if ctx.IsEnd() {
			return ctx.Cont(cont)
		}
		ru, l := ctx.Read()
		if _, err := w.Write(ctx.Input[ctx.Offset : ctx.Offset+l]); err != nil {
			ctx.Err = err
			return ctx
		}
		ctx.Offset += l
		if predict(ru) {
			return ctx.Cont(cont)
		}
		return ctx.Cont(
			ReadTo(predict, w, cont),
		)
	}
}

func ReadToRune(r rune, w io.Writer, cont Proc) Proc {
	return ReadTo(func(c rune) bool {
		return c == r
	}, w, cont)
}
