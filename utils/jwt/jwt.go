package jwt

import (
	"encoding/json"
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// TokenExpired comment
func TokenExpired(tknStr string) (bool, error) {

	token, err := jwt.Parse(tknStr, nil)
	if token == nil {
		fmt.Println(err)
		return true, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		tknExp, err := getExpiredAt(claims)
		if err != nil {
			fmt.Println(err)
			return true, err
		}
		currTime := time.Now().Unix()
		if tknExp <= currTime {
			return true, nil
		}
	} else {
		fmt.Println(err)
		return true, err
	}

	return false, nil

}

// TimeTillExpire s
func TimeTillExpire(tknStr string) int64 {

	token, _ := jwt.Parse(tknStr, nil)

	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		tknExp, _ := getExpiredAt(claims)
		currTime := time.Now().Unix()
		timeTillExpire := tknExp - currTime
		if timeTillExpire <= 0 {
			return 0
		}
		return timeTillExpire
	}

	return 0
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
