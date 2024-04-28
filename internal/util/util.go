package util

import "math/rand"

var randomStringRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandomString(length int) string {
	result := make([]rune, length)
	for i := range result {
		result[i] = randomStringRunes[rand.Intn(len(randomStringRunes))]
	}
	return string(result)
}
