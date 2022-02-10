package redisq_test

import (
	"encoding/json"
	"fmt"
)

type testStringer struct {
	Name string
	Age  int `json:"age"`
}

func (t *testStringer) ToString() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		panic(err)
	}

	return string(b), nil
}

func (t *testStringer) FromString(s string) error {
	err := json.Unmarshal([]byte(s), t)
	if err != nil {
		panic(err)
	}

	return nil
}

type testStringer2 struct{}

func (t *testStringer2) ToString() (string, error) {
	return "", nil
}

func (t *testStringer2) FromString(s string) error {
	return nil
}

type testErrorStringer struct {
	needErr bool
}

func (t *testErrorStringer) ToString() (string, error) {
	if t.needErr {
		return "", fmt.Errorf("testErrorStringer")
	}
	return "", nil
}

func (t *testErrorStringer) FromString(s string) error {
	if t.needErr {
		return fmt.Errorf("testErrorStringer")
	}
	return nil
}
