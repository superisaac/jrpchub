package worker

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/superisaac/jlib"
	"reflect"
)

func typeIsStruct(tp reflect.Type) bool {
	return (tp.Kind() == reflect.Struct ||
		(tp.Kind() == reflect.Ptr && typeIsStruct(tp.Elem())))
}

func interfaceToValue(a interface{}, outputType reflect.Type) (reflect.Value, error) {
	output := reflect.Zero(outputType).Interface()
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		TagName:  "json",
		Result:   &output,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return reflect.Value{}, err
	}
	err = decoder.Decode(a)
	if err != nil {
		return reflect.Value{}, err
	}
	return reflect.ValueOf(output), nil
}

func valueToInterface(tp reflect.Type, val reflect.Value) (interface{}, error) {
	var output interface{}
	if typeIsStruct(tp) {
		output = make(map[string]interface{})
	} else {
		output = reflect.Zero(tp).Interface()
	}
	config := &mapstructure.DecoderConfig{
		Metadata: nil,
		TagName:  "json",
		Result:   &output,
	}
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return nil, err
	}
	err = decoder.Decode(val.Interface())
	if err != nil {
		return nil, err
	}
	return output, nil
}

func wrapTyped(tfunc interface{}, firstArgSpec interface{}) (WorkerCallback, error) {

	funcType := reflect.TypeOf(tfunc)
	if funcType.Kind() != reflect.Func {
		return nil, errors.New("tfunc is not func type")
	}

	firstSpecType := reflect.TypeOf(firstArgSpec)
	// check inputs and 1st argument
	numIn := funcType.NumIn()
	if numIn < 1 {
		return nil, errors.New("func must have 1 more arguments")
	}

	firstArgType := funcType.In(0)
	if !(firstArgType.Kind() == reflect.Ptr && firstArgType.String() == firstSpecType.String()) {
		return nil, errors.New(fmt.Sprintf("the first arg must be %s", firstSpecType.String()))
	}

	// check outputs
	if funcType.NumOut() != 2 {
		return nil, errors.New("func return number must be 2")
	}

	errType := funcType.Out(1)
	errInterface := reflect.TypeOf((*error)(nil)).Elem()

	if !errType.Implements(errInterface) {
		return nil, errors.New("second output does not implement error")
	}

	handler := func(req *WorkerRequest, params []interface{}) (interface{}, error) {
		// check inputs
		if funcType.NumIn() != len(params)+1 {
			return nil, jlib.ParamsError("different params size")
		}

		// params -> []reflect.Value
		fnArgs := []reflect.Value{reflect.ValueOf(req)}
		for i, param := range params {
			argType := funcType.In(i + 1)
			//fmt.Printf("i %d, interface type %s from %#v\n", i, argType, param)
			argValue, err := interfaceToValue(param, argType)
			if err != nil {
				return nil, jlib.ParamsError(
					fmt.Sprintf("params %d %s", i+1, err))
			}
			fnArgs = append(fnArgs, argValue)
		}

		// wrap result
		resValues := reflect.ValueOf(tfunc).Call(fnArgs)
		resType := funcType.Out(0)
		errRes := resValues[1].Interface()
		if errRes != nil {
			if err, ok := errRes.(error); ok {
				return nil, err
			} else {
				return nil, errors.New(fmt.Sprintf("error return is not error %+v", errRes))
			}
		}

		res, err := valueToInterface(
			resType, resValues[0])
		return res, err
	}

	return handler, nil
}
