package jwt

import (
	"encoding/json"
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// ParseIssuedAt comment
func ParseIssuedAt(tknString string) int64 {
	return 0
}

// CheckParse comment
func CheckParse(tknString string) bool {

	token, err := jwt.Parse(tknString, nil)

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		fmt.Println(claims["iat"])
		tknIat, err := getIssuedAt(claims)
		if err != nil {
			fmt.Println(err)
			return false
		}
		fmt.Println(tknIat)
		tknExp, err := getExpiredAt(claims)
		if err != nil {
			fmt.Println(err)
			return false
		}
		fmt.Println(tknExp)
		currTime := time.Now().Unix()
		fmt.Println(currTime)
	} else {
		fmt.Println(err)
		return false
	}

	return true
}

// TokenExpired comment
func TokenExpired(tknStr string) bool {

	token, err := jwt.Parse(tknStr, nil)

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		tknExp, err := getExpiredAt(claims)
		if err != nil {
			fmt.Println(err)
			return false
		}
		currTime := time.Now().Unix()
		if tknExp < currTime {
			return true
		}
	} else {
		fmt.Println(err)
		return false
	}

	return false

}

// getIssuedAt comes from argo-cd/util/jwt/jwt.go
func getIssuedAt(m jwt.MapClaims) (int64, error) {
	switch iat := m["iat"].(type) {
	case float64:
		return int64(iat), nil
	case json.Number:
		return iat.Int64()
	case int64:
		return iat, nil
	default:
		return 0, fmt.Errorf("iat '%v' is not a number", iat)
	}
}

// getExpiredAt comes from argo-cd/util/jwt/jwt.go
func getExpiredAt(m jwt.MapClaims) (int64, error) {
	switch exp := m["exp"].(type) {
	case float64:
		return int64(exp), nil
	case json.Number:
		return exp.Int64()
	case int64:
		return exp, nil
	default:
		return 0, fmt.Errorf("exp '%v' is not a number", exp)
	}
}
