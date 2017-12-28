package fund

import (
	"testing"
)

func TestFund(t *testing.T) {
	fundManager := new(FundManage)
	fundManager.Init()
	fundManager.SaveBalanceBeforeOpen("123456", "okex", 60)
	fundManager.SaveBalanceAfterClose("123456", 123)
}
