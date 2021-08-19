package gorm

import "testing"

func TestLog_Caller(t *testing.T) {
	loc := fileWithLineNum()
	t.Logf("%s", loc)
}
