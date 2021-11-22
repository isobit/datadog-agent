package dwarfutils

import (
	"debug/dwarf"
	"fmt"
	"reflect"
	"strings"

	"github.com/go-delve/delve/pkg/dwarf/godwarf"
)

type TypeFinder struct {
	dwarfData *dwarf.Data
	typeCache map[dwarf.Offset]godwarf.Type
}

func NewTypeFinder(dwarfData *dwarf.Data) *TypeFinder {
	return &TypeFinder{
		dwarfData: dwarfData,
		typeCache: make(map[dwarf.Offset]godwarf.Type),
	}
}

func (f *TypeFinder) FindTypeByName(name string) (godwarf.Type, error) {
	entryReader := f.dwarfData.Reader()
	for entry, err := entryReader.Next(); entry != nil; entry, err = entryReader.Next() {
		if err != nil {
			return nil, fmt.Errorf("error while reading debug info entries to find %q: %w", name, err)
		}

		// Check if this entry is a type.
		// This possible values come from https://pkg.go.dev/debug/dwarf#Tag.
		// A tag is a type if it ends in "Type" or is "TagTypedef".
		// Go only emits a small subset of these,
		// so even if this heuristic isn't perfect,
		// it should work for any type Go uses.
		if entry.Tag == dwarf.TagTypedef || strings.HasSuffix(entry.Tag.String(), "Type") {
			typeName, _ := entry.Val(dwarf.AttrName).(string)
			if typeName == name {
				typ, err := f.FindTypeByOffset(entry.Offset)
				if err != nil {
					return nil, fmt.Errorf("failed to find type %q at its offset: %w", name, err)
				}

				return typ, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find type %q", name)
}

func (f *TypeFinder) FindTypeByOffset(offset dwarf.Offset) (godwarf.Type, error) {
	typ, err := godwarf.ReadType(f.dwarfData, 0, offset, f.typeCache)
	if err != nil {
		return nil, err
	}

	// If the type is a typedef type, recurse to its actual definition
	// when fixing its `.CommonType.ReflectKind` value
	innermostType := typ
	if typedefType, ok := typ.(*godwarf.TypedefType); ok {
		innermostType = recurseTypedefType(typedefType)
	}

	// Fix the internal `godwarf.Type.CommonType.ReflectKind` field for slice types
	// (by default it gets reflect.Invalid as its kind):
	if _, ok := innermostType.(*godwarf.SliceType); ok {
		typ.Common().ReflectKind = reflect.Slice
	}

	// Fix the internal `godwarf.Type.CommonType.ReflectKind` field for interface types
	// (by default it gets reflect.Invalid as its kind):
	if _, ok := innermostType.(*godwarf.InterfaceType); ok {
		typ.Common().ReflectKind = reflect.Interface
	}

	return typ, nil
}

func recurseTypedefType(typ godwarf.Type) godwarf.Type {
	if typedefType, ok := typ.(*godwarf.TypedefType); ok {
		return recurseTypedefType(typedefType.Type)
	} else {
		return typ
	}
}

func (f *TypeFinder) FindStructFieldOffset(structName string, fieldName string) (uint64, error) {
	typ, err := f.FindTypeByName(structName)
	if err != nil {
		return 0, fmt.Errorf("could not find %q type: %w", structName, err)
	}

	var fieldOffset uint64
	foundField := false
	if structType, ok := typ.(*godwarf.StructType); ok {
		for _, field := range structType.Field {
			if field.Name == fieldName {
				fieldOffset = uint64(field.ByteOffset)
				foundField = true
				break
			}
		}
	}

	if !foundField {
		return 0, fmt.Errorf("could not find offset of %q field in %q", fieldName, structName)
	}

	return fieldOffset, nil
}
