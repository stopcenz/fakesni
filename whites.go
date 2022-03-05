package main

import (
	"math/rand"
	"strings"
)

const maxSubdomainLen = 5

var abc = []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l",
	"m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z",}

//rand.Seed(time.Now().UnixNano())
func randString(min int, max int) string {
	l := min + rand.Intn(1 + max - min)
	a := []string{}
	for i := 0; i < l; i++ {
		a = append(a, abc[rand.Intn(len(abc))])
	}
	return strings.Join(a[:], "")
}

func genWhiteHost() string {
	w := strings.Split(WHITES, "|")
	base := w[rand.Intn(len(w))]
	sub := randString(0, maxSubdomainLen)
	if sub == "" {
		return base
	}
	return sub + "." + base
}

