// 版权 @2022 凹语言 作者。保留所有权利。

package wir

import (
	"github.com/wa-lang/wa/internal/backends/compiler_wat/wir/wat"
	"github.com/wa-lang/wa/internal/logger"
)

func toWatType(t ValueType) wat.ValueType {
	switch t.(type) {
	case RUNE:
		return wat.I32{}

	case I32:
		return wat.I32{}

	case U32:
		return wat.U32{}

	case I64:
		return wat.I64{}

	case U64:
		return wat.U64{}

	case F32:
		return wat.F32{}

	case F64:
		return wat.F64{}

	case Pointer:
		return wat.I32{}

	case Block:
		return wat.I32{}

	default:
		logger.Fatalf("Todo:%v\n", t)
	}

	return nil
}

/**************************************
VOID:
**************************************/
type VOID struct{}

func (t VOID) Name() string           { return "void" }
func (t VOID) size() int              { return 0 }
func (t VOID) align() int             { return 0 }
func (t VOID) onFree(m *Module) int   { return 0 }
func (t VOID) Raw() []wat.ValueType   { return []wat.ValueType{} }
func (t VOID) Equal(u ValueType) bool { _, ok := u.(VOID); return ok }

/**************************************
RUNE:
**************************************/
type RUNE struct{}

func (t RUNE) Name() string           { return "rune" }
func (t RUNE) size() int              { return 4 }
func (t RUNE) align() int             { return 4 }
func (t RUNE) onFree(m *Module) int   { return 0 }
func (t RUNE) Raw() []wat.ValueType   { return []wat.ValueType{wat.I32{}} }
func (t RUNE) Equal(u ValueType) bool { _, ok := u.(RUNE); return ok }

/**************************************
I32:
**************************************/
type I32 struct{}

func (t I32) Name() string           { return "i32" }
func (t I32) size() int              { return 4 }
func (t I32) align() int             { return 4 }
func (t I32) onFree(m *Module) int   { return 0 }
func (t I32) Raw() []wat.ValueType   { return []wat.ValueType{wat.I32{}} }
func (t I32) Equal(u ValueType) bool { _, ok := u.(I32); return ok }

/**************************************
U32:
**************************************/
type U32 struct{}

func (t U32) Name() string           { return "u32" }
func (t U32) size() int              { return 4 }
func (t U32) align() int             { return 4 }
func (t U32) onFree(m *Module) int   { return 0 }
func (t U32) Raw() []wat.ValueType   { return []wat.ValueType{wat.U32{}} }
func (t U32) Equal(u ValueType) bool { _, ok := u.(U32); return ok }

/**************************************
I64:
**************************************/
type I64 struct{}

func (t I64) Name() string           { return "i64" }
func (t I64) size() int              { return 8 }
func (t I64) align() int             { return 8 }
func (t I64) onFree(m *Module) int   { return 0 }
func (t I64) Raw() []wat.ValueType   { return []wat.ValueType{wat.I64{}} }
func (t I64) Equal(u ValueType) bool { _, ok := u.(I64); return ok }

/**************************************
Uint64:
**************************************/
type U64 struct{}

func (t U64) Name() string           { return "u64" }
func (t U64) size() int              { return 8 }
func (t U64) align() int             { return 8 }
func (t U64) onFree(m *Module) int   { return 0 }
func (t U64) Raw() []wat.ValueType   { return []wat.ValueType{wat.U64{}} }
func (t U64) Equal(u ValueType) bool { _, ok := u.(U64); return ok }

/**************************************
F32:
**************************************/
type F32 struct{}

func (t F32) Name() string           { return "f32" }
func (t F32) size() int              { return 4 }
func (t F32) align() int             { return 4 }
func (t F32) onFree(m *Module) int   { return 0 }
func (t F32) Raw() []wat.ValueType   { return []wat.ValueType{wat.F32{}} }
func (t F32) Equal(u ValueType) bool { _, ok := u.(F32); return ok }

/**************************************
F64:
**************************************/
type F64 struct{}

func (t F64) Name() string           { return "f64" }
func (t F64) size() int              { return 8 }
func (t F64) align() int             { return 8 }
func (t F64) onFree(m *Module) int   { return 0 }
func (t F64) Raw() []wat.ValueType   { return []wat.ValueType{wat.F64{}} }
func (t F64) Equal(u ValueType) bool { _, ok := u.(F64); return ok }

/**************************************
Pointer:
**************************************/
type Pointer struct {
	Base ValueType
}

func NewPointer(base ValueType) Pointer { return Pointer{Base: base} }
func (t Pointer) Name() string          { return "pointer$" + t.Base.Name() }
func (t Pointer) size() int             { return 4 }
func (t Pointer) align() int            { return 4 }
func (t Pointer) onFree(m *Module) int  { return 0 }
func (t Pointer) Raw() []wat.ValueType  { return []wat.ValueType{wat.I32{}} }
func (t Pointer) Equal(u ValueType) bool {
	if ut, ok := u.(Pointer); ok {
		return t.Base.Equal(ut.Base)
	}
	return false
}

/**************************************
Block:
**************************************/
type Block struct {
	Base ValueType
}

func NewBlock(base ValueType) Block  { return Block{Base: base} }
func (t Block) Name() string         { return "block$" + t.Base.Name() }
func (t Block) size() int            { return 4 }
func (t Block) align() int           { return 4 }
func (t Block) Raw() []wat.ValueType { return []wat.ValueType{wat.I32{}} }
func (t Block) Equal(u ValueType) bool {
	if ut, ok := u.(Block); ok {
		return t.Base.Equal(ut.Base)
	}
	return false
}
func (t Block) onFree(m *Module) int {
	var f Function
	f.Name = "$$onFree_" + t.Name()
	f.Result = VOID{}
	ptr := newVarBasic("$ptr", ValueKindLocal, I32{})
	f.Params = append(f.Params, ptr)

	f.Insts = append(f.Insts, ptr.EmitPush()...)
	f.Insts = append(f.Insts, wat.NewInstLoad(wat.I32{}, 0, 1))
	f.Insts = append(f.Insts, wat.NewInstCall("$wa.RT.Block.Release"))
	f.Insts = append(f.Insts, ptr.EmitPush()...)
	f.Insts = append(f.Insts, wat.NewInstConst(wat.I32{}, "0"))
	f.Insts = append(f.Insts, wat.NewInstStore(wat.I32{}, 0, 1))

	return addTable(&f, m)
}

/**************************************
Struct:
**************************************/
type Struct struct {
	name    string
	Members []Field
	_size   int
	_align  int
}

type Field struct {
	name   string
	typ    ValueType
	_start int
}

func NewField(n string, t ValueType) Field { return Field{name: n, typ: t} }
func (i Field) Name() string               { return i.name }
func (i Field) Type() ValueType            { return i.typ }
func (i Field) Equal(u Field) bool         { return i.name == u.name && i.typ.Equal(u.typ) }

func makeAlign(i, a int) int {
	return (i + a - 1) / a * a
}

func NewStruct(name string, members []Field) Struct {
	var s Struct
	s.name = name

	for _, m := range members {
		ma := m.Type().align()
		m._start = makeAlign(s._size, ma)
		s.Members = append(s.Members, m)

		s._size = m._start + m.Type().size()
		if ma > s._align {
			s._align += ma
		}
	}
	s._size = makeAlign(s._size, s._align)

	return s
}

func (t Struct) Name() string { return t.name }
func (t Struct) size() int    { return t._size }
func (t Struct) align() int   { return t._align }

func (t Struct) onFree(m *Module) int {
	logger.Fatal("Todo")
	return 0
}

func (t Struct) Raw() []wat.ValueType {
	var r []wat.ValueType
	for _, f := range t.Members {
		r = append(r, f.Type().Raw()...)
	}
	return r
}

func (t Struct) Equal(u ValueType) bool {
	if u, ok := u.(Struct); ok {
		if len(t.Members) != len(u.Members) {
			return false
		}

		for i := range t.Members {
			if !t.Members[i].Equal(u.Members[i]) {
				return false
			}
		}

		return true
	}
	return false
}

/**************************************
Ref:
**************************************/
type Ref struct {
	Base       ValueType
	underlying Struct
}

func NewRef(base ValueType) Ref {
	var v Ref
	v.Base = base
	var m []Field
	m = append(m, NewField("block", NewBlock(base)))
	m = append(m, NewField("data", NewPointer(base)))
	v.underlying = NewStruct("", m)
	return v
}
func (t Ref) Name() string         { return "ref$" + t.Base.Name() }
func (t Ref) size() int            { return 8 }
func (t Ref) align() int           { return 4 }
func (t Ref) onFree(m *Module) int { return t.underlying.onFree(m) }
func (t Ref) Raw() []wat.ValueType { return t.underlying.Raw() }
func (t Ref) Equal(u ValueType) bool {
	if ut, ok := u.(Ref); ok {
		return t.Base.Equal(ut.Base)
	}
	return false
}
