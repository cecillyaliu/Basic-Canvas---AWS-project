package test

import (
	"fmt"
	"gorm.io/gorm/utils/tests"
	"net/http"
	"os"
	"testing"
)

func TestHealthz(t *testing.T) {
	url := "http://localhost:8080/healthz"
	response, err := http.Get(url)
	if err != nil {
		fmt.Println("HTTP GET请求失败:", err)
		return
	}
	tests.AssertEqual(t, response.StatusCode, http.StatusOK)
}

func TestMain(m *testing.M) {
	exitCode := m.Run()
	os.Exit(exitCode)
}
