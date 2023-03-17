package util

import (
	"testing"
)

func TestRandomAlphaNumericString(t *testing.T) {
	for i := 5; i < 10; i++ {
		prevString := RandomAlphaNumericString(i)
		nextString := RandomAlphaNumericString(i)

		if prevString == nextString {
			t.Errorf("Same string generated consecutively: %s %s", prevString, nextString)
		}
	}
}