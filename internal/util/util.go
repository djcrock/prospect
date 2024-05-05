package util

import "math/rand/v2"

var randomStringRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomString(length int) string {
	result := make([]rune, length)
	for i := range result {
		result[i] = randomStringRunes[rand.IntN(len(randomStringRunes))]
	}
	return string(result)
}
