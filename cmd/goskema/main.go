package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"plugin"
	"reflect"
	"strings"
	"time"

	"go/ast"
	"go/parser"
	"go/token"

	conv "github.com/reoring/goskema/dsl/irconv"
	gen "github.com/reoring/goskema/internal/gen"
	isir "github.com/reoring/goskema/internal/ir"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	sub := os.Args[1]
	switch sub {
	case "compile":
		compileCmd(os.Args[2:])
	case "compile-dsl":
		compileDSLCmd(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "goskema CLI\n\nUsage:\n  goskema compile -type T1[,T2,...] -o out.go\n  goskema compile-dsl -pkgdir ./path/to/pkg -symbol DSLVarOrFunc -type TypeName -o out.go\n\nNotes:\n  - This is a minimal scaffold. It generates stubs for future compiled parsers.")
}

func compileCmd(args []string) {
	fs := flag.NewFlagSet("compile", flag.ExitOnError)
	var typesCSV string
	var out string
	fs.StringVar(&typesCSV, "type", "", "comma-separated type names to compile")
	fs.StringVar(&out, "o", "", "output filename")
	_ = fs.Parse(args)
	if typesCSV == "" || out == "" {
		fs.Usage()
		os.Exit(2)
	}
	types := splitCSV(typesCSV)
	pkg := detectPackageName()
	if pkg == "" {
		pkg = "main"
	}

	// Try to build IR-like typedefs (object fields) from package AST.
	defs := make([]gen.TypeDef, 0, len(types))
	for _, tname := range types {
		fields, required := collectStructFieldsAndRequired(tname)
		defs = append(defs, gen.TypeDef{Name: tname, Fields: fields, Required: required})
	}
	code, err := gen.RenderFileFromIR(pkg, defs)
	if err != nil {
		// fallback to simple type stubs
		stubs := make([]gen.TypeStub, 0, len(types))
		for _, t := range types {
			stubs = append(stubs, gen.TypeStub{Name: t})
		}
		code, err = gen.RenderFile(gen.File{Package: pkg, Types: stubs})
		if err != nil {
			fatalf("generate: %v", err)
		}
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fatalf("creating output dir: %v", err)
	}
	if err := os.WriteFile(out, code, 0o644); err != nil {
		fatalf("writing output: %v", err)
	}
}

// compileDSLCmd builds a plugin from -pkgdir, loads -symbol (var or func), converts it to IR, and renders code.
func compileDSLCmd(args []string) {
	fs := flag.NewFlagSet("compile-dsl", flag.ExitOnError)
	var pkgdir string
	var symbol string
	var typeName string
	var out string
	var verbose bool
	fs.StringVar(&pkgdir, "pkgdir", "", "directory of the package that defines the DSL schema")
	fs.StringVar(&symbol, "symbol", "", "variable or zero-arg function name that yields a DSL schema")
	fs.StringVar(&typeName, "type", "", "type name to generate for")
	fs.StringVar(&out, "o", "", "output filename")
	fs.BoolVar(&verbose, "v", false, "enable verbose logs")
	_ = fs.Parse(args)
	if pkgdir == "" || symbol == "" || typeName == "" || out == "" {
		fs.Usage()
		os.Exit(2)
	}

	logf := func(format string, a ...any) {
		if verbose {
			fmt.Fprintf(os.Stderr, format+"\n", a...)
		}
	}

	importPath := detectImportPathFor(pkgdir)
	if importPath == "" {
		fatalf("failed to detect import path for %s", pkgdir)
	}
	logf("compile-dsl: pkgdir=%s importPath=%s symbol=%s type=%s out=%s", pkgdir, importPath, symbol, typeName, out)

	// Build a plugin wrapper under the target package directory (inside module workspace)
	tmp := filepath.Join(pkgdir, ".goskema_plugin_wrapper")
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		fatalf("mkdir wrapper: %v", err)
	}
	defer os.RemoveAll(tmp)
	wrapper := "package main\n\nimport pkg \"" + importPath + "\"\n\nvar " + symbol + " = pkg." + symbol + "\n"
	if err := os.WriteFile(filepath.Join(tmp, "main.go"), []byte(wrapper), 0o644); err != nil {
		fatalf("write wrapper: %v", err)
	}
	logf("wrote wrapper: %s", filepath.Join(tmp, "main.go"))

	so := filepath.Join(os.TempDir(), fmt.Sprintf("goskema_dsl_%d.so", time.Now().UnixNano()))
	tmpArg := tmp
	if !strings.HasPrefix(tmpArg, "./") && !filepath.IsAbs(tmpArg) {
		tmpArg = "./" + tmpArg
	}
	logf("building plugin: %s", so)
	cmd := exec.Command("go", "build", "-buildmode=plugin", "-o", so, tmpArg)
	cmd.Env = os.Environ()
	cmd.Dir = "."
	if outBytes, err := cmd.CombinedOutput(); err != nil {
		fatalf("build plugin failed: %v\n%s", err, string(outBytes))
	}
	logf("built plugin: %s", so)

	p, err := plugin.Open(so)
	if err != nil {
		fatalf("open plugin: %v", err)
	}
	logf("opened plugin: %s", so)
	sym, err := p.Lookup(symbol)
	if err != nil {
		fatalf("lookup symbol: %v", err)
	}
	logf("lookup symbol ok: %s (kind=%v)", symbol, reflect.ValueOf(sym).Kind())
	var dslVal any
	v := reflect.ValueOf(sym)
	if v.Kind() == reflect.Func {
		outs := v.Call(nil)
		if len(outs) > 0 {
			dslVal = outs[0].Interface()
		} else {
			dslVal = v.Interface()
		}
	} else {
		// variables are returned as pointers to the variable
		if v.Kind() == reflect.Pointer || v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		if v.Kind() == reflect.Func {
			outs := v.Call(nil)
			if len(outs) > 0 {
				dslVal = outs[0].Interface()
			} else {
				dslVal = v.Interface()
			}
		} else {
			dslVal = v.Interface()
		}
	}
	logf("dsl dynamic type: %T", dslVal)

	irNode := conv.ToIRFromSchemaDynamic(dslVal)
	if irNode == nil {
		t := reflect.TypeOf(dslVal)
		pkgpath := ""
		name := ""
		if t != nil {
			name = t.Name()
			pkgpath = t.PkgPath()
			if t.Kind() == reflect.Pointer || t.Kind() == reflect.Ptr {
				te := t.Elem()
				name += " (elem=" + te.Name() + ")"
				pkgpath += " (elem=" + te.PkgPath() + ")"
			}
		}
		fmt.Fprintf(os.Stderr, "compile-dsl: unsupported schema: symbol=%s dynamic=%T name=%s pkg=%s\n", symbol, dslVal, name, pkgpath)
		fatalf("unsupported DSL schema for symbol %s", symbol)
	}
	if obj, ok := irNode.(*isir.Object); ok {
		logf("ir object: fields=%d required=%d unknownPolicy=%d unknownTarget=%q", len(obj.Fields), len(obj.Required), obj.UnknownPolicy, obj.UnknownTarget)
	}

	pkg := detectPackageNameFor(pkgdir)
	if pkg == "" {
		pkg = "main"
	}

	// Attempt to collect JSON->Go field bindings for the target struct type in pkgdir
	bmap := collectJSONToGoBindings(pkgdir, typeName)
	var code []byte
	if len(bmap) > 0 {
		logf("bindings collected: %d", len(bmap))
		code, err = gen.RenderFileFromIRNodesWithBindings(pkg, map[string]isir.Schema{typeName: irNode}, map[string]map[string]string{typeName: bmap})
	} else {
		code, err = gen.RenderFileFromIRNodes(pkg, map[string]isir.Schema{typeName: irNode})
	}
	if err != nil {
		fatalf("generate from IR: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fatalf("creating output dir: %v", err)
	}
	if err := os.WriteFile(out, code, 0o644); err != nil {
		fatalf("writing output: %v", err)
	}
	logf("wrote generated file: %s", out)
}

// collectStructFields parses the current directory package and returns JSON field
// names for the struct type with the given name. If not found or not a struct,
// returns an empty slice.
func collectStructFields(typeName string) []string {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, ".", nil, parser.ParseComments)
	if err != nil || len(pkgs) == 0 {
		return nil
	}
	// pick any (current working package)
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name == nil || ts.Name.Name != typeName {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return nil
					}
					var names []string
					for _, field := range st.Fields.List {
						// skip anonymous fields without tags
						jsonName := ""
						if field.Tag != nil {
							tagLit := strings.Trim(field.Tag.Value, "`")
							tag := reflect.StructTag(tagLit)
							j := tag.Get("json")
							if j != "" {
								comma := strings.IndexByte(j, ',')
								if comma >= 0 {
									j = j[:comma]
								}
								if j == "-" {
									continue
								}
								jsonName = j
							}
						}
						if jsonName == "" {
							// use the first field name as fallback
							if len(field.Names) == 0 || field.Names[0] == nil {
								continue
							}
							jsonName = field.Names[0].Name
						}
						names = append(names, jsonName)
					}
					return names
				}
			}
		}
	}
	return nil
}

// collectStructFieldsAndRequired returns JSON field names and a heuristic list
// of required fields. Heuristic: fields without ",omitempty" in their json tag
// are treated as required; fields tagged with "-" are skipped. Untagged fields
// are considered required by default.
func collectStructFieldsAndRequired(typeName string) ([]string, []string) {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, ".", nil, parser.ParseComments)
	if err != nil || len(pkgs) == 0 {
		return nil, nil
	}
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name == nil || ts.Name.Name != typeName {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return nil, nil
					}
					var names []string
					var required []string
					for _, field := range st.Fields.List {
						// skip anonymous fields without tags
						jsonName := ""
						hasOmitEmpty := false
						if field.Tag != nil {
							tagLit := strings.Trim(field.Tag.Value, "`")
							tag := reflect.StructTag(tagLit)
							j := tag.Get("json")
							if j != "" {
								parts := strings.Split(j, ",")
								name := parts[0]
								if name == "-" {
									continue
								}
								jsonName = name
								for _, opt := range parts[1:] {
									if strings.TrimSpace(opt) == "omitempty" {
										hasOmitEmpty = true
										break
									}
								}
							}
						}
						if jsonName == "" {
							// use the first field name as fallback
							if len(field.Names) == 0 || field.Names[0] == nil {
								continue
							}
							jsonName = field.Names[0].Name
						}
						names = append(names, jsonName)
						if !hasOmitEmpty {
							required = append(required, jsonName)
						}
					}
					return names, required
				}
			}
		}
	}
	return nil, nil
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func detectPackageName() string {
	cmd := exec.Command("go", "list", "-f", "{{.Name}}")
	cmd.Env = os.Environ()
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func detectPackageNameFor(dir string) string {
	cmd := exec.Command("go", "list", "-f", "{{.Name}}", dir)
	cmd.Env = os.Environ()
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func detectImportPathFor(dir string) string {
	cmd := exec.Command("go", "list", "-f", "{{.ImportPath}}", dir)
	cmd.Env = os.Environ()
	cmd.Dir = "."
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func fatalf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", a...)
	os.Exit(1)
}

// collectJSONToGoBindings parses the given directory package and builds a mapping
// from JSON field name (from `json:"name"` tag or field name fallback) to Go struct field name.
func collectJSONToGoBindings(dir string, typeName string) map[string]string {
	fs := token.NewFileSet()
	pkgs, err := parser.ParseDir(fs, dir, nil, parser.ParseComments)
	if err != nil || len(pkgs) == 0 {
		return nil
	}
	bindings := make(map[string]string)
	for _, pkg := range pkgs {
		for _, f := range pkg.Files {
			for _, decl := range f.Decls {
				gd, ok := decl.(*ast.GenDecl)
				if !ok || gd.Tok != token.TYPE {
					continue
				}
				for _, spec := range gd.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok || ts.Name == nil || ts.Name.Name != typeName {
						continue
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok || st.Fields == nil {
						return bindings
					}
					for _, field := range st.Fields.List {
						if len(field.Names) == 0 || field.Names[0] == nil {
							continue
						}
						goName := field.Names[0].Name
						// Only map exported fields
						if goName == "" || (goName[0] < 'A' || goName[0] > 'Z') {
							continue
						}
						jsonName := ""
						if field.Tag != nil {
							tagLit := strings.Trim(field.Tag.Value, "`")
							tag := reflect.StructTag(tagLit)
							j := tag.Get("json")
							if j != "" {
								parts := strings.Split(j, ",")
								name := parts[0]
								if name == "-" {
									continue
								}
								jsonName = name
							}
						}
						if jsonName == "" {
							jsonName = goName
						}
						bindings[jsonName] = goName
					}
					return bindings
				}
			}
		}
	}
	return bindings
}
