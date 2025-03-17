package bubble

import (
	"cmp"
	"embed"
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"text/template"
)

//go:embed templates
var templates embed.FS

var errType = reflect.TypeOf((*error)(nil)).Elem()

type (
	tmplData struct {
		PkgName  string
		Imports  map[string]string
		Services []serviceDef
	}

	serviceDef struct {
		Name         string
		Type         reflect.Type
		dependencies []string
		ProvFn       provideFunc
	}

	provideFunc struct {
		Params            []string
		Returns           []string
		Args              []string
		ShouldReturnError bool
	}
)

func isValidKind(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Interface || t.Kind() == reflect.Struct
}

func stringifyType(t reflect.Type) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
		return "*" + t.PkgPath() + "." + t.Name()
	}
	return t.PkgPath() + "." + t.Name()
}

func genData(pkgName string, provFns map[string]any) (dat tmplData) {
	svcs := make(map[string]*serviceDef)
	dat = tmplData{
		PkgName: pkgName,
		Imports: map[string]string{
			"fmt":     "",
			"context": "",
		},
	}
	g := newGraph[string]()
	for name, provFn := range provFns {
		svc := serviceDef{
			Name:   name,
			ProvFn: provideFunc{},
		}
		t := reflect.TypeOf(provFn)
		if t.Kind() != reflect.Func {
			panic(fmt.Errorf("expected provider '%s' to be a func but got %s", name, t.Kind()))
		}
		switch t.NumOut() {
		case 0:
			panic(fmt.Errorf("expected provider '%s' to return at least one value", name))
		case 2:
			if t.Out(1) != errType {
				panic(fmt.Errorf("expected 2nd return value of provider '%s' to be an error", name))
			}
			svc.ProvFn.ShouldReturnError = true
		}
		svcType := t.Out(0)
		if !isValidKind(svcType) {
			panic(fmt.Errorf("expected provider '%s' to return an interface or struct but got %s", name, t.Out(0).Kind()))
		}
		svc.ProvFn.Returns = []string{svcType.String()}
		if svc.ProvFn.ShouldReturnError {
			svc.ProvFn.Returns = append(svc.ProvFn.Returns, "error")
		}
		g.addVertex(stringifyType(svcType))
		for j := 0; j < t.NumIn(); j++ {
			if !isValidKind(t.In(j)) {
				panic(fmt.Errorf("expected dependency at index %d of provider '%s' to be an interface or struct but got %s", j, name, t.In(j).Kind()))
			}
			g.addEdge(stringifyType(svcType), stringifyType(t.In(j)))
			svc.ProvFn.Params = append(svc.ProvFn.Params, t.In(j).String())
			svc.dependencies = append(svc.dependencies, stringifyType(t.In(j)))
		}
		svc.Type = svcType
		svcs[stringifyType(svcType)] = &svc
		if svcType.Kind() == reflect.Ptr {
			svcType = svcType.Elem()
		}
		dat.Imports[svcType.PkgPath()] = svcType.String()[:strings.IndexRune(svcType.String(), '.')]
	}
	for _, svc := range svcs {
		for _, dep := range svc.dependencies {
			if _, ok := svcs[dep]; !ok {
				panic(fmt.Errorf("expected dependency '%s' of provider '%s' to be a valid service", dep, svc.Name))
			}
			svc.ProvFn.Args = append(svc.ProvFn.Args, svcs[dep].Name)
		}
	}
	topoOrd := g.sort(func(s1 string, s2 string) int {
		return cmp.Compare(s1, s2)
	})
	for _, svc := range topoOrd {
		dat.Services = append(dat.Services, *svcs[svc])
	}
	return
}

func GenerateWithPkgName(pkgName string, provFns map[string]any, output string) {
	tmpl := template.Must(template.New("bubble").Funcs(template.FuncMap{
		"join": strings.Join,
		"prefix": func(pf string, arr []string) []string {
			for i, v := range arr {
				arr[i] = pf + v
			}
			return arr
		},
	}).ParseFS(templates, "templates/*.tmpl"))
	if e := os.MkdirAll(output, 0755); e != nil {
		panic(e)
	}
	f, e := os.OpenFile(path.Join(output, "container.go"), os.O_WRONLY|os.O_CREATE, 0644)
	if e != nil {
		panic(e)
	}
	tmpl.Lookup("container.tmpl").Execute(f, genData(pkgName, provFns))
}

func Generate(provFns map[string]any, output string) {
	GenerateWithPkgName("di", provFns, output)
}
