package engine

import "testing"

func TestVariableToTemplate(t *testing.T) {
	src := `compress $remote_addr - $remote_user [$time_local] $request $status $body_bytes_sent $http_referer $http_user_agent $gzip_ratio;`
	got := VariablesToTemplates([]byte(src))
	t.Error(string(got))
}
