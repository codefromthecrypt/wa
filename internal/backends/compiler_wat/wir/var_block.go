// 版权 @2022 凹语言 作者。保留所有权利。

package wir

import (
	"strconv"

	"github.com/wa-lang/wa/internal/backends/compiler_wat/wir/wat"
	"github.com/wa-lang/wa/internal/logger"
)

/**************************************
varBlock:
**************************************/
type varBlock struct {
	aVar
}

func newVarBlock(name string, kind ValueKind, base_type ValueType) *varBlock {
	return &varBlock{aVar: aVar{name: name, kind: kind, typ: NewBlock(base_type)}}
}

func (v *varBlock) raw() []wat.Value {
	return []wat.Value{wat.NewVarI32(v.name)}
}

func (v *varBlock) EmitInit() (insts []wat.Inst) {
	insts = append(insts, wat.NewInstConst(wat.I32{}, "0"))
	insts = append(insts, v.pop(v.name))
	return
}

func (v *varBlock) EmitPush() (insts []wat.Inst) {
	insts = append(insts, v.push(v.name))
	insts = append(insts, wat.NewInstCall("$wa.RT.Block.Retain"))
	return
}

func (v *varBlock) EmitPop() (insts []wat.Inst) {
	insts = append(insts, v.EmitRelease()...)
	insts = append(insts, v.pop(v.name))
	return
}

func (v *varBlock) EmitRelease() (insts []wat.Inst) {
	insts = append(insts, v.push(v.name))
	insts = append(insts, wat.NewInstCall("$wa.RT.Block.Release"))
	return
}

func (v *varBlock) emitLoadFromAddr(addr Value) (insts []wat.Inst) {
	insts = append(insts, addr.EmitPush()...)
	insts = append(insts, wat.NewInstLoad(wat.I32{}, 0, 1))
	insts = append(insts, wat.NewInstCall("$wa.RT.Block.Retain"))
	return
}

func (v *varBlock) emitStoreToAddr(addr Value) (insts []wat.Inst) {
	insts = append(insts, v.push(v.name))
	insts = append(insts, wat.NewInstCall("$wa.RT.Block.Retain"))
	insts = append(insts, wat.NewInstDrop())

	insts = append(insts, addr.EmitPush()...)
	insts = append(insts, wat.NewInstLoad(wat.I32{}, 0, 1))
	insts = append(insts, wat.NewInstCall("$wa.RT.Block.Release"))

	insts = append(insts, addr.EmitPush()...)
	insts = append(insts, v.push(v.name))
	insts = append(insts, wat.NewInstStore(toWatType(v.Type()), 0, 1))
	return
}

func (v *varBlock) emitHeapAlloc(item_count Value, module *Module) (insts []wat.Inst) {
	switch item_count.Kind() {
	case ValueKindConst:
		c, err := strconv.Atoi(item_count.Name())
		if err != nil {
			logger.Fatalf("%v\n", err)
			return nil
		}
		insts = append(insts, NewConst(I32{}, strconv.Itoa(v.Type().(Block).Base.size()*c+16)).EmitPush()...)
		insts = append(insts, wat.NewInstCall("$waHeapAlloc"))

		insts = append(insts, item_count.EmitPush()...)                                                          //item_count
		insts = append(insts, NewConst(I32{}, strconv.Itoa(v.Type().(Block).Base.onFree(module))).EmitPush()...) //free_method
		insts = append(insts, wat.NewInstCall("$wa.RT.Block.Init"))

	default:
		if !item_count.Type().Equal(I32{}) {
			logger.Fatal("item_count should be i32")
			return nil
		}

		insts = append(insts, item_count.EmitPush()...)
		insts = append(insts, NewConst(I32{}, strconv.Itoa(v.Type().(Block).Base.size())).EmitPush()...)
		insts = append(insts, wat.NewInstMul(wat.I32{}))
		insts = append(insts, NewConst(I32{}, "16").EmitPush()...)
		insts = append(insts, wat.NewInstAdd(wat.I32{}))
		insts = append(insts, wat.NewInstCall("$waHeapAlloc"))

		insts = append(insts, item_count.EmitPush()...)
		insts = append(insts, NewConst(I32{}, strconv.Itoa(v.Type().(Block).Base.onFree(module))).EmitPush()...) //free_method
		insts = append(insts, wat.NewInstCall("$wa.RT.Block.Init"))
	}

	return
}