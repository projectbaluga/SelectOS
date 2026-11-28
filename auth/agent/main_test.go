package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadState(t *testing.T) {
	path:=filepath.Join(t.TempDir(),"nested","credentials.json")
	want:=state{DeviceToken:"0123456789abcdef0123456789abcdef"}
	if err:=save(path,want);err!=nil{t.Fatal(err)}
	got,err:=load(path);if err!=nil{t.Fatal(err)}
	if got.DeviceToken!=want.DeviceToken{t.Fatalf("got %q, want stored token",got.DeviceToken)}
	info,err:=os.Stat(path);if err!=nil{t.Fatal(err)}
	if info.Mode().Perm()!=0600{t.Fatalf("credential file mode=%o, want 600",info.Mode().Perm())}
}
