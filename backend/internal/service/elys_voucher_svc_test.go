package service

import (
	"context"
	"testing"
)

type fakeElysAPI struct {
	prepare *ElysPrepareResult
	consume *ElysConsumeResult
	err     error
}

func (f *fakeElysAPI) AuthURL(context.Context, uint64, int) (string, error) {
	return "https://example.com/auth", nil
}

func (f *fakeElysAPI) Prepare(context.Context, uint64, int, string) (*ElysPrepareResult, error) {
	return f.prepare, f.err
}

func (f *fakeElysAPI) Consume(context.Context, uint64, int, string, string, int) (*ElysConsumeResult, error) {
	return f.consume, f.err
}

func TestSumPayAmount(t *testing.T) {
	items := []ElysConsumeItem{{PayAmount: 1000}, {PayAmount: 2000}}
	if got := sumPayAmount(items); got != 3000 {
		t.Fatalf("sumPayAmount = %d", got)
	}
}

func TestElysPlatformCode(t *testing.T) {
	if elysPlatformCode("meituan") != 1 || elysPlatformCode("douyin") != 2 {
		t.Fatal("platform mapping wrong")
	}
}
