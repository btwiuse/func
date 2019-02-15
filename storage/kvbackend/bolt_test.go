package kvbackend

import "testing"

func Test_boltBucketKey(t *testing.T) {
	tests := []struct {
		input      string
		wantBucket string
		wantKey    string
		wantErr    bool
	}{
		{input: "", wantErr: true},
		{input: "/foo", wantErr: true},
		{input: "foo", wantErr: true},
		{input: "foo/", wantErr: true},
		{input: "/foo/bar", wantErr: true},
		{input: "foo/bar/", wantErr: true},
		{input: "foo/bar", wantBucket: "foo", wantKey: "bar"},
		{input: "foo/bar/baz", wantBucket: "foo/bar", wantKey: "baz"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			bucket, key, err := boltBucketKey(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(bucket) != tt.wantBucket {
				t.Errorf("Bucket = %q, want = %q", string(bucket), tt.wantBucket)
			}
			if string(key) != tt.wantKey {
				t.Errorf("Key = %q, want = %q", string(key), tt.wantKey)
			}
		})
	}
}
