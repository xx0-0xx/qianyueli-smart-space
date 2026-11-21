package migrate

import "testing"

func TestSplitStatements(t *testing.T) {
	cases := []struct {
		name string
		sql  string
		want int
	}{
		{
			name: "basic",
			sql:  "CREATE TABLE a (id INT);\nCREATE TABLE b (id INT);",
			want: 2,
		},
		{
			name: "semicolon_in_string",
			sql:  "INSERT INTO t (v) VALUES ('a;b');\nINSERT INTO t (v) VALUES ('c');",
			want: 2,
		},
		{
			name: "semicolon_in_backtick",
			sql:  "SELECT `a;b` FROM t;",
			want: 1,
		},
		{
			name: "line_comment_with_semicolon",
			sql:  "-- comment; not a stmt\nCREATE TABLE c (id INT);",
			want: 1,
		},
		{
			name: "block_comment_with_semicolon",
			sql:  "/* a;b */ CREATE TABLE d (id INT);",
			want: 1,
		},
		{
			name: "trailing_whitespace",
			sql:  "CREATE TABLE e (id INT);\n\n   \n",
			want: 1,
		},
	}
	for _, c := range cases {
		got := splitStatements(c.sql)
		if len(got) != c.want {
			t.Errorf("%s: got %d statements %q, want %d", c.name, len(got), got, c.want)
		}
	}
}

func TestIsIdempotentErr(t *testing.T) {
	if !isIdempotentErr(errString("Error 1060 (42S21): Duplicate column name 'x'")) {
		t.Error("duplicate column should be idempotent")
	}
	if isIdempotentErr(errString("Error 1064 (42000): You have an error in your SQL syntax")) {
		t.Error("syntax error must not be idempotent")
	}
	if isIdempotentErr(nil) {
		t.Error("nil must not be idempotent")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
