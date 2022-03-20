package controllers

import (
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func differGeneric(desired, current *unstructured.Unstructured) bool {
	desiredObj := desired.DeepCopy().Object
	delete(desiredObj, "metadata")
	currentObj := current.DeepCopy().Object
	delete(currentObj, "metadata")
	switch desired.GetKind() {
	case "ServiceAccount":
		delete(currentObj, "secrets")
	}
	return !reflect.DeepEqual(desiredObj, currentObj)
}

func ContainsString(slice []string, str string) bool {
	for _, item := range slice {
		if item == str {
			return true
		}
	}
	return false
}

func RemoveString(slice []string, str string) (result []string) {
	for _, item := range slice {
		if item == str {
			continue
		}
		result = append(result, item)
	}
	return result
}
