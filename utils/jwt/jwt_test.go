package jwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseIssuedAt(t *testing.T) {
	issuedAt := ParseIssuedAt("")
	assert.Equal(t, int64(0), issuedAt)
}

func TestCheckParse(t *testing.T) {
	tknString := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE1NjM3Mjk0MDksImlhdCI6MTU2MzEyNDYwOSwiaXNzIjoiYXJnb2NkIiwibmJmIjoxNTYzMTI0NjA5LCJzdWIiOiJwcm9qOmRlZmF1bHQ6VGVzdFJvbGUifQ.wWy2iFZmBnYm_QTfG8IApGnLR2y0z6aHrJeNr-IfHAQ"
	tknBool := CheckParse(tknString)
	assert.Equal(t, true, tknBool)
}
