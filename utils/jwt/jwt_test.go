package jwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTokenExpired(t *testing.T) {
	// this token will expire in a year from 8/4/19
	yrTkn := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1OTY1NTg0MjYsImlhdCI6MTU2NTAyMjQyNiwiaXNzIjoiYXJnb2NkIiwibmJmIjoxNTY1MDIyNDI2LCJzdWIiOiJwcm9qOmRlZmF1bHQ6VGVzdFJvbGUifQ.AY7v6HnSqAIi5sn1tj7-ll4iC0h29I82NT9SpbuJ1x8"
	// this token should be expired
	expTkn := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NjUwMjI0ODAsImlhdCI6MTU2NTAyMjQ3MCwiaXNzIjoiYXJnb2NkIiwibmJmIjoxNTY1MDIyNDcwLCJzdWIiOiJwcm9qOmRlZmF1bHQ6VGVzdFJvbGUifQ.UXml8u9ikwWIMicw6QobQX4dbLP2t7MO4RdOD1W5DNw"
	invalidSt := "invalidstring"

	yrBool, err := TokenExpired(yrTkn)
	assert.Equal(t, false, yrBool)
	assert.Equal(t, nil, err)

	expBool, err := TokenExpired(expTkn)
	assert.Equal(t, true, expBool)
	assert.Equal(t, nil, err)

	invalBool, err := TokenExpired(invalidSt)
	assert.Equal(t, true, invalBool)

	yrExpTime := TimeTillExpire(yrTkn)
	assert.NotEqual(t, 0, yrExpTime)
}
