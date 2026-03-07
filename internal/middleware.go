package internal

import "reflect"

func GetTaskName[T any]() string {
	return reflect.TypeOf((*T)(nil)).Elem().String()
}
