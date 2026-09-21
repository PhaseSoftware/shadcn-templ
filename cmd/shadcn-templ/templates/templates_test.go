package templates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise the actual scaffold asset handler without generating templ files.
func TestScaffoldAssetModes(t *testing.T) {
	cwd := t.TempDir()
	if err := Create(Templates["templ"], cwd, "example.com/app"); err != nil {
		t.Fatal(err)
	}
	write := func(name, content string) {
		t.Helper()
		path := filepath.Join(cwd, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Reuse the pinned module dependencies; this test does not fetch registry data.
	root := filepath.Join("..", "..", "..")
	mod, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	sum, err := os.ReadFile(filepath.Join(root, "go.sum"))
	if err != nil {
		t.Fatal(err)
	}
	write("go.mod", strings.Replace(string(mod), "module github.com/axadrn/shadcn-templ/v2", "module example.com/app", 1))
	write("go.sum", string(sum))
	write("pages/stub.go", `package pages
import "github.com/a-h/templ"
func Home() templ.Component { return templ.NopComponent }
`)
	write("assets/js/shadcn-templ-0123456789abcdef.js", "embedded")
	write("assets/css/output.css", "embedded-css")
	write("main_test.go", `package main
import("net/http";"net/http/httptest";"os";"testing")
func TestAssets(t *testing.T) {
 if err:=os.WriteFile("assets/js/shadcn-templ-0123456789abcdef.js",[]byte("disk"),0644);err!=nil{t.Fatal(err)}
 for _,tt:=range []struct{name,flag,value,body,cache string}{
  {"default","","","embedded","public, max-age=31536000, immutable"},
  {"explicit dev","SHADCN_TEMPL_DEV","true","disk","no-store"},
  {"templ dev","TEMPL_DEV_MODE","true","disk","no-store"},
  {"legacy dev","GO_ENV","development","disk","no-store"},
  {"unknown env","GO_ENV","staging","embedded","public, max-age=31536000, immutable"},
  {"false dev","SHADCN_TEMPL_DEV","false","embedded","public, max-age=31536000, immutable"},
 } {t.Run(tt.name,func(t *testing.T){
  for _,key:=range []string{"GO_ENV","SHADCN_TEMPL_DEV","TEMPL_DEV_MODE"}{t.Setenv(key,"")}
  if tt.flag!=""{t.Setenv(tt.flag,tt.value)}
  mux:=http.NewServeMux();setupAssetsRoutes(mux)
  w:=httptest.NewRecorder();mux.ServeHTTP(w,httptest.NewRequest("GET","/assets/js/shadcn-templ-0123456789abcdef.js",nil))
  if w.Code!=200||w.Body.String()!=tt.body||w.Header().Get("Cache-Control")!=tt.cache{t.Fatalf("status=%d body=%s cache=%s",w.Code,w.Body,w.Header().Get("Cache-Control"))}
 })}
 t.Setenv("GO_ENV","");t.Setenv("SHADCN_TEMPL_DEV","");t.Setenv("TEMPL_DEV_MODE","")
 mux:=http.NewServeMux();setupAssetsRoutes(mux)
 w:=httptest.NewRecorder();mux.ServeHTTP(w,httptest.NewRequest("GET","/assets/css/output.css",nil))
 if w.Header().Get("Cache-Control")!="no-cache"{t.Fatalf("unhashed CSS cache=%s",w.Header().Get("Cache-Control"))}
}
`)
	cmd := exec.Command("go", "test", "-mod=readonly", ".")
	cmd.Dir = cwd
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("scaffold asset tests: %v\n%s", err, output)
	}
}
