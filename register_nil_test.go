package easycel_test

import (
	"strings"
	"testing"

	"github.com/wzshiming/easycel"
)

func TestRegisterNilVariableReturnsErrorNamingIt(t *testing.T) {
	err := easycel.NewRegistry("test").RegisterVariable("myVar", nil)
	if err == nil || !strings.Contains(err.Error(), "myVar") {
		t.Errorf("got error %v, want one naming myVar", err)
	}
}

func TestRegisterNilConversionReturnsErrorNamingEntryPoint(t *testing.T) {
	err := easycel.NewRegistry("test").RegisterConversion(nil)
	if err == nil || !strings.Contains(err.Error(), "RegisterConversion") {
		t.Errorf("got error %v, want one naming RegisterConversion", err)
	}
}
