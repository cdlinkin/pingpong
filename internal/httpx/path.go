package httpx

import (
	"net/http"
	"strconv"
)

func PathInt(r *http.Request, name string) (int, error) {
	return strconv.Atoi(r.PathValue(name))
}
